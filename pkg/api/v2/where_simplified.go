package v2

// Simplified Where clause constructors using short operator names
// These complement the existing Eq, Ne, Gt, Lt, etc. functions

// Where creates a where clause with simplified operator names.
// Example: Where(EQ, "field", "value") or Where(GT, "count", 5)
// Returns nil for unsupported operator/type combinations.
// Supported operations:
//   - String: EQ, NE
//   - Int/Int64: EQ, NE, GT, GTE, LT, LTE
//   - Float32/Float64: EQ, NE, GT, GTE, LT, LTE
//   - []string: IN, NIN
//   - []int: IN, NIN
//   - []float32/[]float64: IN, NIN
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
		if v > int64(^uint(0)>>1) || v < int64(^int(^uint(0)>>1)) {
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

// Eq creates an equality filter (replaces EqString, EqInt, EqFloat)
func Eq(field string, value interface{}) WhereFilter {
	return Where(EQ, field, value)
}

// Ne creates a not-equal filter (replaces NeString, NeInt, NeFloat)
func Ne(field string, value interface{}) WhereFilter {
	return Where(NE, field, value)
}

// Gt creates a greater-than filter (replaces GtInt, GtFloat)
func Gt(field string, value interface{}) WhereFilter {
	return Where(GT, field, value)
}

// Gte creates a greater-than-or-equal filter (replaces GteInt, GteFloat)
func Gte(field string, value interface{}) WhereFilter {
	return Where(GTE, field, value)
}

// Lt creates a less-than filter (replaces LtInt, LtFloat)
func Lt(field string, value interface{}) WhereFilter {
	return Where(LT, field, value)
}

// Lte creates a less-than-or-equal filter (replaces LteInt, LteFloat)
func Lte(field string, value interface{}) WhereFilter {
	return Where(LTE, field, value)
}

// In creates an IN filter (replaces InString, InInt, InFloat)
func In(field string, values interface{}) WhereFilter {
	return Where(IN, field, values)
}

// NotIn creates a NOT IN filter (replaces NotInString, NotInInt, NotInFloat)
func NotIn(field string, values interface{}) WhereFilter {
	return Where(NIN, field, values)
}
