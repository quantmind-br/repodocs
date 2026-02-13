package utils

import (
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// MaxFilenameLength is the maximum length for a filename
const MaxFilenameLength = 200

// Windows reserved names
var windowsReserved = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true,
	"COM5": true, "COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true,
	"LPT5": true, "LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

// invalidCharsRegex matches invalid filename characters
var invalidCharsRegex = regexp.MustCompile(`[<>:"|?*\\/]`)

// multipleSpacesRegex matches multiple consecutive spaces/dashes
var multipleSpacesRegex = regexp.MustCompile(`[-_\s]+`)

// SanitizeFilename sanitizes a string for use as a filename
func SanitizeFilename(name string) string {
	original := name

	// Remove invalid characters
	name = invalidCharsRegex.ReplaceAllString(name, "-")

	// Replace multiple spaces/dashes with single dash
	name = multipleSpacesRegex.ReplaceAllString(name, "-")

	// Separate extension from base name
	ext := filepath.Ext(name)
	baseName := strings.TrimSuffix(name, ext)

	// Trim leading/trailing dashes and spaces from base name
	baseName = strings.Trim(baseName, "- ")

	// Check if we had invalid character substitutions
	// If original had invalid chars that created dashes before extension,
	// and the extension exists, preserve one dash before extension
	hadSubstitutions := (original != name) && invalidCharsRegex.MatchString(original)
	if hadSubstitutions && ext != "" && strings.HasSuffix(name, "-."+ext[1:]) {
		// Reconstruct with dash before extension
		name = baseName + "-" + ext
	} else {
		// Reconstruct normally
		if ext != "" {
			name = baseName + ext
		} else {
			name = baseName
		}
	}

	// Check for Windows reserved names
	upper := strings.ToUpper(name)
	baseNameUpper := strings.TrimSuffix(upper, filepath.Ext(upper))
	if windowsReserved[baseNameUpper] {
		name = "_" + name
	}

	// Limit length
	if len(name) > MaxFilenameLength {
		ext := filepath.Ext(name)
		name = name[:MaxFilenameLength-len(ext)] + ext
	}

	// Ensure the name is not empty
	if name == "" {
		name = "untitled"
	}

	return name
}

// URLToFilename converts a URL to a safe filename
func URLToFilename(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return SanitizeFilename(rawURL)
	}

	// Get path and remove leading/trailing slashes
	path := strings.Trim(u.Path, "/")
	if path == "" {
		path = "index"
	}

	// Replace path separators with dashes for flat structure
	path = strings.ReplaceAll(path, "/", "-")

	// Remove common file extensions
	path = strings.TrimSuffix(path, ".html")
	path = strings.TrimSuffix(path, ".htm")
	path = strings.TrimSuffix(path, ".php")
	path = strings.TrimSuffix(path, ".mdx")

	// Sanitize and add .md extension
	filename := SanitizeFilename(path)
	if !strings.HasSuffix(filename, ".md") {
		filename += ".md"
	}

	return filename
}

// URLToPath converts a URL to a nested directory path
func URLToPath(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return SanitizeFilename(rawURL) + ".md"
	}

	// Get path and remove leading/trailing slashes
	path := strings.Trim(u.Path, "/")
	if path == "" {
		path = "index"
	}

	// Remove common file extensions
	path = strings.TrimSuffix(path, ".html")
	path = strings.TrimSuffix(path, ".htm")
	path = strings.TrimSuffix(path, ".php")
	path = strings.TrimSuffix(path, ".mdx")

	// Split path and sanitize each component
	parts := strings.Split(path, "/")
	for i, part := range parts {
		parts[i] = SanitizeFilename(part)
	}

	// Join with OS-specific separator
	result := filepath.Join(parts...)

	// Add .md extension if not present
	if !strings.HasSuffix(result, ".md") {
		result += ".md"
	}

	return result
}

// GeneratePath generates the output path for a URL
func GeneratePath(baseDir, rawURL string, flat bool) string {
	var relativePath string
	if flat {
		relativePath = URLToFilename(rawURL)
	} else {
		relativePath = URLToPath(rawURL)
	}
	return filepath.Join(baseDir, relativePath)
}

// GeneratePathFromRelative generates the output path from a relative file path
// Used for Git-sourced files to preserve the repository's directory structure
func GeneratePathFromRelative(baseDir, relPath string, flat bool) string {
	if flat {
		// For flat mode, convert full path to filename by replacing "/" with "-"
		// Example: docs/developers/tools/memory.md → docs-developers-tools-memory.md

		// Normalize separators to forward slash
		normalized := filepath.ToSlash(relPath)

		// Remove .md/.mdx extension if present
		ext := filepath.Ext(normalized)
		if ext == ".md" || ext == ".mdx" {
			normalized = strings.TrimSuffix(normalized, ext)
		}

		// Replace "/" with "-" to create flat filename
		flatName := strings.ReplaceAll(normalized, "/", "-")

		// Sanitize the result
		flatName = SanitizeFilename(flatName)

		// Add .md extension
		if !strings.HasSuffix(flatName, ".md") {
			flatName += ".md"
		}

		return filepath.Join(baseDir, flatName)
	}

	// For nested mode, preserve directory structure
	relPath = filepath.FromSlash(relPath)

	parts := strings.Split(relPath, string(filepath.Separator))
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		ext := filepath.Ext(lastPart)
		if ext == ".md" || ext == ".mdx" {
			lastPart = strings.TrimSuffix(lastPart, ext)
			parts[len(parts)-1] = lastPart
		}
	}

	for i, part := range parts {
		parts[i] = SanitizeFilename(part)
	}
	result := filepath.Join(parts...)

	// Add .md extension if not present
	if !strings.HasSuffix(result, ".md") {
		result += ".md"
	}

	return filepath.Join(baseDir, result)
}

// JSONPath returns the corresponding JSON metadata path for a markdown file
func JSONPath(mdPath string) string {
	return strings.TrimSuffix(mdPath, ".md") + ".json"
}

// IsValidFilename checks if a filename is valid
func IsValidFilename(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}

	// Check for invalid characters
	if invalidCharsRegex.MatchString(name) {
		return false
	}

	// Check for Windows reserved names
	upper := strings.ToUpper(name)
	baseName := strings.TrimSuffix(upper, filepath.Ext(upper))
	if windowsReserved[baseName] {
		return false
	}

	// Check for control characters
	for _, r := range name {
		if unicode.IsControl(r) {
			return false
		}
	}

	return true
}

// EnsureDir ensures a directory exists, creating it if necessary
func EnsureDir(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0755)
}

// ExpandPath expands ~ to the user's home directory
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home
	}
	return path
}
