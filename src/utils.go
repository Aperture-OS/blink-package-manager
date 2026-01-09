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
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/Aperture-OS/eyes"
)

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

/****************************************************/
// clean cleans the data folders like recipes and allat, yes thats it
/****************************************************/

func clean() error {

	eyes.Warn("Are you sure you want to delete the cached recipes and sources? [ (Y)es / (N)o ]: ")
	var response string
	fmt.Scanln(&response)

	response = strings.ToLower(response)
	response = strings.TrimSpace(response)

	switch response {

	case "y", "yes", "sure", "yep", "ye", "yea", "yeah", "", "\n":
		eyes.Infof("Acquiring lock at %s", lockPath)
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
		eyes.Fatalf("\nUser declined, exiting...")

	}

	return nil

}

/****************************************************/
// normalizeYesNo takes in a string and returns "yes" or "no" 
// simplified with lowering the input's case and trimming any space
// its a boilerplate function to normalize user input, but its better
// for kiss and readability.
/****************************************************/
func normalizeYesNo(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "n", "no":
		return "no"
	default:
		return "yes"
	}
}
