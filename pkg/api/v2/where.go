package v2

import (
	"encoding/json"

	"github.com/pkg/errors"
)

type WhereFilterOperator string

const (
	// Original long-form operators (kept for backward compatibility)
	EqualOperator              WhereFilterOperator = "$eq"
	NotEqualOperator           WhereFilterOperator = "$ne"
	GreaterThanOperator        WhereFilterOperator = "$gt"
	GreaterThanOrEqualOperator WhereFilterOperator = "$gte"
	LessThanOperator           WhereFilterOperator = "$lt"
	LessThanOrEqualOperator    WhereFilterOperator = "$lte"
	InOperator                 WhereFilterOperator = "$in"
	NotInOperator              WhereFilterOperator = "$nin"
	AndOperator                WhereFilterOperator = "$and"
	OrOperator                 WhereFilterOperator = "$or"

	// Simplified short-form operators (recommended for new code)
	EQ  WhereFilterOperator = "$eq"
	NE  WhereFilterOperator = "$ne"
	GT  WhereFilterOperator = "$gt"
	GTE WhereFilterOperator = "$gte"
	LT  WhereFilterOperator = "$lt"
	LTE WhereFilterOperator = "$lte"
	IN  WhereFilterOperator = "$in"
	NIN WhereFilterOperator = "$nin"
	AND WhereFilterOperator = "$and"
	OR  WhereFilterOperator = "$or"
)

type WhereClause interface {
	Operator() WhereFilterOperator
	Key() string
	Operand() interface{}
	String() string
	Validate() error
	MarshalJSON() ([]byte, error)
	UnmarshalJSON(b []byte) error
}

type WhereClauseBase struct {
	operator WhereFilterOperator
	key      string
}

func (w *WhereClauseBase) Operator() WhereFilterOperator {
	return w.operator
}

func (w *WhereClauseBase) Key() string {
	return w.key
}

func (w *WhereClauseBase) String() string {
	return ""
}

// StringValue

type WhereClauseString struct {
	WhereClauseBase
	operand string
}

func (w *WhereClauseString) Operand() interface{} {
	return w.operand
}

func (w *WhereClauseString) Validate() error {
	if w.key == "" {
		return errors.Errorf("invalid key for %s, expected non-empty", w.operator)
	}
	return nil
}

func (w *WhereClauseString) MarshalJSON() ([]byte, error) {
	var x = map[string]map[WhereFilterOperator]string{
		w.key: {
			w.operator: w.operand,
		},
	}
	return json.Marshal(x)
}

func (w *WhereClauseString) UnmarshalJSON(b []byte) error {
	var x = map[string]map[WhereFilterOperator]string{}
	err := json.Unmarshal(b, &x)
	if err != nil {
		return err
	}

	for key, value := range x {
		w.key = key
		for operator, operand := range value {
			w.operator = operator
			w.operand = operand
		}
	}
	return nil
}

type WhereClauseStrings struct {
	WhereClauseBase
	operand []string
}

func (w *WhereClauseStrings) Operand() interface{} {
	return w.operand
}

func (w *WhereClauseStrings) Validate() error {
	if w.key == "" {
		return errors.Errorf("invalid key for %s, expected non-empty", w.operator)
	}
	if w.operator != InOperator && w.operator != NotInOperator {
		return errors.New("invalid operator, expected in or nin")
	}
	return nil
}

func (w *WhereClauseStrings) MarshalJSON() ([]byte, error) {
	var x = map[string]map[WhereFilterOperator][]string{
		w.key: {
			w.operator: w.operand,
		},
	}
	return json.Marshal(x)
}

func (w *WhereClauseStrings) UnmarshalJSON(b []byte) error {
	var x = map[string]map[WhereFilterOperator][]string{}
	err := json.Unmarshal(b, &x)
	if err != nil {
		return err
	}

	for key, value := range x {
		w.key = key
		for operator, operand := range value {
			w.operator = operator
			w.operand = operand
		}
	}
	return nil
}

// Int

type WhereClauseInt struct {
	WhereClauseBase
	operand int
}

func (w *WhereClauseInt) Operand() interface{} {
	return w.operand
}

func (w *WhereClauseInt) Validate() error {
	if w.key == "" {
		return errors.Errorf("invalid key for %s, expected non-empty", w.operator)
	}
	return nil
}

func (w *WhereClauseInt) MarshalJSON() ([]byte, error) {
	var x = map[string]map[WhereFilterOperator]int{
		w.key: {
			w.operator: w.operand,
		},
	}
	return json.Marshal(x)
}

