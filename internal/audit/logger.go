package audit

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/takakrypt/transparent-encryption/internal/filesystem"
)

type Logger struct {
	logFile *os.File
	enabled bool
}

type AuditLog struct {
	Timestamp  time.Time `json:"timestamp"`
	Operation  string    `json:"operation"`
	Path       string    `json:"path"`
	User       int       `json:"user"`
	Process    string    `json:"process"`
	Permission string    `json:"permission"`
	RuleID     string    `json:"rule_id"`
	Success    bool      `json:"success"`
	Message    string    `json:"message,omitempty"`
}

func NewLogger(logPath string, enabled bool) (*Logger, error) {
	if !enabled {
		return &Logger{enabled: false}, nil
	}

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log file: %w", err)
	}

	return &Logger{
		logFile: file,
		enabled: true,
	}, nil
}

func (l *Logger) LogEvent(event *filesystem.AuditEvent, message string) {
	if !l.enabled {
		return
	}

	auditLog := &AuditLog{
		Timestamp:  time.Unix(event.Timestamp, 0),
		Operation:  event.Operation,
		Path:       event.Path,
		User:       event.User,
		Process:    event.Process,
		Permission: event.Permission,
		RuleID:     event.RuleID,
		Success:    event.Success,
		Message:    message,
	}

	data, err := json.Marshal(auditLog)
	if err != nil {
		log.Printf("Failed to marshal audit log: %v", err)
		return
	}

	if l.logFile != nil {
		fmt.Fprintf(l.logFile, "%s\n", data)
		l.logFile.Sync()
	}

	if !event.Success {
		log.Printf("AUDIT: %s", string(data))
	}
}

func (l *Logger) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}