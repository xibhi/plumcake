package add

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/xibhi/plumcake/internal/core/appimage"
	"github.com/xibhi/plumcake/internal/core/config"
	"github.com/xibhi/plumcake/internal/core/installer"
	"github.com/xibhi/plumcake/internal/core/paths"
	"github.com/xibhi/plumcake/internal/core/registry"
	"github.com/xibhi/plumcake/internal/core/scanner"
	"github.com/xibhi/plumcake/internal/core/updater"
)

type Mode int

const (
	ModeBrowsing Mode = iota
	ModeReview
	ModeInstalling
	ModeCompleted
	ModeFailed
)

type Model struct {
	Paths          *paths.Paths
	Config         config.Config
	Input          textinput.Model
	Scanned        []scanner.Result
	Filtered       []scanner.Result
	Cursor         int
	IsScanning     bool
	DeepScanning   bool
	ScanSeen       map[string]bool
	Mode           Mode
	ReviewItem     *installer.PreInstallInspection
	MoveFileChoice bool
	ExpectedDigest string
	TargetSource   string
	ActiveStep     installer.StepID
	ActiveStepIdx  int
	StepProgress   float64
	StepDetail     string
	InstalledApp   *registry.App
	ErrorMessage   string
	ProgressBar    progress.Model
	PipelineCancel context.CancelFunc
	PipelineEvents chan installer.Event
	Width          int
	Height         int
}

type InitialScanCompleteMsg struct {
	Items []scanner.Result
}

type DeepScanCompleteMsg struct {
	Items []scanner.Result
}

type GitHubAssetsReadyMsg struct {
	Assets []updater.MatchedAsset
}

type GitHubFetchErrorMsg struct {
	Err error
}

type PipelineStepMsg installer.Event

func New(pathsInstance *paths.Paths, cfg config.Config) Model {
	inputComponent := textinput.New()
	inputComponent.Placeholder = "Paste link, path to .AppImage, or github.com/owner/repo..."
	inputComponent.CharLimit = 512
	inputComponent.Width = 64
	inputComponent.Blur()

	progressBar := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(48),
	)

	return Model{
		Paths:          pathsInstance,
		Config:         cfg,
		Input:          inputComponent,
		ScanSeen:       make(map[string]bool),
		ProgressBar:    progressBar,
		Mode:           ModeBrowsing,
		MoveFileChoice: cfg.DefaultMoveFile,
	}
}

func (m *Model) StartInitialScan() tea.Cmd {
	m.IsScanning = true
	return func() tea.Msg {
		apps, _ := registry.LoadAll(m.Paths)
		adoptedMap := make(map[string]*registry.App)
		for i := range apps {
			adoptedMap[apps[i].AppImagePath] = &apps[i]
			adoptedMap[apps[i].OriginalFilename] = &apps[i]
		}

		resultsChannel := make(chan scanner.Result, 32)
		ctx := context.Background()
		go scanner.ScanInitial(ctx, m.Paths, adoptedMap, resultsChannel)

		var items []scanner.Result
		for item := range resultsChannel {
			items = append(items, item)
		}
		return InitialScanCompleteMsg{Items: items}
	}
}

func (m *Model) StartDeepScan() tea.Cmd {
	m.DeepScanning = true
	return func() tea.Msg {
		apps, _ := registry.LoadAll(m.Paths)
		adoptedMap := make(map[string]*registry.App)
		for i := range apps {
			adoptedMap[apps[i].AppImagePath] = &apps[i]
			adoptedMap[apps[i].OriginalFilename] = &apps[i]
		}

		resultsChannel := make(chan scanner.Result, 32)
		ctx := context.Background()
		go scanner.ScanDeep(ctx, m.Paths, adoptedMap, m.ScanSeen, resultsChannel)

		var items []scanner.Result
		for item := range resultsChannel {
			items = append(items, item)
		}
		return DeepScanCompleteMsg{Items: items}
	}
}

func (m *Model) ApplyFilter() {
	query := strings.TrimSpace(m.Input.Value())
	if query == "" || strings.HasPrefix(query, "http://") || strings.HasPrefix(query, "https://") || strings.Contains(query, "/") {
		m.Filtered = m.Scanned
	} else {
		var matched []scanner.Result
		for _, img := range m.Scanned {
			if scanner.FuzzyMatch(query, img.Name) || scanner.FuzzyMatch(query, img.Folder) {
				matched = append(matched, img)
			}
		}
		m.Filtered = matched
	}
	if m.Cursor >= len(m.Filtered) {
		m.Cursor = max(0, len(m.Filtered)-1)
	}
}

func (m *Model) StartReviewOrInstall(source string) tea.Cmd {
	m.TargetSource = source

	if owner, repo, isRepo := updater.ParseGitHubRepoInput(source); isRepo {
		return func() tea.Msg {
			ctx := context.Background()
			assets, err := updater.FetchGitHubReleaseAssets(ctx, owner, repo, m.Config.IncludePrereleases)
			if err != nil {
				return GitHubFetchErrorMsg{Err: err}
			}
			return GitHubAssetsReadyMsg{Assets: assets}
		}
	}

	if m.Config.PreinstallInspect && appimage.IsValid(source) {
		ctx := context.Background()
		review, err := installer.InspectPreInstall(ctx, m.Paths, source, "", m.ExpectedDigest)
		if err == nil {
			m.ReviewItem = review
			m.Mode = ModeReview
			return nil
		}
	}

	return m.StartPipeline(source)
}

