package main // main package, entry point

import (
	"bytes"         // For capturing command output
	"encoding/json" // For decoding JSON package recipes
	"fmt"           // For formatted I/O (printing, formatting strings)
	"io"            // For reading HTTP response bodies
	"log"           // For logging info, warnings, and errors
	"net/http"      // For HTTP requests (download shit and allat)
	"os"            // For file and directory operations
	"os/exec"       // For executing shell commands
	"path/filepath" // For handling file paths in a cross-platform way
	"strings"       // For string manipulation (lowercase, suffix check)
	"time"          // For getting the current year

	"github.com/spf13/cobra" // Cobra CLI framework (as u might guess lol)

	"archive/tar"
	"archive/zip"
	"compress/bzip2"
	"compress/gzip"
	"github.com/klauspost/compress/zstd" // for .zst
	"github.com/ulikunitz/xz"            // for .xz
)

/****************************************************/
// PackageInfo represents the JSON structure of a package recipe
/****************************************************/
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
// ************************ GLOBALS ************************
//

var (
	repoURL = "https://github.com/Aperture-OS/testing-blink-repo/blob/main/pseudoRepo" // Display repo URL

	baseURL = "https://raw.githubusercontent.com/Aperture-OS/testing-blink-repo/refs/heads/main/pseudoRepo/" // Raw JSON base URL for the repository

	// TODO: Change to actual repo when releasing blink
	// TODO: Change to a file instead of variable

	defaultCachePath = "./blink/" // Default: /var/cache/blink/

	currentYear = time.Now().Year() // Current year for copyright

	Version = "v0.0.3-alpha" // Blink version

	lockPath = filepath.Join(defaultCachePath, "etc", "blink.lock") // Path to lock file

	supportPage = // Support information string

	`Having trouble? Join our Discord Server or open a GitHub issue.
	Include any DEBUG INFO logs when reporting issues.
	Discord: https://discord.com/invite/rx82u93hGD
	GitHub Issues: https://github.com/Aperture-OS/Blink-Package-Manager/issues`

	etcPath = "./etc" // TODO: Change to /etc/blink when when actual releasing blink

	manifestPath = filepath.Join(etcPath, "manifest.json") // Path to manifest file

	sourcePath = filepath.Join(defaultCachePath, "sources") // Path to downloaded source

	versionPage = // Version information string
	fmt.Sprintf(`Blink Package Manager - Version %s 
	Licensed under GPL v3.0 by Aperture OS
	https://aperture-os.github.io
	All rights reserved. © Copyright 2025-%d Aperture OS.
	`, Version, currentYear)  // return the formatted string
) // TODO: migrate to /var/blink

// TODO: remove this unused vars shit
var _, _ = etcPath, manifestPath // to avoid unused variable warning for now
//
// ─── FUNCTIONS ──────────────────────────────────────────────────────────────────
//

/***************************************************/
// addLock function: adds a lock file to prevent concurrent executions
// checkLock function: checks if the lock file exists
// removeLock function: removes the lock file
// Why? If multiple instances of Blink run simultaneously, they might interfere with each other,
// leading to corrupted downloads, incomplete installations, or other unexpected behaviors.
// The lock file acts as a semaphore, ensuring that only one instance of Blink can perform
// operations at a time. If another instance is detected (lock file exists), the new instance
// will exit gracefully, informing the user about the existing lock. This mechanism helps maintain
// the integrity of package management operations. This can be modified to hang you in a waiting
// prompt, like this:
//
// user@apertureos:~$ blink install package
// INFO: Another instance of Blink is running. Waiting for it to finish...
// [waits until lock is removed]
// INFO: Lock released. Proceeding with installation...
// (this doesnt happen in support, version and other commands that dont modify the system or cant cause issues)
// TODO: implement this waiting prompt feature and the config shit
/****************************************************/

/********************************************************************************************************/

func addLock(lockPath string) error {

	if _, err := os.Stat(filepath.Join(defaultCachePath, "etc")); os.IsNotExist(err) {
		log.Printf("INFO: Lock directory does not exist. Creating...")
		os.MkdirAll(filepath.Join(defaultCachePath, "etc"), os.ModePerm)
	}

	log.Printf("INFO: Inserting lock file at %s...", lockPath)
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("ERROR: Failed to create lock file. Debug: %v", err)
		return err // lock already exists or another error
	}
	log.Printf("INFO: Lock Inserted successfully.")
	f.Close()
	return nil

}