func (w *WhereClauseInt) UnmarshalJSON(b []byte) error {
	var x = map[string]map[WhereFilterOperator]int{}
	err := json.Unmarshal(b, &x)
	if err != nil {
		return err
	}

	for key, value := range x {
		w.key = key
		for operator, operand := range value {
			w.operator = operator
			w.operand = operand
		}
	}
	return nil
}

type WhereClauseInts struct {
	WhereClauseBase
	operand []int
}

func (w *WhereClauseInts) Operand() interface{} {
	return w.operand
}

func (w *WhereClauseInts) Validate() error {
	if w.key == "" {
		return errors.Errorf("invalid key for %s, expected non-empty", w.operator)
	}
	if w.operator != InOperator && w.operator != NotInOperator {
		return errors.New("invalid operator, expected in or nin")
	}
	return nil
}

func (w *WhereClauseInts) MarshalJSON() ([]byte, error) {
	var x = map[string]map[WhereFilterOperator][]int{
		w.key: {
			w.operator: w.operand,
		},
	}
	return json.Marshal(x)
}

func (w *WhereClauseInts) UnmarshalJSON(b []byte) error {
	var x = map[string]map[WhereFilterOperator][]int{}
	err := json.Unmarshal(b, &x)
	if err != nil {
		return err
	}

	for key, value := range x {
		w.key = key
		for operator, operand := range value {
			w.operator = operator
			w.operand = operand
		}
	}
	return nil
}

// Float

type WhereClauseFloat struct {
	WhereClauseBase
	operand float32
}

func (w *WhereClauseFloat) Operand() interface{} {
	return w.operand
}

func (w *WhereClauseFloat) Validate() error {
	if w.key == "" {
		return errors.Errorf("invalid key for %s, expected non-empty", w.operator)
	}
	return nil
}

func (w *WhereClauseFloat) MarshalJSON() ([]byte, error) {
	var x = map[string]map[WhereFilterOperator]float32{
		w.key: {
			w.operator: w.operand,
		},
	}
	return json.Marshal(x)
}

func (w *WhereClauseFloat) UnmarshalJSON(b []byte) error {
	var x = map[string]map[WhereFilterOperator]float32{}
	err := json.Unmarshal(b, &x)
	if err != nil {
		return err
	}

	for key, value := range x {
		w.key = key
		for operator, operand := range value {
			w.operator = operator
			w.operand = operand
		}
	}
	return nil
}

type WhereClauseFloats struct {
	WhereClauseBase
	operand []float32
}

func (w *WhereClauseFloats) Operand() interface{} {
	return w.operand
}

func (w *WhereClauseFloats) Validate() error {
	if w.key == "" {
		return errors.Errorf("invalid key for %s, expected non-empty", w.operator)
	}
	if w.operator != InOperator && w.operator != NotInOperator {
		return errors.New("invalid operator, expected in or nin")
	}
	return nil
}

func (w *WhereClauseFloats) MarshalJSON() ([]byte, error) {
	var x = map[string]map[WhereFilterOperator][]float32{
		w.key: {
			w.operator: w.operand,
		},
	}
	return json.Marshal(x)
}

func (w *WhereClauseFloats) UnmarshalJSON(b []byte) error {
	var x = map[string]map[WhereFilterOperator][]float32{}
	err := json.Unmarshal(b, &x)
	if err != nil {
		return err
	}

	for key, value := range x {
		w.key = key
		for operator, operand := range value {
			w.operator = operator
			w.operand = operand
		}
	}
	return nil
}

// Bool

type WhereClauseBool struct {
	WhereClauseBase
	operand bool
}

func (w *WhereClauseBool) Operand() interface{} {
	return w.operand
}

func (w *WhereClauseBool) Validate() error {
	if w.key == "" {
		return errors.Errorf("invalid key for %s, expected non-empty", w.operator)
	}
	return nil
}

func (w *WhereClauseBool) MarshalJSON() ([]byte, error) {
	var x = map[string]map[WhereFilterOperator]bool{
		w.key: {
			w.operator: w.operand,
		},
	}
	return json.Marshal(x)
}

func (w *WhereClauseBool) UnmarshalJSON(b []byte) error {
	var x = map[string]map[WhereFilterOperator]bool{}
	err := json.Unmarshal(b, &x)
	if err != nil {
		return err
	}

	for key, value := range x {
		w.key = key
		for operator, operand := range value {
			w.operator = operator
			w.operand = operand
		}
	}
	return nil
}

type WhereClauseBools struct {
	WhereClauseBase
	operand []bool
}

