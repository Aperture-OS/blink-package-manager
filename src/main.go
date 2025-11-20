package main

import (
	"encoding/json" // For decoding JSON package recipes
	"fmt"           // For formatted I/O (printing, formatting strings)
	"io"            // For reading HTTP response bodies
	"log"           // For logging info, warnings, and errors
	"net/http"      // For HTTP requests (fetch package recipes)
	"os"            // For file and directory operations
	"path/filepath" // For handling file paths in a cross-platform way
	"strings"       // For string manipulation (lowercase, suffix check)
	"time"          // For getting the current year

	"github.com/spf13/cobra" // Cobra CLI framework
)

//
// ─── STRUCT ─────────────────────────────────────────────────────────────────────
//

// PackageInfo represents the JSON structure of a package recipe
type PackageInfo struct {
	Name        string   `json:"name"`        // Package name
	Version     string   `json:"version"`     // Package version
	Release     int      `json:"release"`     // Release number
	Description string   `json:"description"` // Short description
	Author      string   `json:"author"`      // Author of package
	License     string   `json:"license"`     // License type (MIT, GPL, etc.)
	Source      struct { // Source code info
		URL    string `json:"url"`    // URL to download source code
		Type   string `json:"type"`   // Archive type (zip, tar, etc.)
		Sha256 string `json:"sha256"` // Checksum for verification
	} `json:"source"`
	Dependencies map[string]string `json:"dependencies"` // Required dependencies
	OptDeps      []struct {        // Optional dependencies groups
		ID          int      `json:"id"`          // Group ID
		Description string   `json:"description"` // Group description
		Options     []string `json:"options"`     // List of options
		Default     string   `json:"default"`     // Default option
	} `json:"opt_dependencies"`
	Build struct { // Build instructions
		Env       map[string]string `json:"env"`       // Environment variables for build
		Prepare   []string          `json:"prepare"`   // Commands to prepare build
		Install   []string          `json:"install"`   // Commands to install package
		Uninstall []string          `json:"uninstall"` // Commands to uninstall package
	} `json:"build"`
}

//
// ─── GLOBALS ────────────────────────────────────────────────────────────────────
//

var (
	repoURL          = "https://github.com/Aperture-OS/testing-blink-repo/blob/main/pseudoRepo"                       // Display repo URL
	baseURL          = "https://raw.githubusercontent.com/Aperture-OS/testing-blink-repo/refs/heads/main/pseudoRepo/" // Raw JSON base URL
	defaultCachePath = "./blink/"                                                                                     // Default local cache directory
	currentYear      = time.Now().Year()                                                                              // Current year for copyright
	Version          = "v0.0.1-alpha"                                                                                 // Blink version
	lockPath         = filepath.Join(defaultCachePath, "etc", "blink.lock")                                           // Path to lock file
	supportPage      = `
Having trouble? Join our Discord or open a GitHub issue.
Include any DEBUG INFO logs when reporting issues.
Discord: https://discord.com/invite/rx82u93hGD
GitHub Issues: https://github.com/Aperture-OS/Blink-Package-Manager/issues
`
)

//
// ─── FUNCTIONS ──────────────────────────────────────────────────────────────────
//

func addLock(lockPath string) error {

	if _, err := os.Stat(filepath.Join(defaultCachePath, "etc")); os.IsNotExist(err) {
		log.Printf("INFO: Lock directory does not exist. Creating...")
		os.MkdirAll(filepath.Join(defaultCachePath, "etc"), os.ModePerm)
	}

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err // lock already exists or another error
	}
	f.Close()
	return nil

}

func checkLock(lockPath string) bool {
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		return false // lock does not exist
	}
	return true // lock exists
}

func removeLock(lockPath string) error {
	if err := os.Remove(lockPath); err != nil {
		return err // failed to remove lock
	}
	return nil
}