func (m *Model) StartPipeline(source string) tea.Cmd {
	m.Mode = ModeInstalling
	m.ActiveStep = installer.StepFetch
	m.ActiveStepIdx = 1
	m.StepProgress = 0.05
	m.StepDetail = "Initiating pipeline..."
	m.ProgressBar.SetPercent(0.05)

	ctx, cancel := context.WithCancel(context.Background())
	m.PipelineCancel = cancel
	m.PipelineEvents = make(chan installer.Event, 16)

	pipelineRunner := installer.NewRunner(m.Paths)
	go pipelineRunner.Run(ctx, source, false, m.MoveFileChoice, m.ExpectedDigest, m.PipelineEvents)

	return m.listenPipeline()
}

func (m *Model) listenPipeline() tea.Cmd {
	return func() tea.Msg {
		event, ok := <-m.PipelineEvents
		if !ok {
			return nil
		}
		return PipelineStepMsg(event)
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var commands []tea.Cmd

	switch event := msg.(type) {
	case InitialScanCompleteMsg:
		m.IsScanning = false
		for _, item := range event.Items {
			if !m.ScanSeen[item.Path] {
				m.ScanSeen[item.Path] = true
				m.Scanned = append(m.Scanned, item)
			}
		}
		m.ApplyFilter()

	case DeepScanCompleteMsg:
		m.DeepScanning = false
		for _, item := range event.Items {
			if !m.ScanSeen[item.Path] {
				m.ScanSeen[item.Path] = true
				m.Scanned = append(m.Scanned, item)
			}
		}
		m.ApplyFilter()

	case GitHubAssetsReadyMsg:
		if len(event.Assets) > 0 {
			chosenAsset := event.Assets[0]
			m.ExpectedDigest = chosenAsset.DigestURL
			return m, m.StartPipeline(chosenAsset.DownloadURL)
		}

	case GitHubFetchErrorMsg:
		m.Mode = ModeFailed
		m.ErrorMessage = event.Err.Error()
		return m, nil

	case PipelineStepMsg:
		pipelineEvent := installer.Event(event)
		if pipelineEvent.Err != nil {
			m.Mode = ModeFailed
			m.ErrorMessage = pipelineEvent.Err.Error()
			return m, nil
		}

		m.ActiveStep = pipelineEvent.Step
		m.ActiveStepIdx = pipelineEvent.StepIndex
		m.StepProgress = pipelineEvent.Progress
		m.StepDetail = pipelineEvent.Detail
		m.ProgressBar.SetPercent(pipelineEvent.Progress)

		if pipelineEvent.Done {
			m.Mode = ModeCompleted
			m.InstalledApp = pipelineEvent.Manifest
			m.ProgressBar.SetPercent(1.0)
			return m, nil
		}

		return m, m.listenPipeline()

	case progress.FrameMsg:
		progressModel, cmd := m.ProgressBar.Update(msg)
		m.ProgressBar = progressModel.(progress.Model)
		return m, cmd

	case tea.KeyMsg:
		switch m.Mode {
		case ModeBrowsing:
			switch event.String() {
			case "enter":
				val := strings.TrimSpace(m.Input.Value())
				if val != "" {
					return m, m.StartReviewOrInstall(val)
				}
				if len(m.Filtered) > 0 && m.Cursor < len(m.Filtered) {
					return m, m.StartReviewOrInstall(m.Filtered[m.Cursor].Path)
				}
			case "up", "k":
				if m.Cursor > 0 {
					m.Cursor--
				}
			case "down", "j":
				if m.Cursor < len(m.Filtered)-1 {
					m.Cursor++
				}
			case "tab":
				if m.Input.Focused() {
					m.Input.Blur()
				} else {
					m.Input.Focus()
				}
			case "/":
				m.Input.Focus()
				return m, textinput.Blink
			case "s", "F5":
				if !m.Input.Focused() {
					return m, m.StartDeepScan()
				}
			case "esc":
				if m.Input.Focused() {
					m.Input.Blur()
				} else {
					m.Input.Reset()
					m.ApplyFilter()
				}
			default:
				if m.Input.Focused() {
					var cmd tea.Cmd
					m.Input, cmd = m.Input.Update(msg)
					m.ApplyFilter()
					return m, cmd
				}
			}

		case ModeReview:
			switch event.String() {
			case "enter":
				return m, m.StartPipeline(m.TargetSource)
			case "m":
				m.MoveFileChoice = !m.MoveFileChoice
			case "esc":
				m.Mode = ModeBrowsing
				m.ReviewItem = nil
			}

		case ModeCompleted:
			switch event.String() {
			case "l", "enter":
				if m.InstalledApp != nil {
					_ = registry.Launch(m.InstalledApp)
				}
			case "esc", "a":
				m.Mode = ModeBrowsing
				m.Input.Reset()
				m.ApplyFilter()
			}

		case ModeFailed:
			switch event.String() {
			case "esc", "r":
				m.Mode = ModeBrowsing
			}
		}
	}

	return m, tea.Batch(commands...)
}
