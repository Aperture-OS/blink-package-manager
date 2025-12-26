/*
  Blink, a powerful source-based package manager. Core of ApertureOS.
	Want to use it for your own project?
	Blink is completely FOSS (Free and Open Source),
	edit, publish, use, contribute to Blink however you prefer.
  Copyright (C) 2025 Aperture OS

  This program is free software: you can redistribute it and/or modify
  it under the terms of the GNU General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  This program is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.

  You should have received a copy of the GNU General Public License
  along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package main // main package, entry point

import (
	"bytes"         // For capturing command output
	"crypto/sha256" // some sha shit
	"encoding/hex"  // same here ^
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
)

//===================================================================//
//							    Structs
//===================================================================//

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

/****************************************************/
// Manifest represents Blink's installed package database
/****************************************************/

type Manifest struct {
	Installed []InstalledPkg `json:"installed"`
}

/****************************************************/
// InstalledPkg represents a package entry in the manifest
/****************************************************/
type InstalledPkg struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Release     int64  `json:"release"`
	InstalledAt int64  `json:"installed_at"`
}

//===================================================================//
//							     Globals
//===================================================================//

var (
	repoURL = "https://github.com/Aperture-OS/testing-blink-repo/blob/main/pseudoRepo" // Display repo URL

	baseURL = "https://raw.githubusercontent.com/Aperture-OS/testing-blink-repo/refs/heads/main/pseudoRepo/" // Raw JSON base URL for the repository

	// TODO: Change to actual repo when releasing blink
	// TODO: Change to a file instead of variable

	defaultCachePath = "./blink/" // Default: /var/blink/

	currentYear = time.Now().Year() // Current year for copyright

	Version = "v0.0.4-alpha" // Blink version

	lockPath = filepath.Join(defaultCachePath, "etc", "blink.lock") // Path to lock file

	supportPage = // Support information string

	`Having trouble? Join our Discord Server or open a GitHub issue.
	Include any DEBUG INFO logs when reporting issues.
	Discord: https://discord.com/invite/rx82u93hGD
	GitHub Issues: https://github.com/Aperture-OS/Blink-Package-Manager/issues`

	sourcePath = filepath.Join(defaultCachePath, "sources") // Path to downloaded source

	recipePath = filepath.Join(defaultCachePath, "recipes")

	manifestPath = filepath.Join(defaultCachePath, "etc", "manifest.json")

	buildRoot = filepath.Join(defaultCachePath, "build")

	versionPage = // Version information string
	fmt.Sprintf(`Blink Package Manager - Version %s 
	Licensed under GPL v3.0 by Aperture OS
	https://aperture-os.github.io
	All rights reserved. © Copyright 2025-%d Aperture OS.
	`, Version, currentYear)  // return the formatted string
) // TODO: migrate to /var/blink

//===================================================================//
//							  Functions
//===================================================================//

/***************************************************/
// check if running as root (user id 0), exit if not
/***************************************************/

