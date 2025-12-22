package version_test

import (
	"testing"

	"github.com/quantmind-br/repodocs-go/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet_String_Short_Full(t *testing.T) {
	// Preserve original values
	origV, origB, origC := version.Version, version.BuildTime, version.Commit
	defer func() { version.Version, version.BuildTime, version.Commit = origV, origB, origC }()

	// Set deterministic values
	version.Version = "1.2.3"
	version.BuildTime = "2025-12-22T00:00:00Z"
	version.Commit = "deadbeef"

	info := version.Get()
	require.Equal(t, "1.2.3", info.Version)
	require.Equal(t, "2025-12-22T00:00:00Z", info.BuildTime)
	require.Equal(t, "deadbeef", info.Commit)

	// Runtime fields should be non-empty
	require.NotEmpty(t, info.GoVersion)
	require.NotEmpty(t, info.OS)
	require.NotEmpty(t, info.Arch)

	// Short should return the Version
	assert.Equal(t, "1.2.3", version.Short())

	// String/Full should contain the expected pieces
	s := info.String()
	assert.Contains(t, s, "repodocs 1.2.3")
	assert.Contains(t, s, "commit: deadbeef")
	assert.Contains(t, version.Full(), "repodocs 1.2.3")

	// Ensure format sanity for prefix
	assert.Contains(t, info.String(), "repodocs 1.2.3 (commit: deadbeef, built: 2025-12-22T00:00:00Z")
}
