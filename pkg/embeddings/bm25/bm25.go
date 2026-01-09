package bm25

import (
	"context"
	"sort"

	"github.com/pkg/errors"
	"github.com/spaolacci/murmur3"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

// Client holds the BM25 configuration
type Client struct {
	K              float64
	B              float64
	AvgDocLength   float64
	TokenMaxLength int
	Stopwords      []string
	IncludeTokens  bool
	tokenizer      *Tokenizer
}

// NewClient creates a new BM25 client with the given options
func NewClient(opts ...Option) (*Client, error) {
	c := &Client{}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, errors.Wrap(err, "failed to apply option")
		}
	}
	applyDefaults(c)
	if err := validate(c); err != nil {
		return nil, errors.Wrap(err, "validation failed")
	}
	c.tokenizer = NewTokenizer(c.Stopwords, c.TokenMaxLength)
	return c, nil
}

// EmbeddingFunction wraps Client to implement SparseEmbeddingFunction
type EmbeddingFunction struct {
	client *Client
}

// NewEmbeddingFunction creates a new BM25 embedding function
func NewEmbeddingFunction(opts ...Option) (*EmbeddingFunction, error) {
	client, err := NewClient(opts...)
	if err != nil {
		return nil, err
	}
	return &EmbeddingFunction{client: client}, nil
}

// embed computes BM25 sparse embeddings for the given texts
func (c *Client) embed(texts []string) ([]*embeddings.SparseVector, error) {
	if len(texts) == 0 {
		return []*embeddings.SparseVector{}, nil
	}

	result := make([]*embeddings.SparseVector, len(texts))
	for i, text := range texts {
		sv, err := c.embedSingle(text)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to embed text at index %d", i)
		}
		result[i] = sv
	}
	return result, nil
}

// embedSingle computes BM25 sparse embedding for a single text
func (c *Client) embedSingle(text string) (*embeddings.SparseVector, error) {
	if text == "" {
		return &embeddings.SparseVector{
			Indices: []int{},
			Values:  []float32{},
		}, nil
	}

	tokens := c.tokenizer.Tokenize(text)
	if len(tokens) == 0 {
		return &embeddings.SparseVector{
			Indices: []int{},
			Values:  []float32{},
		}, nil
	}

	// Count term frequencies
	tf := make(map[string]int)
	for _, token := range tokens {
		tf[token]++
	}

	docLen := float64(len(tokens))

	// Sort tokens for deterministic output
	uniqueTokens := make([]string, 0, len(tf))
	for token := range tf {
		uniqueTokens = append(uniqueTokens, token)
	}
	sort.Strings(uniqueTokens)

	indices := make([]int, 0, len(tf))
	values := make([]float32, 0, len(tf))
	var labels []string
	if c.IncludeTokens {
		labels = make([]string, 0, len(tf))
	}

	for _, token := range uniqueTokens {
		freq := tf[token]

		// Compute BM25 score
		tfFloat := float64(freq)
		denominator := tfFloat + c.K*(1-c.B+c.B*docLen/c.AvgDocLength)
		score := tfFloat * (c.K + 1) / denominator

		// Hash token to index using murmur3 (absolute value)
		hash := murmur3.Sum32([]byte(token))
		index := int(hash)
		if index < 0 {
			index = -index
		}

		indices = append(indices, index)
		values = append(values, float32(score))
		if c.IncludeTokens {
			labels = append(labels, token)
		}
	}

	sv := &embeddings.SparseVector{
		Indices: indices,
		Values:  values,
	}
	if c.IncludeTokens {
		sv.Labels = labels
	}

	if err := sv.Validate(); err != nil {
		return nil, errors.Wrap(err, "generated invalid sparse vector")
	}
	return sv, nil
}

// EmbedDocumentsSparse returns a sparse vector for each text
func (e *EmbeddingFunction) EmbedDocumentsSparse(_ context.Context, texts []string) ([]*embeddings.SparseVector, error) {
	return e.client.embed(texts)
}

// EmbedQuerySparse embeds a single text as a sparse vector
func (e *EmbeddingFunction) EmbedQuerySparse(_ context.Context, text string) (*embeddings.SparseVector, error) {
	results, err := e.client.embed([]string{text})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, errors.New("no embedding returned")
	}
	return results[0], nil
}

// Ensure EmbeddingFunction implements SparseEmbeddingFunction
var _ embeddings.SparseEmbeddingFunction = (*EmbeddingFunction)(nil)
