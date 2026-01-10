package state

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/utils"
)

const StateFileName = ".repodocs-state.json"

type Manager struct {
	baseDir  string
	state    *SyncState
	mu       sync.RWMutex
	dirty    bool
	logger   *utils.Logger
	disabled bool
	seenURLs sync.Map
}

type ManagerOptions struct {
	BaseDir   string
	SourceURL string
	Strategy  string
	Logger    *utils.Logger
	Disabled  bool
}

func NewManager(opts ManagerOptions) *Manager {
	return &Manager{
		baseDir:  opts.BaseDir,
		logger:   opts.Logger,
		disabled: opts.Disabled,
		state:    NewSyncState(opts.SourceURL, opts.Strategy),
	}
}

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

func (m *Manager) Update(url string, page PageState) {
	if m.disabled {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.state.Pages[url] = page
	m.dirty = true
}

func (m *Manager) MarkSeen(url string) {
	m.seenURLs.Store(url, true)
}

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

func (m *Manager) IsDisabled() bool {
	return m.disabled
}

func (m *Manager) statePath() string {
	return filepath.Join(m.baseDir, StateFileName)
}
