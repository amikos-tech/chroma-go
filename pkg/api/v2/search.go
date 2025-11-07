package v2

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

// Search API Implementation
//
// IMPORTANT: The Search API requires Chroma Cloud or a Chroma server with search API support (v1.3.0+).
// This feature is NOT available on older local single-node Chroma deployments.
//
// If you try to use the Search API with an unsupported server, you will receive an error message
// directing you to use the Query API instead.
//
// Supported environments:
//   - Chroma Cloud (fully supported)
//   - Chroma Server v1.3.0+ with search API enabled
//   - NOT supported on local single-node Chroma < v1.3.0
//
// For more information, see: https://docs.trychroma.com/cloud/search-api/overview

// SelectKey represents a field key that can be selected in search results.
// These include both metadata fields and special system fields.
type SelectKey string

const (
	// System fields (prefixed with #)
	SelectID        SelectKey = "#id"
	SelectDocument  SelectKey = "#document"
	SelectEmbedding SelectKey = "#embedding"
	SelectScore     SelectKey = "#score"
	SelectURI       SelectKey = "#uri"
)

// SearchLimit represents pagination configuration for search results
type SearchLimit struct {
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

// RankExpression represents a ranking operation in a search query.
// Implementations include KNN, RRF, and arithmetic combinations.
type RankExpression interface {
	// ToJSON serializes the rank expression to a JSON-compatible structure
	ToJSON() interface{}
	// Validate checks if the rank expression is valid
	Validate() error
}

// KnnRank represents a K-nearest neighbors ranking operation
type KnnRank struct {
	QueryEmbeddings []embeddings.Embedding `json:"query_embeddings,omitempty"`
	QueryTexts      []string               `json:"query_texts,omitempty"`
	K               int                    `json:"k,omitempty"`
}

func (k *KnnRank) ToJSON() interface{} {
	result := map[string]interface{}{
		"type": "knn",
	}
	if len(k.QueryEmbeddings) > 0 {
		result["query_embeddings"] = k.QueryEmbeddings
	}
	if len(k.QueryTexts) > 0 {
		result["query_texts"] = k.QueryTexts
	}
	if k.K > 0 {
		result["k"] = k.K
	}
	return result
}

func (k *KnnRank) Validate() error {
	if len(k.QueryEmbeddings) == 0 && len(k.QueryTexts) == 0 {
		return errors.New("knn rank requires at least one query embedding or query text")
	}
	if k.K <= 0 {
		return errors.New("knn rank k must be greater than 0")
	}
	return nil
}

// RrfRank represents a Reciprocal Rank Fusion operation
// that combines multiple ranking expressions
type RrfRank struct {
	Ranks     []RankExpression `json:"ranks"`
	K         int              `json:"k,omitempty"`
	Normalize bool             `json:"normalize,omitempty"`
}

func (r *RrfRank) ToJSON() interface{} {
	ranks := make([]interface{}, len(r.Ranks))
	for i, rank := range r.Ranks {
		ranks[i] = rank.ToJSON()
	}
	result := map[string]interface{}{
		"type":  "rrf",
		"ranks": ranks,
	}
	if r.K > 0 {
		result["k"] = r.K
	}
	if r.Normalize {
		result["normalize"] = r.Normalize
	}
	return result
}

func (r *RrfRank) Validate() error {
	if len(r.Ranks) < 2 {
		return errors.New("rrf rank requires at least 2 ranking expressions")
	}
	for i, rank := range r.Ranks {
		if err := rank.Validate(); err != nil {
			return errors.Wrapf(err, "invalid rank at index %d", i)
		}
	}
	return nil
}

// ArithmeticRank represents arithmetic operations on rank expressions
type ArithmeticRank struct {
	Operator string         `json:"operator"` // "add", "sub", "mul", "div"
	Left     RankExpression `json:"left"`
	Right    RankExpression `json:"right"`
}

func (a *ArithmeticRank) ToJSON() interface{} {
	return map[string]interface{}{
		"type":     a.Operator,
		"operands": []interface{}{a.Left.ToJSON(), a.Right.ToJSON()},
	}
}

func (a *ArithmeticRank) Validate() error {
	switch a.Operator {
	case "add", "sub", "mul", "div":
		// valid operators
	default:
		return errors.Errorf("invalid arithmetic operator: %s", a.Operator)
	}
	if err := a.Left.Validate(); err != nil {
		return errors.Wrap(err, "invalid left operand")
	}
	if err := a.Right.Validate(); err != nil {
		return errors.Wrap(err, "invalid right operand")
	}
	return nil
}

// FunctionRank represents function operations on rank expressions
type FunctionRank struct {
	Function string         `json:"function"` // "exp", "log", "abs", "max", "min"
	Operand  RankExpression `json:"operand"`
}

func (f *FunctionRank) ToJSON() interface{} {
	return map[string]interface{}{
		"type":    f.Function,
		"operand": f.Operand.ToJSON(),
	}
}

func (f *FunctionRank) Validate() error {
	switch f.Function {
	case "exp", "log", "abs", "max", "min":
		// valid functions
	default:
		return errors.Errorf("invalid function: %s", f.Function)
	}
	if err := f.Operand.Validate(); err != nil {
		return errors.Wrap(err, "invalid operand")
	}
	return nil
}

// CollectionSearchOp represents a search operation on a collection
type CollectionSearchOp struct {
	Where  WhereFilter    `json:"where,omitempty"`
	Rank   RankExpression `json:"rank,omitempty"`
	Limit  *SearchLimit   `json:"limit,omitempty"`
	Select []SelectKey    `json:"select,omitempty"`
}

func NewCollectionSearchOp(opts ...CollectionSearchOption) (*CollectionSearchOp, error) {
	search := &CollectionSearchOp{}
	for _, opt := range opts {
		if err := opt(search); err != nil {
			return nil, err
		}
	}
	return search, nil
}

func (c *CollectionSearchOp) PrepareAndValidate() error {
	if c.Rank == nil {
		return errors.New("rank expression is required for search")
	}
	if err := c.Rank.Validate(); err != nil {
		return errors.Wrap(err, "invalid rank expression")
	}
	if c.Where != nil {
		if err := c.Where.Validate(); err != nil {
			return errors.Wrap(err, "invalid where filter")
		}
	}
	if c.Limit != nil {
		if c.Limit.Limit < 0 {
			return errors.New("limit must be non-negative")
		}
		if c.Limit.Offset < 0 {
			return errors.New("offset must be non-negative")
		}
	}
	return nil
}

func (c *CollectionSearchOp) EmbedData(ctx context.Context, ef embeddings.EmbeddingFunction) error {
	if c.Rank == nil {
		return nil
	}
	return embedRankExpression(ctx, c.Rank, ef)
}

// embedRankExpression recursively traverses rank expressions and embeds query texts in KNN ranks
func embedRankExpression(ctx context.Context, rank RankExpression, ef embeddings.EmbeddingFunction) error {
	switch r := rank.(type) {
	case *KnnRank:
		// Handle KNN rank with query texts
		if len(r.QueryTexts) > 0 && len(r.QueryEmbeddings) == 0 {
			if ef == nil {
				return errors.New("embedding function is required for query texts")
			}
			embeds, err := ef.EmbedDocuments(ctx, r.QueryTexts)
			if err != nil {
				return errors.Wrap(err, "failed to embed query texts")
			}
			r.QueryEmbeddings = embeds
		}
		return nil

	case *RrfRank:
		// Recursively embed all child ranks in RRF
		for i, childRank := range r.Ranks {
			if err := embedRankExpression(ctx, childRank, ef); err != nil {
				return errors.Wrapf(err, "failed to embed rank %d in RRF", i)
			}
		}
		return nil

	case *ArithmeticRank:
		// Recursively embed left and right operands
		if err := embedRankExpression(ctx, r.Left, ef); err != nil {
			return errors.Wrap(err, "failed to embed left operand in arithmetic expression")
		}
		if err := embedRankExpression(ctx, r.Right, ef); err != nil {
			return errors.Wrap(err, "failed to embed right operand in arithmetic expression")
		}
		return nil

	case *FunctionRank:
		// Recursively embed the function operand
		if err := embedRankExpression(ctx, r.Operand, ef); err != nil {
			return errors.Wrap(err, "failed to embed operand in function expression")
		}
		return nil

	default:
		// Unknown rank type - no embedding needed
		return nil
	}
}

func (c *CollectionSearchOp) MarshalJSON() ([]byte, error) {
	result := make(map[string]interface{})

	if c.Where != nil {
		whereJSON, err := c.Where.MarshalJSON()
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal where filter")
		}
		var whereMap interface{}
		if err := json.Unmarshal(whereJSON, &whereMap); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal where filter")
		}
		result["where"] = whereMap
	}

	if c.Rank != nil {
		result["rank"] = c.Rank.ToJSON()
	}

	if c.Limit != nil {
		result["limit"] = map[string]interface{}{
			"limit":  c.Limit.Limit,
			"offset": c.Limit.Offset,
		}
	}

	if len(c.Select) > 0 {
		selectKeys := make([]string, len(c.Select))
		for i, key := range c.Select {
			selectKeys[i] = string(key)
		}
		result["select"] = selectKeys
	}

	return json.Marshal(result)
}

