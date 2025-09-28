# V2 API Simplification Improvements

This document describes the simplified V2 Chroma Go client API that follows Go's "one obvious way" principle for better developer experience.

## Overview

The simplified API provides a single, clear pattern for each operation, reducing cognitive load and decision fatigue. Following Go ecosystem best practices (as seen in AWS SDK v2, standard library), we provide one idiomatic way to accomplish each task.

## Key Improvements

### 1. Metadata Builder Pattern

**Problem:** Creating metadata required verbose constructor functions with redundant naming.

**Before (Deprecated):**
```go
// This pattern is deprecated - use Builder() instead
metadata := chroma.NewMetadata(
    chroma.NewStringAttribute("key", "value"),
    chroma.NewIntAttribute("count", 42),
    chroma.NewFloatAttribute("score", 0.95),
)
```

**After (Single Pattern):**
```go
metadata := chroma.Builder().
    String("key", "value").
    Int("count", 42).
    Float("score", 0.95).
    Build()
```

### 2. Type-Agnostic Where Clauses

**Problem:** Creating where clauses required remembering specific function names for each type.

**Before (Deprecated):**
```go
// These type-specific functions are deprecated
where := chroma.GtInt("priority", 5)
where2 := chroma.EqString("status", "active")
where3 := chroma.InFloat("score", []float32{0.8, 0.9})
```

**After (Single Pattern):**
```go
where := chroma.Gt("priority", 5)        // Auto-detects int
where2 := chroma.Eq("status", "active")  // Auto-detects string
where3 := chroma.In("score", []float32{0.8, 0.9})
```

### 3. Shorter Operator Constants

**Problem:** Operator constants were verbose and repetitive.

**Before:**
```go
chroma.GreaterThanOperator
chroma.LessThanOrEqualOperator
chroma.NotInOperator
```

**After:**
```go
chroma.GT   // Same as GreaterThanOperator
chroma.LTE  // Same as LessThanOrEqualOperator
chroma.NIN  // Same as NotInOperator
```

### 4. Cleaner Option Names

**Problem:** Option functions had operation-specific suffixes that increased cognitive load.

#### Collection Creation
- `WithCollectionMetadataCreate` → `WithMetadata`
- `WithEmbeddingFunctionCreate` → `WithEmbeddingFunction`
- `WithIfNotExistsCreate` → `WithCreateIfNotExists`

#### Query Options
- `WithNResults` → `WithLimit` (clearer naming)
- `WithQueryTexts` → `WithQueryText` (singular for single query)
- `WithQueryEmbeddings` → `WithQueryEmbedding` (singular for single embedding)

### 5. Result Access

**Standard Pattern:**
```go
// Access results using the standard Get methods
ids := result.GetIDs()
docs := result.GetDocuments()
metas := result.GetMetadatas()

// For query results
docs := queryResult.GetDocumentsGroups()[0]
ids := queryResult.GetIDsGroups()[0]
```

**Note:** The simplified converters (AsResult/AsQueryResults) have been removed to maintain API consistency and follow Go's "one obvious way" principle.

## Deprecation Notices

The following methods have been marked as deprecated with recommendations to use the simplified alternatives:

### Collection Operations
- `WithIDsGet`, `WithIDsDelete`, `WithIDsQuery` → Use operation-specific or create unified versions
- `WithWhereGet`, `WithWhereQuery`, `WithWhereDelete` → Use operation-specific or create unified versions
- `WithWhereDocumentGet`, `WithWhereDocumentQuery`, `WithWhereDocumentDelete` → Use operation-specific or create unified versions
- `WithTexts`, `WithTextsUpdate` → Consider more consistent naming in future versions

### Collection Management
- `WithDatabaseCreate`, `WithDatabaseGet`, `WithDatabaseDelete`, `WithDatabaseList`, `WithDatabaseCount` → Use operation-specific or create unified versions
- `WithIncludeGet`, `WithIncludeQuery` → Use operation-specific or create unified versions

## Migration Guide

### Phase 1: Immediate Benefits (No Breaking Changes)
1. Use Builder() pattern for metadata creation
2. Use type-agnostic Where functions (Eq, Gt, Lt, etc.)
3. Use shorter operator constants (GT, LTE, etc.)
4. Use WithLimit instead of WithNResults

### Phase 2: Gradual Adoption
1. Replace verbose option names with simplified versions where applicable
2. Migrate from deprecated functions to their simplified equivalents
3. Update tests to use new patterns

### Phase 3: Future Improvements (v0.3.0)
1. Full unified options across operations
2. Removal of deprecated methods
3. Complete API consistency

## Examples

See `/examples/v2/simplified_api/main.go` for comprehensive examples of all simplified APIs.

## Compatibility

All existing code continues to work without modification. Deprecated functions remain available but should be migrated to the new single-pattern approach for consistency. The simplified API follows Go's philosophy of having one clear, idiomatic way to accomplish each task.