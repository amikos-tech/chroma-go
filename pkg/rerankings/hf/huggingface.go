package huggingface

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	chromago "github.com/amikos-tech/chroma-go/pkg/api/v2"
	chttp "github.com/amikos-tech/chroma-go/pkg/commons/http"
	"github.com/amikos-tech/chroma-go/pkg/rerankings"
)

const (
	DefaultBaseAPIEndpoint = "http://127.0.0.1:8080/rerank"
)

type RerankingRequest struct {
	Model *string  `json:"model,omitempty"`
	Query string   `json:"query"`
	Texts []string `json:"texts"`
}

type RerankingResponse []struct {
	Index int     `json:"index"`
	Score float32 `json:"score"`
}

var _ rerankings.RerankingFunction = (*HFRerankingFunction)(nil)

func getDefaults() *HFRerankingFunction {
	return &HFRerankingFunction{
		httpClient:        http.DefaultClient,
		rerankingEndpoint: DefaultBaseAPIEndpoint,
	}
}

type HFRerankingFunction struct {
	httpClient        *http.Client
	apiKey            string
	defaultModel      *rerankings.RerankingModel
	rerankingEndpoint string
}

func NewHFRerankingFunction(opts ...Option) (*HFRerankingFunction, error) {
	ef := getDefaults()
	for _, opt := range opts {
		err := opt(ef)
		if err != nil {
			return nil, err
		}
	}
	return ef, nil
}

func (r *HFRerankingFunction) sendRequest(ctx context.Context, req *RerankingRequest) (*RerankingResponse, error) {
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
	if r.apiKey != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.apiKey))
	}

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

func (r *HFRerankingFunction) Rerank(ctx context.Context, query string, results []rerankings.Result) (map[string][]rerankings.RankedResult, error) {
	docs := make([]string, 0)
	for _, result := range results {
		d, err := result.ToText()
		if err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	req := &RerankingRequest{
		Model: (*string)(r.defaultModel),
		Texts: docs,
		Query: query,
	}

	rerankResp, err := r.sendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	rankedResults := map[string][]rerankings.RankedResult{r.ID(): make([]rerankings.RankedResult, len(*rerankResp))}
	for i, rr := range *rerankResp {
		originalDoc, err := results[rr.Index].ToText()
		if err != nil {
			return nil, err
		}

		rankedResults[r.ID()][i] = rerankings.RankedResult{
			String: originalDoc,
			Index:  rr.Index,
			Rank:   rr.Score,
		}
	}
	return rankedResults, nil
}

// ID returns the of the reranking function. We use `cohere-` prefix with the default model
func (r *HFRerankingFunction) ID() string {
	if r.defaultModel != nil {
		return fmt.Sprintf("hf-%s", *(*string)(r.defaultModel))
	}
	return "hfei"
}

func (r *HFRerankingFunction) RerankResults(ctx context.Context, queryTexts []string, queryResults *chromago.QueryResultImpl) (*rerankings.RerankedChromaResults, error) {
	rerankedResults := &rerankings.RerankedChromaResults{
		QueryResultImpl: queryResults,
		QueryTexts:      queryTexts,
		Ranks:           map[string][][]float32{r.ID(): make([][]float32, len(queryResults.IDLists))},
	}
	for i, rs := range queryResults.IDLists {
		if len(rs) == 0 {
			return nil, fmt.Errorf("no results to rerank")
		}
		docs := make([]string, 0)
		for _, doc := range queryResults.DocumentsLists[i] {
			docs = append(docs, doc.ContentString())
		}
		req := &RerankingRequest{
			Model: (*string)(r.defaultModel),
			Texts: docs,
			Query: queryTexts[i],
		}
		rerankResp, err := r.sendRequest(ctx, req)
		if err != nil {
			return nil, err
		}
		rerankedResults.Ranks[r.ID()][i] = make([]float32, len(*rerankResp))
		for _, rr := range *rerankResp {
			rerankedResults.Ranks[r.ID()][i][rr.Index] = rr.Score
		}
	}
	return rerankedResults, nil
}
