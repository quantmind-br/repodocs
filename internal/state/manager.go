package state

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/quantmind-br/repodocs/internal/utils"
)

// StateFileName is the filename used to persist incremental sync state.
const StateFileName = ".repodocs-state.json"

// Manager tracks incremental sync state for processed pages and deleted-page detection.
type Manager struct {
	baseDir  string
	state    *SyncState
	mu       sync.RWMutex
	dirty    bool
	logger   *utils.Logger
	disabled bool
	seenURLs sync.Map
}

// ManagerOptions configures sync-state storage, source identity, logging, and disabled mode.
type ManagerOptions struct {
	BaseDir   string
	SourceURL string
	Strategy  string
	Logger    *utils.Logger
	Disabled  bool
}

// NewManager creates a sync-state manager initialized for the configured source.
func NewManager(opts ManagerOptions) *Manager {
	return &Manager{
		baseDir:  opts.BaseDir,
		logger:   opts.Logger,
		disabled: opts.Disabled,
		state:    NewSyncState(opts.SourceURL, opts.Strategy),
	}
}

// Load reads sync state from disk unless the manager is disabled.
func (m *Manager) Load(ctx context.Context) error {
	if m.disabled {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	path := m.statePath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return ErrStateNotFound
	}
	if err != nil {
		return err
	}

	var state SyncState
	if err := json.Unmarshal(data, &state); err != nil {
		return ErrStateCorrupted
	}

	if state.Version != StateVersion {
		if m.logger != nil {
			m.logger.Warn().
				Int("file_version", state.Version).
				Int("expected_version", StateVersion).
				Msg("State version mismatch, will rebuild state")
		}
		return ErrVersionMismatch
	}

	m.state = &state
	return nil
}

// Save writes dirty sync state to disk unless the manager is disabled.
func (m *Manager) Save(ctx context.Context) error {
	if m.disabled {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.dirty {
		return nil
	}

	m.state.LastSync = time.Now()

	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return err
	}

	path := m.statePath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}

	m.dirty = false
	if m.logger != nil {
		m.logger.Debug().
			Int("pages", len(m.state.Pages)).
			Str("path", path).
			Msg("State saved")
	}
	return nil
}

// ShouldProcess reports whether url is new or its stored contentHash differs.
func (m *Manager) ShouldProcess(url, contentHash string) bool {
	if m.disabled {
		return true
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	page, exists := m.state.Pages[url]
	if !exists {
		return true
	}

	return page.ContentHash != contentHash
}

// Update stores page state for url and marks the manager dirty.
func (m *Manager) Update(url string, page PageState) {
	if m.disabled {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.state.Pages[url] = page
	m.dirty = true
}

// MarkSeen records that url was observed during the current sync run.
func (m *Manager) MarkSeen(url string) {
	m.seenURLs.Store(url, true)
}

// GetDeletedPages returns previously known pages not seen during the current sync run.
func (m *Manager) GetDeletedPages() []PageState {
	if m.disabled {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var deleted []PageState
	for url, page := range m.state.Pages {
		if _, seen := m.seenURLs.Load(url); !seen {
			deleted = append(deleted, page)
		}
	}
	return deleted
}

// RemoveDeletedFromState removes unseen pages from the persisted state and marks it dirty.
func (m *Manager) RemoveDeletedFromState() {
	if m.disabled {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for url := range m.state.Pages {
		if _, seen := m.seenURLs.Load(url); !seen {
			delete(m.state.Pages, url)
			m.dirty = true
		}
	}
}

// Stats returns total tracked pages and pages not seen during the current sync run.
func (m *Manager) Stats() (total, cached int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total = len(m.state.Pages)

	var seenCount int
	m.seenURLs.Range(func(_, _ any) bool {
		seenCount++
		return true
	})

	return total, total - seenCount
}

// IsDisabled reports whether sync-state persistence is disabled.
func (m *Manager) IsDisabled() bool {
	return m.disabled
}

func (m *Manager) statePath() string {
	return filepath.Join(m.baseDir, StateFileName)
}
