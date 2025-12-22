package v2

import (
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
)

// Operand represents a value that can be used in rank expressions
type Operand interface {
	IsOperand()
}

// IntOperand represents an integer operand
type IntOperand int64

func (i IntOperand) IsOperand() {}

// FloatOperand represents a float operand
type FloatOperand float64

func (f FloatOperand) IsOperand() {}

// Rank represents a ranking expression that can be serialized to JSON
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

// RankWithWeight pairs a Rank with a weight for use in RRF
type RankWithWeight struct {
	Rank   Rank
	Weight float64
}

// --- Val (constant value) ---

// ValRank represents a constant value in rank expressions
type ValRank struct {
	value float64
}

// Val creates a new constant value rank
func Val(value float64) *ValRank {
	return &ValRank{value: value}
}

func (v *ValRank) IsOperand() {}

func (v *ValRank) Multiply(operand Operand) Rank {
	return &MulRank{ranks: []Rank{v, operandToRank(operand)}}
}

func (v *ValRank) Sub(operand Operand) Rank {
	return &SubRank{left: v, right: operandToRank(operand)}
}

func (v *ValRank) Add(operand Operand) Rank {
	return &SumRank{ranks: []Rank{v, operandToRank(operand)}}
}

func (v *ValRank) Div(operand Operand) Rank {
	return &DivRank{left: v, right: operandToRank(operand)}
}

func (v *ValRank) Negate() Rank {
	return &MulRank{ranks: []Rank{Val(-1), v}}
}

func (v *ValRank) Abs() Rank {
	return &AbsRank{rank: v}
}

func (v *ValRank) Exp() Rank {
	return &ExpRank{rank: v}
}

func (v *ValRank) Log() Rank {
	return &LogRank{rank: v}
}

func (v *ValRank) Max(operand Operand) Rank {
	return &MaxRank{ranks: []Rank{v, operandToRank(operand)}}
}

func (v *ValRank) Min(operand Operand) Rank {
	return &MinRank{ranks: []Rank{v, operandToRank(operand)}}
}

func (v *ValRank) WithWeight(weight float64) RankWithWeight {
	return RankWithWeight{Rank: v, Weight: weight}
}

func (v *ValRank) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]float64{"$val": v.value})
}

func (v *ValRank) UnmarshalJSON(b []byte) error {
	var data map[string]float64
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}
	if val, ok := data["$val"]; ok {
		v.value = val
	}
	return nil
}

// --- Arithmetic Types ---

// SumRank represents addition of multiple ranks: {"$sum": [...]}
type SumRank struct {
	ranks []Rank
}

func (s *SumRank) IsOperand() {}

func (s *SumRank) Multiply(operand Operand) Rank {
	return &MulRank{ranks: []Rank{s, operandToRank(operand)}}
}

func (s *SumRank) Sub(operand Operand) Rank {
	return &SubRank{left: s, right: operandToRank(operand)}
}

func (s *SumRank) Add(operand Operand) Rank {
	r := operandToRank(operand)
	if sum, ok := r.(*SumRank); ok {
		return &SumRank{ranks: append(s.ranks, sum.ranks...)}
	}
	return &SumRank{ranks: append(s.ranks, r)}
}

func (s *SumRank) Div(operand Operand) Rank {
	return &DivRank{left: s, right: operandToRank(operand)}
}

func (s *SumRank) Negate() Rank {
	return &MulRank{ranks: []Rank{Val(-1), s}}
}

func (s *SumRank) Abs() Rank {
	return &AbsRank{rank: s}
}

func (s *SumRank) Exp() Rank {
	return &ExpRank{rank: s}
}

func (s *SumRank) Log() Rank {
	return &LogRank{rank: s}
}

func (s *SumRank) Max(operand Operand) Rank {
	return &MaxRank{ranks: []Rank{s, operandToRank(operand)}}
}

func (s *SumRank) Min(operand Operand) Rank {
	return &MinRank{ranks: []Rank{s, operandToRank(operand)}}
}

func (s *SumRank) WithWeight(weight float64) RankWithWeight {
	return RankWithWeight{Rank: s, Weight: weight}
}

