package workflow

import (
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
)

func RenderTable(repoName string, highlyLikelyRepos []RepoToRepoHighestLikelihoodScores) {
	fmt.Println("-----------------------------------")
	fmt.Println("Repository to compare: " + repoName)
	fmt.Printf("Top %d Repositories\n", len(highlyLikelyRepos))
	fmt.Println("If any of the values are green, then the challenged repo is likely a copy of the repo in question.")

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Repo URL", "Number of Files Similar", "TFIDF Weighted", "Argmin Leven Weighted", "Combined Sim Weighted"})

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
			fmt.Sprintf("%d\\%d", repo.SimilarNumberOfFiles, repo.TotalNumberOfFiles),
			fmt.Sprintf("%.4f", repo.TFIDFSimilarityWeighted),
			fmt.Sprintf("%.4f", repo.LevenSimilarityWeighted),
			fmt.Sprintf("%.4f", repo.CombinedSimilarityWeighted),
		}

		table.Rich(row, []tablewriter.Colors{{}, tfidfSimilarityColors, levenSimilarityColors, combinedSimilarityColors})
	}

	table.Render()
}