func (c *CollectionSearchOp) UnmarshalJSON(b []byte) error {
	var temp map[string]interface{}
	if err := json.Unmarshal(b, &temp); err != nil {
		return err
	}
	// TODO: Implement if needed for response parsing
	return errors.New("unmarshal not implemented for CollectionSearchOp")
}

func (c *CollectionSearchOp) Resource() Resource {
	return ResourceCollection
}

func (c *CollectionSearchOp) Operation() OperationType {
	return "search"
}

// CollectionSearchOption is a functional option for configuring a search operation
type CollectionSearchOption func(search *CollectionSearchOp) error

// WithSearchWhere adds a where filter to the search
func WithSearchWhere(where WhereFilter) CollectionSearchOption {
	return func(search *CollectionSearchOp) error {
		search.Where = where
		return nil
	}
}

// WithSearchRankKnn adds a KNN ranking to the search with query embeddings
func WithSearchRankKnn(queryEmbeddings []embeddings.Embedding, k int) CollectionSearchOption {
	return func(search *CollectionSearchOp) error {
		if k <= 0 {
			return errors.New("k must be greater than 0")
		}
		search.Rank = &KnnRank{
			QueryEmbeddings: queryEmbeddings,
			K:               k,
		}
		return nil
	}
}

// WithSearchRankKnnTexts adds a KNN ranking to the search with query texts
func WithSearchRankKnnTexts(queryTexts []string, k int) CollectionSearchOption {
	return func(search *CollectionSearchOp) error {
		if len(queryTexts) == 0 {
			return errors.New("at least one query text is required")
		}
		if k <= 0 {
			return errors.New("k must be greater than 0")
		}
		search.Rank = &KnnRank{
			QueryTexts: queryTexts,
			K:          k,
		}
		return nil
	}
}

