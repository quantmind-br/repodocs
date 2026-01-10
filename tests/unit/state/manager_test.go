package state_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantmind-br/repodocs-go/internal/state"
)

func TestNewManager(t *testing.T) {
	mgr := state.NewManager(state.ManagerOptions{
		BaseDir:   t.TempDir(),
		SourceURL: "https://example.com",
		Strategy:  "crawler",
	})

	assert.NotNil(t, mgr)
	assert.False(t, mgr.IsDisabled())
}

func TestManager_Disabled(t *testing.T) {
	mgr := state.NewManager(state.ManagerOptions{
		Disabled: true,
	})

	assert.True(t, mgr.IsDisabled())
	assert.True(t, mgr.ShouldProcess("any-url", "any-hash"))
	assert.NoError(t, mgr.Load(context.Background()))
	assert.NoError(t, mgr.Save(context.Background()))
}

func TestManager_LoadNotFound(t *testing.T) {
	dir := t.TempDir()
	mgr := state.NewManager(state.ManagerOptions{
		BaseDir: dir,
	})

	err := mgr.Load(context.Background())
	assert.ErrorIs(t, err, state.ErrStateNotFound)
}

func TestManager_LoadCorrupted(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, state.StateFileName)

	err := os.WriteFile(statePath, []byte("not valid json{{{"), 0644)
	require.NoError(t, err)

	mgr := state.NewManager(state.ManagerOptions{
		BaseDir: dir,
	})

	err = mgr.Load(context.Background())
	assert.ErrorIs(t, err, state.ErrStateCorrupted)
}

func TestManager_LoadVersionMismatch(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, state.StateFileName)

	err := os.WriteFile(statePath, []byte(`{"version": 999, "source_url": "test", "pages": {}}`), 0644)
	require.NoError(t, err)

	mgr := state.NewManager(state.ManagerOptions{
		BaseDir: dir,
	})

	err = mgr.Load(context.Background())
	assert.ErrorIs(t, err, state.ErrVersionMismatch)
}

func TestManager_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	mgr := state.NewManager(state.ManagerOptions{
		BaseDir:   dir,
		SourceURL: "https://example.com",
		Strategy:  "crawler",
	})

	mgr.Update("https://example.com/page1", state.PageState{
		ContentHash: "abc123",
		FetchedAt:   time.Now(),
		FilePath:    "page1.md",
	})
	mgr.Update("https://example.com/page2", state.PageState{
		ContentHash: "def456",
		FetchedAt:   time.Now(),
		FilePath:    "page2.md",
	})

	err := mgr.Save(ctx)
	require.NoError(t, err)

	statePath := filepath.Join(dir, state.StateFileName)
	_, err = os.Stat(statePath)
	require.NoError(t, err)

	mgr2 := state.NewManager(state.ManagerOptions{
		BaseDir: dir,
	})
	err = mgr2.Load(ctx)
	require.NoError(t, err)

	assert.False(t, mgr2.ShouldProcess("https://example.com/page1", "abc123"))
	assert.False(t, mgr2.ShouldProcess("https://example.com/page2", "def456"))
	assert.True(t, mgr2.ShouldProcess("https://example.com/page3", "any"))
}

func TestManager_ShouldProcess_NewPage(t *testing.T) {
	mgr := state.NewManager(state.ManagerOptions{
		BaseDir: t.TempDir(),
	})

	assert.True(t, mgr.ShouldProcess("https://example.com/new", "hash123"))
}

func TestManager_ShouldProcess_UnchangedPage(t *testing.T) {
	mgr := state.NewManager(state.ManagerOptions{
		BaseDir: t.TempDir(),
	})

	mgr.Update("https://example.com/page", state.PageState{
		ContentHash: "hash123",
		FetchedAt:   time.Now(),
		FilePath:    "page.md",
	})

	assert.False(t, mgr.ShouldProcess("https://example.com/page", "hash123"))
}

