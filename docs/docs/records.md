# Records

!!! warning "Removed in V2"

    The `RecordSet` API was part of the V1 API which has been removed in v0.3.0.
    In the V2 API, use the unified options pattern directly with collection operations:

    ```go
    // Add documents directly
    col.Add(ctx,
        chroma.WithIDs("id1", "id2"),
        chroma.WithTexts("Document 1", "Document 2"),
        chroma.WithMetadatas(
            chroma.NewDocumentMetadata(chroma.NewStringAttribute("key", "value1")),
            chroma.NewDocumentMetadata(chroma.NewStringAttribute("key", "value2")),
        ),
    )
    ```

    See the [main README](https://github.com/amikos-tech/chroma-go#unified-options-api) for the full V2 options reference.