/********************************************************************************************************/

func checkLock(lockPath string) bool {
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		return false // lock does not exist
	}
	log.Printf(`FATAL: A lock is inserted. Are there other Blink's 
				Instances running? To check run "ps aux | grep blink", if theres none feel free to remove 
				the lock file at "%s" using "sudo rm -rf %s"`, lockPath, lockPath)
	return true // lock exists
}

/********************************************************************************************************/

func removeLock(lockPath string) error {
	if err := os.Remove(lockPath); err != nil {
		log.Printf(`ERROR: Failed to remove lock file. 
					You might encounter issues when trying to use Blink again. When you do, 
					instructions will show up on how to solve this issue. Debug: %v`, err)
		return err // failed to remove lock
	}
	log.Printf("INFO: Lock file at %s deferred (removed) successfully.", lockPath)
	return nil
}

/********************************************************************************************************/

/****************************************************/
// getPkg downloads a package recipe from the repository and saves it to the specified path
// you can use this standalone to just download recipes if you want, but usually this is
// called internally by other functions, ensuring reusing code, modularity, and less repetition
/****************************************************/

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
		var response string
		fmt.Scanln(&response)
		response = strings.ToLower(response)

		if response == "n" || response == "no" {
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

/********************************************************************************************************/

/****************************************************/
// fetchPkg fetches a package recipe from cache or repository, decodes it, and displays package info
// in addition, it returns the PackageInfo struct for further use, so you can use this function to both
// get the struct and show the info to the user, avoiding code repetition and enhancing modularity
// avoids 2 functions for fetching and displaying info separately
/****************************************************/

func fetchpkg(path string, force bool, pkgName string) (PackageInfo, error) {

	if !strings.HasSuffix(path, string(os.PathSeparator)) {
		path += string(os.PathSeparator)
	}

	recipePath := filepath.Join(path, pkgName+".json")

	if force {
		if err := os.Remove(recipePath); err == nil {
			log.Printf("INFO: Force flag detected, removed cached recipe at %s", recipePath)
		} else if !os.IsNotExist(err) {
			log.Printf("WARNING: Failed to remove cached recipe. Debug: %v", err)
		}
	}

	if _, err := os.Stat(recipePath); os.IsNotExist(err) {
		log.Printf("INFO: Package recipe not found. Downloading...")
		if err := getpkg(pkgName, path); err != nil {
			return PackageInfo{}, err
		}
	}

	f, err := os.Open(recipePath)
	if err != nil {
		log.Printf("FATAL: Failed to open package recipe. Debug: %v", err)
		return PackageInfo{}, fmt.Errorf("error opening file: %v", err)
	}
	defer f.Close()

	var pkg PackageInfo
	if err := json.NewDecoder(f).Decode(&pkg); err != nil {
		log.Printf("FATAL: Failed to parse JSON. Debug: %v", err)
		return PackageInfo{}, fmt.Errorf("error decoding JSON: %v", err)
	}

	fmt.Printf("\nUsing repo: %s\n\nName: %s\nVersion: %s\nDescription: %s\nAuthor: %s\nLicense: %s\n",
		repoURL, pkg.Name, pkg.Version, pkg.Description, pkg.Author, pkg.License)

	return pkg, nil
}

/********************************************************************************************************/

/****************************************************/
// Simple directory check and creation function, useful for ensuring directories exist before operations
// Really useful for checking sourcePath, cachePath, etc.
// This avoids repetitive code and enhances readability, its a simple boilerplate function so i only use it
// for readability and modularity purposes, less repetition of code, dont expect rocket science from this, its
// probably the simplest function in this entire codebase lmao
/****************************************************/

func checkDirAndCreate(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create directory: %v", err)
		}
	}
	return nil
}

/********************************************************************************************************/

/****************************************************/
// getSource downloads the source code archive from the specified URL if it doesn't already exist or if force is true
// This function checks if the source file already exists in the sourcePath directory. If it does not exist or if the isForce flag is set to true,
// it performs an HTTP GET request to download the source archive from the provided URL.
// The downloaded file is saved in the sourcePath directory with its original filename.
// If the file already exists and isForce is false, it logs a warning and skips the download.
// This function returns an error if any step of the process fails, allowing for proper error handling
// in calling functions.
/****************************************************/

