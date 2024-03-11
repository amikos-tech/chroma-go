# Records

Records are a mechanism that allows you to manage Chroma documents as a cohesive unit. This has several advantages over
the traditional approach of managing documents, ids, embeddings, and metadata separately.

Two concepts are important to keep in mind here:

- Record - corresponds to a single document in Chroma which includes id, embedding, metadata, the document or URI
- RecordSet - a single unit of work to insert, upsert, update or delete records.


## Record

A Record contains the following fields:

- ID (string)
- Document (string) - optional
- Metadata (map[string]interface{}) - optional
- Embedding ([]float32 or []int32, wrapped in Embedding struct)
- URI (string) - optional

Here's the `Record` type:

```go
package types

type Record struct {
	ID        string
	Embedding Embedding
	Metadata  map[string]interface{}
	Document  string
	URI       string
	err       error // indicating whether the record is valid
}
```

## RecordSet

A record set is a cohesive unit of work, allowing the user to add, upsert, update, or delete records.


!!! note "Operation support"

    Currently the record set only supports add operation

```go
rs, rerr := types.NewRecordSet(
			types.WithEmbeddingFunction(types.NewConsistentHashEmbeddingFunction()),
			types.WithIDGenerator(types.NewULIDGenerator()),
		)
if err != nil {
    log.Fatalf("Error creating record set: %s", err)
}
// you can loop here to add multiple records
rs.WithRecord(types.WithDocument("Document 1 content"), types.WithMetadata("key1", "value1"))
rs.WithRecord(types.WithDocument("Document 2 content"), types.WithMetadata("key2", "value2"))
records, err = rs.BuildAndValidate(context.Background())

```