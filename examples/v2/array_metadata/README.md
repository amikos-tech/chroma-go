# Array Metadata Example

This example demonstrates how to use array metadata values with Chroma collections. Array metadata allows storing lists of values (strings, integers, floats, booleans) as metadata on documents, and querying them using `$contains` and `$not_contains` operators.

## Features Demonstrated

- Creating collections with array metadata
- Adding documents with string, int, float, and bool array metadata
- Querying with `$contains` to find documents where an array field contains a value
- Querying with `$not_contains` to exclude documents
- Combining array filters with `And`/`Or` operators

```go
// Add documents with array metadata
err = col.Add(context.Background(),
    chroma.WithIDs("doc1"),
    chroma.WithTexts("Einstein's theory of relativity"),
    chroma.WithMetadatas(
        chroma.NewDocumentMetadata(
            chroma.NewStringArrayAttribute("tags", []string{"physics", "science"}),
            chroma.NewIntArrayAttribute("years", []int64{1905, 1915}),
        ),
    ),
)

// Query using $contains
qr, err := col.Query(context.Background(),
    chroma.WithQueryTexts("scientific discoveries"),
    chroma.WithWhere(
        chroma.MetadataContainsString(chroma.K("tags"), "physics"),
    ),
)
```

## Run the example

```bash
cd examples/v2/array_metadata
make run
```
