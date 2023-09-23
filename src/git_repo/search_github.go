package git_repo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type GitHubItem struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
}

type GitHubSearchResult struct {
	TotalCount int          `json:"total_count"`
	Items      []GitHubItem `json:"items"`
}

const maxRetries = 100

func SearchGitHub(query string, numberOfQueries int) (GitHubSearchResult, error) {
	var result GitHubSearchResult

	for i := 0; i < maxRetries; i++ {
		url := fmt.Sprintf("https://api.github.com/search/code?q=%s&per_page=%d", query, numberOfQueries)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return result, err
		}
		req.Header.Set("Accept", "application/vnd.github.v3+json")

		token := os.Getenv("GITHUB_TOKEN")
		if token != "" {
			req.Header.Set("Authorization", "token "+token)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return result, err
		}
		defer resp.Body.Close()

		if resp.StatusCode == 403 {
			// If you hit API rate limit, implement backoff
			wait := time.Duration(10) * time.Second
			fmt.Printf("Hit rate limit. Retrying in %v seconds...\n", wait.Seconds())
			time.Sleep(wait)
			continue
		}

		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			return result, err
		}

		return result, nil
	}

	return result, fmt.Errorf("max retries reached. Could not complete the API request")
}

func FetchRawFileFromGitHub(item GitHubItem) (string, error) {
	// Build the URL to fetch the raw file content
	url := fmt.Sprintf("https://api.github.com/repos/%s/contents/%s", item.Repository.FullName, item.Path)

	// Create an HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3.raw")

	// Add authorization header if token exists
	token := os.Getenv("GITHUB_TOKEN")
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	// Create an HTTP client and perform the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read the response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Convert the data to a string
	return string(data), nil
}