func (w *WhereClauseBools) Operand() interface{} {
	return w.operand
}

func (w *WhereClauseBools) Validate() error {
	if w.key == "" {
		return errors.Errorf("invalid key for %s, expected non-empty", w.operator)
	}
	if w.operator != InOperator && w.operator != NotInOperator {
		return errors.New("invalid operator, expected in or nin")
	}
	return nil
}

func (w *WhereClauseBools) MarshalJSON() ([]byte, error) {
	var x = map[string]map[WhereFilterOperator][]bool{
		w.key: {
			w.operator: w.operand,
		},
	}
	return json.Marshal(x)
}

func (w *WhereClauseBools) UnmarshalJSON(b []byte) error {
	var x = map[string]map[WhereFilterOperator][]bool{}
	err := json.Unmarshal(b, &x)
	if err != nil {
		return err
	}

	for key, value := range x {
		w.key = key
		for operator, operand := range value {
			w.operator = operator
			w.operand = operand
		}
	}
	return nil
}

type WhereClauseWhereClauses struct {
	WhereClauseBase
	operand []WhereClause
}

func (w *WhereClauseWhereClauses) Operand() interface{} {
	return w.operand
}

func (w *WhereClauseWhereClauses) Validate() error {
	if w.operator != OrOperator && w.operator != AndOperator {
		return errors.New("invalid operator, expected in or nin")
	}
	return nil
}

func (w *WhereClauseWhereClauses) MarshalJSON() ([]byte, error) {
	var x = map[WhereFilterOperator][]WhereClause{
		w.operator: w.operand,
	}
	return json.Marshal(x)
}

func (w *WhereClauseWhereClauses) UnmarshalJSON(b []byte) error {
	var x = map[WhereFilterOperator][]WhereClause{}
	err := json.Unmarshal(b, &x)
	if err != nil {
		return err
	}

	for operator, clauses := range x {
		w.operator = operator
		w.operand = clauses
	}
	return nil
}

type WhereFilter interface {
	String() string
	Validate() error
	MarshalJSON() ([]byte, error)
	UnmarshalJSON(b []byte) error
}

// Deprecated: Use Eq(field, value) instead for simpler API
func EqString(field, value string) WhereClause {
	return &WhereClauseString{
		WhereClauseBase: WhereClauseBase{
			operator: EqualOperator,
			key:      field,
		},
		operand: value,
	}
}

// Deprecated: Use Eq(field, value) instead for simpler API
func EqInt(field string, value int) WhereClause {
	return &WhereClauseInt{
		WhereClauseBase: WhereClauseBase{
			operator: EqualOperator,
			key:      field,
		},
		operand: value,
	}
}

// Deprecated: Use Eq(field, value) instead for simpler API
func EqFloat(field string, value float32) WhereClause {
	return &WhereClauseFloat{
		WhereClauseBase: WhereClauseBase{
			operator: EqualOperator,
			key:      field,
		},
		operand: value,
	}
}

// Deprecated: Use Eq(field, value) instead for simpler API
func EqBool(field string, value bool) WhereClause {
	return &WhereClauseBool{
		WhereClauseBase: WhereClauseBase{
			operator: EqualOperator,
			key:      field,
		},
		operand: value,
	}
}

func NotEqString(field, value string) WhereClause {
	return &WhereClauseString{
		WhereClauseBase: WhereClauseBase{
			operator: NotEqualOperator,
			key:      field,
		},
		operand: value,
	}
}
func NotEqInt(field string, value int) WhereClause {
	return &WhereClauseInt{
		WhereClauseBase: WhereClauseBase{
			operator: NotEqualOperator,
			key:      field,
		},
		operand: value,
	}
}
func NotEqFloat(field string, value float32) WhereClause {
	return &WhereClauseFloat{
		WhereClauseBase: WhereClauseBase{
			operator: NotEqualOperator,
			key:      field,
		},
		operand: value,
	}
}
func NotEqBool(field string, value bool) WhereClause {
	return &WhereClauseBool{
		WhereClauseBase: WhereClauseBase{
			operator: NotEqualOperator,
			key:      field,
		},
		operand: value,
	}
}

// Deprecated: Use Gt(field, value) instead for simpler API
func GtInt(field string, value int) WhereClause {
	return &WhereClauseInt{
		WhereClauseBase: WhereClauseBase{
			operator: GreaterThanOperator,
			key:      field,
		},
		operand: value,
	}
}

// Deprecated: Use Gt(field, value) instead for simpler API
func GtFloat(field string, value float32) WhereClause {
	return &WhereClauseFloat{
		WhereClauseBase: WhereClauseBase{
			operator: GreaterThanOperator,
			key:      field,
		},
		operand: value,
	}
}

