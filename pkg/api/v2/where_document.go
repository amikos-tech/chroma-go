package v2

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

type WhereDocumentFilterOperator string

const (
	ContainsOperator    WhereDocumentFilterOperator = "$contains"
	NotContainsOperator WhereDocumentFilterOperator = "$not_contains"
	RegexOperator       WhereDocumentFilterOperator = "$regex"
	NotRegexOperator    WhereDocumentFilterOperator = "$not_regex"
	OrDocumentOperator  WhereDocumentFilterOperator = "$or"
	AndDocumentOperator WhereDocumentFilterOperator = "$and"
)

type WhereDocumentFilter interface {
	Operator() WhereDocumentFilterOperator
	Operand() interface{}
	Validate() error
	String() string
	MarshalJSON() ([]byte, error)
	UnmarshalJSON(b []byte) error
}

type WhereDocumentFilterBase struct {
	operator WhereDocumentFilterOperator
	content  interface{}
}

func (w *WhereDocumentFilterBase) Operator() WhereDocumentFilterOperator {
	return w.operator
}

func (w *WhereDocumentFilterBase) Operand() interface{} {
	return w.content
}

func (w *WhereDocumentFilterBase) Validate() error {
	return nil
}

func (w *WhereDocumentFilterBase) String() string {
	return ""
}

func (w *WhereDocumentFilterBase) MarshalJSON() ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (w *WhereDocumentFilterBase) UnmarshalJSON(b []byte) error {
	return errors.New("not implemented")
}

type WhereDocumentClauseContainsOrNotContains struct {
	WhereDocumentFilterBase
	content string
}

func (w *WhereDocumentClauseContainsOrNotContains) MarshalJSON() ([]byte, error) {
	err := w.Validate()
	if err != nil {
		return nil, err
	}
	var x = map[WhereDocumentFilterOperator]string{
		w.operator: w.content,
	}
	return json.Marshal(x)
}

func (w *WhereDocumentClauseContainsOrNotContains) UnmarshalJSON(b []byte) error {
	var x map[WhereDocumentFilterOperator]string
	if err := json.Unmarshal(b, &x); err != nil {
		return err
	}
	for operator, content := range x {
		w.operator = operator
		w.content = content
	}
	return nil
}

func (w *WhereDocumentClauseContainsOrNotContains) Validate() error {
	if w.operator != ContainsOperator && w.operator != NotContainsOperator {
		return errors.New("invalid operator, expected in contains or not contains")
	}
	return nil
}

func (w *WhereDocumentClauseContainsOrNotContains) String() string {
	return fmt.Sprintf("%s: %s", w.operator, w.content)
}

type WhereDocumentClauseRegexNotRegex struct {
	WhereDocumentFilterBase
	content string
}

func (w *WhereDocumentClauseRegexNotRegex) MarshalJSON() ([]byte, error) {
	err := w.Validate()
	if err != nil {
		return nil, err
	}
	var x = map[WhereDocumentFilterOperator]string{
		w.operator: w.content,
	}
	return json.Marshal(x)
}

func (w *WhereDocumentClauseRegexNotRegex) UnmarshalJSON(b []byte) error {
	var x map[WhereDocumentFilterOperator]string
	if err := json.Unmarshal(b, &x); err != nil {
		return err
	}
	for operator, content := range x {
		w.operator = operator
		w.content = content
	}
	return nil
}

func (w *WhereDocumentClauseRegexNotRegex) Validate() error {
	if w.operator != RegexOperator && w.operator != NotRegexOperator {
		return errors.New("invalid operator, expected in regex or not regex")
	}
	return nil
}

func (w *WhereDocumentClauseRegexNotRegex) String() string {
	return fmt.Sprintf("%s: %s", w.operator, w.content)
}

type WhereDocumentClauseOr struct {
	WhereDocumentFilterBase
	content []WhereDocumentFilter
}

func (w *WhereDocumentClauseOr) MarshalJSON() ([]byte, error) {
	return w.marshalJSONWithDepth(0)
}

func (w *WhereDocumentClauseOr) marshalJSONWithDepth(depth int) ([]byte, error) {
	if w.operator != OrDocumentOperator {
		return nil, errors.New("invalid operator, expected in or")
	}
	if len(w.content) == 0 {
		return nil, errors.New("invalid content, expected at least one")
	}
	rawContent, err := marshalWhereDocumentFilterChildren(w.content, w.operator, depth)
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[WhereDocumentFilterOperator][]json.RawMessage{w.operator: rawContent})
}

