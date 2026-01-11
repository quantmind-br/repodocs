package state_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/state"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager_ValidOptions(t *testing.T) {
	tmpDir := t.TempDir()
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})

	manager := state.NewManager(state.ManagerOptions{
		BaseDir:   tmpDir,
		SourceURL: "https://example.com",
		Strategy:  "crawler",
		Logger:    logger,
		Disabled:  false,
	})

	assert.NotNil(t, manager)
	assert.False(t, manager.IsDisabled())
}

func TestNewManager_Disabled(t *testing.T) {
	tmpDir := t.TempDir()

	manager := state.NewManager(state.ManagerOptions{
		BaseDir:  tmpDir,
		Disabled: true,
	})

	assert.NotNil(t, manager)
	assert.True(t, manager.IsDisabled())
}

func TestNewManager_MinimalOptions(t *testing.T) {
	manager := state.NewManager(state.ManagerOptions{})

	assert.NotNil(t, manager)
	assert.False(t, manager.IsDisabled())
}

func TestManager_Load_StateNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	manager := state.NewManager(state.ManagerOptions{
		BaseDir: tmpDir,
	})

	err := manager.Load(context.Background())
	assert.ErrorIs(t, err, state.ErrStateNotFound)
}

func TestManager_Load_Disabled(t *testing.T) {
	tmpDir := t.TempDir()

	manager := state.NewManager(state.ManagerOptions{
		BaseDir:  tmpDir,
		Disabled: true,
	})

	err := manager.Load(context.Background())
	assert.NoError(t, err)
}

func TestManager_Load_ValidState(t *testing.T) {
	tmpDir := t.TempDir()

	stateData := state.SyncState{
		Version:   state.StateVersion,
		SourceURL: "https://example.com",
		Strategy:  "crawler",
		LastSync:  time.Now(),
		Pages: map[string]state.PageState{
			"https://example.com/page1": {
				ContentHash: "abc123",
				FetchedAt:   time.Now(),
				FilePath:    "page1.md",
			},
		},
	}

	data, err := json.MarshalIndent(stateData, "", "  ")
	require.NoError(t, err)

	statePath := filepath.Join(tmpDir, state.StateFileName)
	require.NoError(t, os.WriteFile(statePath, data, 0644))

	manager := state.NewManager(state.ManagerOptions{
		BaseDir: tmpDir,
	})

	err = manager.Load(context.Background())
	require.NoError(t, err)

	total, _ := manager.Stats()
	assert.Equal(t, 1, total)
}

func TestManager_Load_CorruptedState(t *testing.T) {
	tmpDir := t.TempDir()

	statePath := filepath.Join(tmpDir, state.StateFileName)
	require.NoError(t, os.WriteFile(statePath, []byte("invalid json{"), 0644))

	manager := state.NewManager(state.ManagerOptions{
		BaseDir: tmpDir,
	})

	err := manager.Load(context.Background())
	assert.ErrorIs(t, err, state.ErrStateCorrupted)
}

func TestManager_Load_VersionMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	logger := utils.NewLogger(utils.LoggerOptions{Level: "error"})

	stateData := map[string]interface{}{
		"version":    999,
		"source_url": "https://example.com",
		"last_sync":  time.Now(),
		"pages":      map[string]interface{}{},
	}

	data, err := json.MarshalIndent(stateData, "", "  ")
	require.NoError(t, err)

	statePath := filepath.Join(tmpDir, state.StateFileName)
	require.NoError(t, os.WriteFile(statePath, data, 0644))

	manager := state.NewManager(state.ManagerOptions{
		BaseDir: tmpDir,
		Logger:  logger,
	})

	err = manager.Load(context.Background())
	assert.ErrorIs(t, err, state.ErrVersionMismatch)
}

func TestManager_Save_Disabled(t *testing.T) {
	tmpDir := t.TempDir()

	manager := state.NewManager(state.ManagerOptions{
		BaseDir:  tmpDir,
		Disabled: true,
	})

	err := manager.Save(context.Background())
	assert.NoError(t, err)

	statePath := filepath.Join(tmpDir, state.StateFileName)
	_, err = os.Stat(statePath)
	assert.True(t, os.IsNotExist(err))
}

func TestManager_Save_NotDirty(t *testing.T) {
	tmpDir := t.TempDir()

	manager := state.NewManager(state.ManagerOptions{
		BaseDir: tmpDir,
	})

	err := manager.Save(context.Background())
	assert.NoError(t, err)

	statePath := filepath.Join(tmpDir, state.StateFileName)
	_, err = os.Stat(statePath)
	assert.True(t, os.IsNotExist(err))
}

