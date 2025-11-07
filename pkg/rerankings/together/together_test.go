//go:build rf

package together

import (
	"context"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chromago "github.com/amikos-tech/chroma-go"
	"github.com/amikos-tech/chroma-go/pkg/rerankings"
)

func TestRerank(t *testing.T) {
	apiKey := os.Getenv("TOGETHER_API_KEY")
	if apiKey == "" {
		err := godotenv.Load("../../../.env")
		if err != nil {
			assert.Failf(t, "Error loading .env file", "%s", err)
		}
		apiKey = os.Getenv("TOGETHER_API_KEY")
	}

	tests := []struct {
		name                 string
		rankingFunction      func() *TogetherRerankingFunction
		query                string
		results              []any
		resultsType          string
		validate             func(t *testing.T, rf *TogetherRerankingFunction, results map[string][]rerankings.RankedResult)
		expectedErrorStrings []string
		highestScoreIndex    int
	}{
		{
			name: "Test Rerank basic",
			rankingFunction: func() *TogetherRerankingFunction {
				rf, err := NewTogetherRerankingFunction(WithAPIKey(apiKey))
				require.NoError(t, err, "Failed to create TogetherRerankingFunction")
				return rf
			},
			query: "What is the capital of the United States?",
			results: []any{
				"Carson City is the capital city of the American state of Nevada.",
				"The Commonwealth of the Northern Mariana Islands is a group of islands in the Pacific Ocean that are a political division controlled by the United States. Its capital is Saipan.",
				"Charlotte Amalie is the capital and largest city of the United States Virgin Islands. It has about 20,000 people. The city is on the island of Saint Thomas.",
				"Washington, D.C. (also known as simply Washington or D.C., and officially as the District of Columbia) is the capital of the United States.",
				"Capital punishment (the death penalty) has existed in the United States since before the United States was a country.",
			},
			resultsType: "text",
			validate: func(t *testing.T, rf *TogetherRerankingFunction, results map[string][]rerankings.RankedResult) {
			},
			highestScoreIndex: 3,
		},
		{
			name: "Test Rerank With Env Key",
			rankingFunction: func() *TogetherRerankingFunction {
				rf, err := NewTogetherRerankingFunction(WithEnvAPIKey())
				require.NoError(t, err, "Failed to create TogetherRerankingFunction")
				return rf
			},
			query: "What is the capital of the United States?",
			results: []any{
				"Carson City is the capital city of the American state of Nevada.",
				"The Commonwealth of the Northern Mariana Islands is a group of islands in the Pacific Ocean that are a political division controlled by the United States. Its capital is Saipan.",
				"Charlotte Amalie is the capital and largest city of the United States Virgin Islands. It has about 20,000 people. The city is on the island of Saint Thomas.",
				"Washington, D.C. (also known as simply Washington or D.C., and officially as the District of Columbia) is the capital of the United States.",
				"Capital punishment (the death penalty) has existed in the United States since before the United States was a country.",
			},
			resultsType: "text",
			validate: func(t *testing.T, rf *TogetherRerankingFunction, results map[string][]rerankings.RankedResult) {
			},
			highestScoreIndex: 3,
		},
		{
			name: "Test Rerank With TopN",
			rankingFunction: func() *TogetherRerankingFunction {
				rf, err := NewTogetherRerankingFunction(WithEnvAPIKey(), WithTopN(2))
				require.NoError(t, err, "Failed to create TogetherRerankingFunction")
				return rf
			},
			query: "What is the capital of the United States?",
			results: []any{
				"Carson City is the capital city of the American state of Nevada.",
				"The Commonwealth of the Northern Mariana Islands is a group of islands in the Pacific Ocean that are a political division controlled by the United States. Its capital is Saipan.",
				"Charlotte Amalie is the capital and largest city of the United States Virgin Islands. It has about 20,000 people. The city is on the island of Saint Thomas.",
				"Washington, D.C. (also known as simply Washington or D.C., and officially as the District of Columbia) is the capital of the United States.",
				"Capital punishment (the death penalty) has existed in the United States since before the United States was a country.",
			},
			resultsType: "text",
			validate: func(t *testing.T, rf *TogetherRerankingFunction, results map[string][]rerankings.RankedResult) {
				require.Equal(t, 2, len(results[rf.ID()]))
			},
			highestScoreIndex: 3,
		},
		{
			name: "Test Rerank With ReturnDocuments",
			rankingFunction: func() *TogetherRerankingFunction {
				rf, err := NewTogetherRerankingFunction(WithEnvAPIKey(), WithReturnDocuments(false))
				require.NoError(t, err, "Failed to create TogetherRerankingFunction")
				return rf
			},
			query: "What is the capital of the United States?",
			results: []any{
				"Carson City is the capital city of the American state of Nevada.",
				"The Commonwealth of the Northern Mariana Islands is a group of islands in the Pacific Ocean that are a political division controlled by the United States. Its capital is Saipan.",
				"Charlotte Amalie is the capital and largest city of the United States Virgin Islands. It has about 20,000 people. The city is on the island of Saint Thomas.",
				"Washington, D.C. (also known as simply Washington or D.C., and officially as the District of Columbia) is the capital of the United States.",
				"Capital punishment (the death penalty) has existed in the United States since before the United States was a country.",
			},
			resultsType: "text",
			validate: func(t *testing.T, rf *TogetherRerankingFunction, results map[string][]rerankings.RankedResult) {
				require.Equal(t, 5, len(results[rf.ID()]))
				for _, rr := range results[rf.ID()] {
					require.NotEmpty(t, rr.String)
				}
			},
			highestScoreIndex: 3,
		},
		{
			name: "Test Rerank With Different Model",
			rankingFunction: func() *TogetherRerankingFunction {
				rf, err := NewTogetherRerankingFunction(WithEnvAPIKey(), WithModel("mixedbread-ai/mxbai-rerank-large-v1"))
				require.NoError(t, err, "Failed to create TogetherRerankingFunction")
				return rf
			},
			query: "What is the capital of the United States?",
			results: []any{
				"Carson City is the capital city of the American state of Nevada.",
				"The Commonwealth of the Northern Mariana Islands is a group of islands in the Pacific Ocean that are a political division controlled by the United States. Its capital is Saipan.",
				"Charlotte Amalie is the capital and largest city of the United States Virgin Islands. It has about 20,000 people. The city is on the island of Saint Thomas.",
				"Washington, D.C. (also known as simply Washington or D.C., and officially as the District of Columbia) is the capital of the United States.",
				"Capital punishment (the death penalty) has existed in the United States since before the United States was a country.",
			},
			resultsType: "text",
			validate: func(t *testing.T, rf *TogetherRerankingFunction, results map[string][]rerankings.RankedResult) {
				require.Equal(t, 5, len(results[rf.ID()]))
				for _, rr := range results[rf.ID()] {
					require.NotEmpty(t, rr.String)
				}
			},
			highestScoreIndex: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rf := tt.rankingFunction()
			var resultsToRerank []rerankings.Result
			if tt.resultsType == "text" {
				var textResults []string
				for _, result := range tt.results {
					textResults = append(textResults, result.(string))
				}
				resultsToRerank = rerankings.FromTexts(textResults)
			} else {
				resultsToRerank = rerankings.FromObjects(tt.results)
			}
			res, err := rf.Rerank(context.Background(), tt.query, resultsToRerank)
			if len(tt.expectedErrorStrings) > 0 {
				for _, expectedErrorString := range tt.expectedErrorStrings {
					require.Error(t, err, "Expected error but got nil")
					require.Contains(t, err.Error(), expectedErrorString, "Error message does not contain expected string")
				}
			} else {
				require.NoError(t, err, "Failed to rerank")
				require.NotNil(t, res, "Rerank result is nil")
				require.Contains(t, res, rf.ID(), "Rerank result does not contain the ID of the reranking function")
				tt.validate(t, rf, res)
			}
			require.NoError(t, err, "Failed to rerank")
			require.NotNil(t, res, "Rerank result is nil")
			require.Contains(t, res, rf.ID(), "Rerank result does not contain the ID of the reranking function")
			tt.validate(t, rf, res)
			maxIdx, _ := getMaxIDAndRank(res[rf.ID()])

			require.Equal(t, 3, maxIdx, "The most relevant result is not the expected one")
		})
	}
}

