package arg_parser

import (
	"flag"
	"fmt"
	"hercules/src/git_repo"
	"hercules/src/workflow"
	"os"
	"path/filepath"
)

func ArgParser() {
	// Define flags
	var dir string
	var url string

	flag.StringVar(&dir, "dir", "", "The path to the directory.")
	flag.StringVar(&url, "url", "", "The GitHub URL.")

	// Parse the flags
	flag.Parse()

	// Check if either flag was provided
	if dir == "" && url == "" {
		fmt.Println("Please provide either a directory path using --dir=<DIR_PATH> or a GitHub URL using --url=<GITHUB_URL>")
		os.Exit(1)
	}

	// Check if both flags were provided
	if dir != "" && url != "" {
		fmt.Println("Do not provide both --dir and --url flags.")
		os.Exit(1)
	}

	if dir != "" {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			fmt.Printf("Error converting to full path: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Running execution on directory: " + absDir)
		workflow.RunWorkflow(absDir, absDir, false)
	}

	if url != "" {
		if git_repo.IsValidGitHubURL(url) {
			workflow.RunGitCloneWorkflow(url)
		} else {
			fmt.Println("The provided URL is not a Github url or not valid.")
			os.Exit(1)
		}
	}
}