func requireRoot() {
	if os.Geteuid() != 0 {
		fmt.Fprintf(os.Stderr, `
FATAL: This command must be run as Root or Super User (also known as Admin, Administrator, SU, etc.)
Please try again with 'sudo' infront of the command or as the root user ('su -').
`)
		os.Exit(1)
	}
}

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
		os.MkdirAll(filepath.Join(defaultCachePath, "etc"), 0755)
	}

	log.Printf("INFO: Inserting lock file at %s...", lockPath)
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0555)
	if err != nil {
		log.Printf("ERROR: Failed to create lock file.\nERR: %v", err)
		fmt.Fprintf(f, "%d\n", os.Getpid()) // write PID into the lock file
		return err                          // lock already exists or another error
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
	instructions will show up on how to solve this issue.
	ERR: %v`, err)
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

	log.Printf("INFO: Getting package recipe.")

	log.Printf("INFO: Acquiring lock at %s", lockPath)
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

	checkDirAndCreate(path)
	checkDirAndCreate(filepath.Join(path, "recipes"))

	// Full path to recipe
	recipePath := filepath.Join(path, "recipes", pkgName+".json")

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

	log.Printf("INFO: Package recipe downloaded to %s", recipePath)

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

	log.Printf("INFO: Fetching package %q", pkgName)

	if !strings.HasSuffix(path, string(os.PathSeparator)) {
		path += string(os.PathSeparator)
	}

	recipePath := filepath.Join(path, "recipes", pkgName+".json")

	if force {
		if err := os.Remove(recipePath); err == nil {
			log.Printf("INFO: Force flag detected, removed cached recipe at %s", recipePath)
		} else if !os.IsNotExist(err) {
			log.Printf("WARNING: Failed to remove cached recipe.\nERR: %v", err)
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
		log.Printf("FATAL: Failed to open package recipe.\nERR: %v", err)
		return PackageInfo{}, fmt.Errorf("error opening file: %v", err)
	}
	defer f.Close()

	var pkg PackageInfo
	if err := json.NewDecoder(f).Decode(&pkg); err != nil {
		log.Printf("FATAL: Failed to parse JSON.\nERR: %v", err)
		return PackageInfo{}, fmt.Errorf("error decoding JSON: %v", err)
	}

	fmt.Printf("\nRepository: %s\n\nName: %s\nVersion: %s\nDescription: %s\nAuthor: %s\nLicense: %s\n",
		repoURL, pkg.Name, pkg.Version, pkg.Description, pkg.Author, pkg.License)

	log.Printf("INFO: Package fetching completed.")

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
// This takes in a PackageInfo struct and a URL, checks if the source
// is already extracted, if not, it extracts the source based on the
// specified type (tar, zip, etc.) uses the previous funcs for
// improves modularity and readability by encapsulating extraction logic in a single function
/****************************************************/

func decompressSource(pkg PackageInfo, dest string) error {

	log.Printf("INFO: Decompressing source for %s into %s", pkg.Name, dest)

	srcFile := filepath.Join(sourcePath, filepath.Base(pkg.Source.URL))

	if _, err := os.Stat(srcFile); err != nil {
		return fmt.Errorf("source archive not found: %s", srcFile)
	}

	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	var cmd *exec.Cmd

	switch {
	case strings.HasSuffix(srcFile, ".tar.gz"), strings.HasSuffix(srcFile, ".tgz"):
		cmd = exec.Command("tar", "-xzf", srcFile, "-C", dest)

	case strings.HasSuffix(srcFile, ".tar.xz"):
		cmd = exec.Command("tar", "-xJf", srcFile, "-C", dest)

	case strings.HasSuffix(srcFile, ".tar.bz2"):
		cmd = exec.Command("tar", "-xjf", srcFile, "-C", dest)

	case strings.HasSuffix(srcFile, ".zip"):
		cmd = exec.Command("unzip", "-q", srcFile, "-d", dest)

	default:
		return fmt.Errorf("unsupported archive format: %s", srcFile)
	}

	log.Printf("INFO: Running extract command: %v", cmd.Args)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

/********************************************************************************************************/

/****************************************************/
// postExtractDir returns the actual build directory inside dest.
// If the archive extracted exactly one directory, it returns that.
// Otherwise, it returns dest itself.
/****************************************************/

func postExtractDir(extractRoot string) (string, error) {
	log.Printf("INFO: Scanning extract root %s", extractRoot)

	entries, err := os.ReadDir(extractRoot)
	if err != nil {
		return "", err
	}

	if len(entries) == 1 && entries[0].IsDir() {
		dir := filepath.Join(extractRoot, entries[0].Name())
		log.Printf("INFO: Using single top-level dir %s", dir)
		return dir, nil
	}

	log.Printf("INFO: Using extract root as build dir")
	return extractRoot, nil
}

/********************************************************************************************************/

/****************************************************/
// compareSHA256 takes in a expectedHash (so a string which is a sha256), and
// a file, it decodes the file's hash and checks if it matches the expectedHash,
/****************************************************/

func compareSHA256(expectedHash, file string) (bool, error) { // takes a expectedHash and a file, it generates the file's sha256 and compares it with expectedHash
	f, err := os.Open(file)
	if err != nil {
		return false, err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false, err
	}

	actual := hex.EncodeToString(h.Sum(nil))
	return strings.EqualFold(actual, expectedHash), nil
}

/********************************************************************************************************/

/****************************************************/
// Manifest creation and dependency handling functions
// a manifest is a JSON file that keeps track of installed packages, their versions, and other metadata
// this is useful for managing installed packages, checking for updates, and handling dependencies
// the manifest will be stored in /var/blink/etc/manifest.json (see variable delarations at the start of the file)
/****************************************************/

func ensureManifest() error {
	log.Printf("INFO: Ensuring manifest exists at %s", manifestPath)

	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		return err
	}

	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		m := Manifest{Installed: []InstalledPkg{}}
		data, _ := json.MarshalIndent(m, "", "  ")
		return os.WriteFile(manifestPath, data, 0644)
	}

	return nil
}

func loadManifest() (Manifest, error) {
	log.Printf("DEBUG: loading manifest")

	var m Manifest

	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return Manifest{Installed: []InstalledPkg{}}, nil
	}

	f, err := os.Open(manifestPath)
	if err != nil {
		return m, err
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&m); err != nil {
		return m, err
	}

	return m, nil
}

func saveManifest(m Manifest) error {
	log.Printf("DEBUG: saving manifest (%d packages)", len(m.Installed))

	// ENSURE DIRECTORY EXISTS (THIS WAS MISSING)
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		return err
	}

	tmp := manifestPath + ".tmp"

	f, err := os.Create(tmp)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")

	if err := enc.Encode(&m); err != nil {
		f.Close()
		return err
	}
	f.Close()

	return os.Rename(tmp, manifestPath)
}

func manifestHas(name string) (*InstalledPkg, bool, error) {
	m, err := loadManifest()
	if err != nil {
		return nil, false, err
	}

	for _, p := range m.Installed {
		if p.Name == name {
			return &p, true, nil
		}
	}

	return nil, false, nil
}

func addToManifest(pkg PackageInfo) error {
	log.Printf("INFO: adding %s to manifest", pkg.Name)

	m, err := loadManifest()
	if err != nil {
		return err
	}

	for _, p := range m.Installed {
		if p.Name == pkg.Name {
			log.Printf("WARN: %s already recorded in manifest", pkg.Name)
			return nil
		}
	}

	m.Installed = append(m.Installed, InstalledPkg{
		Name:        pkg.Name,
		Version:     pkg.Version,
		Release:     int64(pkg.Release),
		InstalledAt: time.Now().Unix(),
	})

	return saveManifest(m)
}

/****************************************************/
// install function downloads, decompresses, builds, and installs a package
// it fetches package info, downloads source, decompresses it
// it uses the getSource, decompressSource functions for modularity and to satisfy my KISS principle
// i wish golang had macros so i could avoid writing the same error handling code every single time and just have a single line for it
/****************************************************/

func install(pkgName string, force bool, path string) error {
	log.Printf("INFO: ===== INSTALL START =====")
	log.Printf("INFO: pkg=%s force=%v", pkgName, force)

	// manifest must exist BEFORE touching it
	if err := ensureManifest(); err != nil {
		return err
	}

	// fetch recipe
	pkg, err := fetchpkg(path, force, pkgName)
	if err != nil {
		return err
	}

	installed, exists, err := manifestHas(pkg.Name)
	if err != nil {
		return err
	}

	if exists && !force {
		return fmt.Errorf(
			"package %s already installed (version=%s release=%d)",
			installed.Name,
			installed.Version,
			installed.Release,
		)
	}

	// prepare build root
	if err := os.MkdirAll(buildRoot, 0755); err != nil {
		return err
	}

	extractRoot := filepath.Join(buildRoot, pkg.Name)
	log.Printf("INFO: extract root = %s", extractRoot)

	_ = os.RemoveAll(extractRoot)
	if err := os.MkdirAll(extractRoot, 0755); err != nil {
		return err
	}

	// download source
	if err := getSource(pkg.Source.URL, force); err != nil {
		return err
	}

	srcFile := filepath.Join(sourcePath, filepath.Base(pkg.Source.URL))
	ok, err := compareSHA256(pkg.Source.Sha256, srcFile)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("source hash mismatch for %s", srcFile)
	}

	// extract
	if err := decompressSource(pkg, extractRoot); err != nil {
		return err
	}

	buildDir, err := postExtractDir(extractRoot)
	if err != nil {
		return err
	}

	log.Printf("INFO: build dir = %s", buildDir)

	if err := os.Chdir(buildDir); err != nil {
		return err
	}

	// env
	for k, v := range pkg.Build.Env {
		log.Printf("DEBUG: env %s=%s", k, v)
		os.Setenv(k, v)
	}

	// prepare
	for _, cmd := range pkg.Build.Prepare {
		log.Printf("INFO: prepare → %s", cmd)
		if err := runCmd("sh", "-c", cmd); err != nil {
			return err
		}
	}

	// install
	for _, cmd := range pkg.Build.Install {
		log.Printf("INFO: install → %s", cmd)
		if err := runCmd("sh", "-c", cmd); err != nil {
			return err
		}
	}

	// record install (THIS was missing/broken before)
	if err := addToManifest(pkg); err != nil {
		return err
	}

	log.Printf("INFO: ===== INSTALL COMPLETE =====")
	return nil
}

/*
 *  i wonder what "finding hidden gems in blink source code" would feel like lmao
 *  well just know that this code is open source, so feel free to explore it and find any hidden gems
 *  or pull request and add a couple ;) (no mister "i wanna contribute to foss", this doesnt count as a
 *  proper contribution but if u add gems good job!)
 */

/********************************************************************************************************/

/****************************************************/
// clean cleans the cache folders, yes thats it
/****************************************************/

func clean() error {

	fmt.Printf("WARNING: Are you sure you want to delete the cached recipes and sources? [ (Y)es / (N)o ] ")
	var response string
	fmt.Scanln(&response)

	response = strings.ToLower(response)
	response = strings.TrimSpace(response)

	switch response {

	case "y", "yes", "sure", "yep", "ye", "yea", "yeah", "", " ", "  ", "   ", "\n":
		log.Printf("INFO: Acquiring lock at %s", lockPath)
		if checkLock(lockPath) {
			return fmt.Errorf("another instance is running, lock file exists at %s", lockPath)
		}

		lockErr := addLock(lockPath) // add lock file
		defer removeLock(lockPath)   // remove lock file at the end

		if lockErr != nil { // check for errors while adding lock
			return fmt.Errorf("failed to create lock file at %s: %v", lockPath, lockErr)
		}

		os.RemoveAll(recipePath)
		os.MkdirAll(recipePath, 0755)

		os.RemoveAll(sourcePath)
		os.MkdirAll(sourcePath, 0755)

		os.RemoveAll(buildRoot)
		os.MkdirAll(buildRoot, 0755)

	default:
		log.Fatalf("\nFATAL: User declined, exiting...")

	}

	return nil

}

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

	//  blink get <pkg>
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

	//  blink info <pkg>
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

	//  blink install <pkg>
	installCmd := &cobra.Command{
		Use:     "install <pkg>",
		Short:   "Download and install a package",
		Args:    cobra.ExactArgs(1),
		Aliases: []string{"i", "add", "inst"},
		Run: func(cmd *cobra.Command, args []string) {

			requireRoot() // ensure running as root

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

	//  blink support
	supportCmd := &cobra.Command{
		Use:     "support",
		Aliases: []string{"issue", "bug", "contact", "discord", "s", "-s", "--support", "--bug"},
		Short:   "Show support information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s", supportPage)
		},
	}

	cleanCmd := &cobra.Command{
		Use:     "clean",
		Aliases: []string{"cleanup", "clear", "c", "-c", "--clean", "--cleanup"},
		Short:   "Clean cache info.",
		Run: func(cmd *cobra.Command, args []string) {

			requireRoot() // ensure running as root

			clean()
		},
	}

	//  blink version
	versionCmd := &cobra.Command{
		Use:     "version",
		Aliases: []string{"v", "ver", "--version", "-v"},
		Short:   "Show Blink version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s", versionPage)
		},
	}

	// Add commands to root
	rootCmd.AddCommand(getCmd, infoCmd, installCmd, supportCmd, versionCmd, cleanCmd)

	// Disable default Cobra completion
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	//  Shell completion command
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
	fmt.Printf("Blink Package Manager Version: %s\n", Version)
	fmt.Printf("© Copyright 2025-%d Aperture OS. All rights reserved.\n", currentYear)

	// Execute root command
	if err := rootCmd.Execute(); err != nil {
		log.Printf("FATAL: Command Line Interface failed to run. (Is there any syntax error(s)?)\nERR: %v ", err)
		os.Exit(1)
	}
}

/********************************************************************************************************/

// if ur reading this pls contribute to the repository if its out :sob:
