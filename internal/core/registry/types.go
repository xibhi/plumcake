package registry

import (
	"fmt"
	"github.com/xibhi/plumcake/internal/core/trust"
)

type SandboxConfig struct {
	Enabled      bool   `json:"enabled"`
	Tool         string `json:"tool"`
	Network      bool   `json:"network"`
	OriginalExec string `json:"original_exec"`
}

type VersionRecord struct {
	Version     string      `json:"version"`
	Path        string      `json:"path"`
	InstalledAt string      `json:"installed_at"`
	Size        int64       `json:"size"`
	SHA256      string      `json:"sha256"`
	Trust       trust.State `json:"trust"`
}

type UpdateConfig struct {
	SourceType    string `json:"source_type"`
	SourceURL     string `json:"source_url"`
	LatestVersion string `json:"latest_version"`
	HasUpdate     bool   `json:"has_update"`
	CheckedAt     string `json:"checked_at"`
}

type App struct {
	SchemaVersion    int             `json:"schema_version"`
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Version          string          `json:"version"`
	ExecName         string          `json:"exec_name"`
	WMClass          string          `json:"wm_class"`
	Comment          string          `json:"comment"`
	Categories       []string        `json:"categories"`
	InstalledAt      string          `json:"installed_at"`
	UpdatedAt        string          `json:"updated_at"`
	AppImagePath     string          `json:"appimage_path"`
	OriginalFilename string          `json:"original_filename"`
	BinarySHA256     string          `json:"binary_sha256"`
	BinarySize       int64           `json:"binary_size"`
	DesktopFile      string          `json:"desktop_file"`
	IconFiles        []string        `json:"icon_files"`
	WrapperFile      string          `json:"wrapper_file"`
	Portable         bool            `json:"portable"`
	TraceIdentifiers []string        `json:"trace_identifiers"`
	Terminal         bool            `json:"terminal"`
	MimeTypes        []string        `json:"mime_types"`
	SourceURL        *string         `json:"source_url,omitempty"`
	Trust            trust.State     `json:"trust"`
	Checksum         string          `json:"checksum"`
	Signature        string          `json:"signature"`
	Sandbox          SandboxConfig   `json:"sandbox"`
	UpdateInfo       UpdateConfig    `json:"update_info"`
	VersionHistory   []VersionRecord `json:"version_history"`
}

func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	divider, exponent := int64(unit), 0
	for value := bytes / unit; value >= unit; value /= unit {
		divider *= unit
		exponent++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(divider), "KMGTPE"[exponent])
}

func Truncate(str string, maxLen int) string {
	if len(str) <= maxLen {
		return str
	}
	if maxLen <= 3 {
		return str[:maxLen]
	}
	return str[:maxLen-2] + "…"
}