func getSource(url string, isForce bool) error {

	if _, err := os.Stat(filepath.Join(sourcePath, filepath.Base(url))); os.IsNotExist(err) || isForce { // if recipe does not exist or force is true, download

		if isForce { // if isForce is true, log it (isForce == true is useless because isForce already implies it exists and is true, so we simplify it to just isForce)
			log.Printf("INFO: Force flag detected, re-downloading source from %s", url)
		}

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

		checkDirAndCreate(sourcePath)

		// Create file to save source
		outFile, err := os.Create(filepath.Join(sourcePath, filepath.Base(url)))
		if err != nil {
			return fmt.Errorf("failed to create recipe file: %v", err)
		}
		defer outFile.Close()

		// Copy response body to file
		_, err = io.Copy(outFile, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to write recipe file: %v", err)
		}
	} else {
		log.Printf("WARNING: Source already exists, skipping download. Use --force or -f to re-download.")
	}

	return nil
}

/********************************************************************************************************/

/****************************************************/
// runCmd is another boilerplate function to run shell commands with error handling
// captures stderr output for meaningful error messages
// useful for running commands like tar, unzip, etc. with proper error handling
// i love this because it improves readability and modularity, less repetitive code
// and satisfies my KISS (Keep it simple stupid) principle, you just have a single function for running a command with ful on error handling
// without reusing the same code for 8 billion times
/****************************************************/

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)

	// Capture stderr for meaningful error messages
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command failed: %s %v\nstderr: %s\nerror: %w",
			name, args, stderr.String(), err)
	}

	return nil
}

/********************************************************************************************************/

/****************************************************/
// lots of functions to extract shit and more shit and other shit
// more extensions functions ah too bad i dont give a shit about
// what they actually do
/****************************************************/

func baseNameWithoutExt(archive string) string {
	name := filepath.Base(archive)

	exts := []string{
		".tar.gz",
		".tar.xz",
		".tar.bz2",
		".tar.zst",
		".tgz",
		".zip",
		".tar",
	}

	for _, ext := range exts {
		if strings.HasSuffix(name, ext) {
			return strings.TrimSuffix(name, ext)
		}
	}
	return name
}

func extractDirFor(archive, destRoot string) (string, error) {
	dir := filepath.Join(destRoot, baseNameWithoutExt(archive))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func extractTar(r io.Reader, dest string) error {
	tr := tar.NewReader(r)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, hdr.Name)

		switch hdr.Typeflag {

		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}

		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}

			out, err := os.Create(target)
			if err != nil {
				return err
			}

			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()

		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return err
			}

		case tar.TypeLink:
			linkTarget := filepath.Join(dest, hdr.Linkname)
			if err := os.Link(linkTarget, target); err != nil {
				return err
			}
		default:
			// ignore other types
		}
	}
}

func extractTarOnly(path, dest string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return extractTar(f, dest)
}

func extractTarGz(path, dest string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	return extractTar(gzr, dest)
}

func extractTarXz(path, dest string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	xzr, err := xz.NewReader(f)
	if err != nil {
		return err
	}

	return extractTar(xzr, dest)
}

func extractTarBz2(path, dest string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	bzr := bzip2.NewReader(f)
	return extractTar(bzr, dest)
}

func extractTarZst(path, dest string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	zr, err := zstd.NewReader(f)
	if err != nil {
		return err
	}
	defer zr.Close()

	return extractTar(zr, dest)
}

func extractZip(path, dest string) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {

		target := filepath.Join(dest, f.Name)

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		out, err := os.Create(target)
		if err != nil {
			rc.Close()
			return err
		}

		if _, err := io.Copy(out, rc); err != nil {
			out.Close()
			rc.Close()
			return err
		}

		out.Close()
		rc.Close()
	}

	return nil
}

/********************************************************************************************************/

/****************************************************/
// This takes in a PackageInfo struct and a URL, checks if the source
// is already extracted, if not, it extracts the source based on the
// specified type (tar, zip, etc.) uses the previous funcs for extraction
// improves modularity and readability by encapsulating extraction logic in a single function
/****************************************************/

