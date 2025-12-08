package benchmark

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
)

// Test repositories of different sizes
var testRepos = []struct {
	name        string
	url         string
	archiveURL  string
	description string
}{
	{
		name:        "small",
		url:         "https://github.com/kelseyhightower/nocode",
		archiveURL:  "https://github.com/kelseyhightower/nocode/archive/refs/heads/master.tar.gz",
		description: "Very small repo (~10 files)",
	},
	{
		name:        "medium",
		url:         "https://github.com/charmbracelet/bubbletea",
		archiveURL:  "https://github.com/charmbracelet/bubbletea/archive/refs/heads/master.tar.gz",
		description: "Medium repo (~100 files)",
	},
	{
		name:        "large",
		url:         "https://github.com/gin-gonic/gin",
		archiveURL:  "https://github.com/gin-gonic/gin/archive/refs/heads/master.tar.gz",
		description: "Large repo (~500 files)",
	},
	{
		name:        "very_large",
		url:         "https://github.com/kubernetes/kubernetes",
		archiveURL:  "https://github.com/kubernetes/kubernetes/archive/refs/heads/master.tar.gz",
		description: "Very large repo (Kubernetes ~30k files)",
	},
}

// BenchmarkGitClone benchmarks the current git clone approach
func BenchmarkGitClone(b *testing.B) {
	for _, repo := range testRepos[:3] { // Skip very large for quick benchmarks
		b.Run(repo.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				tmpDir, err := os.MkdirTemp("", "bench-clone-*")
				if err != nil {
					b.Fatal(err)
				}

				b.StartTimer()
				_, err = git.PlainClone(tmpDir, false, &git.CloneOptions{
					URL:          repo.url,
					Depth:        1,
					SingleBranch: true,
					Tags:         git.NoTags,
					Progress:     nil, // No progress output during benchmark
				})
				b.StopTimer()

				os.RemoveAll(tmpDir)

				if err != nil {
					b.Fatalf("Clone failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkArchiveDownload benchmarks the archive download approach
func BenchmarkArchiveDownload(b *testing.B) {
	for _, repo := range testRepos[:3] { // Skip very large for quick benchmarks
		b.Run(repo.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				tmpDir, err := os.MkdirTemp("", "bench-archive-*")
				if err != nil {
					b.Fatal(err)
				}

				b.StartTimer()
				err = downloadAndExtractArchive(repo.archiveURL, tmpDir)
				b.StopTimer()

				os.RemoveAll(tmpDir)

				if err != nil {
					b.Fatalf("Archive download failed: %v", err)
				}
			}
		})
	}
}

// downloadAndExtractArchive downloads a .tar.gz archive and extracts it
func downloadAndExtractArchive(archiveURL, destDir string) error {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Minute,
	}

	// Download archive
	resp, err := client.Get(archiveURL)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// Create gzip reader
	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("gzip reader failed: %w", err)
	}
	defer gzr.Close()

	// Create tar reader
	tr := tar.NewReader(gzr)

	// Extract files
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read failed: %w", err)
		}

		// Skip the root directory (GitHub adds repo-branch/ prefix)
		parts := strings.SplitN(header.Name, "/", 2)
		if len(parts) < 2 || parts[1] == "" {
			continue
		}
		relativePath := parts[1]

		targetPath := filepath.Join(destDir, relativePath)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return fmt.Errorf("mkdir failed: %w", err)
			}
		case tar.TypeReg:
			// Create parent directory
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("mkdir failed: %w", err)
			}

			// Create file
			f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("create file failed: %w", err)
			}

			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("copy failed: %w", err)
			}
			f.Close()
		}
	}

	return nil
}

