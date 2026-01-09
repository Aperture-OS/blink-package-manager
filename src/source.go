/*
  Blink, a powerful source-based package manager. Core of ApertureOS.
	Want to use it for your own project?
	Blink is completely FOSS (Free and Open Source),
	edit, publish, use, contribute to Blink however you prefer.
  Copyright (C) 2025-2026 Aperture OS

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

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Aperture-OS/eyes"
)

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
			eyes.Infof("Force flag detected, re-downloading source from %s", url)
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
		eyes.Warnf("Source already exists, skipping download. Use --force or -f to re-download.")
	}

	return nil
}

/****************************************************/
// This takes in a PackageInfo struct and a URL, checks if the source
// is already extracted, if not, it extracts the source based on the
// specified type (tar, zip, etc.) uses the previous funcs for
// improves modularity and readability by encapsulating extraction logic in a single function
/****************************************************/

func decompressSource(pkg PackageInfo, dest string) error {

	eyes.Infof("Decompressing source for %s into %s", pkg.Name, dest)

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

	eyes.Infof("Running extract command: %v", cmd.Args)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

/****************************************************/
// postExtractDir returns the actual build directory inside dest.
// If the archive extracted exactly one directory, it returns that.
// Otherwise, it returns dest itself.
/****************************************************/

func postExtractDir(extractRoot string) (string, error) {
	eyes.Infof("Scanning extract root %s", extractRoot)

	entries, err := os.ReadDir(extractRoot)
	if err != nil {
		return "", err
	}

	if len(entries) == 1 && entries[0].IsDir() {
		dir := filepath.Join(extractRoot, entries[0].Name())
		eyes.Infof("Using single top-level dir %s", dir)
		return dir, nil
	}

	eyes.Infof("Using extract root as build dir")
	return extractRoot, nil
}

/****************************************************/
// safeExtractToRoot checks the extracted files for path traversal
// vulnerabilities and returns an error if any are found.
// It takes in a PackageInfo struct and the extractRoot directory
// and returns an error if any unsafe paths are found.
/****************************************************/

func safeExtractToRoot(pkg PackageInfo, extractRoot string) error {
	// reuse existing extractor
	if err := decompressSource(pkg, extractRoot); err != nil {
		return err
	}

	// walk extracted files and block path traversal
	return filepath.Walk(extractRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(extractRoot, path)
		if err != nil {
			return err
		}

		// no absolute paths, no ..
		if strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
			return fmt.Errorf("unsafe path detected in binary package: %s", path)
		}

		return nil
	})
}