func TestManager_Save_AfterUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	logger := utils.NewLogger(utils.LoggerOptions{Level: "debug"})

	manager := state.NewManager(state.ManagerOptions{
		BaseDir:   tmpDir,
		SourceURL: "https://example.com",
		Logger:    logger,
	})

	manager.Update("https://example.com/page1", state.PageState{
		ContentHash: "hash123",
		FetchedAt:   time.Now(),
		FilePath:    "page1.md",
	})

	err := manager.Save(context.Background())
	require.NoError(t, err)

	statePath := filepath.Join(tmpDir, state.StateFileName)
	data, err := os.ReadFile(statePath)
	require.NoError(t, err)

	var savedState state.SyncState
	require.NoError(t, json.Unmarshal(data, &savedState))

	assert.Equal(t, state.StateVersion, savedState.Version)
	assert.Equal(t, "https://example.com", savedState.SourceURL)
	assert.Len(t, savedState.Pages, 1)
}

func TestManager_Save_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "nested", "path")

	manager := state.NewManager(state.ManagerOptions{
		BaseDir: nestedDir,
	})

	manager.Update("https://example.com/page1", state.PageState{
		ContentHash: "hash123",
		FetchedAt:   time.Now(),
		FilePath:    "page1.md",
	})

	err := manager.Save(context.Background())
	require.NoError(t, err)

	statePath := filepath.Join(nestedDir, state.StateFileName)
	_, err = os.Stat(statePath)
	assert.NoError(t, err)
}

func TestManager_ShouldProcess_Disabled(t *testing.T) {
	manager := state.NewManager(state.ManagerOptions{
		Disabled: true,
	})

	result := manager.ShouldProcess("https://example.com", "hash")
	assert.True(t, result)
}

func TestManager_ShouldProcess_NewURL(t *testing.T) {
	tmpDir := t.TempDir()

	manager := state.NewManager(state.ManagerOptions{
		BaseDir: tmpDir,
	})

	result := manager.ShouldProcess("https://example.com/new", "hash123")
	assert.True(t, result)
}

func TestManager_ShouldProcess_ExistingURL_SameHash(t *testing.T) {
	tmpDir := t.TempDir()

	manager := state.NewManager(state.ManagerOptions{
		BaseDir: tmpDir,
	})

	manager.Update("https://example.com/page1", state.PageState{
		ContentHash: "hash123",
		FetchedAt:   time.Now(),
		FilePath:    "page1.md",
	})

	result := manager.ShouldProcess("https://example.com/page1", "hash123")
	assert.False(t, result)
}

func TestManager_ShouldProcess_ExistingURL_DifferentHash(t *testing.T) {
	tmpDir := t.TempDir()

	manager := state.NewManager(state.ManagerOptions{
		BaseDir: tmpDir,
	})

	manager.Update("https://example.com/page1", state.PageState{
		ContentHash: "hash123",
		FetchedAt:   time.Now(),
		FilePath:    "page1.md",
	})

	result := manager.ShouldProcess("https://example.com/page1", "newhash456")
	assert.True(t, result)
}

func TestManager_Update_Disabled(t *testing.T) {
	manager := state.NewManager(state.ManagerOptions{
		Disabled: true,
	})

	manager.Update("https://example.com/page1", state.PageState{
		ContentHash: "hash123",
	})

	total, _ := manager.Stats()
	assert.Equal(t, 0, total)
}

func TestManager_Update_AddsPage(t *testing.T) {
	tmpDir := t.TempDir()

	manager := state.NewManager(state.ManagerOptions{
		BaseDir: tmpDir,
	})

	manager.Update("https://example.com/page1", state.PageState{
		ContentHash: "hash123",
		FetchedAt:   time.Now(),
		FilePath:    "page1.md",
	})

	total, _ := manager.Stats()
	assert.Equal(t, 1, total)
}

func TestManager_Update_UpdatesExistingPage(t *testing.T) {
	tmpDir := t.TempDir()

	manager := state.NewManager(state.ManagerOptions{
		BaseDir: tmpDir,
	})

	manager.Update("https://example.com/page1", state.PageState{
		ContentHash: "hash123",
		FetchedAt:   time.Now(),
		FilePath:    "page1.md",
	})

	manager.Update("https://example.com/page1", state.PageState{
		ContentHash: "newhash456",
		FetchedAt:   time.Now(),
		FilePath:    "page1.md",
	})

	total, _ := manager.Stats()
	assert.Equal(t, 1, total)
}