// TestCompareApproaches runs a comparison test (not a benchmark)
// Run with: go test -v -run TestCompareApproaches ./tests/benchmark/...
func TestCompareApproaches(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping comparison test in short mode")
	}

	repo := testRepos[1] // Use medium repo for comparison
	ctx := context.Background()
	_ = ctx // for future use

	fmt.Printf("\n=== Performance Comparison: %s ===\n", repo.description)
	fmt.Printf("Repository: %s\n\n", repo.url)

	// Test Git Clone
	fmt.Println("Testing Git Clone (shallow, single branch, no tags)...")
	cloneDir, _ := os.MkdirTemp("", "compare-clone-*")
	defer os.RemoveAll(cloneDir)

	cloneStart := time.Now()
	_, err := git.PlainClone(cloneDir, false, &git.CloneOptions{
		URL:          repo.url,
		Depth:        1,
		SingleBranch: true,
		Tags:         git.NoTags,
		Progress:     nil,
	})
	cloneDuration := time.Since(cloneStart)

	if err != nil {
		t.Fatalf("Clone failed: %v", err)
	}

	cloneSize := getDirSize(cloneDir)
	cloneFiles := countFiles(cloneDir)

	fmt.Printf("  Duration: %v\n", cloneDuration)
	fmt.Printf("  Total size: %.2f MB\n", float64(cloneSize)/(1024*1024))
	fmt.Printf("  Total files: %d\n\n", cloneFiles)

	// Test Archive Download
	fmt.Println("Testing Archive Download (tar.gz)...")
	archiveDir, _ := os.MkdirTemp("", "compare-archive-*")
	defer os.RemoveAll(archiveDir)

	archiveStart := time.Now()
	err = downloadAndExtractArchive(repo.archiveURL, archiveDir)
	archiveDuration := time.Since(archiveStart)

	if err != nil {
		t.Fatalf("Archive download failed: %v", err)
	}

	archiveSize := getDirSize(archiveDir)
	archiveFiles := countFiles(archiveDir)

	fmt.Printf("  Duration: %v\n", archiveDuration)
	fmt.Printf("  Total size: %.2f MB\n", float64(archiveSize)/(1024*1024))
	fmt.Printf("  Total files: %d\n\n", archiveFiles)

	// Comparison
	fmt.Println("=== Results ===")
	speedup := float64(cloneDuration) / float64(archiveDuration)
	fmt.Printf("Archive is %.2fx %s than Git Clone\n",
		max(speedup, 1/speedup),
		map[bool]string{true: "faster", false: "slower"}[speedup > 1])

	sizeRatio := float64(archiveSize) / float64(cloneSize)
	fmt.Printf("Size ratio (archive/clone): %.2f\n", sizeRatio)
}

// TestCompareAllSizes runs comparison for all repository sizes
// Run with: go test -v -run TestCompareAllSizes ./tests/benchmark/...
func TestCompareAllSizes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping comparison test in short mode")
	}

	fmt.Println("\n========================================"[1:])
	fmt.Println("  Git Clone vs Archive Download")
	fmt.Println("========================================")

	results := make([]struct {
		name           string
		cloneDuration  time.Duration
		archiveDuration time.Duration
		cloneSize      int64
		archiveSize    int64
		cloneFiles     int
		archiveFiles   int
	}, 0)

	for _, repo := range testRepos[:3] { // Skip kubernetes for now
		fmt.Printf("\n--- %s: %s ---\n", repo.name, repo.description)
		fmt.Printf("URL: %s\n\n", repo.url)

		// Test Git Clone
		fmt.Println("Git Clone (shallow, single branch, no tags)...")
		cloneDir, _ := os.MkdirTemp("", "compare-clone-*")

		cloneStart := time.Now()
		_, err := git.PlainClone(cloneDir, false, &git.CloneOptions{
			URL:          repo.url,
			Depth:        1,
			SingleBranch: true,
			Tags:         git.NoTags,
			Progress:     nil,
		})
		cloneDuration := time.Since(cloneStart)

		if err != nil {
			fmt.Printf("  Clone FAILED: %v\n", err)
			os.RemoveAll(cloneDir)
			continue
		}

		cloneSize := getDirSize(cloneDir)
		cloneFiles := countFiles(cloneDir)
		os.RemoveAll(cloneDir)

		fmt.Printf("  Duration: %v\n", cloneDuration)
		fmt.Printf("  Size: %.2f MB, Files: %d\n", float64(cloneSize)/(1024*1024), cloneFiles)

		// Test Archive Download
		fmt.Println("\nArchive Download (tar.gz)...")
		archiveDir, _ := os.MkdirTemp("", "compare-archive-*")

		archiveStart := time.Now()
		err = downloadAndExtractArchive(repo.archiveURL, archiveDir)
		archiveDuration := time.Since(archiveStart)

		if err != nil {
			fmt.Printf("  Archive FAILED: %v\n", err)
			os.RemoveAll(archiveDir)
			continue
		}

		archiveSize := getDirSize(archiveDir)
		archiveFiles := countFiles(archiveDir)
		os.RemoveAll(archiveDir)

		fmt.Printf("  Duration: %v\n", archiveDuration)
		fmt.Printf("  Size: %.2f MB, Files: %d\n", float64(archiveSize)/(1024*1024), archiveFiles)

		// Store results
		results = append(results, struct {
			name           string
			cloneDuration  time.Duration
			archiveDuration time.Duration
			cloneSize      int64
			archiveSize    int64
			cloneFiles     int
			archiveFiles   int
		}{
			name:           repo.name,
			cloneDuration:  cloneDuration,
			archiveDuration: archiveDuration,
			cloneSize:      cloneSize,
			archiveSize:    archiveSize,
			cloneFiles:     cloneFiles,
			archiveFiles:   archiveFiles,
		})

		// Print comparison
		// speedup > 1 means clone took longer, so Archive is faster
		speedup := float64(cloneDuration) / float64(archiveDuration)
		winner := "Git Clone"
		ratio := speedup
		if speedup > 1 {
			winner = "Archive"
			ratio = speedup
		} else {
			ratio = 1 / speedup
		}
		fmt.Printf("\n  Winner: %s (%.2fx faster)\n", winner, ratio)
	}

	// Summary table
	fmt.Println("\n========================================"[1:])
	fmt.Println("  SUMMARY")
	fmt.Println("========================================")
	fmt.Printf("\n%-12s | %-12s | %-12s | %-10s | Winner\n", "Repo", "Clone", "Archive", "Speedup")
	fmt.Println("-------------|--------------|--------------|------------|--------")

	for _, r := range results {
		// speedup > 1 means clone took longer, so Archive is faster
		speedup := float64(r.cloneDuration) / float64(r.archiveDuration)
		winner := "Clone"
		speedupStr := fmt.Sprintf("%.2fx", 1/speedup)
		if speedup > 1 {
			winner = "Archive"
			speedupStr = fmt.Sprintf("%.2fx", speedup)
		}
		fmt.Printf("%-12s | %-12v | %-12v | %-10s | %s\n",
			r.name, r.cloneDuration.Round(time.Millisecond),
			r.archiveDuration.Round(time.Millisecond), speedupStr, winner)
	}
}

