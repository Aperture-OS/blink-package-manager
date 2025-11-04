package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

type PackageInfo struct {
	Name        string `json:"name"`        // self-explanatory
	Version     string `json:"version"`     // self-explanatory
	Release     int    `json:"release"`     // self-explanatory
	Description string `json:"description"` // self-explanatory
	Author      string `json:"author"`      // self-explanatory
	License     string `json:"license"`     // self-explanatory
	Source      struct {
		URL    string `json:"url"`    // URL of the source code
		Type   string `json:"type"`   // Type of source (e.g., zip, tarball, proprietary)
		Sha256 string `json:"sha256"` // SHA256 checksum for integrity verification
	} `json:"source"`

	Dependencies map[string]string `json:"dependencies"` // key-value pairs of dependencies and their versions

	OptDeps []struct {
		ID int `json:"id"` // unique identifier for the optional dependency list, you can have multiple opt deps,
		// but this id helps differentiate the groups you need to list upon installation
		Description string   `json:"description"` // description of what this optional dependency group is for
		Options     []string `json:"options"`     // list of optional dependencies
		Default     string   `json:"default"`     // default option to select if user doesn't specify
	} `json:"opt_dependencies"`

	Build struct {
		Env       map[string]string `json:"env"`       // environment variables for the build process
		Prepare   []string          `json:"prepare"`   // commands to prepare the build environment
		Install   []string          `json:"install"`   // commands to compile and install the package
		Uninstall []string          `json:"uninstall"` // commands to uninstall the package
	} `json:"build"`
}

var repoURL = "https://github.com/Aperture-OS/testing-blink-repo/blob/main/pseudoRepo" // general repo URL
var baseURL = "https://raw.githubusercontent.com/Aperture-OS/testing-blink-repo/refs/heads/main/pseudoRepo/" // add the package name.json at the end of this URL and you have the full URL to the package recipe
var cachePath = "./blink/" // path to cache directory

// getpkg gets the package recipe from the git repository
func getpkg() error {
	if _, err := os.Stat(cachePath); os.IsNotExist(err) { // check if cache directory exists
		response := "" // either:   y, n, yes, no, new line
		fmt.Printf("FATAL: Cache directory does not exist, this is needed to store package recipes (the files that give instructions on how to install a package). Would you like to create one? [Y/n] ")
		fmt.Scanln(&response) // get user input
		response = strings.ToLower(response) // convert to lowercase for easier comparison

		fmt.Printf("%s", response) // temp
		
		switch response { // if statement but better because it can handle multiple cases

			case "y", "yes", "": // if yes (default to yes on empty input):
				err := os.MkdirAll(cachePath, os.ModePerm) // create cache directory

				if err != nil {

					log.Printf("FATAL: Failed to create cache directory:\nDEBUG INFO: (include in support thread):\n%v", err)
					return fmt.Errorf("failed to create cache directory. DEBUG INFO: (include in support thread):\n\"%s\"", err)
				} // error handling

				log.Printf("Cache directory created successfully at %s", cachePath)

			case "n", "no": //if no:

				log.Printf("FATAL: Cache directory is required to proceed. Response: %q .Exiting.", response)
				return fmt.Errorf("declined to create cache directory. Response: \"%s\" ", response)

			default: // anything else:

				log.Printf("FATAL: Invalid response. Please enter 'y' or 'n'. Response: %q .Exiting.", response)
				return fmt.Errorf("invalid response. Response: \"%s\" ", response)
		
		} // end switch

	} // end if

	pkgName := os.Args[1] // get package name from command line argument
	downloader := exec.Command("curl", baseURL+pkgName+".json", "-o", cachePath+pkgName+".json") // curl command to download the package recipe
	out, err := downloader.CombinedOutput()
	if err != nil {
		log.Printf("FATAL: Failed to download package recipe:\n DEBUG INFO: (include in support thread):\n%v", err)
	}
	log.Printf("%s", out)
	return fmt.Errorf("%v", out)
}


// fetchpkg reads a JSON file and prints basic info.
// It handles errors.
func fetchpkg(path string) error {

	// Open file
	f, err := os.Open(path)
	// if the error is NOT empty, handle it by printing a debug message and cleaning.
	if err != nil {
		// Error opening file: maybe it doesn't exist or permission denied
		log.Printf("FATAL: Failed to open package recipe, if you're unsure why this is happening consider opening a support thread in our Discord server. For more info type ` blink support ` .") // self-explanatory message
		log.Printf("DEBUG INFO: (include in support thread): \n %v", err) // Print error details for debugging and support threads.
		return fmt.Errorf("error: %v", err) // run cleanup
	}
	defer f.Close() // close file when done

	// Decode JSON into struct
	var pkg PackageInfo                   // pkg = PackageInfo struct
	err = json.NewDecoder(f).Decode(&pkg) // Decode JSON from file into pkg struct
	if err != nil {
		// Handle bad JSON
		log.Printf("FATAL: Failed to parse JSON file. if you're unsure why this is happening consider opening a support thread in our Discord server. For more info type ` blink support ` .") // self-explanatory message
		log.Printf("DEBUG INFO: (include in support thread): \n %v", err) // Print error details for debugging and support threads.
		return fmt.Errorf("error: %v", err) // run cleanup
	}

	// Success — print package info
	log.Printf("DEBUG: Package Loaded Successfully.")
	log.Printf("\nDEBUG: Package URL: %s%s.json", baseURL, pkg.Name)
	fmt.Printf("\nUsing: %s\n\nName: %s\nVersion: %s\nDescription: %s\nAuthor: %s\nLicense: %s", repoURL, pkg.Name, pkg.Version, pkg.Description, pkg.Author, pkg.License) //temp
	return nil
}

func main() {
	// Call fetchpkg with path
	
}
