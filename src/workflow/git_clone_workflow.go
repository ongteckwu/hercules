package workflow

import (
	"fmt"
	"hercules/src/git_repo"
	"hercules/src/util"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func RunGitCloneWorkflow(repoUrl string) error {
	// Create a temporary directory
	dir, err := os.MkdirTemp("", util.TEMP_REPO_PREFIX)
	if err != nil {
		log.Fatalf("Error creating temp directory: %v", err)
	}
	defer util.Cleanup(dir)

	// Setup to catch interrupt or kill signal
	sigChannel := make(chan os.Signal, 1)
	signal.Notify(sigChannel, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChannel)

	// Cleanup on interrupt or kill signal
	go func() {
		<-sigChannel
		util.Cleanup(dir)
		os.Exit(0)
	}()

	git_repo.GitClone(repoUrl, dir)

	repoName, err := git_repo.GetRepoNameFromUrl(repoUrl)
	if err != nil {
		return fmt.Errorf("error getting repo name from URL: %v", err)
	}
	RunWorkflow(dir, repoName, true)
	return nil
}
