package main

import (
	"context"
	"fmt"
	"log"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	// Create a new Chroma client
	// Note: WithDebug() is deprecated - use WithLogger with debug level for logging
	client, err := chroma.NewHTTPClient()
	if err != nil {
		log.Printf("Error creating client: %s \n", err)
		return
	}
	// Close the client to release any resources such as local embedding functions
	defer func() {
		err = client.Close()
		if err != nil {
			log.Printf("Error closing client: %s \n", err)
			return
		}
	}()

	// Create a new collection with options. We don't provide an embedding function here, so the default embedding function will be used
	col, err := client.GetOrCreateCollection(context.Background(), "col1",
		chroma.WithMetadata(
			chroma.Builder().
				String("str", "hello2").
				Int("int", 1).
				Float("float", 1.1).
				Build(),
		),
	)
	if err != nil {
		_ = client.Close() // Ensure the client is closed before exiting
		log.Printf("Error creating collection: %s \n", err)
		return
	}

	err = col.Add(context.Background(),
		// chroma.WithIDGenerator(chroma.NewULIDGenerator()),
		chroma.WithIDs("1", "2"),
		chroma.WithTexts("hello world", "goodbye world"),
		chroma.WithMetadatas(
			chroma.DocumentBuilder().Int("int", 1).Build(),
			chroma.DocumentBuilder().String("str1", "hello2").Build(),
		))
	if err != nil {
		log.Printf("Error adding collection: %s \n", err)
		return
	}

	count, err := col.Count(context.Background())
	if err != nil {
		log.Printf("Error counting collection: %s \n", err)
		return
	}
	fmt.Printf("Count collection: %d\n", count)
	IntFilter := chroma.EqInt("int", 1)
	StringFilter := chroma.EqString("str1", "hello2")
	qr, err := col.Query(context.Background(),
		chroma.WithQueryTexts("say hello"),
		chroma.WithIncludeQuery(chroma.IncludeDocuments, chroma.IncludeMetadatas),
		// Example with a single filter:
		// chroma.WithWhereQuery(StringFilter)

		// Example with multiple combined filters:
		chroma.WithWhereQuery(
			chroma.Or(StringFilter, IntFilter),
		),
	)
	if err != nil {
		log.Printf("Error querying collection: %s \n", err)
		return
	}
	fmt.Printf("Query result expected: 'hello world', actual: '%v'\n", qr.GetDocumentsGroups()[0][0]) // goodbye world is also returned because of the OR filter

	err = col.Delete(context.Background(), chroma.WithIDsDelete("1", "2"))
	if err != nil {
		log.Printf("Error deleting collection: %s \n", err)
		return
	}
}
