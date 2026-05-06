// Package git provides a thin wrapper around go-git for cloning and reading
// repositories.
//
// The wrapper keeps repository operations injectable and testable while hiding
// go-git details from callers. It is primarily used by the git extraction
// strategy for clone, archive, and file traversal workflows.
package git