func (s *SumRank) MarshalJSON() ([]byte, error) {
	rankMaps := make([]json.RawMessage, len(s.ranks))
	for i, r := range s.ranks {
		data, err := r.MarshalJSON()
		if err != nil {
			return nil, err
		}
		rankMaps[i] = data
	}
	return json.Marshal(map[string][]json.RawMessage{"$sum": rankMaps})
}

func (s *SumRank) UnmarshalJSON(b []byte) error {
	return nil // TODO: implement if needed
}

// SubRank represents subtraction: {"$sub": {"left": ..., "right": ...}}
type SubRank struct {
	left  Rank
	right Rank
}

func (s *SubRank) IsOperand() {}

func (s *SubRank) Multiply(operand Operand) Rank {
	return &MulRank{ranks: []Rank{s, operandToRank(operand)}}
}

func (s *SubRank) Sub(operand Operand) Rank {
	return &SubRank{left: s, right: operandToRank(operand)}
}

func (s *SubRank) Add(operand Operand) Rank {
	return &SumRank{ranks: []Rank{s, operandToRank(operand)}}
}

func (s *SubRank) Div(operand Operand) Rank {
	return &DivRank{left: s, right: operandToRank(operand)}
}

func (s *SubRank) Negate() Rank {
	return &MulRank{ranks: []Rank{Val(-1), s}}
}

func (s *SubRank) Abs() Rank {
	return &AbsRank{rank: s}
}

func (s *SubRank) Exp() Rank {
	return &ExpRank{rank: s}
}

func (s *SubRank) Log() Rank {
	return &LogRank{rank: s}
}

func (s *SubRank) Max(operand Operand) Rank {
	return &MaxRank{ranks: []Rank{s, operandToRank(operand)}}
}

func (s *SubRank) Min(operand Operand) Rank {
	return &MinRank{ranks: []Rank{s, operandToRank(operand)}}
}

func (s *SubRank) WithWeight(weight float64) RankWithWeight {
	return RankWithWeight{Rank: s, Weight: weight}
}

func (s *SubRank) MarshalJSON() ([]byte, error) {
	leftData, err := s.left.MarshalJSON()
	if err != nil {
		return nil, err
	}
	rightData, err := s.right.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]map[string]json.RawMessage{
		"$sub": {"left": leftData, "right": rightData},
	})
}

func (s *SubRank) UnmarshalJSON(b []byte) error {
	return nil // TODO: implement if needed
}

// MulRank represents multiplication of multiple ranks: {"$mul": [...]}
type MulRank struct {
	ranks []Rank
}

func (m *MulRank) IsOperand() {}

func (m *MulRank) Multiply(operand Operand) Rank {
	r := operandToRank(operand)
	if mul, ok := r.(*MulRank); ok {
		return &MulRank{ranks: append(m.ranks, mul.ranks...)}
	}
	return &MulRank{ranks: append(m.ranks, r)}
}

func (m *MulRank) Sub(operand Operand) Rank {
	return &SubRank{left: m, right: operandToRank(operand)}
}

func (m *MulRank) Add(operand Operand) Rank {
	return &SumRank{ranks: []Rank{m, operandToRank(operand)}}
}

func (m *MulRank) Div(operand Operand) Rank {
	return &DivRank{left: m, right: operandToRank(operand)}
}

func (m *MulRank) Negate() Rank {
	return &MulRank{ranks: []Rank{Val(-1), m}}
}

func (m *MulRank) Abs() Rank {
	return &AbsRank{rank: m}
}

func (m *MulRank) Exp() Rank {
	return &ExpRank{rank: m}
}

func (m *MulRank) Log() Rank {
	return &LogRank{rank: m}
}

func (m *MulRank) Max(operand Operand) Rank {
	return &MaxRank{ranks: []Rank{m, operandToRank(operand)}}
}

func (m *MulRank) Min(operand Operand) Rank {
	return &MinRank{ranks: []Rank{m, operandToRank(operand)}}
}

func (m *MulRank) WithWeight(weight float64) RankWithWeight {
	return RankWithWeight{Rank: m, Weight: weight}
}

func (m *MulRank) MarshalJSON() ([]byte, error) {
	rankMaps := make([]json.RawMessage, len(m.ranks))
	for i, r := range m.ranks {
		data, err := r.MarshalJSON()
		if err != nil {
			return nil, err
		}
		rankMaps[i] = data
	}
	return json.Marshal(map[string][]json.RawMessage{"$mul": rankMaps})
}

