package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/quantmind-br/repodocs-go/internal/app"
	"github.com/quantmind-br/repodocs-go/internal/config"
	"github.com/quantmind-br/repodocs-go/internal/domain"
	"github.com/quantmind-br/repodocs-go/internal/utils"
	"github.com/quantmind-br/repodocs-go/pkg/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	verbose bool
	log     *utils.Logger

	// Dependencies for testing
	osStat       = os.Stat
	execLookPath = exec.LookPath
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "repodocs [url]",
	Short: "Extract documentation from any source",
	Long: `RepoDocs is a CLI tool for extracting documentation from websites,
git repositories, sitemaps, pkg.go.dev, and llms.txt files.

It supports stealth mode for avoiding bot detection and JavaScript rendering
for single-page applications.`,
	Version: version.Short(),
	Args:    cobra.MaximumNArgs(1),
	RunE:    run,
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ~/.repodocs/config.yaml)")
	rootCmd.PersistentFlags().StringP("output", "o", "./docs", "Output directory")
	rootCmd.PersistentFlags().IntP("concurrency", "j", 5, "Number of concurrent workers")
	rootCmd.PersistentFlags().IntP("limit", "l", 0, "Max pages to process (0=unlimited)")
	rootCmd.PersistentFlags().IntP("max-depth", "d", 4, "Max crawl depth")
	rootCmd.PersistentFlags().StringSlice("exclude", nil, "Regex patterns to exclude")
	rootCmd.PersistentFlags().String("filter", "", "Path filter (web: base URL; git: subdirectory)")
	rootCmd.PersistentFlags().Bool("nofolders", false, "Flat output structure")
	rootCmd.PersistentFlags().Bool("force", false, "Overwrite existing files")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	// Cache flags
	rootCmd.PersistentFlags().Bool("no-cache", false, "Disable caching")
	rootCmd.PersistentFlags().Duration("cache-ttl", 24*time.Hour, "Cache TTL")
	rootCmd.PersistentFlags().Bool("refresh-cache", false, "Force cache refresh")

	// Rendering flags
	rootCmd.PersistentFlags().Bool("render-js", false, "Force JS rendering")
	rootCmd.PersistentFlags().Duration("timeout", 30*time.Second, "Request timeout")

	// Output flags
	rootCmd.PersistentFlags().Bool("json-meta", false, "Generate JSON metadata files")
	rootCmd.PersistentFlags().Bool("dry-run", false, "Simulate without writing files")

	// Specific flags
	rootCmd.PersistentFlags().Bool("split", false, "Split output by sections (pkg.go.dev)")
	rootCmd.PersistentFlags().Bool("include-assets", false, "Include referenced images (git)")
	rootCmd.PersistentFlags().String("user-agent", "", "Custom User-Agent")
	rootCmd.PersistentFlags().String("content-selector", "", "CSS selector for main content")
	rootCmd.PersistentFlags().String("exclude-selector", "", "CSS selector for elements to exclude from content")

	// Bind flags to viper
	_ = viper.BindPFlag("output.directory", rootCmd.PersistentFlags().Lookup("output"))
	_ = viper.BindPFlag("concurrency.workers", rootCmd.PersistentFlags().Lookup("concurrency"))
	_ = viper.BindPFlag("concurrency.max_depth", rootCmd.PersistentFlags().Lookup("max-depth"))
	_ = viper.BindPFlag("concurrency.timeout", rootCmd.PersistentFlags().Lookup("timeout"))
	_ = viper.BindPFlag("output.flat", rootCmd.PersistentFlags().Lookup("nofolders"))
	_ = viper.BindPFlag("output.overwrite", rootCmd.PersistentFlags().Lookup("force"))
	_ = viper.BindPFlag("cache.enabled", rootCmd.PersistentFlags().Lookup("no-cache"))
	_ = viper.BindPFlag("cache.ttl", rootCmd.PersistentFlags().Lookup("cache-ttl"))
	_ = viper.BindPFlag("rendering.force_js", rootCmd.PersistentFlags().Lookup("render-js"))
	_ = viper.BindPFlag("output.json_metadata", rootCmd.PersistentFlags().Lookup("json-meta"))
	_ = viper.BindPFlag("stealth.user_agent", rootCmd.PersistentFlags().Lookup("user-agent"))

	// Add subcommands
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(versionCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	}
}

