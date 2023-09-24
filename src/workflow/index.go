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
	"github.com/olekukonko/tablewriter"

	"github.com/wilcosheh/tfidf/similarity"
)

const TEXT_MAX_LENGTH = 25000
const NO_OF_FILES_FOR_PARSING = 20

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
	charLevelTFIDF.AddDocs(&allDataArray, tfidf.TokenizeCharLevelNoAlpha)
	charLevelTFIDFMutex := &sync.Mutex{}

	// create normal tfidf with the files
	keywordsTFIDF := tfidf.New()
	keywordsTFIDF.AddDocs(&allDataArray)
	keywordsTFIDFMutex := &sync.Mutex{}

	randomlyDrawnFilesN := util.RandomDrawWithoutReplacement(&filePaths, NO_OF_FILES_FOR_PARSING)

	// for each file, parse
	possibleRepoMap := make(map[string]int)
	possibleRepoMapMutex := &sync.Mutex{}

	///////////////////////////////
	// PARSE FILES TO FIND REPOS //
	///////////////////////////////

	mainMessage := "Parsing these randomly picked files:\n"
	for _, path := range randomlyDrawnFilesN {
		mainMessage += fmt.Sprintf("    - %s\n", path)
	}

	fileSearchProgressBar := ProgressModel{
		progress:    progress.New(progress.WithDefaultGradient()),
		mainMessage: mainMessage,
		length:      len(randomlyDrawnFilesN),
	}

	fileSearchProgressBarModel := tea.NewProgram(fileSearchProgressBar)

	go func() {
		countOfDone := 0
		for _, path := range randomlyDrawnFilesN {
			// dont need to goroutine since github has a rate limit
			ParseCodeWorkflow(
				repoName,
				path, isTempDir, allDataMap[path],
				keywordsTFIDF, keywordsTFIDFMutex,
				charLevelTFIDF, charLevelTFIDFMutex,
				possibleRepoMap, possibleRepoMapMutex,
			)
			countOfDone++
			fileSearchProgressBarModel.Send(progressMsg{workflowsDone: countOfDone})
			fileSearchProgressBarModel.Send(updateMessageMsg{message: fmt.Sprintf("Parsing %s... (Might slow down due to Github API Rate Limit)", path)})
		}
	}()

	if _, err := fileSearchProgressBarModel.Run(); err != nil {
		fmt.Println("Error running progress bar for parsing code:", err)
		os.Exit(1)
	}

	///////////////////////////
	// EVALUATE REPOSITORIES //
	///////////////////////////

	possibleReposTopN := getTopNRepos(possibleRepoMap, CHOOSE_TOP_N_REPOS)

	var highlyLikelyRepos []RepoToRepoHighestLikelihoodScores

	if len(possibleReposTopN) > 0 {
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

		// sort by combined similarity, descending order
		sort.Slice(highlyLikelyRepos, func(i, j int) bool {
			return highlyLikelyRepos[i].CombinedSimilarityWeighted > highlyLikelyRepos[j].CombinedSimilarityWeighted
		})
		if _, err := repoEvaluationProgressBarModel.Run(); err != nil {
			fmt.Println("Error running progress bar for parsing code:", err)
			os.Exit(1)
		}
	}

	renderTable(repoName, highlyLikelyRepos)

}

func renderTable(repoName string, highlyLikelyRepos []RepoToRepoHighestLikelihoodScores) {
	fmt.Println("-----------------------------------")
	fmt.Println("Repository to compare: " + repoName)
	fmt.Printf("Top %d Repositories\n", len(highlyLikelyRepos))
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
	combinedCharLevelTFIDF.AddDocs(&challengeeAllDataArray, tfidf.TokenizeCharLevelNoAlpha)
	combinedCharLevelTFIDF.AddDocs(&allDataArray, tfidf.TokenizeCharLevelNoAlpha)

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
			if similarity > 0.6 { // just need an entry point, since 0.5 is lowest
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

	tempMemory = nil

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
	return &RepoToRepoHighestLikelihoodScores{
		RepoUrl:                    challengeeRepoUrl,
		RepoName:                   challengeeRepoName,
		TFIDFSimilarityWeighted:    weightedTFIDFSimilarity,
		LevenSimilarityWeighted:    weightedLevenSimilarity,
		CombinedSimilarityWeighted: weightedCombinedSimilarity,
	}
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