func (m *MulRank) UnmarshalJSON(b []byte) error {
	return nil // TODO: implement if needed
}

// DivRank represents division: {"$div": {"left": ..., "right": ...}}
type DivRank struct {
	left  Rank
	right Rank
}

func (d *DivRank) IsOperand() {}

func (d *DivRank) Multiply(operand Operand) Rank {
	return &MulRank{ranks: []Rank{d, operandToRank(operand)}}
}

func (d *DivRank) Sub(operand Operand) Rank {
	return &SubRank{left: d, right: operandToRank(operand)}
}

func (d *DivRank) Add(operand Operand) Rank {
	return &SumRank{ranks: []Rank{d, operandToRank(operand)}}
}

func (d *DivRank) Div(operand Operand) Rank {
	return &DivRank{left: d, right: operandToRank(operand)}
}

func (d *DivRank) Negate() Rank {
	return &MulRank{ranks: []Rank{Val(-1), d}}
}

func (d *DivRank) Abs() Rank {
	return &AbsRank{rank: d}
}

func (d *DivRank) Exp() Rank {
	return &ExpRank{rank: d}
}

func (d *DivRank) Log() Rank {
	return &LogRank{rank: d}
}

func (d *DivRank) Max(operand Operand) Rank {
	return &MaxRank{ranks: []Rank{d, operandToRank(operand)}}
}

func (d *DivRank) Min(operand Operand) Rank {
	return &MinRank{ranks: []Rank{d, operandToRank(operand)}}
}

func (d *DivRank) WithWeight(weight float64) RankWithWeight {
	return RankWithWeight{Rank: d, Weight: weight}
}

func (d *DivRank) MarshalJSON() ([]byte, error) {
	leftData, err := d.left.MarshalJSON()
	if err != nil {
		return nil, err
	}
	rightData, err := d.right.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]map[string]json.RawMessage{
		"$div": {"left": leftData, "right": rightData},
	})
}

func (d *DivRank) UnmarshalJSON(b []byte) error {
	return nil // TODO: implement if needed
}

// --- Math Function Types ---

// AbsRank represents absolute value: {"$abs": rank}
type AbsRank struct {
	rank Rank
}

func (a *AbsRank) IsOperand() {}

func (a *AbsRank) Multiply(operand Operand) Rank {
	return &MulRank{ranks: []Rank{a, operandToRank(operand)}}
}

func (a *AbsRank) Sub(operand Operand) Rank {
	return &SubRank{left: a, right: operandToRank(operand)}
}

func (a *AbsRank) Add(operand Operand) Rank {
	return &SumRank{ranks: []Rank{a, operandToRank(operand)}}
}

func (a *AbsRank) Div(operand Operand) Rank {
	return &DivRank{left: a, right: operandToRank(operand)}
}

func (a *AbsRank) Negate() Rank {
	return &MulRank{ranks: []Rank{Val(-1), a}}
}

func (a *AbsRank) Abs() Rank {
	return a // abs(abs(x)) = abs(x)
}

func (a *AbsRank) Exp() Rank {
	return &ExpRank{rank: a}
}

func (a *AbsRank) Log() Rank {
	return &LogRank{rank: a}
}

func (a *AbsRank) Max(operand Operand) Rank {
	return &MaxRank{ranks: []Rank{a, operandToRank(operand)}}
}

func (a *AbsRank) Min(operand Operand) Rank {
	return &MinRank{ranks: []Rank{a, operandToRank(operand)}}
}

func (a *AbsRank) WithWeight(weight float64) RankWithWeight {
	return RankWithWeight{Rank: a, Weight: weight}
}

func (a *AbsRank) MarshalJSON() ([]byte, error) {
	data, err := a.rank.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]json.RawMessage{"$abs": data})
}

func (a *AbsRank) UnmarshalJSON(b []byte) error {
	return nil // TODO: implement if needed
}

// ExpRank represents exponential: {"$exp": rank}
type ExpRank struct {
	rank Rank
}

func (e *ExpRank) IsOperand() {}

func (e *ExpRank) Multiply(operand Operand) Rank {
	return &MulRank{ranks: []Rank{e, operandToRank(operand)}}
}

