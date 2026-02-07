package api

import (
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"
)

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}

// LogBuffer is a thread-safe ring buffer for log entries
type LogBuffer struct {
	mu      sync.RWMutex
	entries []LogEntry
	cap     int
}

// NewLogBuffer creates a new log buffer with the given capacity
func NewLogBuffer(capacity int) *LogBuffer {
	return &LogBuffer{
		entries: make([]LogEntry, 0, capacity),
		cap:     capacity,
	}
}

// Add adds a log entry to the buffer
func (lb *LogBuffer) Add(level, message string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
	}

	if len(lb.entries) >= lb.cap {
		// Shift everything left by 1, drop oldest
		copy(lb.entries, lb.entries[1:])
		lb.entries[len(lb.entries)-1] = entry
	} else {
		lb.entries = append(lb.entries, entry)
	}
}

// Entries returns all entries, optionally filtered by level
func (lb *LogBuffer) Entries(levels []string) []LogEntry {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if len(levels) == 0 {
		result := make([]LogEntry, len(lb.entries))
		copy(result, lb.entries)
		return result
	}

	levelSet := make(map[string]bool)
	for _, l := range levels {
		levelSet[strings.ToLower(l)] = true
	}

	result := make([]LogEntry, 0)
	for _, e := range lb.entries {
		if levelSet[strings.ToLower(e.Level)] {
			result = append(result, e)
		}
	}
	return result
}

// Clear removes all entries
func (lb *LogBuffer) Clear() {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.entries = lb.entries[:0]
}

// logWriter adapts LogBuffer to io.Writer for use with Go's log package
type logWriter struct {
	buf *LogBuffer
}

func (lw *logWriter) Write(p []byte) (n int, err error) {
	msg := strings.TrimSpace(string(p))
	if msg == "" {
		return len(p), nil
	}

	// Parse log level from message
	level := "info"
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "error") || strings.Contains(lower, "fail") {
		level = "error"
	} else if strings.Contains(lower, "warn") {
		level = "warn"
	}

	// Strip standard log prefix (date/time) if present
	// Go's log package prefixes with "2006/01/02 15:04:05 "
	if len(msg) > 20 && msg[4] == '/' && msg[7] == '/' && msg[10] == ' ' {
		msg = msg[20:]
	}

	lw.buf.Add(level, msg)
	return len(p), nil
}

// InstallLogCapture sets up Go's log package to write to the LogBuffer
// and also to stderr. Returns the multi-writer for additional use.
func InstallLogCapture(buf *LogBuffer) io.Writer {
	lw := &logWriter{buf: buf}
	multi := io.MultiWriter(lw, log.Writer())
	log.SetOutput(multi)
	log.SetFlags(log.LstdFlags)
	return multi
}

// LogInfo logs an info message
func (lb *LogBuffer) LogInfo(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	lb.Add("info", msg)
	log.Println(msg)
}

// LogWarn logs a warning message
func (lb *LogBuffer) LogWarn(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	lb.Add("warn", msg)
	log.Printf("WARN: %s", msg)
}

// LogError logs an error message
func (lb *LogBuffer) LogError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	lb.Add("error", msg)
	log.Printf("ERROR: %s", msg)
}