// WithSearchRankRrf adds a Reciprocal Rank Fusion ranking to the search
func WithSearchRankRrf(ranks []RankExpression, k int, normalize bool) CollectionSearchOption {
	return func(search *CollectionSearchOp) error {
		if len(ranks) < 2 {
			return errors.New("rrf requires at least 2 ranking expressions")
		}
		search.Rank = &RrfRank{
			Ranks:     ranks,
			K:         k,
			Normalize: normalize,
		}
		return nil
	}
}

// WithSearchRank sets a custom rank expression
func WithSearchRank(rank RankExpression) CollectionSearchOption {
	return func(search *CollectionSearchOp) error {
		if rank == nil {
			return errors.New("rank expression cannot be nil")
		}
		search.Rank = rank
		return nil
	}
}

// WithSearchLimit sets the limit and offset for pagination
func WithSearchLimit(limit, offset int) CollectionSearchOption {
	return func(search *CollectionSearchOp) error {
		if limit < 0 {
			return errors.New("limit must be non-negative")
		}
		if offset < 0 {
			return errors.New("offset must be non-negative")
		}
		search.Limit = &SearchLimit{
			Limit:  limit,
			Offset: offset,
		}
		return nil
	}
}

// WithSearchSelect sets the fields to select in the results
func WithSearchSelect(keys ...SelectKey) CollectionSearchOption {
	return func(search *CollectionSearchOp) error {
		if len(keys) == 0 {
			return errors.New("at least one select key is required")
		}
		search.Select = keys
		return nil
	}
}

// Arithmetic rank constructors

// AddRanks creates a rank expression that adds two rank expressions
func AddRanks(left, right RankExpression) RankExpression {
	return &ArithmeticRank{
		Operator: "add",
		Left:     left,
		Right:    right,
	}
}

// SubRanks creates a rank expression that subtracts two rank expressions
func SubRanks(left, right RankExpression) RankExpression {
	return &ArithmeticRank{
		Operator: "sub",
		Left:     left,
		Right:    right,
	}
}

// MulRanks creates a rank expression that multiplies two rank expressions
func MulRanks(left, right RankExpression) RankExpression {
	return &ArithmeticRank{
		Operator: "mul",
		Left:     left,
		Right:    right,
	}
}

// DivRanks creates a rank expression that divides two rank expressions
func DivRanks(left, right RankExpression) RankExpression {
	return &ArithmeticRank{
		Operator: "div",
		Left:     left,
		Right:    right,
	}
}

// Function rank constructors

// ExpRank creates a rank expression that applies the exponential function
func ExpRank(operand RankExpression) RankExpression {
	return &FunctionRank{
		Function: "exp",
		Operand:  operand,
	}
}

// LogRank creates a rank expression that applies the logarithm function
func LogRank(operand RankExpression) RankExpression {
	return &FunctionRank{
		Function: "log",
		Operand:  operand,
	}
}

// AbsRank creates a rank expression that applies the absolute value function
func AbsRank(operand RankExpression) RankExpression {
	return &FunctionRank{
		Function: "abs",
		Operand:  operand,
	}
}

// MaxRank creates a rank expression that applies the maximum function
func MaxRank(operand RankExpression) RankExpression {
	return &FunctionRank{
		Function: "max",
		Operand:  operand,
	}
}

// MinRank creates a rank expression that applies the minimum function
func MinRank(operand RankExpression) RankExpression {
	return &FunctionRank{
		Function: "min",
		Operand:  operand,
	}
}
