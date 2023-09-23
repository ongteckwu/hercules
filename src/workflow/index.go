package workflow

import (
	"fmt"
	"hercules/src/code_parser"
	"hercules/src/git_repo"
	"hercules/src/similarity_compute"
	"hercules/src/tfidf"
	"hercules/src/util"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"

	"github.com/olekukonko/tablewriter"

	"github.com/wilcosheh/tfidf/similarity"
)

const NO_OF_FILES_FOR_PARSING = 15

type RepoToRepoPotentialChallengeeData struct {
	path            string
	data            string
	tfidfSimilarity float64
}

type RepoToRepoMatchedChallengeeData struct {
	NumberOfLinesCopied int
	Path                string
	TFIDFSimilarity     float64
	LevenSimilarity     float64
	CombinedSimilarity  float64
}

type RepoToRepoHighestLikelihoodScores struct {
	RepoUrl                    string
	RepoName                   string
	TFIDFSimilarityWeighted    float64
	LevenSimilarityWeighted    float64
	CombinedSimilarityWeighted float64
}

const TFIDF_SIMILARITY_THRESHOLD = 0.8
const LEVEN_SIMILARITY_THRESHOLD = 0.7
const COMBINED_SIMILARITY_THRESHOLD = 0.45
const CHOOSE_TOP_N_REPOS = 8

