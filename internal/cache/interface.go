package cache

import (
	"time"

	"github.com/quantmind-br/repodocs-go/internal/domain"
)

// Ensure BadgerCache implements domain.Cache
var _ domain.Cache = (*BadgerCache)(nil)

// Entry represents a cached entry with metadata
type Entry struct {
	URL         string    `json:"url"`
	Content     []byte    `json:"content"`
	ContentType string    `json:"content_type"`
	FetchedAt   time.Time `json:"fetched_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// IsExpired returns true if the entry has expired
func (e *Entry) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

// TTL returns the remaining time-to-live
func (e *Entry) TTL() time.Duration {
	remaining := time.Until(e.ExpiresAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Options contains cache configuration options
type Options struct {
	Directory string
	InMemory  bool
	Logger    bool
}

// DefaultOptions returns default cache options
func DefaultOptions() Options {
	return Options{
		Directory: "",
		InMemory:  false,
		Logger:    false,
	}
}