// getpkg downloads a package recipe from the repository
func getpkg(pkgName string, path string) error {

	log.Printf("DEBUG: Acquiring lock at %s", lockPath)
	if checkLock(lockPath) {
		return fmt.Errorf("another instance is running, lock file exists at %s", lockPath)
	}

	lockErr := addLock(lockPath) // add lock file
	defer removeLock(lockPath)   // remove lock file at the end

	if lockErr != nil { // check for errors while adding lock
		return fmt.Errorf("failed to create lock file at %s: %v", lockPath, lockErr)
	}

	// Ensure path ends with OS-specific separator
	if !strings.HasSuffix(path, string(os.PathSeparator)) {
		path += string(os.PathSeparator)
	}

	// Check if cache directory exists

	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Printf("Cache directory does not exist. Create it? [Y/n]: ")
		var resp string
		fmt.Scanln(&resp)
		resp = strings.ToLower(resp)

		if resp == "n" || resp == "no" {
			return fmt.Errorf("cache directory required, user declined creation") // idk how the fuck this exits the program but it works ig
		}

		// Create directory if yes/default
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create cache directory: %v", err)
		}
		log.Printf("DEBUG: Cache directory created at %s", path)
	}

	// Full path to recipe
	recipePath := filepath.Join(path, pkgName+".json")

	// Build URL for package JSON
	url := baseURL + pkgName + ".json"

	// Perform HTTP GET request
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download recipe: %v", err)
	}
	defer resp.Body.Close()

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download recipe, status: %s", resp.Status)
	}

	// Create file to save recipe
	outFile, err := os.Create(recipePath)
	if err != nil {
		return fmt.Errorf("failed to create recipe file: %v", err)
	}
	defer outFile.Close()

	// Copy response body to file
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write recipe file: %v", err)
	}

	log.Printf("DEBUG: Package recipe downloaded to %s", recipePath)
	return nil
}

// fetchpkg opens a package recipe and prints its information
func fetchpkg(path string, force bool, pkgName string) error {

	// Ensure path ends with OS-specific separator

	if !strings.HasSuffix(path, string(os.PathSeparator)) {
		path += string(os.PathSeparator)
	}

	// Full path to package JSON
	recipePath := filepath.Join(path, pkgName+".json")

	// If force flag is true, remove existing cached recipe
	if force {
		if err := os.Remove(recipePath); err == nil {
			log.Printf("DEBUG: Force flag detected, removed cached recipe at %s", recipePath)
		} else if !os.IsNotExist(err) {
			log.Printf("WARNING: Failed to remove cached recipe. Debug: %v", err)
		}
	}

	// If recipe does not exist, download it
	if _, err := os.Stat(recipePath); os.IsNotExist(err) {
		log.Printf("INFO: Package recipe not found. Downloading...")
		if err := getpkg(pkgName, path); err != nil {
			return err // Return error if download fails
		}
	}

	// Open recipe JSON file
	f, err := os.Open(recipePath)
	if err != nil {
		log.Printf("FATAL: Failed to open package recipe. Debug: %v", err)
		return fmt.Errorf("error opening file: %v", err)
	}
	defer f.Close() // Ensure file is closed after function ends

	// Decode JSON into PackageInfo struct
	var pkg PackageInfo
	if err := json.NewDecoder(f).Decode(&pkg); err != nil {
		log.Printf("FATAL: Failed to parse JSON. Debug: %v", err)
		return fmt.Errorf("error decoding JSON: %v", err)
	}

	// Print package information
	fmt.Printf("\nUsing repo: %s\n\nName: %s\nVersion: %s\nDescription: %s\nAuthor: %s\nLicense: %s\n",
		repoURL, pkg.Name, pkg.Version, pkg.Description, pkg.Author, pkg.License)

	return nil
}

// sendSupport prints support info
func sendSupport() {
	fmt.Printf("%s", supportPage)
}

// sendVersion prints Blink version info
func sendVersion() {
	fmt.Printf(`
Blink Package Manager - Version %s
Licensed under GPL v3.0 by Aperture OS
https://aperture-os.github.io
All rights reserved. © Copyright 2025-%d Aperture OS.
`, Version, currentYear)
}

// install downloads (fetches) a package and would install it (placeholder)
func install(pkgName string, force bool, path string) error {
	// Fetch package info from cache or repo
	if err := fetchpkg(path, force, pkgName); err != nil {
		return err // Return error if fetch fails
	}

	// TODO implement actual installation logic here
	fmt.Printf("Installation logic is not yet implemented.\n", pkgName)	

	return nil // Return nil to indicate success
}

//
// ─── MAIN ───────────────────────────────────────────────────────────────────────
//