func TestManager_MarkSeen(t *testing.T) {
	tmpDir := t.TempDir()

	manager := state.NewManager(state.ManagerOptions{
		BaseDir: tmpDir,
	})

	manager.Update("https://example.com/page1", state.PageState{
		ContentHash: "hash123",
	})
	manager.Update("https://example.com/page2", state.PageState{
		ContentHash: "hash456",
	})

	manager.MarkSeen("https://example.com/page1")

	deleted := manager.GetDeletedPages()
	assert.Len(t, deleted, 1)
}

func TestManager_GetDeletedPages_Disabled(t *testing.T) {
	manager := state.NewManager(state.ManagerOptions{
		Disabled: true,
	})

	deleted := manager.GetDeletedPages()
	assert.Nil(t, deleted)
}

func TestManager_GetDeletedPages_NoneDeleted(t *testing.T) {
	tmpDir := t.TempDir()

	manager := state.NewManager(state.ManagerOptions{
		BaseDir: tmpDir,
	})

	manager.Update("https://example.com/page1", state.PageState{
		ContentHash: "hash123",
	})

	manager.MarkSeen("https://example.com/page1")

	deleted := manager.GetDeletedPages()
	assert.Empty(t, deleted)
}

func TestManager_GetDeletedPages_SomeDeleted(t *testing.T) {
	tmpDir := t.TempDir()

	manager := state.NewManager(state.ManagerOptions{
		BaseDir: tmpDir,
	})

	manager.Update("https://example.com/page1", state.PageState{
		ContentHash: "hash123",
		FilePath:    "page1.md",
	})
	manager.Update("https://example.com/page2", state.PageState{
		ContentHash: "hash456",
		FilePath:    "page2.md",
	})
	manager.Update("https://example.com/page3", state.PageState{
		ContentHash: "hash789",
		FilePath:    "page3.md",
	})

	manager.MarkSeen("https://example.com/page1")

	deleted := manager.GetDeletedPages()
	assert.Len(t, deleted, 2)
}

func TestManager_RemoveDeletedFromState_Disabled(t *testing.T) {
	manager := state.NewManager(state.ManagerOptions{
		Disabled: true,
	})

	manager.RemoveDeletedFromState()
}

func TestManager_RemoveDeletedFromState_RemovesDeleted(t *testing.T) {
	tmpDir := t.TempDir()

	manager := state.NewManager(state.ManagerOptions{
		BaseDir: tmpDir,
	})

	manager.Update("https://example.com/page1", state.PageState{
		ContentHash: "hash123",
	})
	manager.Update("https://example.com/page2", state.PageState{
		ContentHash: "hash456",
	})

	manager.MarkSeen("https://example.com/page1")

	manager.RemoveDeletedFromState()

	total, _ := manager.Stats()
	assert.Equal(t, 1, total)
}

func TestManager_Stats_Empty(t *testing.T) {
	tmpDir := t.TempDir()

	manager := state.NewManager(state.ManagerOptions{
		BaseDir: tmpDir,
	})

	total, cached := manager.Stats()
	assert.Equal(t, 0, total)
	assert.Equal(t, 0, cached)
}

func TestManager_Stats_WithPages(t *testing.T) {
	tmpDir := t.TempDir()

	manager := state.NewManager(state.ManagerOptions{
		BaseDir: tmpDir,
	})

	manager.Update("https://example.com/page1", state.PageState{
		ContentHash: "hash123",
	})
	manager.Update("https://example.com/page2", state.PageState{
		ContentHash: "hash456",
	})

	total, cached := manager.Stats()
	assert.Equal(t, 2, total)
	assert.Equal(t, 2, cached)
}

func TestManager_Stats_WithSeenPages(t *testing.T) {
	tmpDir := t.TempDir()

	manager := state.NewManager(state.ManagerOptions{
		BaseDir: tmpDir,
	})

	manager.Update("https://example.com/page1", state.PageState{
		ContentHash: "hash123",
	})
	manager.Update("https://example.com/page2", state.PageState{
		ContentHash: "hash456",
	})

	manager.MarkSeen("https://example.com/page1")

	total, cached := manager.Stats()
	assert.Equal(t, 2, total)
	assert.Equal(t, 1, cached)
}

func TestManager_IsDisabled(t *testing.T) {
	tests := []struct {
		name     string
		disabled bool
	}{
		{"enabled", false},
		{"disabled", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			manager := state.NewManager(state.ManagerOptions{
				Disabled: tc.disabled,
			})
			assert.Equal(t, tc.disabled, manager.IsDisabled())
		})
	}
}

