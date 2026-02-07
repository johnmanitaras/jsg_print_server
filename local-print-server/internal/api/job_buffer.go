package api

import (
	"sync"
	"time"
)

// JobRecord represents a completed print job
type JobRecord struct {
	ID          string     `json:"id"`
	PrinterID   string     `json:"printer_id"`
	PrinterName string     `json:"printer_name"`
	Status      string     `json:"status"` // completed, failed, printing, pending
	DataSize    int        `json:"data_size"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Error       string     `json:"error,omitempty"`
}

// JobBuffer is a thread-safe ring buffer for job records
type JobBuffer struct {
	mu      sync.RWMutex
	entries []JobRecord
	cap     int
}

// NewJobBuffer creates a new job buffer with the given capacity
func NewJobBuffer(capacity int) *JobBuffer {
	return &JobBuffer{
		entries: make([]JobRecord, 0, capacity),
		cap:     capacity,
	}
}

// Add adds a job record to the buffer
func (jb *JobBuffer) Add(job JobRecord) {
	jb.mu.Lock()
	defer jb.mu.Unlock()

	if len(jb.entries) >= jb.cap {
		copy(jb.entries, jb.entries[1:])
		jb.entries[len(jb.entries)-1] = job
	} else {
		jb.entries = append(jb.entries, job)
	}
}

// Entries returns all job records (newest first)
func (jb *JobBuffer) Entries() []JobRecord {
	jb.mu.RLock()
	defer jb.mu.RUnlock()

	result := make([]JobRecord, len(jb.entries))
	// Reverse order so newest is first
	for i, j := 0, len(jb.entries)-1; j >= 0; i, j = i+1, j-1 {
		result[i] = jb.entries[j]
	}
	return result
}

// UpdateStatus updates the status of a job by ID
func (jb *JobBuffer) UpdateStatus(jobID, status, errMsg string) {
	jb.mu.Lock()
	defer jb.mu.Unlock()

	for i := len(jb.entries) - 1; i >= 0; i-- {
		if jb.entries[i].ID == jobID {
			jb.entries[i].Status = status
			if errMsg != "" {
				jb.entries[i].Error = errMsg
			}
			if status == "completed" || status == "failed" {
				now := time.Now()
				jb.entries[i].CompletedAt = &now
			}
			return
		}
	}
}