func (e *ExpRank) Sub(operand Operand) Rank {
	return &SubRank{left: e, right: operandToRank(operand)}
}

func (e *ExpRank) Add(operand Operand) Rank {
	return &SumRank{ranks: []Rank{e, operandToRank(operand)}}
}

func (e *ExpRank) Div(operand Operand) Rank {
	return &DivRank{left: e, right: operandToRank(operand)}
}

func (e *ExpRank) Negate() Rank {
	return &MulRank{ranks: []Rank{Val(-1), e}}
}

func (e *ExpRank) Abs() Rank {
	return &AbsRank{rank: e}
}

func (e *ExpRank) Exp() Rank {
	return &ExpRank{rank: e}
}

func (e *ExpRank) Log() Rank {
	return &LogRank{rank: e}
}

func (e *ExpRank) Max(operand Operand) Rank {
	return &MaxRank{ranks: []Rank{e, operandToRank(operand)}}
}

func (e *ExpRank) Min(operand Operand) Rank {
	return &MinRank{ranks: []Rank{e, operandToRank(operand)}}
}

func (e *ExpRank) WithWeight(weight float64) RankWithWeight {
	return RankWithWeight{Rank: e, Weight: weight}
}

func (e *ExpRank) MarshalJSON() ([]byte, error) {
	data, err := e.rank.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]json.RawMessage{"$exp": data})
}

func (e *ExpRank) UnmarshalJSON(b []byte) error {
	return nil // TODO: implement if needed
}

// LogRank represents logarithm: {"$log": rank}
type LogRank struct {
	rank Rank
}

func (l *LogRank) IsOperand() {}

func (l *LogRank) Multiply(operand Operand) Rank {
	return &MulRank{ranks: []Rank{l, operandToRank(operand)}}
}

func (l *LogRank) Sub(operand Operand) Rank {
	return &SubRank{left: l, right: operandToRank(operand)}
}

func (l *LogRank) Add(operand Operand) Rank {
	return &SumRank{ranks: []Rank{l, operandToRank(operand)}}
}

func (l *LogRank) Div(operand Operand) Rank {
	return &DivRank{left: l, right: operandToRank(operand)}
}

func (l *LogRank) Negate() Rank {
	return &MulRank{ranks: []Rank{Val(-1), l}}
}

func (l *LogRank) Abs() Rank {
	return &AbsRank{rank: l}
}

func (l *LogRank) Exp() Rank {
	return &ExpRank{rank: l}
}

func (l *LogRank) Log() Rank {
	return l // log(log(x)) is valid but unusual
}

func (l *LogRank) Max(operand Operand) Rank {
	return &MaxRank{ranks: []Rank{l, operandToRank(operand)}}
}

func (l *LogRank) Min(operand Operand) Rank {
	return &MinRank{ranks: []Rank{l, operandToRank(operand)}}
}

func (l *LogRank) WithWeight(weight float64) RankWithWeight {
	return RankWithWeight{Rank: l, Weight: weight}
}

func (l *LogRank) MarshalJSON() ([]byte, error) {
	data, err := l.rank.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]json.RawMessage{"$log": data})
}

func (l *LogRank) UnmarshalJSON(b []byte) error {
	return nil // TODO: implement if needed
}

// MaxRank represents maximum: {"$max": [...]}
type MaxRank struct {
	ranks []Rank
}

func (m *MaxRank) IsOperand() {}

func (m *MaxRank) Multiply(operand Operand) Rank {
	return &MulRank{ranks: []Rank{m, operandToRank(operand)}}
}

func (m *MaxRank) Sub(operand Operand) Rank {
	return &SubRank{left: m, right: operandToRank(operand)}
}

func (m *MaxRank) Add(operand Operand) Rank {
	return &SumRank{ranks: []Rank{m, operandToRank(operand)}}
}

func (m *MaxRank) Div(operand Operand) Rank {
	return &DivRank{left: m, right: operandToRank(operand)}
}

func (m *MaxRank) Negate() Rank {
	return &MulRank{ranks: []Rank{Val(-1), m}}
}

func (m *MaxRank) Abs() Rank {
	return &AbsRank{rank: m}
}

func (m *MaxRank) Exp() Rank {
	return &ExpRank{rank: m}
}

func (m *MaxRank) Log() Rank {
	return &LogRank{rank: m}
}