func run(cmd *cobra.Command, args []string) error {
	// Initialize logger
	logLevel := "info"
	if verbose {
		logLevel = "debug"
	}
	log = utils.NewLogger(utils.LoggerOptions{
		Level:   logLevel,
		Format:  "pretty",
		Verbose: verbose,
	})

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if URL was provided
	if len(args) == 0 {
		return cmd.Help()
	}

	url := args[0]

	// Handle output directory:
	// 1. If user explicitly provided -o flag, use that value
	// 2. Otherwise, generate from URL
	if cmd.Flags().Changed("output") {
		// User explicitly set output directory via -o flag
		cfg.Output.Directory, _ = cmd.Flags().GetString("output")
	} else {
		// Use URL-based output directory
		cfg.Output.Directory = utils.GenerateOutputDirFromURL(url)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Info().Msg("Shutting down gracefully...")
		cancel()
	}()

	// Get flags
	limit, _ := cmd.Flags().GetInt("limit")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	split, _ := cmd.Flags().GetBool("split")
	includeAssets, _ := cmd.Flags().GetBool("include-assets")
	contentSelector, _ := cmd.Flags().GetString("content-selector")
	excludeSelector, _ := cmd.Flags().GetString("exclude-selector")
	excludePatterns, _ := cmd.Flags().GetStringSlice("exclude")
	renderJS, _ := cmd.Flags().GetBool("render-js")
	force, _ := cmd.Flags().GetBool("force")
	filterURL, _ := cmd.Flags().GetString("filter")

	// Create orchestrator options
	orchOpts := app.OrchestratorOptions{
		CommonOptions: domain.CommonOptions{
			Verbose:  verbose,
			DryRun:   dryRun,
			Force:    force,
			RenderJS: renderJS,
			Limit:    limit,
		},
		Config:          cfg,
		Split:           split,
		IncludeAssets:   includeAssets,
		ContentSelector: contentSelector,
		ExcludeSelector: excludeSelector,
		ExcludePatterns: excludePatterns,
		FilterURL:       filterURL,
	}

	// Create orchestrator
	orchestrator, err := app.NewOrchestrator(orchOpts)
	if err != nil {
		return fmt.Errorf("failed to create orchestrator: %w", err)
	}
	defer orchestrator.Close()

	// Validate URL
	if err := orchestrator.ValidateURL(url); err != nil {
		return err
	}

	// Run extraction
	return orchestrator.Run(ctx, url, orchOpts)
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system dependencies",
	Long:  "Verifies that all system dependencies are properly installed and configured.",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Checking system dependencies...")
		allPassed := true

		// Check 1: Internet connection
		fmt.Print("  Internet connection: ")
		if checkInternet() {
			fmt.Println("OK")
		} else {
			fmt.Println("FAILED")
			allPassed = false
		}

		// Check 2: Chrome/Chromium
		fmt.Print("  Chrome/Chromium: ")
		if chromePath := checkChrome(); chromePath != "" {
			fmt.Printf("OK (%s)\n", chromePath)
		} else {
			fmt.Println("NOT FOUND (JS rendering will be unavailable)")
		}

		// Check 3: Write permissions for output dir
		fmt.Print("  Write permissions: ")
		if checkWritePermissions() {
			fmt.Println("OK")
		} else {
			fmt.Println("FAILED")
			allPassed = false
		}

		// Check 4: Config file
		fmt.Print("  Config file: ")
		_, err := config.Load()
		if err != nil {
			fmt.Printf("WARN (%v)\n", err)
		} else {
			fmt.Println("OK")
		}

		// Check 5: Cache directory
		fmt.Print("  Cache directory: ")
		cacheDir := utils.ExpandPath("~/.repodocs/cache")
		if checkCacheDir(cacheDir) {
			fmt.Printf("OK (%s)\n", cacheDir)
		} else {
			fmt.Println("WARN (will be created on first use)")
		}

		fmt.Println()
		if allPassed {
			fmt.Println("All critical checks passed!")
		} else {
			fmt.Println("Some checks failed. Please resolve the issues above.")
		}
		return nil
	},
}

// checkInternet checks if there's an internet connection
func checkInternet() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, "https://www.google.com", nil)
	if err != nil {
		return false
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode < 400
}

// checkChrome checks if Chrome/Chromium is available
func checkChrome() string {
	// Common Chrome/Chromium paths
	paths := []string{
		// Linux
		"/usr/bin/google-chrome",
		"/usr/bin/google-chrome-stable",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		"/snap/bin/chromium",
		// macOS
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
		// Windows (via PATH)
		"chrome.exe",
		"chromium.exe",
	}

	for _, path := range paths {
		if _, err := osStat(path); err == nil {
			return path
		}
	}

	// Try to find via which/where command
	if path, err := execLookPath("google-chrome"); err == nil {
		return path
	}
	if path, err := execLookPath("chromium"); err == nil {
		return path
	}
	if path, err := execLookPath("chromium-browser"); err == nil {
		return path
	}

	return ""
}

// checkWritePermissions checks if we can write to the current directory
func checkWritePermissions() bool {
	tmpFile := ".repodocs_test_write"
	f, err := os.Create(tmpFile)
	if err != nil {
		return false
	}
	f.Close()
	os.Remove(tmpFile)
	return true
}

// checkCacheDir checks if the cache directory exists or can be created
func checkCacheDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.Full())
	},
}
