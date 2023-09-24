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
	RepositoryName      string
	NumberOfLinesCopied int
	TFIDFSimilarity     float64
	LevenSimilarity     float64
	CombinedSimilarity  float64
}

const NUMBER_OF_FILES_TO_QUERY = 10

func ParseCodeWorkflow(
	repoName string,
	path string,
	isTempPath bool,
	codeText string,
	keywordsTFIDF *tfidf.TFIDF,
	keywordsTFIDFMutex *sync.Mutex,
	charLevelTFIDF *tfidf.TFIDF,
	charLevelTFIDFMutex *sync.Mutex,
	possibleRepoMap map[string][]*MiniParseCodeWorkflowScanResult,
	possibleRepoMapMutex *sync.Mutex,
) error {
	fileExt := filepath.Ext(path)
	parsedCodeText := code_parser.ParseCodeText(codeText)
	keywordsTFIDFMutex.Lock()
	codeTextWeights := keywordsTFIDF.Cal(codeText)
	keywordsTFIDFMutex.Unlock()

	topKeywords := tfidf.GetTopNKeywordsTfIdf(4, codeTextWeights)
	var extQuery string
	if fileExt == "" {
		extQuery = ""
	} else {
		extQuery = "+language:" + fileExt[1:]
	}
	query := strings.Join(topKeywords, "+") + extQuery
	queryResults, err := git_repo.SearchGitHub(query, NUMBER_OF_FILES_TO_QUERY)
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
			challengeeCodeText = challengeeCodeText[:util.Min(TEXT_MAX_LENGTH, len(challengeeCodeText))] // to prevent OOM
			util.Check(err)
			// if challengeeCodeText is too long, or vice versa, ignore
			// 2x difference max
			// since codeText and challengeeCodeText is already capped at TEXT_MAX_LENGTH
			// it's okay to do this comparison for the sake of no OOM
			if (len(challengeeCodeText) > len(codeText)*2) || (len(codeText) > len(challengeeCodeText)*2) {
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
				RepositoryName:      item.Repository.FullName,
				NumberOfLinesCopied: similarityResults.Text1SubstringIndexes.EndIndex - similarityResults.Text1SubstringIndexes.StartIndex,
				TFIDFSimilarity:     tfidfSimilarity,
				LevenSimilarity:     similarityResults.Percentage,
				CombinedSimilarity:  similarityResults.Percentage * tfidfSimilarity,
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
		if result.CombinedSimilarity > COMBINED_SIMILARITY_THRESHOLD ||
			result.TFIDFSimilarity > TFIDF_SIMILARITY_THRESHOLD ||
			result.LevenSimilarity > LEVEN_SIMILARITY_THRESHOLD {
			possibleRepoMapMutex.Lock()
			possibleRepoMap[result.RepositoryName] = append(possibleRepoMap[result.RepositoryName], result)
			possibleRepoMapMutex.Unlock()
			count++
		}
	}
	return nil
}