func (m *MaxRank) Max(operand Operand) Rank {
	r := operandToRank(operand)
	if max, ok := r.(*MaxRank); ok {
		return &MaxRank{ranks: append(m.ranks, max.ranks...)}
	}
	return &MaxRank{ranks: append(m.ranks, r)}
}

func (m *MaxRank) Min(operand Operand) Rank {
	return &MinRank{ranks: []Rank{m, operandToRank(operand)}}
}

func (m *MaxRank) WithWeight(weight float64) RankWithWeight {
	return RankWithWeight{Rank: m, Weight: weight}
}

func (m *MaxRank) MarshalJSON() ([]byte, error) {
	rankMaps := make([]json.RawMessage, len(m.ranks))
	for i, r := range m.ranks {
		data, err := r.MarshalJSON()
		if err != nil {
			return nil, err
		}
		rankMaps[i] = data
	}
	return json.Marshal(map[string][]json.RawMessage{"$max": rankMaps})
}

func (m *MaxRank) UnmarshalJSON(b []byte) error {
	return nil // TODO: implement if needed
}

// MinRank represents minimum: {"$min": [...]}
type MinRank struct {
	ranks []Rank
}

func (m *MinRank) IsOperand() {}

func (m *MinRank) Multiply(operand Operand) Rank {
	return &MulRank{ranks: []Rank{m, operandToRank(operand)}}
}

func (m *MinRank) Sub(operand Operand) Rank {
	return &SubRank{left: m, right: operandToRank(operand)}
}

func (m *MinRank) Add(operand Operand) Rank {
	return &SumRank{ranks: []Rank{m, operandToRank(operand)}}
}

func (m *MinRank) Div(operand Operand) Rank {
	return &DivRank{left: m, right: operandToRank(operand)}
}

func (m *MinRank) Negate() Rank {
	return &MulRank{ranks: []Rank{Val(-1), m}}
}

func (m *MinRank) Abs() Rank {
	return &AbsRank{rank: m}
}

func (m *MinRank) Exp() Rank {
	return &ExpRank{rank: m}
}

func (m *MinRank) Log() Rank {
	return &LogRank{rank: m}
}

func (m *MinRank) Max(operand Operand) Rank {
	return &MaxRank{ranks: []Rank{m, operandToRank(operand)}}
}

func (m *MinRank) Min(operand Operand) Rank {
	r := operandToRank(operand)
	if min, ok := r.(*MinRank); ok {
		return &MinRank{ranks: append(m.ranks, min.ranks...)}
	}
	return &MinRank{ranks: append(m.ranks, r)}
}

func (m *MinRank) WithWeight(weight float64) RankWithWeight {
	return RankWithWeight{Rank: m, Weight: weight}
}

func (m *MinRank) MarshalJSON() ([]byte, error) {
	rankMaps := make([]json.RawMessage, len(m.ranks))
	for i, r := range m.ranks {
		data, err := r.MarshalJSON()
		if err != nil {
			return nil, err
		}
		rankMaps[i] = data
	}
	return json.Marshal(map[string][]json.RawMessage{"$min": rankMaps})
}

func (m *MinRank) UnmarshalJSON(b []byte) error {
	return nil // TODO: implement if needed
}

// --- KnnRank ---

type KnnOption func(req *KnnRank) error
type KnnQueryOption func(req *KnnRank) error

func KnnQueryVector(queryVector embeddings.KnnVector) KnnQueryOption {
	return func(req *KnnRank) error {
		req.Query = queryVector.ValuesAsFloat32()
		return nil
	}
}

func KnnQuerySparseVector(sparseVector *embeddings.SparseVector) KnnQueryOption {
	return func(req *KnnRank) error {
		req.Query = sparseVector
		return nil
	}
}

func KnnQueryText(text string) KnnQueryOption {
	return func(req *KnnRank) error {
		req.Query = text
		return nil
	}
}

func WithKnnLimit(limit int) KnnOption {
	return func(req *KnnRank) error {
		if limit < 1 {
			return errors.New("knn limit must be >= 1")
		}
		req.Limit = limit
		return nil
	}
}

func WithKnnKey(key ProjectionKey) KnnOption {
	return func(req *KnnRank) error {
		req.Key = key
		return nil
	}
}

