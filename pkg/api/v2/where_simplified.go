package v2

// Simplified Where clause constructors using short operator names
// These complement the existing Eq, Ne, Gt, Lt, etc. functions

// Where creates a where clause with simplified operator names.
// Example: Where(EQ, "field", "value") or Where(GT, "count", 5)
//
// Returns nil for unsupported operator/type combinations or invalid inputs.
// This includes:
//   - Unsupported operators for the given type
//   - Empty field names or nil values
//   - Empty slices for IN/NIN operators
//   - Unsupported value types
//
// Type conversions:
//   - int64: Values exceeding int range are converted to float64 to preserve magnitude
//   - float64: Always converted to float32 (may lose precision)
//
// Supported operations:
//   - String: EQ, NE
//   - Int/Int64: EQ, NE, GT, GTE, LT, LTE
//   - Float32/Float64: EQ, NE, GT, GTE, LT, LTE
//   - []string: IN, NIN
//   - []int: IN, NIN
//   - []float32: IN, NIN
func Where(operator WhereFilterOperator, field string, value interface{}) WhereFilter {
	if field == "" || value == nil {
		return nil
	}
	switch v := value.(type) {
	case string:
		switch operator {
		case EQ:
			return EqString(field, v)
		case NE:
			// NE not implemented for strings in base API, create manually
			return &WhereClauseString{
				WhereClauseBase: WhereClauseBase{
					operator: NE,
					key:      field,
				},
				operand: v,
			}
		default:
			// Strings only support EQ and NE
			return nil
		}
	case int:
		switch operator {
		case EQ:
			return EqInt(field, v)
		case NE:
			// NE not implemented for ints in base API, create manually
			return &WhereClauseInt{
				WhereClauseBase: WhereClauseBase{
					operator: NE,
					key:      field,
				},
				operand: v,
			}
		case GT:
			return GtInt(field, v)
		case GTE:
			return GteInt(field, v)
		case LT:
			return LtInt(field, v)
		case LTE:
			return LteInt(field, v)
		default:
			return nil
		}
	case int64:
		// Check for overflow before converting to int
		const maxInt = int64(^uint(0) >> 1)
		const minInt = -maxInt - 1
		if v > maxInt || v < minInt {
			// Value would overflow int, convert to float64 to preserve magnitude
			// This allows filtering on large int64 values at the cost of potential precision
			return Where(operator, field, float64(v))
		}
		return Where(operator, field, int(v))
	case float32:
		switch operator {
		case EQ:
			return EqFloat(field, v)
		case NE:
			// NE not implemented for floats in base API, create manually
			return &WhereClauseFloat{
				WhereClauseBase: WhereClauseBase{
					operator: NE,
					key:      field,
				},
				operand: v,
			}
		case GT:
			return GtFloat(field, v)
		case GTE:
			return GteFloat(field, v)
		case LT:
			return LtFloat(field, v)
		case LTE:
			return LteFloat(field, v)
		default:
			return nil
		}
	case float64:
		// Warning: Converting float64 to float32 may result in precision loss
		return Where(operator, field, float32(v))
	case []string:
		if len(v) == 0 {
			return nil
		}
		switch operator {
		case IN:
			return InString(field, v...)
		case NIN:
			return NinString(field, v...)
		default:
			return nil
		}
	case []int:
		if len(v) == 0 {
			return nil
		}
		switch operator {
		case IN:
			return InInt(field, v...)
		case NIN:
			return NinInt(field, v...)
		default:
			return nil
		}
	case []float32:
		if len(v) == 0 {
			return nil
		}
		switch operator {
		case IN:
			return InFloat(field, v...)
		case NIN:
			return NinFloat(field, v...)
		default:
			return nil
		}
	default:
		return nil
	}
}

// Convenience functions with short names

// Eq creates an equality filter for any supported type.
// Returns nil if the value type is not supported or field is empty.
// Note: int64 values exceeding int range are converted to float64.
// Note: float64 values are converted to float32 (potential precision loss).
func Eq(field string, value interface{}) WhereFilter {
	if field == "" || value == nil {
		return nil
	}
	return Where(EQ, field, value)
}

// Ne creates a not-equal filter for any supported type.
// Returns nil if the value type is not supported or field is empty.
// Note: int64 values exceeding int range are converted to float64.
// Note: float64 values are converted to float32 (potential precision loss).
func Ne(field string, value interface{}) WhereFilter {
	if field == "" || value == nil {
		return nil
	}
	return Where(NE, field, value)
}

// Gt creates a greater-than filter for numeric types.
// Returns nil if the value type is not numeric, not supported, or field is empty.
// Note: int64 values exceeding int range are converted to float64.
// Note: float64 values are converted to float32 (potential precision loss).
func Gt(field string, value interface{}) WhereFilter {
	if field == "" || value == nil {
		return nil
	}
	return Where(GT, field, value)
}

// Gte creates a greater-than-or-equal filter for numeric types.
// Returns nil if the value type is not numeric, not supported, or field is empty.
// Note: int64 values exceeding int range are converted to float64.
// Note: float64 values are converted to float32 (potential precision loss).
func Gte(field string, value interface{}) WhereFilter {
	if field == "" || value == nil {
		return nil
	}
	return Where(GTE, field, value)
}

// Lt creates a less-than filter for numeric types.
// Returns nil if the value type is not numeric, not supported, or field is empty.
// Note: int64 values exceeding int range are converted to float64.
// Note: float64 values are converted to float32 (potential precision loss).
func Lt(field string, value interface{}) WhereFilter {
	if field == "" || value == nil {
		return nil
	}
	return Where(LT, field, value)
}

// Lte creates a less-than-or-equal filter for numeric types.
// Returns nil if the value type is not numeric, not supported, or field is empty.
// Note: int64 values exceeding int range are converted to float64.
// Note: float64 values are converted to float32 (potential precision loss).
func Lte(field string, value interface{}) WhereFilter {
	if field == "" || value == nil {
		return nil
	}
	return Where(LTE, field, value)
}

// In creates an IN filter for slice types.
// Returns nil if the value is not a supported slice type or field is empty.
// Supported: []string, []int, []float32
func In(field string, values interface{}) WhereFilter {
	if field == "" || values == nil {
		return nil
	}
	return Where(IN, field, values)
}

// NotIn creates a NOT IN filter for slice types.
// Returns nil if the value is not a supported slice type or field is empty.
// Supported: []string, []int, []float32
func NotIn(field string, values interface{}) WhereFilter {
	if field == "" || values == nil {
		return nil
	}
	return Where(NIN, field, values)
}