func RunWorkflow(repoDir string, repoName string) {
	filePaths, err := util.GetFilePaths(repoDir)
	util.Check(err)

	filePaths = util.RemoveNonCodeFiles(filePaths)

	// read all the files and return a map of path to data
	allDataMap, err := util.MultipleFileRead(filePaths) // map[path]data
	util.Check(err)

	// create a slice of all the data
	allDataArray := make([]string, 0)
	for _, data := range allDataMap {
		allDataArray = append(allDataArray, data)
	}

	// create a char level tfidf with the files
	charLevelTFIDF := tfidf.New()
	charLevelTFIDF.AddDocs(allDataArray, tfidf.TokenizeCharLevelNoAlpha)
	charLevelTFIDFMutex := &sync.Mutex{}

	// create normal tfidf with the files
	keywordsTFIDF := tfidf.New()
	keywordsTFIDF.AddDocs(allDataArray)
	keywordsTFIDFMutex := &sync.Mutex{}

	randomlyDrawnFilesN := util.RandomDrawWithoutReplacement(filePaths, NO_OF_FILES_FOR_PARSING)

	// for each file, parse
	possibleRepoMap := make(map[string]int)
	possibleRepoMapMutex := &sync.Mutex{}

	for _, path := range randomlyDrawnFilesN {
		// dont need to goroutine since github has a rate limit
		MiniParseCodeWorkflow(
			repoName,
			path, allDataMap[path],
			keywordsTFIDF, keywordsTFIDFMutex,
			charLevelTFIDF, charLevelTFIDFMutex,
			possibleRepoMap, possibleRepoMapMutex)
	}

	possibleReposTopN := getTopNRepos(possibleRepoMap, CHOOSE_TOP_N_REPOS)

	var highlyLikelyRepos []RepoToRepoHighestLikelihoodScores

	// dont go routine each due to memory usage
	for _, challengeeRepoName := range possibleReposTopN {
		challengeeRepoUrl := "https://github.com/" + challengeeRepoName
		challengeeDir, err := os.MkdirTemp("", util.TEMP_REPO_PREFIX)
		if err != nil {
			log.Fatalf("Error creating temp directory: %v", err)
		}
		defer util.Cleanup(challengeeDir)

		// Setup to catch interrupt or kill signal
		sigChannel := make(chan os.Signal, 1)
		signal.Notify(sigChannel, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigChannel)

		// Cleanup on interrupt or kill signal
		go func() {
			<-sigChannel
			util.Cleanup(challengeeDir)
			os.Exit(0)
		}()

		git_repo.GitClone(challengeeRepoUrl, challengeeDir)
		filePaths, err := util.GetFilePaths(challengeeDir)
		util.Check(err)

		filePaths = util.RemoveNonCodeFiles(filePaths)

		// read all the files and return a map of path to data
		challengeeAllDataMap, err := util.MultipleFileRead(filePaths) // map[path]data
		util.Check(err)

		// create a slice of all the data
		challengeeAllDataArray := make([]string, 0)
		for _, data := range challengeeAllDataMap {
			challengeeAllDataArray = append(challengeeAllDataArray, data)
		}
		// if challengee has too many files compared to challenger, or vice versa, ignore
		// 2x difference max
		if (len(challengeeAllDataArray) > len(allDataArray)*2) || (len(allDataArray) > len(challengeeAllDataArray)*2) {
			continue
		}
		// create a char level tfidf with the files
		combinedCharLevelTFIDF := tfidf.New()
		combinedCharLevelTFIDF.AddDocs(challengeeAllDataArray, tfidf.TokenizeCharLevelNoAlpha)
		combinedCharLevelTFIDF.AddDocs(allDataArray, tfidf.TokenizeCharLevelNoAlpha)

		matchedMap := make(map[string]RepoToRepoMatchedChallengeeData) // map[challengePath]RepoToRepoMatchedChallengeeData
		tempMemory := make(map[string](map[string]float64))

		for path, data := range allDataMap {
			w1 := combinedCharLevelTFIDF.Cal(data)
			var challengeeArray []RepoToRepoPotentialChallengeeData
			for challengeePath, challengeeData := range challengeeAllDataMap {
				w2, ok := tempMemory[challengeePath]
				if !ok {
					w2 = combinedCharLevelTFIDF.Cal(challengeeData)
					tempMemory[challengeePath] = w2
				}
				similarity := similarity.Cosine(w1, w2)
				if similarity > 0.6 {
					obj := RepoToRepoPotentialChallengeeData{
						path:            challengeePath,
						data:            challengeeData,
						tfidfSimilarity: similarity,
					}
					challengeeArray = append(challengeeArray, obj)
				}
			}
			if len(challengeeArray) > 0 {
				// get the challengee with the highest similarity with sort
				sort.Slice(challengeeArray, func(i, j int) bool {
					return challengeeArray[i].tfidfSimilarity > challengeeArray[j].tfidfSimilarity
				})
				mostMatchedChallengeeData := challengeeArray[0]
				challengerParsedCodeText := code_parser.ParseCodeText(data)
				challengeeParsedCodeText := code_parser.ParseCodeText(mostMatchedChallengeeData.data)
				levenSimilarityResults := similarity_compute.ComputeLevenSimilarity(
					challengerParsedCodeText.ParsedCodeText,
					challengeeParsedCodeText.ParsedCodeText,
				)

				combinedSimilarity := levenSimilarityResults.Percentage * mostMatchedChallengeeData.tfidfSimilarity
				matchedMap[path] = RepoToRepoMatchedChallengeeData{
					NumberOfLinesCopied: levenSimilarityResults.Text1SubstringIndexes.EndIndex -
						levenSimilarityResults.Text1SubstringIndexes.StartIndex,
					Path:               mostMatchedChallengeeData.path,
					TFIDFSimilarity:    mostMatchedChallengeeData.tfidfSimilarity,
					LevenSimilarity:    levenSimilarityResults.Percentage,
					CombinedSimilarity: combinedSimilarity,
				}
			} // else, no data in matchedMap
		}

		// compute a weighted average score based on NumberOfLinesCopied and CombinedSimilarity
		totalNumberOfLinesCopied := 0
		for _, matchedChallengeeData := range matchedMap {
			totalNumberOfLinesCopied += matchedChallengeeData.NumberOfLinesCopied
		}
		weightedCombinedSimilarity := 0.0
		weightedTFIDFSimilarity := 0.0
		weightedLevenSimilarity := 0.0
		for _, matchedChallengeeData := range matchedMap {
			weight := float64(matchedChallengeeData.NumberOfLinesCopied) / float64(totalNumberOfLinesCopied)
			weightedCombinedSimilarity += weight * matchedChallengeeData.CombinedSimilarity
			weightedTFIDFSimilarity += weight * matchedChallengeeData.TFIDFSimilarity
			weightedLevenSimilarity += weight * matchedChallengeeData.LevenSimilarity
		}
		highlyLikelyRepos = append(highlyLikelyRepos, RepoToRepoHighestLikelihoodScores{
			RepoUrl:                    challengeeRepoUrl,
			RepoName:                   challengeeRepoName,
			TFIDFSimilarityWeighted:    weightedTFIDFSimilarity,
			LevenSimilarityWeighted:    weightedLevenSimilarity,
			CombinedSimilarityWeighted: weightedCombinedSimilarity,
		})
	}
	sort.Slice(highlyLikelyRepos, func(i, j int) bool {
		return highlyLikelyRepos[i].CombinedSimilarityWeighted > highlyLikelyRepos[j].CombinedSimilarityWeighted
	})
	fmt.Println("-----------------------------------")
	fmt.Println("Repository to compare: " + repoName)
	fmt.Printf("Top %d Repositories\n", CHOOSE_TOP_N_REPOS)
	fmt.Println("If any of the values are green, then the repo to compare is likely a copy of the repo in question.")

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Repo URL", "TFIDF Weighted", "Argmin Leven Weighted", "Combined Sim Weighted"})

	for _, repo := range highlyLikelyRepos {
		tfidfSimilarityColors := tablewriter.Colors{tablewriter.BgBlackColor}
		if repo.TFIDFSimilarityWeighted > TFIDF_SIMILARITY_THRESHOLD {
			tfidfSimilarityColors = tablewriter.Colors{tablewriter.FgGreenColor}
		}

		levenSimilarityColors := tablewriter.Colors{tablewriter.BgBlackColor}
		if repo.LevenSimilarityWeighted > LEVEN_SIMILARITY_THRESHOLD {
			levenSimilarityColors = tablewriter.Colors{tablewriter.FgGreenColor}
		}

		combinedSimilarityColors := tablewriter.Colors{tablewriter.BgBlackColor}
		if repo.CombinedSimilarityWeighted > COMBINED_SIMILARITY_THRESHOLD {
			combinedSimilarityColors = tablewriter.Colors{tablewriter.FgGreenColor}
		}

		row := []string{
			repo.RepoUrl,
			fmt.Sprintf("%.4f", repo.TFIDFSimilarityWeighted),
			fmt.Sprintf("%.4f", repo.LevenSimilarityWeighted),
			fmt.Sprintf("%.4f", repo.CombinedSimilarityWeighted),
		}

		table.Rich(row, []tablewriter.Colors{{}, tfidfSimilarityColors, levenSimilarityColors, combinedSimilarityColors})
	}

	table.Render()
}

