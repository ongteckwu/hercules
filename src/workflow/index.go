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
	"sort"
	"sync"
	"syscall"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/wilcosheh/tfidf/similarity"
)

const TEXT_MAX_LENGTH = 25000
const NO_OF_FILES_FOR_PARSING = 18
const NO_OF_MAX_SEARCHED_FILES_TO_PARSE = 180 // can be set if you want to parse less files

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
	TotalNumberOfFiles         int
	SimilarNumberOfFiles       int
	TFIDFSimilarityWeighted    float64
	LevenSimilarityWeighted    float64
	CombinedSimilarityWeighted float64
}

const TFIDF_SIMILARITY_THRESHOLD = 0.7
const LEVEN_SIMILARITY_THRESHOLD = 0.7
const COMBINED_SIMILARITY_THRESHOLD = 0.4
const CHOOSE_TOP_N_REPOS = 8

func RunWorkflow(repoDir string, repoName string, isTempDir bool) {
	filePaths, err := util.GetFilePaths(repoDir)
	util.Check(err)

	filePaths = util.RemoveNonCodeFiles(filePaths)

	// read all the files and return a map of path to data
	allDataMap, err := util.MultipleFileRead(filePaths, TEXT_MAX_LENGTH) // map[path]data
	// truncated to prevent OOM
	util.Check(err)

	// create a slice of all the data
	allDataArray := loadAllData(allDataMap)

	fmt.Printf("Number of files: %d\n", len(allDataArray))

	// create a char level tfidf with the files
	charLevelTFIDF := tfidf.New()
	charLevelTFIDF.AddDocs(allDataArray, tfidf.TokenizeCharLevelNoAlpha)
	charLevelTFIDFMutex := &sync.Mutex{}

	// create normal tfidf with the files
	keywordsTFIDF := tfidf.New()
	keywordsTFIDF.AddDocs(allDataArray)
	keywordsTFIDFMutex := &sync.Mutex{}

	// randomly draw half that is code files first
	codeFilePaths := util.Filter(filePaths, func(path string) bool {
		return util.IsCodeFile(path)
	})
	if len(codeFilePaths) > NO_OF_FILES_FOR_PARSING/2 {
		codeFilePaths = util.RandomDrawWithoutReplacement(codeFilePaths, NO_OF_FILES_FOR_PARSING/2)
	}
	// draw the other half randomly
	remainingFilePaths := util.Filter(filePaths, func(path string) bool {
		return !util.Contains(codeFilePaths, path)
	})
	randomlyDrawnFilesN := util.RandomDrawWithoutReplacement(remainingFilePaths, NO_OF_FILES_FOR_PARSING-len(codeFilePaths))
	randomlyDrawnFilesN = append(randomlyDrawnFilesN, codeFilePaths...)

	// for each file, parse
	possibleRepoMap := make(map[string][]*MiniParseCodeWorkflowScanResult)
	possibleRepoMapMutex := &sync.Mutex{}

	///////////////////////////////
	// PARSE FILES TO FIND REPOS //
	///////////////////////////////

	mainMessage := "Parsing these randomly picked files:\n"
	for _, path := range randomlyDrawnFilesN {
		mainMessage += fmt.Sprintf("    - %s\n", path)
	}

	totalNumberOfPossibleSearchedFilesToBeParsed := len(randomlyDrawnFilesN) * NUMBER_OF_FILES_TO_QUERY
	maxNumberOfSearchedFilesToBeParsed := util.Min(
		totalNumberOfPossibleSearchedFilesToBeParsed,
		NO_OF_MAX_SEARCHED_FILES_TO_PARSE,
	)
	fileSearchProgressBar := ProgressModel{
		progress:    progress.New(progress.WithDefaultGradient()),
		mainMessage: mainMessage,
		length:      len(randomlyDrawnFilesN),
	}

	fileSearchProgressBarModel := tea.NewProgram(fileSearchProgressBar)

	go func() {
		totalNumberOfFilesParsed := 0
		count := 0
		for _, path := range randomlyDrawnFilesN {
			// dont need to goroutine since github has a rate limit
			numberOfFilesParsed, _ := ParseCodeWorkflow(
				repoName,
				path, isTempDir, allDataMap[path],
				keywordsTFIDF, keywordsTFIDFMutex,
				charLevelTFIDF, charLevelTFIDFMutex,
				possibleRepoMap, possibleRepoMapMutex,
			)
			count++
			totalNumberOfFilesParsed = util.Min(totalNumberOfFilesParsed+numberOfFilesParsed, maxNumberOfSearchedFilesToBeParsed)
			fileSearchProgressBarModel.Send(progressMsg{workflowsDone: count})
			fileSearchProgressBarModel.Send(updateMessageMsg{message: fmt.Sprintf("Parsing %s... (Might slow down due to Github API Rate Limit)", path)})
			if totalNumberOfFilesParsed >= maxNumberOfSearchedFilesToBeParsed {
				break
			}
		}
		if count < len(randomlyDrawnFilesN) {
			fileSearchProgressBarModel.Send(progressMsg{workflowsDone: len(randomlyDrawnFilesN)})
			fileSearchProgressBarModel.Send(updateMessageMsg{message: "Parsing complete!"})
		}
	}()

	// awaits here
	if _, err := fileSearchProgressBarModel.Run(); err != nil {
		fmt.Println("Error running progress bar for parsing code:", err)
		os.Exit(1)
	}

	///////////////////////////
	// EVALUATE REPOSITORIES //
	///////////////////////////

	possibleReposTopN := getTopNRepos(possibleRepoMap, CHOOSE_TOP_N_REPOS)

	if len(possibleReposTopN) == 0 {
		fmt.Println("No repositories found, hence no plagiarism detected!")
		os.Exit(0)
	}

	possibleReposTopNMap := make(map[string][]*MiniParseCodeWorkflowScanResult)
	for _, repoName := range possibleReposTopN {
		possibleReposTopNMap[repoName] = possibleRepoMap[repoName]
	}
	possibleRepoMap = nil // free memory

	var preliminaryHighlyLikelyRepos []RepoToRepoHighestLikelihoodScores

	for challengeeRepoName, challengeeRepoData := range possibleReposTopNMap {
		totalNumberOfLinesCopied := 0
		for _, data := range challengeeRepoData {
			totalNumberOfLinesCopied += data.NumberOfLinesCopied
		}
		result := computePreliminarySimilarityScoresWeighted(
			len(allDataArray),
			totalNumberOfLinesCopied,
			challengeeRepoData,
			challengeeRepoName,
		)
		preliminaryHighlyLikelyRepos = append(preliminaryHighlyLikelyRepos, *result)
	}

	// sort by combined similarity, descending order
	sort.Slice(preliminaryHighlyLikelyRepos, func(i, j int) bool {
		return preliminaryHighlyLikelyRepos[i].CombinedSimilarityWeighted > preliminaryHighlyLikelyRepos[j].CombinedSimilarityWeighted
	})

	fmt.Println("-----------------------------------")
	fmt.Println("Preliminary Results")
	RenderTable(repoName, preliminaryHighlyLikelyRepos)
	fmt.Println("-----------------------------------")
	// ask user if want to continue advanced repo-to-repo match evaluation
	fmt.Println("Do you want to continue to advanced repo-to-repo match evaluation? (y/n)")
	var input string
	fmt.Scanln(&input)
	if input != "y" {
		fmt.Println("Exiting...")
		os.Exit(0)
	}

	var highlyLikelyRepos []RepoToRepoHighestLikelihoodScores

	repoEvaluationProgressBar := ProgressModel{
		progress:    progress.New(progress.WithDefaultGradient()),
		mainMessage: fmt.Sprintf("Evaluating %d Repositories Found (Max 8)...", len(possibleReposTopN)),
		length:      len(possibleReposTopN),
	}

	repoEvaluationProgressBarModel := tea.NewProgram(repoEvaluationProgressBar)

	go func() {
		// for each repo, clone and compare
		// dont go routine each due to memory usage
		countOfDone := 0
		for _, challengeeRepoName := range possibleReposTopN {
			repoEvaluationProgressBarModel.Send(updateMessageMsg{message: fmt.Sprintf("Evaluating repo number %d...", countOfDone)})
			result := cloneAndCompare(challengeeRepoName, allDataArray, allDataMap)
			if result != nil {
				highlyLikelyRepos = append(highlyLikelyRepos, *result)
			}
			countOfDone++
			repoEvaluationProgressBarModel.Send(progressMsg{workflowsDone: countOfDone})
		}
	}()

	// awaits here
	if _, err := repoEvaluationProgressBarModel.Run(); err != nil {
		fmt.Println("Error running progress bar for parsing code:", err)
		os.Exit(1)
	}

	// sort by combined similarity, descending order
	sort.Slice(highlyLikelyRepos, func(i, j int) bool {
		return highlyLikelyRepos[i].CombinedSimilarityWeighted > highlyLikelyRepos[j].CombinedSimilarityWeighted
	})
	RenderTable(repoName, highlyLikelyRepos)

}

