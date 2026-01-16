package cohere

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	chromago "github.com/amikos-tech/chroma-go/pkg/api/v2"
	ccommons "github.com/amikos-tech/chroma-go/pkg/commons/cohere"
	"github.com/amikos-tech/chroma-go/pkg/rerankings"
)

const (
	DefaultRerankingEndpoint                             = "rerank"
	ModelRerankEnglishV30      rerankings.RerankingModel = "rerank-english-v3.0"
	DefaultModel               rerankings.RerankingModel = ModelRerankEnglishV30
	ModelRerankMultilingualV30 rerankings.RerankingModel = "rerank-multilingual-v3.0"
	ModelRerankEnglishV20      rerankings.RerankingModel = "rerank-english-v2.0"
	ModelRerankMultilingualV20 rerankings.RerankingModel = "rerank-multilingual-v2.0"
)

type RerankRequest struct {
	Model           string   `json:"model"`
	Query           string   `json:"query"`
	Documents       []any    `json:"documents"`
	TopN            int      `json:"top_n,omitempty"`
	RerankFields    []string `json:"rerank_fields,omitempty"`
	ReturnDocuments bool     `json:"return_documents,omitempty"`
	MaxChunksPerDoc int      `json:"max_chunks_per_doc,omitempty"`
}

type RerankResult struct {
	Document struct {
		Text string `json:"text"`
	} `json:"document"`
	RelevanceScore float32 `json:"relevance_score"`
	Index          int     `json:"index"`
}

type RerankResponse struct {
	ID      string         `json:"id"`
	Results []RerankResult `json:"results"`
	Meta    map[string]any `json:"meta"`
}

type CohereRerankingFunction struct {
	ccommons.CohereClient
	TopN            int
	RerankFields    []string
	ReturnDocuments bool
	MaxChunksPerDoc int
	RerankEndpoint  string
}

var _ rerankings.RerankingFunction = &CohereRerankingFunction{}

func NewCohereRerankingFunction(opts ...Option) (*CohereRerankingFunction, error) {
	rf := &CohereRerankingFunction{}
	ccOpts := make([]ccommons.Option, 0)
	opts = append(opts, WithDefaultModel(DefaultModel))
	// stagger the options to pass to the cohere client
	for _, opt := range opts {
		ccOpts = append(ccOpts, opt(rf))
	}
	cohereCommonClient, err := ccommons.NewCohereClient(ccOpts...)
	if err != nil {
		return nil, err
	}
	rf.CohereClient = *cohereCommonClient
	rf.RerankEndpoint = cohereCommonClient.GetAPIEndpoint(DefaultRerankingEndpoint)
	return rf, nil
}

func (c CohereRerankingFunction) Rerank(ctx context.Context, query string, results []rerankings.Result) (map[string][]rerankings.RankedResult, error) {
	docs := make([]any, 0)
	for _, result := range results {
		d, err := result.ToText()
		if err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	req := &RerankRequest{
		Model:           string(c.DefaultModel),
		Query:           query,
		Documents:       docs,
		TopN:            c.TopN,
		RerankFields:    c.RerankFields,
		ReturnDocuments: c.ReturnDocuments,
		MaxChunksPerDoc: c.MaxChunksPerDoc,
	}
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := c.GetRequest(ctx, "POST", c.RerankEndpoint, string(reqJSON))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.DoRequest(httpReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		all, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("rerank failed with status code: %d, failed to read response: %v", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("rerank failed with status code: %d, response: %s", resp.StatusCode, string(all))
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("Error closing body: %v\n", err)
		}
	}(resp.Body)
	var rerankResp RerankResponse
	err = json.NewDecoder(resp.Body).Decode(&rerankResp)
	if err != nil {
		return nil, err
	}
	rankedResults := map[string][]rerankings.RankedResult{c.ID(): make([]rerankings.RankedResult, len(rerankResp.Results))}
	for i, rr := range rerankResp.Results {
		if rr.Index < 0 || rr.Index >= len(results) {
			return nil, fmt.Errorf("invalid index %d from reranking API (valid range: 0-%d)", rr.Index, len(results)-1)
		}
		originalDoc, err := results[rr.Index].ToText()
		if err != nil {
			return nil, err
		}
		if rr.Document.Text != "" {
			originalDoc = rr.Document.Text
		}
		rankedResults[c.ID()][i] = rerankings.RankedResult{
			String: originalDoc,
			Index:  rr.Index,
			Rank:   rr.RelevanceScore,
		}
	}

	return rankedResults, nil
}

// ID returns the of the reranking function. We use `cohere-` prefix with the default model
func (c CohereRerankingFunction) ID() string {
	return fmt.Sprintf("cohere-%s", c.DefaultModel)
}

func (c CohereRerankingFunction) RerankResults(ctx context.Context, queryTexts []string, queryResults *chromago.QueryResultImpl) (*rerankings.RerankedChromaResults, error) {
	if len(queryTexts) != len(queryResults.IDLists) {
		return nil, fmt.Errorf("queryTexts length (%d) does not match IDLists length (%d)", len(queryTexts), len(queryResults.IDLists))
	}
	if len(queryResults.DocumentsLists) != len(queryResults.IDLists) {
		return nil, fmt.Errorf("DocumentsLists length (%d) does not match IDLists length (%d)", len(queryResults.DocumentsLists), len(queryResults.IDLists))
	}
	rerankedResults := &rerankings.RerankedChromaResults{
		QueryResultImpl: queryResults,
		QueryTexts:      queryTexts,
		Ranks:           map[string][][]float32{c.ID(): make([][]float32, len(queryResults.IDLists))},
	}
	for i, r := range queryResults.IDLists {
		if len(r) == 0 {
			return nil, fmt.Errorf("no results to rerank")
		}
		docs := make([]any, 0)
		for _, doc := range queryResults.DocumentsLists[i] {
			docs = append(docs, doc.ContentString())
		}
		req := &RerankRequest{
			Model:           string(c.DefaultModel),
			Query:           queryTexts[i],
			Documents:       docs,
			TopN:            c.TopN,
			RerankFields:    c.RerankFields,
			ReturnDocuments: c.ReturnDocuments,
			MaxChunksPerDoc: c.MaxChunksPerDoc,
		}

		var bytesBuffer bytes.Buffer
		err := json.NewEncoder(&bytesBuffer).Encode(req)
		if err != nil {
			return nil, err
		}
		httpReq, err := c.GetRequest(ctx, "POST", c.RerankEndpoint, bytesBuffer.String())
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Accept", "application/json")
		httpReq.Header.Set("Content-Type", "application/json")
		resp, err := c.DoRequest(httpReq)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != 200 {
			all, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return nil, fmt.Errorf("rerank failed with status code: %d, failed to read response: %v", resp.StatusCode, err)
			}
			return nil, fmt.Errorf("rerank failed with status code: %d, response: %s", resp.StatusCode, string(all))
		}
		var rerankResp RerankResponse
		err = json.NewDecoder(resp.Body).Decode(&rerankResp)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		rerankedResults.Ranks[c.ID()][i] = make([]float32, len(rerankResp.Results))
		for _, rr := range rerankResp.Results {
			if rr.Index < 0 || rr.Index >= len(rerankedResults.Ranks[c.ID()][i]) {
				return nil, fmt.Errorf("invalid index %d from reranking API (valid range: 0-%d)", rr.Index, len(rerankedResults.Ranks[c.ID()][i])-1)
			}
			rerankedResults.Ranks[c.ID()][i][rr.Index] = rr.RelevanceScore
		}
	}
	return rerankedResults, nil
}
