package v2

import "github.com/amikos-tech/chroma-go/pkg/embeddings"

type Operand interface {
	IsOperand()
}
type IntOperand int64

func (i IntOperand) IsOperand() {}

type FloatOperand float64

func (f FloatOperand) IsOperand() {}

type Rank interface {
	Operand
	Multiply(operand Operand) Rank
	Sub(operand Operand) Rank
	Add(operand Operand) Rank
	Div(operand Operand) Rank
	Negate() Rank
	Abs() Rank
	Exp() Rank
	Log() Rank
	Max(operand Operand) Rank
	Min(operand Operand) Rank
	WithWeight(weight float64) RankWithWeight
	MarshalJSON() ([]byte, error)
	UnmarshalJSON(b []byte) error
}

type KnnOption func(req *KnnRank) error
type KnnQueryOption func(req *KnnRank) error

func KnnQueryVector(queryVector embeddings.KnnVector) KnnQueryOption {
	return func(req *KnnRank) error {
		return nil
	}
}

func KnnQueryText(text string) KnnQueryOption {
	return func(req *KnnRank) error {
		return nil
	}
}

func WithKnnLimit(limit int) KnnOption {
	return func(req *KnnRank) error {
		return nil
	}
}

func WithKnnKey(key ProjectionKey) KnnOption {
	return func(req *KnnRank) error {
		return nil
	}
}

func WithKnnDefault(defaultScore float64) KnnOption {
	return func(req *KnnRank) error {
		return nil
	}
}

func WithKnnReturnRank() KnnOption {
	return func(req *KnnRank) error {
		return nil
	}
}

type KnnClause struct {
	Left  Operand
	Op    string
	Right Operand
}

type KnnRank struct {
	Query        interface{}   `json:"query"`
	Key          ProjectionKey `json:"key" default:"#embedding"`
	Limit        int           `json:"limit" default:"16"`
	DefaultScore float64       `json:"default_score"`
	ReturnRank   bool          `json:"return_rank" default:"false"`
	KnnClause    *KnnClause    `json:"-"` // not serialized but used internally to build expressions
}

func WithKnnRank(query KnnQueryOption, knnOptions ...KnnOption) SearchOption {
	_ = &KnnRank{} // TODO: use the KnnRank struct
	return func(req *SearchRequest) error {
		return nil
	}
}

func NewKnnRank(query KnnQueryOption, knnOptions ...KnnOption) *KnnRank {
	_ = &KnnRank{} // TODO: use the KnnRank struct
	return &KnnRank{}
}

func (k *KnnRank) IsOperand() {}

func (k *KnnRank) Multiply(operand Operand) Rank {
	return k
}
func (k *KnnRank) Sub(operand Operand) Rank {
	return k
}
func (k *KnnRank) Add(operand Operand) Rank {
	return k
}
func (k *KnnRank) Div(operand Operand) Rank {
	return k
}
func (k *KnnRank) Negate() Rank {
	return k
}
func (k *KnnRank) Abs() Rank {
	return k
}
func (k *KnnRank) Exp() Rank {
	return k
}
func (k *KnnRank) Log() Rank {
	return k
}
func (k *KnnRank) Max(operand Operand) Rank {
	return k
}
func (k *KnnRank) Min(operand Operand) Rank {
	return k
}

func (k *KnnRank) WithWeight(weight float64) RankWithWeight {
	return RankWithWeight{
		Rank:   k,
		Weight: weight,
	}
}

func (k *KnnRank) MarshalJSON() ([]byte, error) {
	return nil, nil
}

func (k *KnnRank) UnmarshalJSON(b []byte) error {
	return nil
}

type RffOption func(req *RffRank) error

type RankWithWeight struct {
	Rank   Rank
	Weight float64
}

func WithRffRanks(ranks ...RankWithWeight) RffOption {
	return func(req *RffRank) error {
		return nil
	}
}

func WithRffK(k int) RffOption {
	return func(req *RffRank) error {
		return nil
	}
}

func WithRffNormalize() RffOption {
	return func(req *RffRank) error {
		return nil
	}
}

type RffRank struct {
	ranks []RankWithWeight
}

func WithRffRank(opts ...RffOption) SearchOption {
	_ = &RffRank{} // TODO: use the RffRank struct
	return func(req *SearchRequest) error {
		return nil
	}
}

func (r *RffRank) IsOperand() {}

func (r *RffRank) Multiply(operand Operand) Rank {
	return r
}
func (r *RffRank) Sub(operand Operand) Rank {
	return r
}
func (r *RffRank) Add(operand Operand) Rank {
	return r
}
func (r *RffRank) Div(operand Operand) Rank {
	return r
}
func (r *RffRank) Negate() Rank {
	return r
}
func (r *RffRank) Abs() Rank {
	return r
}
func (r *RffRank) Exp() Rank {
	return r
}
func (r *RffRank) Log() Rank {
	return r
}
func (r *RffRank) Max(operand Operand) Rank {
	return r
}
func (r *RffRank) Min(operand Operand) Rank {
	return r
}

func (r *RffRank) MarshalJSON() ([]byte, error) {
	return nil, nil
}

func (r *RffRank) UnmarshalJSON(b []byte) error {
	return nil
}

// For RffRank this is a no-op placeholder
func (r *RffRank) WithWeight(weight float64) RankWithWeight {
	return RankWithWeight{}
}