func TestManager_Concurrency_Update(t *testing.T) {
	tmpDir := t.TempDir()

	manager := state.NewManager(state.ManagerOptions{
		BaseDir: tmpDir,
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			manager.Update("https://example.com/page"+string(rune('0'+i%10)), state.PageState{
				ContentHash: "hash",
			})
		}(i)
	}
	wg.Wait()

	total, _ := manager.Stats()
	assert.LessOrEqual(t, total, 10)
}

func TestManager_Concurrency_ShouldProcess(t *testing.T) {
	tmpDir := t.TempDir()

	manager := state.NewManager(state.ManagerOptions{
		BaseDir: tmpDir,
	})

	manager.Update("https://example.com/page1", state.PageState{
		ContentHash: "hash123",
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			manager.ShouldProcess("https://example.com/page1", "hash123")
		}()
	}
	wg.Wait()
}

func TestManager_Concurrency_MarkSeen(t *testing.T) {
	tmpDir := t.TempDir()

	manager := state.NewManager(state.ManagerOptions{
		BaseDir: tmpDir,
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			manager.MarkSeen("https://example.com/page" + string(rune('0'+i%10)))
		}(i)
	}
	wg.Wait()
}

func TestManager_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()

	manager1 := state.NewManager(state.ManagerOptions{
		BaseDir:   tmpDir,
		SourceURL: "https://example.com",
		Strategy:  "crawler",
	})

	manager1.Update("https://example.com/page1", state.PageState{
		ContentHash: "hash123",
		FetchedAt:   time.Now(),
		FilePath:    "page1.md",
	})
	manager1.Update("https://example.com/page2", state.PageState{
		ContentHash: "hash456",
		FetchedAt:   time.Now(),
		FilePath:    "page2.md",
	})

	err := manager1.Save(context.Background())
	require.NoError(t, err)

	manager2 := state.NewManager(state.ManagerOptions{
		BaseDir: tmpDir,
	})

	err = manager2.Load(context.Background())
	require.NoError(t, err)

	total, _ := manager2.Stats()
	assert.Equal(t, 2, total)

	assert.False(t, manager2.ShouldProcess("https://example.com/page1", "hash123"))
	assert.False(t, manager2.ShouldProcess("https://example.com/page2", "hash456"))
	assert.True(t, manager2.ShouldProcess("https://example.com/page1", "different"))
	assert.True(t, manager2.ShouldProcess("https://example.com/page3", "newpage"))
}

func TestNewSyncState(t *testing.T) {
	syncState := state.NewSyncState("https://example.com", "crawler")

	assert.NotNil(t, syncState)
	assert.Equal(t, state.StateVersion, syncState.Version)
	assert.Equal(t, "https://example.com", syncState.SourceURL)
	assert.Equal(t, "crawler", syncState.Strategy)
	assert.NotNil(t, syncState.Pages)
	assert.Empty(t, syncState.Pages)
	assert.False(t, syncState.LastSync.IsZero())
}

func TestSyncState_PageCount(t *testing.T) {
	syncState := state.NewSyncState("https://example.com", "crawler")

	assert.Equal(t, 0, syncState.PageCount())

	syncState.SetPage("https://example.com/page1", state.PageState{ContentHash: "hash1"})
	assert.Equal(t, 1, syncState.PageCount())

	syncState.SetPage("https://example.com/page2", state.PageState{ContentHash: "hash2"})
	assert.Equal(t, 2, syncState.PageCount())
}

func TestSyncState_HasPage(t *testing.T) {
	syncState := state.NewSyncState("https://example.com", "crawler")

	assert.False(t, syncState.HasPage("https://example.com/page1"))

	syncState.SetPage("https://example.com/page1", state.PageState{ContentHash: "hash1"})
	assert.True(t, syncState.HasPage("https://example.com/page1"))
	assert.False(t, syncState.HasPage("https://example.com/page2"))
}

func TestSyncState_GetPage(t *testing.T) {
	syncState := state.NewSyncState("https://example.com", "crawler")

	page, exists := syncState.GetPage("https://example.com/page1")
	assert.False(t, exists)
	assert.Equal(t, state.PageState{}, page)

	syncState.SetPage("https://example.com/page1", state.PageState{
		ContentHash: "hash1",
		FilePath:    "page1.md",
	})

	page, exists = syncState.GetPage("https://example.com/page1")
	assert.True(t, exists)
	assert.Equal(t, "hash1", page.ContentHash)
	assert.Equal(t, "page1.md", page.FilePath)
}