func TestRerankChromaResults(t *testing.T) {
	apiKey := os.Getenv("TOGETHER_API_KEY")
	if apiKey == "" {
		err := godotenv.Load("../../../.env")
		if err != nil {
			assert.Failf(t, "Error loading .env file", "%s", err)
		}
		apiKey = os.Getenv("TOGETHER_API_KEY")
	}

	tests := []struct {
		name                 string
		rankingFunction      func() *TogetherRerankingFunction
		results              *chromago.QueryResults
		resultsType          string
		validate             func(t *testing.T, rf *TogetherRerankingFunction, results *rerankings.RerankedChromaResults)
		expectedErrorStrings []string
		highestScoreIndex    int
	}{
		{
			name: "Test Rerank basic",
			rankingFunction: func() *TogetherRerankingFunction {
				rf, err := NewTogetherRerankingFunction(WithAPIKey(apiKey))
				require.NoError(t, err, "Failed to create TogetherRerankingFunction")
				return rf
			},
			results: &chromago.QueryResults{
				Ids: [][]string{
					{"1", "2", "3", "4", "5"},
				},
				Documents: [][]string{
					{
						"Carson City is the capital city of the American state of Nevada.",
						"The Commonwealth of the Northern Mariana Islands is a group of islands in the Pacific Ocean that are a political division controlled by the United States. Its capital is Saipan.",
						"Charlotte Amalie is the capital and largest city of the United States Virgin Islands. It has about 20,000 people. The city is on the island of Saint Thomas.",
						"Washington, D.C. (also known as simply Washington or D.C., and officially as the District of Columbia) is the capital of the United States.",
						"Capital punishment (the death penalty) has existed in the United States since before the United States was a country.",
					},
				},
				QueryTexts: []string{"What is the capital of the United States?"},
			},
			resultsType: "text",
			validate: func(t *testing.T, rf *TogetherRerankingFunction, results *rerankings.RerankedChromaResults) {
			},
			highestScoreIndex: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rf := tt.rankingFunction()
			res, err := rf.RerankResults(context.Background(), tt.results)
			if len(tt.expectedErrorStrings) > 0 {
				for _, expectedErrorString := range tt.expectedErrorStrings {
					require.Error(t, err, "Expected error but got nil")
					require.Contains(t, err.Error(), expectedErrorString, "Error message does not contain expected string")
				}
			} else {
				require.NoError(t, err, "Failed to rerank")
				require.NotNil(t, res, "Rerank result is nil")
				require.Contains(t, res.Ranks, rf.ID(), "Rerank result does not contain the ID of the reranking function")
				tt.validate(t, rf, res)
			}
			require.NoError(t, err, "Failed to rerank")
			require.NotNil(t, res, "Rerank result is nil")
			require.Contains(t, res.Ranks, rf.ID(), "Rerank result does not contain the ID of the reranking function")
			tt.validate(t, rf, res)
			maxIdx := getIDForMaxRank(res.Ranks[rf.ID()][0]) // we have only one query
			require.Equal(t, 3, maxIdx, "The most relevant result is not the expected one")
		})
	}
}

func getIDForMaxRank(ranks []float32) int {
	var maxRank = float32(-1)
	var id = -1
	for i, rr := range ranks {
		if maxRank <= rr {
			id = i
			maxRank = rr
		}
	}
	return id
}

func getMaxIDAndRank(results []rerankings.RankedResult) (int, float64) {
	var maxRank = float64(-1)
	var id = -1
	for _, rr := range results {
		if maxRank <= float64(rr.Rank) {
			id = rr.Index
			maxRank = float64(rr.Rank)
		}
	}

	return id, maxRank
}