func WithKnnDefault(defaultScore float64) KnnOption {
	return func(req *KnnRank) error {
		req.DefaultScore = &defaultScore
		return nil
	}
}

func WithKnnReturnRank() KnnOption {
	return func(req *KnnRank) error {
		req.ReturnRank = true
		return nil
	}
}

// KnnRank represents a KNN-based ranking expression
type KnnRank struct {
	Query        interface{}   // string (auto-embedded), []float32, []float64, or SparseVector
	Key          ProjectionKey // default "#embedding"
	Limit        int           // default 16
	DefaultScore *float64      // nil means exclude documents not in KNN results
	ReturnRank   bool          // return rank position instead of distance
}

func NewKnnRank(query KnnQueryOption, knnOptions ...KnnOption) *KnnRank {
	knn := &KnnRank{
		Key:   KEmbedding,
		Limit: 16,
	}
	if query != nil {
		_ = query(knn)
	}
	for _, opt := range knnOptions {
		_ = opt(knn)
	}
	return knn
}

func (k *KnnRank) IsOperand() {}

func (k *KnnRank) Multiply(operand Operand) Rank {
	return &MulRank{ranks: []Rank{k, operandToRank(operand)}}
}

func (k *KnnRank) Sub(operand Operand) Rank {
	return &SubRank{left: k, right: operandToRank(operand)}
}

func (k *KnnRank) Add(operand Operand) Rank {
	return &SumRank{ranks: []Rank{k, operandToRank(operand)}}
}

func (k *KnnRank) Div(operand Operand) Rank {
	return &DivRank{left: k, right: operandToRank(operand)}
}

func (k *KnnRank) Negate() Rank {
	return &MulRank{ranks: []Rank{Val(-1), k}}
}

func (k *KnnRank) Abs() Rank {
	return &AbsRank{rank: k}
}

func (k *KnnRank) Exp() Rank {
	return &ExpRank{rank: k}
}

func (k *KnnRank) Log() Rank {
	return &LogRank{rank: k}
}

func (k *KnnRank) Max(operand Operand) Rank {
	return &MaxRank{ranks: []Rank{k, operandToRank(operand)}}
}

func (k *KnnRank) Min(operand Operand) Rank {
	return &MinRank{ranks: []Rank{k, operandToRank(operand)}}
}

func (k *KnnRank) WithWeight(weight float64) RankWithWeight {
	return RankWithWeight{Rank: k, Weight: weight}
}

func (k *KnnRank) MarshalJSON() ([]byte, error) {
	inner := map[string]interface{}{
		"query": k.Query,
		"key":   string(k.Key),
		"limit": k.Limit,
	}
	if k.DefaultScore != nil {
		inner["default"] = *k.DefaultScore
	}
	if k.ReturnRank {
		inner["return_rank"] = true
	}
	return json.Marshal(map[string]interface{}{"$knn": inner})
}

func (k *KnnRank) UnmarshalJSON(b []byte) error {
	var data map[string]map[string]interface{}
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}
	if knnData, ok := data["$knn"]; ok {
		if query, ok := knnData["query"]; ok {
			k.Query = query
		}
		if key, ok := knnData["key"].(string); ok {
			k.Key = ProjectionKey(key)
		}
		if limit, ok := knnData["limit"].(float64); ok {
			k.Limit = int(limit)
		}
		if def, ok := knnData["default"].(float64); ok {
			k.DefaultScore = &def
		}
		if returnRank, ok := knnData["return_rank"].(bool); ok {
			k.ReturnRank = returnRank
		}
	}
	return nil
}

// --- RRF (Reciprocal Rank Fusion) ---

type RffOption func(req *RrfRank) error

func WithRffRanks(ranks ...RankWithWeight) RffOption {
	return func(req *RrfRank) error {
		req.Ranks = append(req.Ranks, ranks...)
		return nil
	}
}

func WithRffK(k int) RffOption {
	return func(req *RrfRank) error {
		if k <= 0 {
			return errors.New("rrf k must be > 0")
		}
		req.K = k
		return nil
	}
}

func WithRffNormalize() RffOption {
	return func(req *RrfRank) error {
		req.Normalize = true
		return nil
	}
}

