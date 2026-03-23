package testutil

import (
	"runtime"
	"testing"
)

// SetTestHome overrides the home directory for the duration of a test.
// On Windows, this sets USERPROFILE (used by os.UserHomeDir()).
// On Unix, this sets HOME (used by os.UserHomeDir()).
// The original value is restored automatically when the test completes.
func SetTestHome(t *testing.T, dir string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", dir)
	} else {
		t.Setenv("HOME", dir)
	}
}