func (w *WhereDocumentClauseOr) UnmarshalJSON(b []byte) error {
	var x map[WhereDocumentFilterOperator]json.RawMessage
	if err := json.Unmarshal(b, &x); err != nil {
		return err
	}
	for operator, raw := range x {
		clauses, err := unmarshalWhereDocumentFilters(raw)
		if err != nil {
			return err
		}
		w.operator = operator
		w.content = clauses
	}
	return nil
}

func (w *WhereDocumentClauseOr) Validate() error {
	return w.validateWithDepth(0)
}

func (w *WhereDocumentClauseOr) validateWithDepth(depth int) error {
	if w.operator != OrDocumentOperator {
		return errors.New("invalid operator, expected in or")
	}
	if len(w.content) == 0 {
		return errors.New("invalid content, expected at least one")
	}
	for _, v := range w.content {
		if err := validateWhereDocumentFilterChild(v, w.operator, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func (w *WhereDocumentClauseOr) String() string {
	return fmt.Sprintf("%s: %s", w.operator, w.content)
}

type WhereDocumentClauseAnd struct {
	WhereDocumentFilterBase
	content []WhereDocumentFilter
}

func (w *WhereDocumentClauseAnd) MarshalJSON() ([]byte, error) {
	return w.marshalJSONWithDepth(0)
}

func (w *WhereDocumentClauseAnd) marshalJSONWithDepth(depth int) ([]byte, error) {
	if w.operator != AndDocumentOperator {
		return nil, errors.New("invalid operator, expected in and")
	}
	if len(w.content) == 0 {
		return nil, errors.New("invalid content, expected at least one")
	}
	rawContent, err := marshalWhereDocumentFilterChildren(w.content, w.operator, depth)
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[WhereDocumentFilterOperator][]json.RawMessage{w.operator: rawContent})
}

func (w *WhereDocumentClauseAnd) UnmarshalJSON(b []byte) error {
	return errors.New("not implemented")
}

func (w *WhereDocumentClauseAnd) Validate() error {
	return w.validateWithDepth(0)
}

func (w *WhereDocumentClauseAnd) validateWithDepth(depth int) error {
	if w.operator != AndDocumentOperator {
		return errors.New("invalid operator, expected in and")
	}
	if len(w.content) == 0 {
		return errors.New("invalid content, expected at least one")
	}
	for _, v := range w.content {
		if err := validateWhereDocumentFilterChild(v, w.operator, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func validateWhereDocumentFilterChild(filter WhereDocumentFilter, operator WhereDocumentFilterOperator, depth int) error {
	if isNilInterface(filter) {
		return errors.Errorf("nil clause in %s expression", operator)
	}
	if depth > MaxExpressionDepth {
		return errors.Errorf("where document expression exceeds maximum depth of %d", MaxExpressionDepth)
	}
	switch filter := filter.(type) {
	case *WhereDocumentClauseOr:
		return filter.validateWithDepth(depth)
	case *WhereDocumentClauseAnd:
		return filter.validateWithDepth(depth)
	default:
		return filter.Validate()
	}
}

// marshalWhereDocumentFilterChildren validates and encodes each child at the
// depth it was reached, instead of delegating to json.Marshal and letting
// each nested compound child's own MarshalJSON restart validation from depth
// 0 (which would revalidate every remaining subtree once per level).
func marshalWhereDocumentFilterChildren(content []WhereDocumentFilter, operator WhereDocumentFilterOperator, depth int) ([]json.RawMessage, error) {
	rawContent := make([]json.RawMessage, len(content))
	for i, v := range content {
		data, err := marshalWhereDocumentFilterChild(v, operator, depth+1)
		if err != nil {
			return nil, err
		}
		rawContent[i] = data
	}
	return rawContent, nil
}

func marshalWhereDocumentFilterChild(filter WhereDocumentFilter, operator WhereDocumentFilterOperator, depth int) ([]byte, error) {
	if isNilInterface(filter) {
		return nil, errors.Errorf("nil clause in %s expression", operator)
	}
	if depth > MaxExpressionDepth {
		return nil, errors.Errorf("where document expression exceeds maximum depth of %d", MaxExpressionDepth)
	}
	switch filter := filter.(type) {
	case *WhereDocumentClauseOr:
		return filter.marshalJSONWithDepth(depth)
	case *WhereDocumentClauseAnd:
		return filter.marshalJSONWithDepth(depth)
	default:
		if err := filter.Validate(); err != nil {
			return nil, err
		}
		return filter.MarshalJSON()
	}
}

func (w *WhereDocumentClauseAnd) String() string {
	return fmt.Sprintf("%s: %s", w.operator, w.content)
}

// unmarshalWhereDocumentFilters decodes a JSON array of document filters into
// concrete values. A plain []WhereDocumentFilter cannot be decoded by
// encoding/json because the element type is a non-empty interface.
func unmarshalWhereDocumentFilters(data []byte) ([]WhereDocumentFilter, error) {
	var rawFilters []json.RawMessage
	if err := json.Unmarshal(data, &rawFilters); err != nil {
		return nil, err
	}
	filters := make([]WhereDocumentFilter, len(rawFilters))
	for i, raw := range rawFilters {
		filter, err := unmarshalWhereDocumentFilter(raw)
		if err != nil {
			return nil, err
		}
		filters[i] = filter
	}
	return filters, nil
}

func unmarshalWhereDocumentFilter(data []byte) (WhereDocumentFilter, error) {
	var operators map[WhereDocumentFilterOperator]json.RawMessage
	if err := json.Unmarshal(data, &operators); err != nil {
		return nil, err
	}
	if len(operators) == 0 {
		return nil, errors.New("where document filter has no operator")
	}
	for operator, raw := range operators {
		switch operator {
		case ContainsOperator, NotContainsOperator:
			var content string
			if err := json.Unmarshal(raw, &content); err != nil {
				return nil, err
			}
			return &WhereDocumentClauseContainsOrNotContains{
				WhereDocumentFilterBase: WhereDocumentFilterBase{operator: operator},
				content:                 content,
			}, nil
		case RegexOperator, NotRegexOperator:
			var content string
			if err := json.Unmarshal(raw, &content); err != nil {
				return nil, err
			}
			return &WhereDocumentClauseRegexNotRegex{
				WhereDocumentFilterBase: WhereDocumentFilterBase{operator: operator},
				content:                 content,
			}, nil
		case OrDocumentOperator, AndDocumentOperator:
			clauses, err := unmarshalWhereDocumentFilters(raw)
			if err != nil {
				return nil, err
			}
			base := WhereDocumentFilterBase{operator: operator}
			if operator == OrDocumentOperator {
				return &WhereDocumentClauseOr{WhereDocumentFilterBase: base, content: clauses}, nil
			}
			return &WhereDocumentClauseAnd{WhereDocumentFilterBase: base, content: clauses}, nil
		default:
			return nil, errors.Errorf("unknown where document operator %q", operator)
		}
	}
	return nil, errors.New("where document filter has no operator")
}

func Contains(content string) WhereDocumentFilter {
	return &WhereDocumentClauseContainsOrNotContains{
		WhereDocumentFilterBase: WhereDocumentFilterBase{
			operator: ContainsOperator,
		},
		content: content,
	}
}

func NotContains(content string) WhereDocumentFilter {
	return &WhereDocumentClauseContainsOrNotContains{
		WhereDocumentFilterBase: WhereDocumentFilterBase{
			operator: NotContainsOperator,
		},
		content: content,
	}
}

func Regex(content string) WhereDocumentFilter {
	return &WhereDocumentClauseRegexNotRegex{
		WhereDocumentFilterBase: WhereDocumentFilterBase{
			operator: RegexOperator,
		},
		content: content,
	}
}

func NotRegex(content string) WhereDocumentFilter {
	return &WhereDocumentClauseRegexNotRegex{
		WhereDocumentFilterBase: WhereDocumentFilterBase{
			operator: NotRegexOperator,
		},
		content: content,
	}
}

func OrDocument(clauses ...WhereDocumentFilter) WhereDocumentFilter {
	return &WhereDocumentClauseOr{
		WhereDocumentFilterBase: WhereDocumentFilterBase{
			operator: OrDocumentOperator,
		},
		content: clauses,
	}
}

func AndDocument(clauses ...WhereDocumentFilter) WhereDocumentFilter {
	return &WhereDocumentClauseAnd{
		WhereDocumentFilterBase: WhereDocumentFilterBase{
			operator: AndDocumentOperator,
		},
		content: clauses,
	}
}
