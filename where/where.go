package where

import (
	"fmt"
)

type Builder struct {
	WhereClause map[string]interface{}
	err         error
}
type InvalidWhereValueError struct {
	Key   string
	Value interface{}
}

func (e *InvalidWhereValueError) Error() string {
	return fmt.Sprintf("Invalid value for where clause for key %s: %v. Allowed values are string, int, float, bool", e.Key, e.Value)
}

func NewWhereBuilder() *Builder {
	return &Builder{WhereClause: make(map[string]interface{})}
}

func (w *Builder) operation(operation string, field string, value interface{}) *Builder {
	if w.err != nil {
		return w
	}
	inner := make(map[string]interface{})
	if v, ok := value.([]interface{}); ok {
		inner[operation] = v
	} else {
		switch value.(type) {
		case string, int, float32, bool:
		default:
			w.err = &InvalidWhereValueError{Key: field, Value: value}
			return w
		}
		inner[operation] = value
	}
	w.WhereClause[field] = inner
	return w
}
func (w *Builder) Build() (map[string]interface{}, error) {
	if w.err != nil {
		return nil, w.err
	}
	return w.WhereClause, nil
}
func (w *Builder) Eq(key string, value interface{}) *Builder {
	return w.operation("$eq", key, value)
}
func (w *Builder) Ne(key string, value interface{}) *Builder {
	return w.operation("$ne", key, value)
}
func (w *Builder) Gt(key string, value interface{}) *Builder {
	return w.operation("$gt", key, value)
}
func (w *Builder) Gte(key string, value interface{}) *Builder {
	return w.operation("$gte", key, value)
}
func (w *Builder) Lt(key string, value interface{}) *Builder {
	return w.operation("$lt", key, value)
}
func (w *Builder) Lte(key string, value interface{}) *Builder {
	return w.operation("$lte", key, value)
}
func (w *Builder) In(key string, value []interface{}) *Builder {
	return w.operation("$in", key, value)
}
func (w *Builder) Nin(key string, value []interface{}) *Builder {
	return w.operation("$nin", key, value)
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

type WhereOperation func(*Builder) error

func Eq(key string, value interface{}) WhereOperation {
	return func(w *Builder) error {
		w.Eq(key, value)
		return nil
	}
}

func Ne(key string, value interface{}) WhereOperation {
	return func(w *Builder) error {
		w.Ne(key, value)
		return nil
	}
}

func Gt(key string, value interface{}) WhereOperation {
	return func(w *Builder) error {
		w.Gt(key, value)
		return nil
	}
}

func Gte(key string, value interface{}) WhereOperation {
	return func(w *Builder) error {
		w.Gte(key, value)
		return nil
	}
}

func Lt(key string, value interface{}) WhereOperation {
	return func(w *Builder) error {
		w.Lt(key, value)
		return nil
	}
}

func Lte(key string, value interface{}) WhereOperation {
	return func(w *Builder) error {
		w.Lte(key, value)
		return nil
	}
}

func In(key string, value []interface{}) WhereOperation {
	return func(w *Builder) error {
		w.In(key, value)
		return nil
	}
}

func Nin(key string, value []interface{}) WhereOperation {
	return func(w *Builder) error {
		w.Nin(key, value)
		return nil
	}
}

func And(ops ...WhereOperation) WhereOperation {
	return func(w *Builder) error {
		subBuilders := make([]*Builder, 0, len(ops))
		for _, op := range ops {
			wx := NewWhereBuilder()
			if err := op(wx); err != nil {
				return err
			}
			subBuilders = append(subBuilders, wx)
		}
		w.And(subBuilders...)
		return nil
	}
}

func Or(ops ...WhereOperation) WhereOperation {
	return func(w *Builder) error {
		subBuilders := make([]*Builder, 0, len(ops))
		for _, op := range ops {
			wx := NewWhereBuilder()
			if err := op(wx); err != nil {
				return err
			}
			subBuilders = append(subBuilders, wx)
		}
		w.Or(subBuilders...)
		return nil
	}
}

func Where(operation WhereOperation) (map[string]interface{}, error) {
	w := NewWhereBuilder()

	if err := operation(w); err != nil {
		return nil, err
	}
	return w.Build()
}
