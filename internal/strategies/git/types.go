package git

// Platform represents a git hosting platform
type Platform string

const (
	// PlatformGitHub identifies repositories hosted on github.com.
	PlatformGitHub Platform = "github"
	// PlatformGitLab identifies repositories hosted on gitlab.com.
	PlatformGitLab Platform = "gitlab"
	// PlatformBitbucket identifies repositories hosted on bitbucket.org.
	PlatformBitbucket Platform = "bitbucket"
	// PlatformGeneric identifies HTTP(S) git repositories without a recognized hosted platform.
	PlatformGeneric Platform = "generic"
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

// DocumentExtensions are file extensions to process as Markdown documents.
// `.rst` files are converted to Markdown by `converter.ConvertRST` in the
// processor before being written.
var DocumentExtensions = map[string]bool{
	".md":  true,
	".mdx": true,
	".rst": true,
}

// ConfigExtensions are configuration file extensions to include as raw files.
var ConfigExtensions = map[string]bool{
	".json": true,
	".yaml": true,
	".yml":  true,
	".toml": true,
	".env":  true,
}

// IgnoreDirs are directories to skip during file discovery.
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
