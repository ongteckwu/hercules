package workflow

import (
	"fmt"
	"hercules/src/code_parser"
	"hercules/src/git_repo"
	"hercules/src/similarity_compute"
	"hercules/src/tfidf"
	"hercules/src/util"
	"path/filepath"
	"strings"
	"sync"

	"github.com/wilcosheh/tfidf/similarity"
)

type MiniParseCodeWorkflowScanResult struct {
	repositoryName     string
	tfidfSimilarity    float64
	levenSimilarity    float64
	combinedSimilarity float64
}

const NUMBER_OF_PAGES_TO_SEARCH = 10
const TEXT_MAX_LENGTH = 25000

func MiniParseCodeWorkflow(
	repoName string,
	path string,
	codeText string,
	keywordsTFIDF *tfidf.TFIDF,
	keywordsTFIDFMutex *sync.Mutex,
	charLevelTFIDF *tfidf.TFIDF,
	charLevelTFIDFMutex *sync.Mutex,
	possibleRepoMap map[string]int,
	possibleRepoMapMutex *sync.Mutex,
) error {
	fileExt := filepath.Ext(path)
	originalCodeTextLength := len(codeText)
	slicedCodeText := codeText[:util.Min(TEXT_MAX_LENGTH, originalCodeTextLength)] // to prevent OOM
	parsedCodeText := code_parser.ParseCodeText(slicedCodeText)
	keywordsTFIDFMutex.Lock()
	codeTextWeights := keywordsTFIDF.Cal(slicedCodeText)
	keywordsTFIDFMutex.Unlock()

	topKeywords := tfidf.GetTopNKeywordsTfIdf(4, codeTextWeights)
	var extQuery string
	if fileExt == "" {
		extQuery = ""
	} else {
		extQuery = "+language:" + fileExt[1:]
	}
	query := strings.Join(topKeywords, "+") + extQuery
	queryResults, err := git_repo.SearchGitHub(query, NUMBER_OF_PAGES_TO_SEARCH)
	if err != nil {
		fmt.Printf("Error searching GitHub: %v\n", err)
		return err
	}

	resultChannel := make(chan *MiniParseCodeWorkflowScanResult, len(queryResults.Items))
	wg := sync.WaitGroup{}
	// Semaphore to limit concurrency to 3 to reduce OOM.
	// Okay since bottleneck is rate limiter.
	sem := make(chan struct{}, 3)
	for _, item := range queryResults.Items {
		// skip own repo
		if item.Repository.FullName == repoName {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(item git_repo.GitHubItem) {
			defer wg.Done()
			defer func() { <-sem }()

			challengeeCodeText, err := git_repo.FetchRawFileFromGitHub(item)
			util.Check(err)
			// if challengeeCodeText is too long, or vice versa, ignore
			// 2x difference max
			if (len(challengeeCodeText) > originalCodeTextLength*2) || (originalCodeTextLength > len(challengeeCodeText)*2) {
				return
			}
			challengeeCodeText = challengeeCodeText[:util.Min(TEXT_MAX_LENGTH, len(challengeeCodeText))] // to prevent OOM

			parsedCodeTextToCompare := code_parser.ParseCodeText(challengeeCodeText)

			similarityResults := similarity_compute.ComputeLevenSimilarity(
				parsedCodeText,
				parsedCodeTextToCompare,
			)

			charLevelTFIDFMutex.Lock()
			charLevelTFIDF.AddDocs(&[]string{challengeeCodeText})
			w1 := charLevelTFIDF.Cal(parsedCodeText.ParsedCodeText)
			w2 := charLevelTFIDF.Cal(parsedCodeTextToCompare.ParsedCodeText)
			charLevelTFIDFMutex.Unlock()

			tfidfSimilarity := similarity.Cosine(w1, w2)

			result := MiniParseCodeWorkflowScanResult{
				repositoryName:     item.Repository.FullName,
				tfidfSimilarity:    tfidfSimilarity,
				levenSimilarity:    similarityResults.Percentage,
				combinedSimilarity: similarityResults.Percentage * tfidfSimilarity,
			}

			resultChannel <- &result
		}(item)
	}

	go func() {
		wg.Wait()
		close(resultChannel)
		close(sem)
	}()

	// collect results as they become available
	// and increase the count of the repo name
	count := 0
	for result := range resultChannel {
		if result.combinedSimilarity > COMBINED_SIMILARITY_THRESHOLD ||
			result.tfidfSimilarity > TFIDF_SIMILARITY_THRESHOLD ||
			result.levenSimilarity > LEVEN_SIMILARITY_THRESHOLD {
			possibleRepoMapMutex.Lock()
			possibleRepoMap[result.repositoryName]++
			possibleRepoMapMutex.Unlock()
			count++
		}
	}
	basePath := util.RemoveTempFilePath(path)
	fmt.Printf("Results for %s in: %d\n", basePath, count)
	return nil
}