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

TBD