// RrfRank implements Reciprocal Rank Fusion
// Formula: -sum(weight_i / (k + rank_i))
type RrfRank struct {
	Ranks     []RankWithWeight
	K         int  // smoothing constant, default 60
	Normalize bool // normalize weights to sum to 1.0
}

func NewRrfRank(opts ...RffOption) (*RrfRank, error) {
	rrf := &RrfRank{
		K: 60,
	}
	for _, opt := range opts {
		if err := opt(rrf); err != nil {
			return nil, err
		}
	}
	return rrf, nil
}

func (r *RrfRank) IsOperand() {}

func (r *RrfRank) Multiply(operand Operand) Rank {
	return &MulRank{ranks: []Rank{r, operandToRank(operand)}}
}

func (r *RrfRank) Sub(operand Operand) Rank {
	return &SubRank{left: r, right: operandToRank(operand)}
}

func (r *RrfRank) Add(operand Operand) Rank {
	return &SumRank{ranks: []Rank{r, operandToRank(operand)}}
}

func (r *RrfRank) Div(operand Operand) Rank {
	return &DivRank{left: r, right: operandToRank(operand)}
}

func (r *RrfRank) Negate() Rank {
	return &MulRank{ranks: []Rank{Val(-1), r}}
}

func (r *RrfRank) Abs() Rank {
	return &AbsRank{rank: r}
}

func (r *RrfRank) Exp() Rank {
	return &ExpRank{rank: r}
}

func (r *RrfRank) Log() Rank {
	return &LogRank{rank: r}
}

func (r *RrfRank) Max(operand Operand) Rank {
	return &MaxRank{ranks: []Rank{r, operandToRank(operand)}}
}

func (r *RrfRank) Min(operand Operand) Rank {
	return &MinRank{ranks: []Rank{r, operandToRank(operand)}}
}

func (r *RrfRank) WithWeight(weight float64) RankWithWeight {
	return RankWithWeight{Rank: r, Weight: weight}
}

// MarshalJSON builds: -sum(weight_i / (k + rank_i))
func (r *RrfRank) MarshalJSON() ([]byte, error) {
	if len(r.Ranks) == 0 {
		return nil, errors.New("rrf requires at least one rank")
	}

	// Compute weights
	weights := make([]float64, len(r.Ranks))
	for i, rw := range r.Ranks {
		if rw.Weight == 0 {
			weights[i] = 1.0
		} else {
			weights[i] = rw.Weight
		}
	}

	// Normalize if requested
	if r.Normalize {
		sum := 0.0
		for _, w := range weights {
			sum += w
		}
		if sum == 0 {
			return nil, errors.New("sum of weights must be positive when normalize=true")
		}
		for i := range weights {
			weights[i] /= sum
		}
	}

	// Build terms: weight / (k + rank)
	terms := make([]Rank, len(r.Ranks))
	for i, rw := range r.Ranks {
		// term = weight / (k + rank)
		kVal := Val(float64(r.K))
		denominator := kVal.Add(rw.Rank)
		terms[i] = Val(weights[i]).Div(denominator)
	}

	// Sum all terms
	rrfSum := terms[0]
	for _, term := range terms[1:] {
		rrfSum = rrfSum.Add(term)
	}

	// Negate (RRF gives higher scores for better, Chroma needs lower for better)
	result := rrfSum.Negate()
	return result.MarshalJSON()
}

func (r *RrfRank) UnmarshalJSON(b []byte) error {
	return nil // RRF is constructed programmatically, not from JSON
}

// --- Helper Functions ---

// operandToRank converts an Operand to a Rank
func operandToRank(operand Operand) Rank {
	switch v := operand.(type) {
	case Rank:
		return v
	case IntOperand:
		return Val(float64(v))
	case FloatOperand:
		return Val(float64(v))
	default:
		return Val(0)
	}
}

// --- Search Option Functions ---

func WithKnnRank(query KnnQueryOption, knnOptions ...KnnOption) SearchOption {
	return func(req *SearchRequest) error {
		knn := NewKnnRank(query, knnOptions...)
		req.Rank = knn
		return nil
	}
}

func WithRffRank(opts ...RffOption) SearchOption {
	return func(req *SearchRequest) error {
		rrf, err := NewRrfRank(opts...)
		if err != nil {
			return err
		}
		req.Rank = rrf
		return nil
	}
}