func TestSyncState_SetPage(t *testing.T) {
	syncState := state.NewSyncState("https://example.com", "crawler")

	syncState.SetPage("https://example.com/page1", state.PageState{
		ContentHash: "hash1",
		FilePath:    "page1.md",
	})

	assert.True(t, syncState.HasPage("https://example.com/page1"))

	syncState.SetPage("https://example.com/page1", state.PageState{
		ContentHash: "newhash",
		FilePath:    "newpath.md",
	})

	page, _ := syncState.GetPage("https://example.com/page1")
	assert.Equal(t, "newhash", page.ContentHash)
	assert.Equal(t, "newpath.md", page.FilePath)
}

func TestSyncState_RemovePage(t *testing.T) {
	syncState := state.NewSyncState("https://example.com", "crawler")

	syncState.SetPage("https://example.com/page1", state.PageState{ContentHash: "hash1"})
	syncState.SetPage("https://example.com/page2", state.PageState{ContentHash: "hash2"})

	assert.Equal(t, 2, syncState.PageCount())

	syncState.RemovePage("https://example.com/page1")

	assert.Equal(t, 1, syncState.PageCount())
	assert.False(t, syncState.HasPage("https://example.com/page1"))
	assert.True(t, syncState.HasPage("https://example.com/page2"))
}

func TestSyncState_RemovePage_NonExistent(t *testing.T) {
	syncState := state.NewSyncState("https://example.com", "crawler")

	syncState.RemovePage("https://example.com/nonexistent")
	assert.Equal(t, 0, syncState.PageCount())
}

func TestPageState_Fields(t *testing.T) {
	now := time.Now()
	page := state.PageState{
		ContentHash: "hash123",
		FetchedAt:   now,
		FilePath:    "docs/page.md",
	}

	assert.Equal(t, "hash123", page.ContentHash)
	assert.Equal(t, now, page.FetchedAt)
	assert.Equal(t, "docs/page.md", page.FilePath)
}

func TestErrors(t *testing.T) {
	assert.True(t, errors.Is(state.ErrStateNotFound, state.ErrStateNotFound))
	assert.True(t, errors.Is(state.ErrStateCorrupted, state.ErrStateCorrupted))
	assert.True(t, errors.Is(state.ErrVersionMismatch, state.ErrVersionMismatch))

	assert.False(t, errors.Is(state.ErrStateNotFound, state.ErrStateCorrupted))
	assert.False(t, errors.Is(state.ErrStateCorrupted, state.ErrVersionMismatch))
}

func TestStateFileName(t *testing.T) {
	assert.Equal(t, ".repodocs-state.json", state.StateFileName)
}

func TestStateVersion(t *testing.T) {
	assert.Equal(t, 1, state.StateVersion)
}

func BenchmarkManager_Update(b *testing.B) {
	tmpDir := b.TempDir()

	manager := state.NewManager(state.ManagerOptions{
		BaseDir: tmpDir,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.Update("https://example.com/page", state.PageState{
			ContentHash: "hash",
		})
	}
}

func BenchmarkManager_ShouldProcess(b *testing.B) {
	tmpDir := b.TempDir()

	manager := state.NewManager(state.ManagerOptions{
		BaseDir: tmpDir,
	})

	manager.Update("https://example.com/page", state.PageState{
		ContentHash: "hash123",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.ShouldProcess("https://example.com/page", "hash123")
	}
}

func BenchmarkManager_MarkSeen(b *testing.B) {
	tmpDir := b.TempDir()

	manager := state.NewManager(state.ManagerOptions{
		BaseDir: tmpDir,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.MarkSeen("https://example.com/page")
	}
}

func BenchmarkManager_Save(b *testing.B) {
	tmpDir := b.TempDir()

	manager := state.NewManager(state.ManagerOptions{
		BaseDir: tmpDir,
	})

	for i := 0; i < 100; i++ {
		manager.Update("https://example.com/page"+string(rune('0'+i)), state.PageState{
			ContentHash: "hash",
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.Update("https://example.com/page0", state.PageState{
			ContentHash: "newhash",
		})
		manager.Save(context.Background())
	}
}

func BenchmarkManager_Load(b *testing.B) {
	tmpDir := b.TempDir()

	manager := state.NewManager(state.ManagerOptions{
		BaseDir:   tmpDir,
		SourceURL: "https://example.com",
	})

	for i := 0; i < 100; i++ {
		manager.Update("https://example.com/page"+string(rune('0'+i)), state.PageState{
			ContentHash: "hash",
		})
	}
	manager.Save(context.Background())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		newManager := state.NewManager(state.ManagerOptions{
			BaseDir: tmpDir,
		})
		newManager.Load(context.Background())
	}
}
