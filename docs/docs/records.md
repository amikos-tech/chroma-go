# Records

Records are a coherency mechanism that allows you to manage Chroma data in a structured way.

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