func TestManager_ShouldProcess_ChangedPage(t *testing.T) {
	mgr := state.NewManager(state.ManagerOptions{
		BaseDir: t.TempDir(),
	})

	mgr.Update("https://example.com/page", state.PageState{
		ContentHash: "old-hash",
		FetchedAt:   time.Now(),
		FilePath:    "page.md",
	})

	assert.True(t, mgr.ShouldProcess("https://example.com/page", "new-hash"))
}

func TestManager_MarkSeen_GetDeletedPages(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	mgr := state.NewManager(state.ManagerOptions{
		BaseDir: dir,
	})

	mgr.Update("https://example.com/page1", state.PageState{
		ContentHash: "hash1",
		FetchedAt:   time.Now(),
		FilePath:    "page1.md",
	})
	mgr.Update("https://example.com/page2", state.PageState{
		ContentHash: "hash2",
		FetchedAt:   time.Now(),
		FilePath:    "page2.md",
	})
	mgr.Update("https://example.com/page3", state.PageState{
		ContentHash: "hash3",
		FetchedAt:   time.Now(),
		FilePath:    "page3.md",
	})

	require.NoError(t, mgr.Save(ctx))

	mgr2 := state.NewManager(state.ManagerOptions{
		BaseDir: dir,
	})
	require.NoError(t, mgr2.Load(ctx))

	mgr2.MarkSeen("https://example.com/page1")
	mgr2.MarkSeen("https://example.com/page3")

	deleted := mgr2.GetDeletedPages()
	require.Len(t, deleted, 1)
	assert.Equal(t, "page2.md", deleted[0].FilePath)
}

func TestManager_RemoveDeletedFromState(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	mgr := state.NewManager(state.ManagerOptions{
		BaseDir: dir,
	})

	mgr.Update("https://example.com/keep", state.PageState{
		ContentHash: "hash1",
		FetchedAt:   time.Now(),
		FilePath:    "keep.md",
	})
	mgr.Update("https://example.com/delete", state.PageState{
		ContentHash: "hash2",
		FetchedAt:   time.Now(),
		FilePath:    "delete.md",
	})

	mgr.MarkSeen("https://example.com/keep")

	deleted := mgr.GetDeletedPages()
	require.Len(t, deleted, 1)

	mgr.RemoveDeletedFromState()

	require.NoError(t, mgr.Save(ctx))

	mgr2 := state.NewManager(state.ManagerOptions{
		BaseDir: dir,
	})
	require.NoError(t, mgr2.Load(ctx))

	assert.False(t, mgr2.ShouldProcess("https://example.com/keep", "hash1"))
	assert.True(t, mgr2.ShouldProcess("https://example.com/delete", "hash2"))
}

func TestManager_Stats(t *testing.T) {
	mgr := state.NewManager(state.ManagerOptions{
		BaseDir: t.TempDir(),
	})

	mgr.Update("https://example.com/page1", state.PageState{ContentHash: "h1"})
	mgr.Update("https://example.com/page2", state.PageState{ContentHash: "h2"})
	mgr.Update("https://example.com/page3", state.PageState{ContentHash: "h3"})

	total, _ := mgr.Stats()
	assert.Equal(t, 3, total)
}

func TestManager_SaveNoDirty(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	mgr := state.NewManager(state.ManagerOptions{
		BaseDir: dir,
	})

	err := mgr.Save(ctx)
	require.NoError(t, err)

	statePath := filepath.Join(dir, state.StateFileName)
	_, err = os.Stat(statePath)
	assert.True(t, os.IsNotExist(err))
}

func TestSyncState_Methods(t *testing.T) {
	s := state.NewSyncState("https://example.com", "crawler")

	assert.Equal(t, 0, s.PageCount())
	assert.False(t, s.HasPage("https://example.com/page"))

	s.SetPage("https://example.com/page", state.PageState{
		ContentHash: "hash",
		FilePath:    "page.md",
	})

	assert.Equal(t, 1, s.PageCount())
	assert.True(t, s.HasPage("https://example.com/page"))

	page, exists := s.GetPage("https://example.com/page")
	assert.True(t, exists)
	assert.Equal(t, "hash", page.ContentHash)

	s.RemovePage("https://example.com/page")
	assert.Equal(t, 0, s.PageCount())
}
