# Metadata filtering example

This example demonstrate how to use metadata filters to query Chroma collections.

```go
package main
import (
    "context"
    "fmt"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

qr, err := col.Query(context.Background(),
    chroma.WithQueryTexts("say hello"),
    chroma.WithInclude(chroma.IncludeDocuments, chroma.IncludeMetadatas),
    // Example with a single filter:
    // chroma.WithWhere(StringFilter)

    // Example with multiple combined filters:
    chroma.WithWhere(
        chroma.Or(chroma.EqString("str1", "hello2"), chroma.EqInt("int", 1)),
    ),
)
```

## Run the example

```bash
cd examples/v2/metadata_filters
make run
```