func decompressSource(pkg PackageInfo, force bool) error {
	archive := filepath.Join(sourcePath, filepath.Base(pkg.Source.URL))

	// Determine the extraction folder based on archive name
	dest, err := extractDirFor(archive, sourcePath)
	if err != nil {
		return err
	}

	// If force is false and folder exists, skip extraction
	if !force {
		if _, err := os.Stat(dest); err == nil {
			return nil
		}
	}

	// If force is true, remove the existing folder first
	if force {
		if _, err := os.Stat(dest); err == nil {
			if err := os.RemoveAll(dest); err != nil {
				return fmt.Errorf("failed to remove existing folder: %w", err)
			}
			// recreate the folder
			if err := os.MkdirAll(dest, 0o755); err != nil {
				return fmt.Errorf("failed to recreate folder: %w", err)
			}
		}
	}

	// Extract based on type
	switch pkg.Source.Type {
	case "tar":
		return extractTarOnly(archive, dest)
	case "tar.gz", "tgz":
		return extractTarGz(archive, dest)
	case "tar.xz":
		return extractTarXz(archive, dest)
	case "tar.bz2":
		return extractTarBz2(archive, dest)
	case "tar.zst":
		return extractTarZst(archive, dest)
	case "zip":
		return extractZip(archive, dest)
	default:
		return fmt.Errorf("unknown source type: %q", pkg.Source.Type)
	}
}

/********************************************************************************************************/

/****************************************************/
// install function downloads, decompresses, builds, and installs a package
// it fetches package info, downloads source, decompresses it
// it uses the getSource, decompressSource functions for modularity and to satisfy my KISS principle
// i wish golang had macros so i could avoid writing the same error handling code every single time and just have a single line for it
/****************************************************/

func install(pkgName string, force bool, path string) error {

	// fetchPkg + error handling

	log.Printf("INFO: Fetching package info for %s", pkgName)
	pkg, err := fetchpkg(path, force, pkgName)
	if err != nil {
		return err // Return error if fetching package fails
	}

	// getSource + error handling

	log.Printf("INFO: Downloading source from %s", pkg.Source.URL)
	if err := getSource(pkg.Source.URL, force); err != nil {
		return err // Return error if decompressing source fails
	}

	// decompressSource + error handling

	log.Printf("INFO: Decompressing source of type %s", pkg.Source.Type)
	if err := decompressSource(pkg, force); err != nil {
		return err // Return error if decompressing source fails
	} // TODO: Implement build and install steps here
	return nil // Return nil to indicate success
}

/*
 *  i wonder what "finding hidden gems in blink source code" would feel like lmao
 *  well just know that this code is open source, so feel free to explore it and find any hidden gems
 *  or pull request and add a couple ;)
 */

/********************************************************************************************************/

/****************************************************/
// main function - entry point of the Blink package manager
// sets up Cobra CLI commands and flags
// handles user input and executes corresponding functions
// provides commands for downloading, fetching info, installing packages
// also includes support and version commands
// uses modular functions for package operations to enhance readability and maintainability
/****************************************************/

// these comments on the main func should prob be added but theyre so boring so imma skip them
// if anyone wanna add them feel free ;)
func main() {

	log.SetFlags(log.Ltime)         // Log only time, no date as its useless
	log.SetPrefix("[Blink Debug] ") // Log prefix for debug messages (eg. [Blink Debug] INFO: { ... } )

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
				path = filepath.Join(defaultCachePath, "recipes")
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
				path = filepath.Join(defaultCachePath, "recipes")
			}
			if _, err := fetchpkg(path, force, pkgName); err != nil {
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
				path = filepath.Join(defaultCachePath, "recipes")
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
			fmt.Printf("%s", supportPage)
		},
	}

	// ─── blink version ─────────────────────────────────────────────────────────
	versionCmd := &cobra.Command{
		Use:     "version",
		Aliases: []string{"v", "ver", "--version", "-v"},
		Short:   "Show Blink version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s", versionPage)
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
		log.Printf("! PANIC ! : CobraCLI Root command failed to run\nDEBUG: %v ", err)
		os.Exit(1)
	}
}

/********************************************************************************************************/

// if ur reading this pls contribute to the repository if its out :sob:
