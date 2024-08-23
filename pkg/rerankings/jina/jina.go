package jina

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	chromago "github.com/amikos-tech/chroma-go"
	chttp "github.com/amikos-tech/chroma-go/pkg/commons/http"
	"github.com/amikos-tech/chroma-go/pkg/rerankings"
	"github.com/amikos-tech/chroma-go/types"
)

const (
	DefaultBaseAPIEndpoint                      = "https://api.jina.ai/v1/rerank"
	DefaultRerankingModel  types.RerankingModel = "jina-reranker-v1-base-en"
)

type RerankingRequest struct {
	Model           string   `json:"model"`
	Query           string   `json:"query"`
	Documents       []string `json:"documents"` // TODO there seems Documents (objects) (Documents) TBD
	TopN            *int     `json:"top_n,omitempty"`
	ReturnDocuments *bool    `json:"return_documents,omitempty"`
}

type RerankingResponse struct {
	Model  string `json:"model"`
	Object string `json:"object"`
	Usage  struct {
		TotalTokens  int `json:"total_tokens"`
		PromptTokens int `json:"prompt_tokens"`
	}
	Results []struct {
		Index          int     `json:"index"`
		RelevanceScore float32 `json:"relevance_score"`
		Document       struct {
			Text string `json:"text"`
		} `json:"document,omitempty"`
	} `json:"results"`
}

var _ rerankings.RerankingFunction = (*JinaRerankingFunction)(nil)

func getDefaults() *JinaRerankingFunction {
	var returnDocuments = true
	return &JinaRerankingFunction{
		httpClient:        http.DefaultClient,
		defaultModel:      DefaultRerankingModel,
		rerankingEndpoint: DefaultBaseAPIEndpoint,
		returnDocuments:   &returnDocuments,
		topN:              nil,
	}
}

type JinaRerankingFunction struct {
	httpClient        *http.Client
	apiKey            string
	defaultModel      types.RerankingModel
	rerankingEndpoint string
	returnDocuments   *bool
	topN              *int
}

func NewJinaRerankingFunction(opts ...Option) (*JinaRerankingFunction, error) {
	ef := getDefaults()
	for _, opt := range opts {
		err := opt(ef)
		if err != nil {
			return nil, err
		}
	}
	return ef, nil
}

func (r *JinaRerankingFunction) sendRequest(ctx context.Context, req *RerankingRequest) (*RerankingResponse, error) {
	if req.TopN == nil {
		var dlen = len(req.Documents)
		req.TopN = &dlen
	}
	if req.ReturnDocuments == nil {
		req.ReturnDocuments = r.returnDocuments
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", r.rerankingEndpoint, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", chttp.ChromaGoClientUserAgent)
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.apiKey))

	resp, err := r.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		// TODO serialize body in error
		return nil, fmt.Errorf("unexpected response %v: %s", resp.Status, respData)
	}
	var response *RerankingResponse
	if err := json.Unmarshal(respData, &response); err != nil {
		return nil, err
	}

	return response, nil
}

func (r *JinaRerankingFunction) Rerank(ctx context.Context, query string, results []rerankings.Result) (map[string][]rerankings.RankedResult, error) {
	docs := make([]string, 0)
	for _, result := range results {
		d, err := result.ToText()
		if err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}

	req := &RerankingRequest{
		Model:     string(r.defaultModel),
		Documents: docs,
		Query:     query,
		TopN:      r.topN,
	}

	rerankResp, err := r.sendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	rankedResults := map[string][]rerankings.RankedResult{r.ID(): make([]rerankings.RankedResult, len(rerankResp.Results))}
	for i, rr := range rerankResp.Results {
		originalDoc, err := results[rr.Index].ToText()
		if err != nil {
			return nil, err
		}
		if rr.Document.Text != "" {
			originalDoc = rr.Document.Text
		}
		rankedResults[r.ID()][i] = rerankings.RankedResult{
			String: originalDoc,
			Index:  rr.Index,
			Rank:   rr.RelevanceScore,
		}
	}
	return rankedResults, nil
}

// ID returns the of the reranking function. We use `cohere-` prefix with the default model
func (r *JinaRerankingFunction) ID() string {
	return fmt.Sprintf("jinaai-%s", r.defaultModel)
}

func (r *JinaRerankingFunction) RerankResults(ctx context.Context, queryResults *chromago.QueryResults) (*rerankings.RerankedChromaResults, error) {
	rerankedResults := &rerankings.RerankedChromaResults{
		QueryResults: *queryResults,
		Ranks:        map[string][][]float32{r.ID(): make([][]float32, len(queryResults.Ids))},
	}
	for i, rs := range queryResults.Ids {
		if len(rs) == 0 {
			return nil, fmt.Errorf("no results to rerank")
		}
		docs := make([]string, 0)
		for _, result := range queryResults.Documents[i] {
			docs = append(docs, result)
		}
		req := &RerankingRequest{
			Model:     string(r.defaultModel),
			Documents: docs,
			Query:     queryResults.QueryTexts[i],
			TopN:      r.topN,
		}
		rerankResp, err := r.sendRequest(ctx, req)
		if err != nil {
			return nil, err
		}
		rerankedResults.Ranks[r.ID()][i] = make([]float32, len(rerankResp.Results))
		for _, rr := range rerankResp.Results {
			rerankedResults.Ranks[r.ID()][i][rr.Index] = rr.RelevanceScore
		}
	}
	return rerankedResults, nil
}
