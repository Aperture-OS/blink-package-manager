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
	"path/filepath"
	"time"
)

//===================================================================//
//							     Globals
//===================================================================//

// !!! NEVER USE RELATIVE PATHS FOR GLOBALS !!!
// ALWAYS USE ABSOLUTE PATHS TO AVOID ISSUES
// EVEN WHEN TESTING LOCALLY, ALWAYS ABSOLUTE PATHS!

var (
	distroName = "ApertureOS"

	defaultCachePath = "/home/elia/Desktop/ApertureOS/blink/var-blink"      // Default: /var/blink/
	currentYear      = time.Now().Year()                                    // Current year for copyright
	Version          = "v0.1.0-alpha"                                       // Blink version
	lockPath         = filepath.Join(defaultCachePath, "etc", "blink.lock") // Path to lock file

	configPath        = filepath.Join(defaultCachePath, "etc", "config.toml")
	defaultRepoConfig = `
[pseudoRepository]
git_url = "https://github.com/Aperture-OS/testing-blink-repo.git"
branch = "main"
`

	repoCachePath = filepath.Join(defaultCachePath, "repositories")
	sourcePath    = filepath.Join(defaultCachePath, "sources") // Path to downloaded source
	recipePath    = filepath.Join(defaultCachePath, "recipes")
	manifestPath  = filepath.Join(defaultCachePath, "etc", "manifest.toml")
	buildRoot     = filepath.Join(defaultCachePath, "build")

	supportPage = // Support information string
	`Having trouble? Join our Discord Server or open a GitHub issue.
	Include any DEBUG INFO logs when reporting issues.
	Discord: https://discord.com/invite/rx82u93hGD
	GitHub Issues: https://github.com/Aperture-OS/Blink-Package-Manager/issues`

	versionPage = // Version information string
	fmt.Sprintf(`Blink Package Manager - Version %s 
	Licensed under GPL v3.0 by Aperture OS
	https://aperture-os.github.io
	All rights reserved. © Copyright 2025-%d Aperture OS.
	`, Version, currentYear)  // return the formatted string
) // TODO: migrate to /var/blink
