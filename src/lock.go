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
	"os"
	"path/filepath"
	"github.com/Aperture-OS/eyes"
)

/***************************************************/
// check if running as root (user id 0), exit if not
/***************************************************/

func requireRoot() {
	if os.Geteuid() != 0 {
		eyes.Fatalf(`This command must be run as Root or Super User (also known as Admin, Administrator, SU, etc.)
		Please try again with 'sudo' infront of the command or as the root user ('su -').
		`)
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
// user@blinkTesting:~$ blink install package
// INFO: Another instance of Blink is running. Waiting for it to finish...
// [waits until lock is removed]
// INFO: Lock released. Proceeding with installation...
// (this doesnt happen in support, version and other commands that dont modify the system or cant cause issues)
// TODO: implement this waiting prompt feature and the config shit
/****************************************************/


/****************************************************/
// the following functions do as they say. Add, check and remove lock
/****************************************************/
func addLock(lockPath string) error {

	if _, err := os.Stat(filepath.Join(defaultCachePath, "etc")); os.IsNotExist(err) {
		eyes.Infof("Lock directory does not exist. Creating...")
		os.MkdirAll(filepath.Join(defaultCachePath, "etc"), 0755)
	}

	eyes.Infof("Inserting lock file at %s...", lockPath)
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0555)
	if err != nil {
		return err
	}
	defer f.Close()
	fmt.Fprintf(f, "%d\n", os.Getpid())

	eyes.Infof("Lock Inserted successfully.")
	f.Close()
	return nil

}



func checkLock(lockPath string) bool {
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		return false // lock does not exist
	}
	eyes.Errorf(`A lock is inserted. Are there other Blink's 
Instances running? To check run "ps aux | grep blink", if theres none feel free to remove 
the lock file at "%s" using "sudo rm -rf %s"`, lockPath, lockPath)
	return true // lock exists
}



func removeLock(lockPath string) error {
	if err := os.Remove(lockPath); err != nil {
		eyes.Errorf(`Failed to remove lock file. 
You might encounter issues when trying to use Blink again. When you do, 
instructions will show up on how to solve this issue.
ERR: %v`, err)
		return err // failed to remove lock
	}
	eyes.Infof("Lock file at %s deferred (removed) successfully.", lockPath)
	return nil
}