func getTopNRepos(possibleRepoMap map[string]int, n int) []string {
	// Create a slice of Pairs and populate it from the map
	possibleRepoPairs := make(util.PairList[int], len(possibleRepoMap))
	i := 0
	for k, v := range possibleRepoMap {
		possibleRepoPairs[i] = util.Pair[int]{Key: k, Value: v}
		i++
	}

	sort.Sort(possibleRepoPairs)

	// filter all the repos with < 1 count
	possibleRepoPairs = util.Filter(possibleRepoPairs, func(p util.Pair[int]) bool {
		return p.Value > 1
	})

	// Extract the top N elements
	lenPossibleReposTopN := util.Min(n, len(possibleRepoPairs))
	possibleReposTopN := make([]string, lenPossibleReposTopN)
	for i := len(possibleRepoPairs) - 1; i > len(possibleRepoPairs)-lenPossibleReposTopN-1; i-- {
		possibleReposTopN[len(possibleRepoPairs)-i-1] = possibleRepoPairs[i].Key
	}
	return possibleReposTopN
}

func RunGitCloneWorkflow(repoUrl string) {
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
		log.Fatalf("Error getting repo name from URL: %v", err)
	}
	RunWorkflow(dir, repoName)
}

type MiniParseCodeWorkflowScanResult struct {
	repositoryName     string
	tfidfSimilarity    float64
	levenSimilarity    float64
	combinedSimilarity float64
}

const NUMBER_OF_PAGES_TO_SEARCH = 10
const TEXT_MAX_LENGTH = 20000

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
) {
	fileExt := filepath.Ext(path)
	originalCodeTextLength := len(codeText)
	codeText = codeText[:util.Min(TEXT_MAX_LENGTH, originalCodeTextLength)] // to prevent OOM
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
	queryResults, err := git_repo.SearchGitHub(query, NUMBER_OF_PAGES_TO_SEARCH)
	if err != nil {
		fmt.Printf("Error searching GitHub: %v\n", err)
		return
	}

	resultChannel := make(chan MiniParseCodeWorkflowScanResult, len(queryResults.Items))
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
				parsedCodeText.ParsedCodeText,
				parsedCodeTextToCompare.ParsedCodeText,
			)

			charLevelTFIDFMutex.Lock()
			charLevelTFIDF.AddDocs([]string{challengeeCodeText})
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

			resultChannel <- result
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
}