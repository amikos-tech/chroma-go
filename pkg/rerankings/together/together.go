package together

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
	DefaultBaseAPIEndpoint                           = "https://api.together.xyz/v1/rerank"
	DefaultRerankingModel  rerankings.RerankingModel = "Salesforce/Llama-Rank-V1"
)

type RerankingRequest struct {
	Model           string   `json:"model"`
	Query           string   `json:"query"`
	Documents       []string `json:"documents"`
	TopN            *int     `json:"top_n,omitempty"`
	ReturnDocuments *bool    `json:"return_documents,omitempty"`
}

type RerankingResponse struct {
	Model   string `json:"model"`
	Object  string `json:"object"`
	Results []struct {
		Index          int     `json:"index"`
		RelevanceScore float32 `json:"relevance_score"`
		Document       struct {
			Text string `json:"text"`
		} `json:"document,omitempty"`
	} `json:"results"`
}

var _ rerankings.RerankingFunction = (*TogetherRerankingFunction)(nil)

func getDefaults() *TogetherRerankingFunction {
	var returnDocuments = true
	return &TogetherRerankingFunction{
		httpClient:        http.DefaultClient,
		defaultModel:      DefaultRerankingModel,
		rerankingEndpoint: DefaultBaseAPIEndpoint,
		returnDocuments:   &returnDocuments,
		topN:              nil,
	}
}

type TogetherRerankingFunction struct {
	httpClient        *http.Client
	apiKey            string
	defaultModel      rerankings.RerankingModel
	rerankingEndpoint string
	returnDocuments   *bool
	topN              *int
}

func NewTogetherRerankingFunction(opts ...Option) (*TogetherRerankingFunction, error) {
	rf := getDefaults()
	for _, opt := range opts {
		err := opt(rf)
		if err != nil {
			return nil, err
		}
	}
	return rf, nil
}

func (r *TogetherRerankingFunction) sendRequest(ctx context.Context, req *RerankingRequest) (*RerankingResponse, error) {
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

	httpReq, err := http.NewRequestWithContext(ctx, "POST", r.rerankingEndpoint, bytes.NewBuffer(payload))
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
		return nil, fmt.Errorf("unexpected response %v: %s", resp.Status, respData)
	}
	var response *RerankingResponse
	if err := json.Unmarshal(respData, &response); err != nil {
		return nil, err
	}

	return response, nil
}

func (r *TogetherRerankingFunction) Rerank(ctx context.Context, query string, results []rerankings.Result) (map[string][]rerankings.RankedResult, error) {
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
		rankedResults[r.ID()][i] = rerankings.RankedResult{
			String: originalDoc,
			Index:  rr.Index,
			Rank:   rr.RelevanceScore,
		}
	}
	return rankedResults, nil
}

func (r *TogetherRerankingFunction) ID() string {
	return fmt.Sprintf("together-%s", r.defaultModel)
}

func (r *TogetherRerankingFunction) RerankResults(ctx context.Context, queryTexts []string, queryResults *chromago.QueryResultImpl) (*rerankings.RerankedChromaResults, error) {
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
			Model:     string(r.defaultModel),
			Documents: docs,
			Query:     queryTexts[i],
			TopN:      r.topN,
		}
		rerankResp, err := r.sendRequest(ctx, req)
		if err != nil {
			return nil, err
		}
		rerankedResults.Ranks[r.ID()][i] = make([]float32, len(queryResults.DocumentsLists[i]))
		for _, rr := range rerankResp.Results {
			if rr.Index < 0 || rr.Index >= len(rerankedResults.Ranks[r.ID()][i]) {
				return nil, fmt.Errorf("invalid index %d from reranking API (valid range: 0-%d)", rr.Index, len(rerankedResults.Ranks[r.ID()][i])-1)
			}
			rerankedResults.Ranks[r.ID()][i][rr.Index] = rr.RelevanceScore
		}
	}
	return rerankedResults, nil
}
