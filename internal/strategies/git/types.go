package git

// Platform represents a git hosting platform
type Platform string

const (
	PlatformGitHub    Platform = "github"
	PlatformGitLab    Platform = "gitlab"
	PlatformBitbucket Platform = "bitbucket"
	PlatformGeneric   Platform = "generic"
)

// RepoInfo contains parsed repository information
type RepoInfo struct {
	Platform Platform
	Owner    string
	Repo     string
	URL      string // Original URL
}

// GitURLInfo contains parsed Git URL information including optional path
type GitURLInfo struct {
	RepoURL  string // Clean repository URL (without /tree/... suffix)
	Platform Platform
	Owner    string
	Repo     string
	Branch   string // Branch from URL (empty if not specified)
	SubPath  string // Subdirectory path (empty if root)
}

// FetchResult contains the result of a repository fetch operation
type FetchResult struct {
	LocalPath string // Path to extracted/cloned repo
	Branch    string // Detected or specified branch
	Method    string // "archive" or "clone"
}

// DocumentExtensions are file extensions to process (markdown only)
var DocumentExtensions = map[string]bool{
	".md":  true,
	".mdx": true,
}

// ConfigExtensions are configuration file extensions to include as raw files
var ConfigExtensions = map[string]bool{
	".json": true,
	".yaml": true,
	".yml":  true,
	".toml": true,
	".env":  true,
}

// IgnoreDirs are directories to skip during file discovery
var IgnoreDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"__pycache__":  true,
	".venv":        true,
	"venv":         true,
	"dist":         true,
	"build":        true,
	".next":        true,
	".nuxt":        true,
}
