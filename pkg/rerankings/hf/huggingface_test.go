//go:build rf

package huggingface

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	chromago "github.com/amikos-tech/chroma-go"
	"github.com/amikos-tech/chroma-go/pkg/rerankings"
)

func getTEIImage() string {
	teiVersion := "1.8.3"
	teiImage := "ghcr.io/huggingface/text-embeddings-inference"
	if v := os.Getenv("TEI_VERSION"); v != "" {
		teiVersion = v
	}
	if img := os.Getenv("TEI_IMAGE"); img != "" {
		teiImage = img
	}
	return fmt.Sprintf("%s:cpu-%s", teiImage, teiVersion)
}

func TestRerankHFEI(t *testing.T) {
	apiKey := os.Getenv("HF_API_KEY")
	if apiKey == "" {
		err := godotenv.Load("../../../.env")
		if err != nil {
			assert.Failf(t, "Error loading .env file", "%s", err)
		}
		apiKey = os.Getenv("HF_API_KEY")
	}

	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:         getTEIImage(),
		ExposedPorts:  []string{"80/tcp"},
		WaitingFor:    wait.ForHTTP("/health").WithPort("80/tcp").WithStartupTimeout(5 * time.Minute),
		ImagePlatform: "linux/amd64",
		Cmd:           []string{"--model-id", "BAAI/bge-reranker-base"},
		HostConfigModifier: func(hostConfig *container.HostConfig) {
			dockerMounts := make([]mount.Mount, 0)
			cwd, err := os.Getwd()
			require.NoError(t, err)

			// Join the current working directory with the relative path
			joinedPath := filepath.Join(cwd, "data")
			if _, err := os.Stat(joinedPath); os.IsNotExist(err) {
				// Create the directory if it doesn't exist
				err = os.MkdirAll(joinedPath, 0755)
				require.NoError(t, err)
			}
			dockerMounts = append(dockerMounts, mount.Mount{
				Type:   mount.TypeBind,
				Source: joinedPath,
				Target: "/data",
			})
			hostConfig.Mounts = dockerMounts
		},
	}
	hfei, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, hfei.Terminate(ctx))
	})
	ip, err := hfei.Host(ctx)
	require.NoError(t, err)
	port, err := hfei.MappedPort(ctx, "80")
	require.NoError(t, err)
	endpoint := fmt.Sprintf("http://%s:%s", ip, port.Port())

	tests := []struct {
		name                 string
		rankingFunction      func() *HFRerankingFunction
		query                string
		results              []any
		resultsType          string
		validate             func(t *testing.T, rf *HFRerankingFunction, results map[string][]rerankings.RankedResult)
		expectedErrorStrings []string
		highestScoreIndex    int
	}{
		{
			name: "Test Rerank Inference Endpoint",
			rankingFunction: func() *HFRerankingFunction {
				rf, err := NewHFRerankingFunction(WithRerankingEndpoint(endpoint))
				require.NoError(t, err, "Failed to create HFRerankingFunction")
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
			validate: func(t *testing.T, rf *HFRerankingFunction, results map[string][]rerankings.RankedResult) {
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

				maxIdx, _ := getMaxIDAndRank(res[rf.ID()])
				require.Equal(t, 3, maxIdx, "The most relevant result is not the expected one")
			}
		})
	}
}

func TestRerankChromaResults(t *testing.T) {
	apiKey := os.Getenv("HF_API_KEY")
	if apiKey == "" {
		err := godotenv.Load("../../../.env")
		if err != nil {
			assert.Failf(t, "Error loading .env file", "%s", err)
		}
		apiKey = os.Getenv("HF_API_KEY")
	}

	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:         getTEIImage(),
		ExposedPorts:  []string{"80/tcp"},
		WaitingFor:    wait.ForHTTP("/health").WithPort("80/tcp").WithStartupTimeout(5 * time.Minute),
		ImagePlatform: "linux/amd64",
		Cmd:           []string{"--model-id", "BAAI/bge-reranker-base"},
		HostConfigModifier: func(hostConfig *container.HostConfig) {
			dockerMounts := make([]mount.Mount, 0)
			cwd, err := os.Getwd()
			require.NoError(t, err)

			// Join the current working directory with the relative path
			joinedPath := filepath.Join(cwd, "data")
			if _, err := os.Stat(joinedPath); os.IsNotExist(err) {
				// Create the directory if it doesn't exist
				err = os.MkdirAll(joinedPath, 0755)
				require.NoError(t, err)
			}
			dockerMounts = append(dockerMounts, mount.Mount{
				Type:   mount.TypeBind,
				Source: joinedPath,
				Target: "/data",
			})
			hostConfig.Mounts = dockerMounts
		},
	}
	hfei, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, hfei.Terminate(ctx))
	})
	ip, err := hfei.Host(ctx)
	require.NoError(t, err)
	port, err := hfei.MappedPort(ctx, "80")
	require.NoError(t, err)
	endpoint := fmt.Sprintf("http://%s:%s", ip, port.Port())
	tests := []struct {
		name                 string
		rankingFunction      func() *HFRerankingFunction
		results              *chromago.QueryResults
		resultsType          string
		validate             func(t *testing.T, rf *HFRerankingFunction, results *rerankings.RerankedChromaResults)
		expectedErrorStrings []string
		highestScoreIndex    int
	}{
		{
			name: "Test Rerank basic",
			rankingFunction: func() *HFRerankingFunction {
				rf, err := NewHFRerankingFunction(WithRerankingEndpoint(endpoint))
				require.NoError(t, err, "Failed to create HFRerankingFunction")
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
			validate: func(t *testing.T, rf *HFRerankingFunction, results *rerankings.RerankedChromaResults) {
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

				maxIdx := getIDForMaxRank(res.Ranks[rf.ID()][0]) // we have only one query
				require.Equal(t, 3, maxIdx, "The most relevant result is not the expected one")
			}
		})
	}
}

// TODO extract this into separate reranking testing commons file
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

// TODO extract this into separate reranking testing commons file
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
