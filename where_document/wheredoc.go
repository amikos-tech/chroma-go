package wheredoc

import (
	"fmt"
)

type InvalidWhereDocumentValueError struct {
	Value interface{}
}

func (e *InvalidWhereDocumentValueError) Error() string {
	return fmt.Sprintf("Invalid value for where document clause for value %v. Allowed values are string", e.Value)
}

type Builder struct {
	WhereClause map[string]interface{}
	err         error
}

func NewWhereDocumentBuilder() *Builder {
	return &Builder{WhereClause: make(map[string]interface{})}
}

func (w *Builder) operation(operation string, value interface{}) *Builder {
	if w.err != nil {
		return w
	}
	inner := make(map[string]interface{})

	switch value.(type) {
	case string:
	default:
		w.err = &InvalidWhereDocumentValueError{Value: value}
		return w
	}
	inner[operation] = value
	w.WhereClause[operation] = value
	return w
}

func (w *Builder) Contains(value interface{}) *Builder {
	return w.operation("$contains", value)
}

func (w *Builder) NotContains(value interface{}) *Builder {
	return w.operation("$not_contains", value)
}

func (w *Builder) And(builders ...*Builder) *Builder {
	if w.err != nil {
		return w
	}
	var andClause []map[string]interface{}
	for _, b := range builders {
		buildExpr, err := b.Build()
		if err != nil {
			w.err = err
			return w
		}
		andClause = append(andClause, buildExpr)
	}
	w.WhereClause["$and"] = andClause
	return w
}

func (w *Builder) Or(builders ...*Builder) *Builder {
	if w.err != nil {
		return w
	}
	var orClause []map[string]interface{}
	for _, b := range builders {
		buildExpr, err := b.Build()
		if err != nil {
			w.err = err
			return w
		}
		orClause = append(orClause, buildExpr)
	}
	w.WhereClause["$or"] = orClause
	return w
}

func (w *Builder) Build() (map[string]interface{}, error) {
	if w.err != nil {
		return nil, w.err
	}
	return w.WhereClause, nil
}

type WhereDocumentOperation func(builder *Builder) error

func Contains(value interface{}) WhereDocumentOperation {
	return func(w *Builder) error {
		w.Contains(value)
		return nil
	}
}

func NotContains(value interface{}) WhereDocumentOperation {
	return func(w *Builder) error {
		w.NotContains(value)
		return nil
	}
}

func And(ops ...WhereDocumentOperation) WhereDocumentOperation {
	return func(w *Builder) error {
		subBuilders := make([]*Builder, 0, len(ops))
		for _, op := range ops {
			wdx := NewWhereDocumentBuilder()
			if err := op(wdx); err != nil {
				return err
			}
			subBuilders = append(subBuilders, wdx)
		}
		w.And(subBuilders...)
		return nil
	}
}

func Or(ops ...WhereDocumentOperation) WhereDocumentOperation {
	return func(w *Builder) error {
		subBuilders := make([]*Builder, 0, len(ops))
		for _, op := range ops {
			wdx := NewWhereDocumentBuilder()
			if err := op(wdx); err != nil {
				return err
			}
			subBuilders = append(subBuilders, wdx)
		}
		w.Or(subBuilders...)
		return nil
	}
}

func WhereDocument(operation WhereDocumentOperation) (map[string]interface{}, error) {
	w := NewWhereDocumentBuilder()
	if err := operation(w); err != nil {
		return nil, err
	}
	return w.Build()
}