func loadAllData(allDataMap map[string]string) []string {
	// Function to load all data into a slice
	allDataArray := make([]string, 0, len(allDataMap))
	for _, data := range allDataMap {
		allDataArray = append(allDataArray, data)
	}
	return allDataArray
}

func cloneAndCompare(challengeeRepoName string, allDataArray []string, allDataMap map[string]string) *RepoToRepoHighestLikelihoodScores {
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
	challengeeAllDataMap, err := util.MultipleFileRead(filePaths, TEXT_MAX_LENGTH) // map[path]data
	// truncated to prevent OOM
	util.Check(err)

	// create a slice of all the data
	challengeeAllDataArray := loadAllData(challengeeAllDataMap)

	// if challengee has too many files compared to challenger, or vice versa, ignore
	// 2x difference max
	if (len(challengeeAllDataArray) > len(allDataArray)*2) || (len(allDataArray) > len(challengeeAllDataArray)*2) {
		return nil
	}
	// create a char level tfidf with the files
	combinedCharLevelTFIDF := tfidf.New()
	combinedCharLevelTFIDF.AddDocs(challengeeAllDataArray, tfidf.TokenizeCharLevelNoAlpha)
	combinedCharLevelTFIDF.AddDocs(allDataArray, tfidf.TokenizeCharLevelNoAlpha)

	matchedMap := make(map[string]RepoToRepoMatchedChallengeeData) // map[challengePath]RepoToRepoMatchedChallengeeData
	tempMemory := make(map[string](map[string]float64))

	// for each file, find the best challengee file to match with
	for path, data := range allDataMap {
		w1 := combinedCharLevelTFIDF.Cal(data)
		var challengeeArray []RepoToRepoPotentialChallengeeData
		for challengeePath, challengeeData := range challengeeAllDataMap {
			if !util.IsExtensionSame(path, challengeePath) {
				continue
			}
			w2, ok := tempMemory[challengeePath]
			if !ok {
				w2 = combinedCharLevelTFIDF.Cal(challengeeData)
				tempMemory[challengeePath] = w2
			}
			similarity := similarity.Cosine(w1, w2)
			if similarity > TFIDF_SIMILARITY_THRESHOLD {
				obj := RepoToRepoPotentialChallengeeData{
					path:            challengeePath,
					data:            challengeeData,
					tfidfSimilarity: similarity,
				}
				challengeeArray = append(challengeeArray, obj)
				break
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
				challengerParsedCodeText,
				challengeeParsedCodeText,
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
	// free memory
	tempMemory = nil

	// compute a weighted average score based on NumberOfLinesCopied and CombinedSimilarity
	totalNumberOfLinesCopied := 0
	for _, matchedChallengeeData := range matchedMap {
		totalNumberOfLinesCopied += matchedChallengeeData.NumberOfLinesCopied
	}

	resultPtr := computeSimilarityScoresWeighted(
		len(allDataArray), totalNumberOfLinesCopied,
		matchedMap, challengeeRepoUrl, challengeeRepoName,
	)
	return resultPtr
}

func computePreliminarySimilarityScoresWeighted(
	totalNumberOfFiles int,
	totalNumberOfLinesCopied int,
	challengeeRepoData []*MiniParseCodeWorkflowScanResult,
	challengeeRepoName string,
) *RepoToRepoHighestLikelihoodScores {

	weightedCombinedSimilarity := 0.0
	weightedTFIDFSimilarity := 0.0
	weightedLevenSimilarity := 0.0
	for _, data := range challengeeRepoData {
		weight := float64(data.NumberOfLinesCopied) / float64(totalNumberOfLinesCopied)
		weightedCombinedSimilarity += weight * data.CombinedSimilarity
		weightedTFIDFSimilarity += weight * data.TFIDFSimilarity
		weightedLevenSimilarity += weight * data.LevenSimilarity
	}
	return &RepoToRepoHighestLikelihoodScores{
		RepoUrl:                    "https://github.com/" + challengeeRepoName,
		RepoName:                   challengeeRepoName,
		TotalNumberOfFiles:         totalNumberOfFiles,
		SimilarNumberOfFiles:       len(challengeeRepoData),
		TFIDFSimilarityWeighted:    weightedTFIDFSimilarity,
		LevenSimilarityWeighted:    weightedLevenSimilarity,
		CombinedSimilarityWeighted: weightedCombinedSimilarity,
	}
}

func computeSimilarityScoresWeighted(
	totalNumberOfFiles int,
	totalNumberOfLinesCopied int,
	matchedMap map[string]RepoToRepoMatchedChallengeeData,
	challengeeRepoUrl string,
	challengeeRepoName string,
) *RepoToRepoHighestLikelihoodScores {
	weightedCombinedSimilarity := 0.0
	weightedTFIDFSimilarity := 0.0
	weightedLevenSimilarity := 0.0
	for _, matchedChallengeeData := range matchedMap {
		weight := float64(matchedChallengeeData.NumberOfLinesCopied) / float64(totalNumberOfLinesCopied)
		weightedCombinedSimilarity += weight * matchedChallengeeData.CombinedSimilarity
		weightedTFIDFSimilarity += weight * matchedChallengeeData.TFIDFSimilarity
		weightedLevenSimilarity += weight * matchedChallengeeData.LevenSimilarity
	}
	return &RepoToRepoHighestLikelihoodScores{
		RepoUrl:                    challengeeRepoUrl,
		RepoName:                   challengeeRepoName,
		TotalNumberOfFiles:         totalNumberOfFiles,
		SimilarNumberOfFiles:       len(matchedMap),
		TFIDFSimilarityWeighted:    weightedTFIDFSimilarity,
		LevenSimilarityWeighted:    weightedLevenSimilarity,
		CombinedSimilarityWeighted: weightedCombinedSimilarity,
	}
}

func getTopNRepos(possibleRepoMap map[string][]*MiniParseCodeWorkflowScanResult, n int) []string {
	// Create a slice of Pairs and populate it from the map
	possibleRepoPairs := make(util.PairList[int], len(possibleRepoMap))
	i := 0
	for k, v := range possibleRepoMap {
		possibleRepoPairs[i] = util.Pair[int]{Key: k, Value: len(v)}
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