func main() {

	log.SetFlags(log.Ltime)
	log.SetPrefix("[Blink Debug] ")

	// Flags for CLI commands
	var force bool  // Force re-download or reinstall
	var path string // Custom cache path

	// Root Cobra command
	rootCmd := &cobra.Command{
		Use:   "blink",
		Short: "Blink - lightweight, source-based package manager for Aperture OS",
		Long:  "Blink - lightweight, fast, source-based package manager for Aperture OS and Unix-like systems.",
	}

	// ─── blink get <pkg> ───────────────────────────────────────────────────────
	getCmd := &cobra.Command{
		Use:     "get <pkg>",
		Short:   "Download a package recipe (JSON file)",
		Args:    cobra.ExactArgs(1),
		Aliases: []string{"d", "download", "g", "dl"},
		Run: func(cmd *cobra.Command, args []string) {
			pkgName := args[0]
			if path == "" {
				path = defaultCachePath
			}
			if err := getpkg(pkgName, path); err != nil {
				log.Fatalf("Error fetching package: %v", err)
			}
		},
	}
	getCmd.Flags().BoolVarP(&force, "force", "f", false, "Force re-download")
	getCmd.Flags().StringVarP(&path, "path", "p", defaultCachePath, "Specify cache directory")

	// ─── blink info <pkg> ──────────────────────────────────────────────────────
	infoCmd := &cobra.Command{
		Use:     "info <pkg>",
		Short:   "Fetch & display package information",
		Args:    cobra.ExactArgs(1),
		Aliases: []string{"information", "pkginfo", "details", "fetch"},
		Run: func(cmd *cobra.Command, args []string) {
			pkgName := args[0]
			if path == "" {
				path = defaultCachePath
			}
			if err := fetchpkg(path, force, pkgName); err != nil {
				log.Fatalf("Error reading package info: %v", err)
			}
		},
	}
	infoCmd.Flags().BoolVarP(&force, "force", "f", false, "Force re-download")
	infoCmd.Flags().StringVarP(&path, "path", "p", defaultCachePath, "Specify cache directory")

	// ─── blink install <pkg> ───────────────────────────────────────────────────
	installCmd := &cobra.Command{
		Use:     "install <pkg>",
		Short:   "Download and install a package",
		Args:    cobra.ExactArgs(1),
		Aliases: []string{"i", "add", "inst"},
		Run: func(cmd *cobra.Command, args []string) {
			pkgName := args[0]
			if path == "" {
				path = defaultCachePath
			}
			if err := install(pkgName, force, path); err != nil {
				log.Fatalf("Error installing package: %v", err)
			}
		},
	}
	installCmd.Flags().BoolVarP(&force, "force", "f", false, "Force reinstall")
	installCmd.Flags().StringVarP(&path, "path", "p", defaultCachePath, "Specify cache directory")

	// ─── blink support ─────────────────────────────────────────────────────────
	supportCmd := &cobra.Command{
		Use:     "support",
		Aliases: []string{"issue", "bug", "help", "contact", "discord"},
		Short:   "Show support information",
		Run: func(cmd *cobra.Command, args []string) {
			sendSupport()
		},
	}

	// ─── blink version ─────────────────────────────────────────────────────────
	versionCmd := &cobra.Command{
		Use:     "version",
		Aliases: []string{"v", "ver", "--version", "-v"},
		Short:   "Show Blink version",
		Run: func(cmd *cobra.Command, args []string) {
			sendVersion()
		},
	}

	// Add commands to root
	rootCmd.AddCommand(getCmd, infoCmd, installCmd, supportCmd, versionCmd)

	// Disable default Cobra completion
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// ─── Shell completion command ───────────────────────────────────────────────
	completionCmd := &cobra.Command{
		Use:       "completion [bash|zsh|fish]",
		Short:     "Generate shell completion scripts",
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish"},
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletion(os.Stdout)
			case "zsh":
				return rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				return rootCmd.GenFishCompletion(os.Stdout, true)
			default:
				return cmd.Help()
			}
		},
	}
	rootCmd.AddCommand(completionCmd)

	// Print welcome message
	fmt.Printf("Welcome to Blink Package Manager! Version: %s\n", Version)
	fmt.Printf("© Copyright 2025-%d Aperture OS. All rights reserved.\n", currentYear)

	// Execute root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
