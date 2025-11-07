# Search API Examples

This directory contains examples demonstrating the Search API's advanced querying capabilities.

> **Official Documentation**: For the upstream Chroma Search API documentation, see:
> - [Search API Overview](https://docs.trychroma.com/cloud/search-api/overview)
> - [Search API Guide](https://docs.trychroma.com/guides/search)

## Prerequisites

1. Go 1.21 or higher
2. Running Chroma server (default: `http://localhost:8000`)

Start a local Chroma server:
```bash
# Using Docker
docker run -p 8000:8000 ghcr.io/chroma-core/chroma:latest

# Or using the Makefile from the project root
make server
```

## Examples

### 1. Simple KNN Search (`simple_knn/`)

Demonstrates basic K-Nearest Neighbors search functionality.

**Features:**
- Basic KNN search with query texts
- Multiple search queries
- Different K values

**Run:**
```bash
cd simple_knn
go run main.go
```

**What it shows:**
- How to perform simple similarity search
- How query text affects results
- How K parameter controls number of results

---

### 2. RRF Multi-Query (`rrf_multi_query/`)

Shows Reciprocal Rank Fusion (RRF) for combining multiple search strategies.

**Features:**
- Single vs. multiple query comparison
- RRF with 2-3 different queries
- Diverse query combination

**Run:**
```bash
cd rrf_multi_query
go run main.go
```

**What it shows:**
- Documents appearing in multiple searches rank higher
- RRF acts like intelligent "OR" operation
- Combining semantic variations improves recall

**Use cases:**
- Multi-lingual search
- Synonym expansion
- Query variation handling

---

### 3. Filtered Search (`filtered_search/`)

Demonstrates metadata filtering combined with semantic search.

**Features:**
- Simple metadata filters
- Complex AND/OR filter combinations
- Numeric comparisons (rating, year)
- IN operator for multiple values

**Run:**
```bash
cd filtered_search
go run main.go
```

**What it shows:**
- How to filter by metadata fields
- Combining multiple filter conditions
- Using comparison operators (>, <, >=, <=)
- IN/NOT IN for set membership

**Use cases:**
- Category filtering
- Price range filtering
- Date range queries
- Status-based filtering

---

### 4. Hybrid Search (`hybrid_search/`)

Advanced hybrid search strategies combining semantic search with business logic.

**Features:**
- Semantic + business rules
- Multi-query RRF with filters
- Score transformations (exponential boost)
- Intent-based search
- Complex multi-strategy combinations

**Run:**
```bash
cd hybrid_search
go run main.go
```

**What it shows:**
- E-commerce product search patterns
- Combining multiple ranking strategies
- Using filters for business constraints
- Score transformation for emphasis

**Use cases:**
- E-commerce product search
- Content recommendation
- Document retrieval with constraints
- Multi-faceted search

---

## Common Patterns

### Basic Search Pattern

```go
results, err := collection.Search(ctx,
    v2.WithSearchRankKnnTexts([]string{"query"}, 10),
    v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
)
```

### With Filters

```go
results, err := collection.Search(ctx,
    v2.WithSearchRankKnnTexts([]string{"query"}, 10),
    v2.WithSearchWhere(v2.EqString("category", "tech")),
    v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
)
```

### RRF Combination

```go
rank1 := &v2.KnnRank{QueryTexts: []string{"query1"}, K: 10}
rank2 := &v2.KnnRank{QueryTexts: []string{"query2"}, K: 10}

results, err := collection.Search(ctx,
    v2.WithSearchRankRrf([]v2.RankExpression{rank1, rank2}, 60, true),
    v2.WithSearchLimit(10, 0),
    v2.WithSearchSelect(v2.SelectID, v2.SelectDocument, v2.SelectScore),
)
```

### Processing Results

```go
// Get result groups
ids := results.GetIDGroups()[0]
docs := results.GetDocumentsGroups()[0]
scores := results.GetScoresGroups()[0]

// Iterate through results
for i, id := range ids {
    fmt.Printf("ID: %s, Score: %.4f\n", id, scores[i])
    fmt.Printf("Doc: %s\n", docs[i].ContentString())
}
```

## Tips

1. **Start Simple**: Begin with `simple_knn` before exploring advanced features
2. **Experiment with K**: Try different K values to find optimal results
3. **Use Filters Wisely**: Filters are applied after ranking, so use them for business constraints
4. **RRF Parameters**: k=60 is a good default for RRF; normalize=true usually works better
5. **Monitor Performance**: Complex rank expressions may be slower than simple KNN

## Documentation

For complete API documentation, see:
- [Search API Documentation](../../../docs/docs/search.md)
- [Filtering Documentation](../../../docs/docs/filtering.md)

## Troubleshooting

**Connection refused:**
```
Make sure Chroma server is running on http://localhost:8000
```

**Import errors:**
```bash
go mod tidy
```

**No results found:**
- Check that documents were added successfully
- Try broader queries or higher K values
- Verify filters aren't too restrictive

## Next Steps

After exploring these examples, try:
1. Modifying queries to match your use case
2. Adding your own documents and metadata
3. Experimenting with different rank expressions
4. Combining techniques from multiple examples

For more advanced usage, check the main [examples directory](../) for additional V2 API examples.
