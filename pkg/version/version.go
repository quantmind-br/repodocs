package version

import (
	"fmt"
	"runtime"
)

// Build-time variables (set via ldflags)
var (
	Version   = "dev"
	BuildTime = "unknown"
	Commit    = "unknown"
)

// Info contains version information
type Info struct {
	Version   string `json:"version"`
	BuildTime string `json:"build_time"`
	Commit    string `json:"commit"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// Get returns the current version info
func Get() Info {
	return Info{
		Version:   Version,
		BuildTime: BuildTime,
		Commit:    Commit,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

// String returns a formatted version string
func (i Info) String() string {
	return fmt.Sprintf("repodocs %s (commit: %s, built: %s, %s %s/%s)",
		i.Version, i.Commit, i.BuildTime, i.GoVersion, i.OS, i.Arch)
}

// Short returns a short version string
func Short() string {
	return Version
}

// Full returns a full version string
func Full() string {
	return Get().String()
}
