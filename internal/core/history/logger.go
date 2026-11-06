package history

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xibhi/plumcake/internal/core/paths"
)

type Entry struct {
	Timestamp string `json:"timestamp"`
	Action    string `json:"action"`
	AppID     string `json:"app_id"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}

func FilePath(pathsInstance *paths.Paths) string {
	stateHome := pathsInstance.StateHome
	if stateHome == "" {
		home, _ := os.UserHomeDir()
		stateHome = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(stateHome, "plumcake", "operations.log")
}

func Record(pathsInstance *paths.Paths, action, appID, status, message string) {
	logFile := FilePath(pathsInstance)
	_ = os.MkdirAll(filepath.Dir(logFile), 0755)

	sanitizedMessage := message
	if strings.Contains(strings.ToLower(sanitizedMessage), "token=") {
		sanitizedMessage = "redacted sensitive parameter"
	}

	entryLine := fmt.Sprintf("[%s] action=%s app=%s status=%s msg=\"%s\"\n",
		time.Now().Format(time.RFC3339),
		action,
		appID,
		status,
		sanitizedMessage,
	)

	if fileInfo, err := os.Stat(logFile); err == nil && fileInfo.Size() > 1024*1024 {
		oldLog := logFile + ".1"
		_ = os.Rename(logFile, oldLog)
	}

	fileDescriptor, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		defer fileDescriptor.Close()
		_, _ = fileDescriptor.WriteString(entryLine)
	}
}

func ReadRecent(pathsInstance *paths.Paths, maxCount int) ([]Entry, error) {
	logFile := FilePath(pathsInstance)
	fileDescriptor, err := os.Open(logFile)
	if err != nil {
		oldLogFile := filepath.Join(pathsInstance.StateHome, "adopt", "operations.log")
		fileDescriptor, err = os.Open(oldLogFile)
		if err != nil {
			return nil, err
		}
	}
	defer fileDescriptor.Close()

	var entries []Entry
	scanner := bufio.NewScanner(fileDescriptor)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		entry := Entry{}
		if strings.HasPrefix(line, "[") && strings.Contains(line, "] ") {
			parts := strings.SplitN(line[1:], "] ", 2)
			entry.Timestamp = parts[0]
			tokens := strings.Split(parts[1], " ")

			for _, token := range tokens {
				if strings.HasPrefix(token, "action=") {
					entry.Action = strings.TrimPrefix(token, "action=")
				} else if strings.HasPrefix(token, "app=") {
					entry.AppID = strings.TrimPrefix(token, "app=")
				} else if strings.HasPrefix(token, "status=") {
					entry.Status = strings.TrimPrefix(token, "status=")
				} else if strings.HasPrefix(token, "msg=") {
					msgRaw := strings.TrimPrefix(token, "msg=")
					entry.Message = strings.Trim(msgRaw, "\"")
				}
			}
			entries = append(entries, entry)
		}
	}

	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	if maxCount > 0 && len(entries) > maxCount {
		entries = entries[:maxCount]
	}

	return entries, nil
}
