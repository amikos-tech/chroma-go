package v2

// Simplified Where clause constructors using short operator names
// These complement the existing Eq, Ne, Gt, Lt, etc. functions

// Where creates a where clause with simplified operator names.
// Example: Where(EQ, "field", "value") or Where(GT, "count", 5)
//
// Returns nil for unsupported operator/type combinations or invalid inputs.
// This includes:
//   - Unsupported operators for the given type
//   - Int64 values that would overflow when converted to int
//   - Unsupported value types
//
// Note: float64 values are converted to float32, which may result in precision loss.
// For applications requiring high precision, consider using float32 values directly.
//
// Supported operations:
//   - String: EQ, NE
//   - Int/Int64: EQ, NE, GT, GTE, LT, LTE (int64 must fit in int range)
//   - Float32/Float64: EQ, NE, GT, GTE, LT, LTE (float64 converted to float32)
//   - []string: IN, NIN
//   - []int: IN, NIN
//   - []float32: IN, NIN
func Where(operator WhereFilterOperator, field string, value interface{}) WhereFilter {
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
			// Value would overflow, return nil to indicate unsupported
			return nil
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
		switch operator {
		case IN:
			return InString(field, v...)
		case NIN:
			return NinString(field, v...)
		default:
			return nil
		}
	case []int:
		switch operator {
		case IN:
			return InInt(field, v...)
		case NIN:
			return NinInt(field, v...)
		default:
			return nil
		}
	case []float32:
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
// Returns nil if the value type is not supported.
// Note: float64 values are converted to float32 (potential precision loss).
func Eq(field string, value interface{}) WhereFilter {
	return Where(EQ, field, value)
}

// Ne creates a not-equal filter for any supported type.
// Returns nil if the value type is not supported.
// Note: float64 values are converted to float32 (potential precision loss).
func Ne(field string, value interface{}) WhereFilter {
	return Where(NE, field, value)
}

// Gt creates a greater-than filter for numeric types.
// Returns nil if the value type is not numeric or not supported.
// Note: float64 values are converted to float32 (potential precision loss).
func Gt(field string, value interface{}) WhereFilter {
	return Where(GT, field, value)
}

// Gte creates a greater-than-or-equal filter for numeric types.
// Returns nil if the value type is not numeric or not supported.
// Note: float64 values are converted to float32 (potential precision loss).
func Gte(field string, value interface{}) WhereFilter {
	return Where(GTE, field, value)
}

// Lt creates a less-than filter for numeric types.
// Returns nil if the value type is not numeric or not supported.
// Note: float64 values are converted to float32 (potential precision loss).
func Lt(field string, value interface{}) WhereFilter {
	return Where(LT, field, value)
}

// Lte creates a less-than-or-equal filter for numeric types.
// Returns nil if the value type is not numeric or not supported.
// Note: float64 values are converted to float32 (potential precision loss).
func Lte(field string, value interface{}) WhereFilter {
	return Where(LTE, field, value)
}

// In creates an IN filter for slice types.
// Returns nil if the value is not a supported slice type.
// Supported: []string, []int, []float32
func In(field string, values interface{}) WhereFilter {
	return Where(IN, field, values)
}

// NotIn creates a NOT IN filter for slice types.
// Returns nil if the value is not a supported slice type.
// Supported: []string, []int, []float32
func NotIn(field string, values interface{}) WhereFilter {
	return Where(NIN, field, values)
}
