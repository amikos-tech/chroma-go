//go:build rf

package cohere

import (
	"context"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chromago "github.com/guiperry/chroma-go_cerebras"
	"github.com/guiperry/chroma-go_cerebras/pkg/rerankings"
)

func TestRerank(t *testing.T) {
	apiKey := os.Getenv("COHERE_API_KEY")
	if apiKey == "" {
		err := godotenv.Load("../../../.env")
		if err != nil {
			assert.Failf(t, "Error loading .env file", "%s", err)
		}
		apiKey = os.Getenv("COHERE_API_KEY")
	}

	tests := []struct {
		name                 string
		rankingFunction      func() *CohereRerankingFunction
		query                string
		results              []any
		resultsType          string
		validate             func(t *testing.T, rf *CohereRerankingFunction, results map[string][]rerankings.RankedResult)
		expectedErrorStrings []string
		highestScoreIndex    int
	}{
		{
			name: "Test Rerank basic",
			rankingFunction: func() *CohereRerankingFunction {
				rf, err := NewCohereRerankingFunction(WithAPIKey(apiKey))
				require.NoError(t, err, "Failed to create CohereRerankingFunction")
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
			validate: func(t *testing.T, rf *CohereRerankingFunction, results map[string][]rerankings.RankedResult) {
			},
			highestScoreIndex: 3,
		},
		{
			name: "Test Rerank With Env Key",
			rankingFunction: func() *CohereRerankingFunction {
				rf, err := NewCohereRerankingFunction(WithEnvAPIKey())
				require.NoError(t, err, "Failed to create CohereRerankingFunction")
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
			validate: func(t *testing.T, rf *CohereRerankingFunction, results map[string][]rerankings.RankedResult) {
			},
			highestScoreIndex: 3,
		},
		{
			name: "Test Rerank With TopN",
			rankingFunction: func() *CohereRerankingFunction {
				rf, err := NewCohereRerankingFunction(WithEnvAPIKey(), WithTopN(2))
				require.NoError(t, err, "Failed to create CohereRerankingFunction")
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
			validate: func(t *testing.T, rf *CohereRerankingFunction, results map[string][]rerankings.RankedResult) {
				require.Equal(t, 2, len(results[rf.ID()]))
			},
			highestScoreIndex: 3,
		},
		{
			name: "Test Rerank With ReturnDocuments",
			rankingFunction: func() *CohereRerankingFunction {
				rf, err := NewCohereRerankingFunction(WithEnvAPIKey(), WithReturnDocuments())
				require.NoError(t, err, "Failed to create CohereRerankingFunction")
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
			validate: func(t *testing.T, rf *CohereRerankingFunction, results map[string][]rerankings.RankedResult) {
				require.Equal(t, 5, len(results[rf.ID()]))
				for _, rr := range results[rf.ID()] {
					require.NotEmpty(t, rr.String)
				}
			},
			highestScoreIndex: 3,
		},
		{
			name: "Test Rerank With MaxChunksPerDoc",
			rankingFunction: func() *CohereRerankingFunction {
				rf, err := NewCohereRerankingFunction(WithEnvAPIKey(), WithMaxChunksPerDoc(5))
				require.NoError(t, err, "Failed to create CohereRerankingFunction")
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
			validate: func(t *testing.T, rf *CohereRerankingFunction, results map[string][]rerankings.RankedResult) {
				require.Equal(t, 5, len(results[rf.ID()]))
				for _, rr := range results[rf.ID()] {
					require.NotEmpty(t, rr.String)
				}
			},
			highestScoreIndex: 3,
		},
		{
			name: "Test Rerank With Different Model",
			rankingFunction: func() *CohereRerankingFunction {
				rf, err := NewCohereRerankingFunction(WithEnvAPIKey(), WithDefaultModel(ModelRerankMultilingualV30))
				require.NoError(t, err, "Failed to create CohereRerankingFunction")
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
			validate: func(t *testing.T, rf *CohereRerankingFunction, results map[string][]rerankings.RankedResult) {
				require.Equal(t, 5, len(results[rf.ID()]))
				for _, rr := range results[rf.ID()] {
					require.NotEmpty(t, rr.String)
				}
			},
			highestScoreIndex: 3,
		},
		{
			name: "Test Rerank With RerankFields",
			rankingFunction: func() *CohereRerankingFunction {
				rf, err := NewCohereRerankingFunction(WithEnvAPIKey(), WithRerankFields([]string{"content"}))
				require.NoError(t, err, "Failed to create CohereRerankingFunction")
				return rf
			},
			query: "What is the capital of the United States?",
			results: []any{
				map[string]string{
					"content": "Carson City is the capital city of the American state of Nevada.",
				},
				map[string]string{
					"content": "The Commonwealth of the Northern Mariana Islands is a group of islands in the Pacific Ocean that are a political division controlled by the United States. Its capital is Saipan.",
				},
				map[string]string{
					"content": "Charlotte Amalie is the capital and largest city of the United States Virgin Islands. It has about 20,000 people. The city is on the island of Saint Thomas.",
				},
				map[string]string{
					"content": "Washington, D.C. (also known as simply Washington or D.C., and officially as the District of Columbia) is the capital of the United States.",
				},
				map[string]string{
					"content": "Capital punishment (the death penalty) has existed in the United States since before the United States was a country.",
				},
			},
			resultsType: "object",
			validate: func(t *testing.T, rf *CohereRerankingFunction, results map[string][]rerankings.RankedResult) {
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
	apiKey := os.Getenv("COHERE_API_KEY")
	if apiKey == "" {
		err := godotenv.Load("../../../.env")
		if err != nil {
			assert.Failf(t, "Error loading .env file", "%s", err)
		}
		apiKey = os.Getenv("COHERE_API_KEY")
	}

	tests := []struct {
		name                 string
		rankingFunction      func() *CohereRerankingFunction
		results              *chromago.QueryResults
		resultsType          string
		validate             func(t *testing.T, rf *CohereRerankingFunction, results *rerankings.RerankedChromaResults)
		expectedErrorStrings []string
		highestScoreIndex    int
	}{
		{
			name: "Test Rerank basic",
			rankingFunction: func() *CohereRerankingFunction {
				rf, err := NewCohereRerankingFunction(WithAPIKey(apiKey))
				require.NoError(t, err, "Failed to create CohereRerankingFunction")
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
			validate: func(t *testing.T, rf *CohereRerankingFunction, results *rerankings.RerankedChromaResults) {
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