// Deprecated: Use Lt(field, value) instead for simpler API
func LtInt(field string, value int) WhereClause {
	return &WhereClauseInt{
		WhereClauseBase: WhereClauseBase{
			operator: LessThanOperator,
			key:      field,
		},
		operand: value,
	}
}

// Deprecated: Use Lt(field, value) instead for simpler API
func LtFloat(field string, value float32) WhereClause {
	return &WhereClauseFloat{
		WhereClauseBase: WhereClauseBase{
			operator: LessThanOperator,
			key:      field,
		},
		operand: value,
	}
}

// Deprecated: Use Gte(field, value) instead for simpler API
func GteInt(field string, value int) WhereClause {
	return &WhereClauseInt{
		WhereClauseBase: WhereClauseBase{
			operator: GreaterThanOrEqualOperator,
			key:      field,
		},
		operand: value,
	}
}

// Deprecated: Use Gte(field, value) instead for simpler API
func GteFloat(field string, value float32) WhereClause {
	return &WhereClauseFloat{
		WhereClauseBase: WhereClauseBase{
			operator: GreaterThanOrEqualOperator,
			key:      field,
		},
		operand: value,
	}
}

// Deprecated: Use Lte(field, value) instead for simpler API
func LteInt(field string, value int) WhereClause {
	return &WhereClauseInt{
		WhereClauseBase: WhereClauseBase{
			operator: LessThanOrEqualOperator,
			key:      field,
		},
		operand: value,
	}
}

// Deprecated: Use Lte(field, value) instead for simpler API
func LteFloat(field string, value float32) WhereClause {
	return &WhereClauseFloat{
		WhereClauseBase: WhereClauseBase{
			operator: LessThanOrEqualOperator,
			key:      field,
		},
		operand: value,
	}
}

// Deprecated: Use In(field, values) instead for simpler API
func InString(field string, values ...string) WhereClause {
	return &WhereClauseStrings{
		WhereClauseBase: WhereClauseBase{
			operator: InOperator,
			key:      field,
		},
		operand: values,
	}
}

// Deprecated: Use In(field, values) instead for simpler API
func InInt(field string, values ...int) WhereClause {
	return &WhereClauseInts{
		WhereClauseBase: WhereClauseBase{
			operator: InOperator,
			key:      field,
		},
		operand: values,
	}
}

// Deprecated: Use In(field, values) instead for simpler API
func InFloat(field string, values ...float32) WhereClause {
	return &WhereClauseFloats{
		WhereClauseBase: WhereClauseBase{
			operator: InOperator,
			key:      field,
		},
		operand: values,
	}
}

// Deprecated: Use In(field, values) instead for simpler API
func InBool(field string, values ...bool) WhereClause {
	return &WhereClauseBools{
		WhereClauseBase: WhereClauseBase{
			operator: InOperator,
			key:      field,
		},
		operand: values,
	}
}

// Deprecated: Use Nin(field, values) instead for simpler API
func NinString(field string, values ...string) WhereClause {
	return &WhereClauseStrings{
		WhereClauseBase: WhereClauseBase{
			operator: NotInOperator,
			key:      field,
		},
		operand: values,
	}
}

// Deprecated: Use Nin(field, values) instead for simpler API
func NinInt(field string, values ...int) WhereClause {
	return &WhereClauseInts{
		WhereClauseBase: WhereClauseBase{
			operator: NotInOperator,
			key:      field,
		},
		operand: values,
	}
}

// Deprecated: Use Nin(field, values) instead for simpler API
func NinFloat(field string, values ...float32) WhereClause {
	return &WhereClauseFloats{
		WhereClauseBase: WhereClauseBase{
			operator: NotInOperator,
			key:      field,
		},
		operand: values,
	}
}

// Deprecated: Use Nin(field, values) instead for simpler API
func NinBool(field string, values ...bool) WhereClause {
	return &WhereClauseBools{
		WhereClauseBase: WhereClauseBase{
			operator: NotInOperator,
			key:      field,
		},
		operand: values,
	}
}
func Or(clauses ...WhereClause) WhereClause {
	return &WhereClauseWhereClauses{
		WhereClauseBase: WhereClauseBase{
			operator: OrOperator,
		},
		operand: clauses,
	}
}
func And(clauses ...WhereClause) WhereClause {
	return &WhereClauseWhereClauses{
		WhereClauseBase: WhereClauseBase{
			operator: AndOperator,
		},
		operand: clauses,
	}
}
