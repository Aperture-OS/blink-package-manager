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
		Kind      string            `json:"kind"`      // toCompile or preCompiled
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
	Name    string `json:"name"`
	Version string `json:"version"`
	Release int64  `json:"release"`
}

/****************************************************/
// RepoConfig represents the structure of the config.toml
/****************************************************/
type RepoConfig struct {
	Name string
	URL  string
	Ref  string
}