// TestLargeRepoComparison tests with a very large repository (kubernetes)
// Run with: go test -v -timeout 30m -run TestLargeRepoComparison ./tests/benchmark/...
func TestLargeRepoComparison(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large repo test in short mode")
	}

	repo := testRepos[3] // Kubernetes

	fmt.Printf("\n=== LARGE REPO TEST: %s ===\n", repo.description)
	fmt.Printf("Repository: %s\n", repo.url)
	fmt.Println("WARNING: This test may take several minutes!")

	// Test Git Clone
	fmt.Println("Testing Git Clone (shallow, single branch, no tags)...")
	cloneDir, _ := os.MkdirTemp("", "compare-clone-*")
	defer os.RemoveAll(cloneDir)

	cloneStart := time.Now()
	_, err := git.PlainClone(cloneDir, false, &git.CloneOptions{
		URL:          repo.url,
		Depth:        1,
		SingleBranch: true,
		Tags:         git.NoTags,
		Progress:     os.Stdout, // Show progress for long operation
	})
	cloneDuration := time.Since(cloneStart)

	if err != nil {
		t.Fatalf("Clone failed: %v", err)
	}

	cloneSize := getDirSize(cloneDir)
	cloneFiles := countFiles(cloneDir)

	fmt.Printf("\n  Duration: %v\n", cloneDuration)
	fmt.Printf("  Total size: %.2f MB\n", float64(cloneSize)/(1024*1024))
	fmt.Printf("  Total files: %d\n\n", cloneFiles)

	// Test Archive Download
	fmt.Println("Testing Archive Download (tar.gz)...")
	archiveDir, _ := os.MkdirTemp("", "compare-archive-*")
	defer os.RemoveAll(archiveDir)

	archiveStart := time.Now()
	err = downloadAndExtractArchive(repo.archiveURL, archiveDir)
	archiveDuration := time.Since(archiveStart)

	if err != nil {
		t.Fatalf("Archive download failed: %v", err)
	}

	archiveSize := getDirSize(archiveDir)
	archiveFiles := countFiles(archiveDir)

	fmt.Printf("  Duration: %v\n", archiveDuration)
	fmt.Printf("  Total size: %.2f MB\n", float64(archiveSize)/(1024*1024))
	fmt.Printf("  Total files: %d\n\n", archiveFiles)

	// Comparison
	fmt.Println("=== Results ===")
	speedup := float64(cloneDuration) / float64(archiveDuration)
	winner := "Archive"
	if speedup > 1 {
		winner = "Git Clone"
	}
	fmt.Printf("Winner: %s\n", winner)
	fmt.Printf("Speed difference: %.2fx\n", max(speedup, 1/speedup))
	fmt.Printf("Size ratio (archive/clone): %.2f\n", float64(archiveSize)/float64(cloneSize))
}

func getDirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

func countFiles(path string) int {
	var count int
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			count++
		}
		return nil
	})
	return count
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
