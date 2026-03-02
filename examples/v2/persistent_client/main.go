// Package main shows a concise local persistent client workflow.
package main

import (
	"context"
	"fmt"
	"log"

	v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
	defaultef "github.com/amikos-tech/chroma-go/pkg/embeddings/default_ef"
)

const (
	persistPath    = "./chroma_data_local_persistent"
	collectionName = "persistent_local_demo"
)

func main() {
	ctx := context.Background()

	client, err := v2.NewPersistentClient(
		v2.WithPersistentPath(persistPath),
	)
	if err != nil {
		log.Fatalf("Error creating persistent client: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("Warning: failed to close client: %v", err)
		}
	}()

	embeddingFunction, closeEF, err := defaultef.NewDefaultEmbeddingFunction()
	if err != nil {
		log.Fatalf("Error creating default embedding function: %v", err)
	}
	defer func() {
		if err := closeEF(); err != nil {
			log.Printf("Warning: failed to close default embedding function: %v", err)
		}
	}()

	collection, err := client.GetOrCreateCollection(
		ctx,
		collectionName,
		v2.WithEmbeddingFunctionCreate(embeddingFunction),
	)
	if err != nil {
		log.Fatalf("Error creating/getting collection: %v", err)
	}

	countBeforeUpsert, err := collection.Count(ctx)
	if err != nil {
		log.Fatalf("Error counting documents before upsert: %v", err)
	}
	fmt.Printf("Persistence path: %s\n", persistPath)
	fmt.Printf("Collection: %s\n", collectionName)
	fmt.Printf("Existing docs before upsert: %d\n", countBeforeUpsert)

	err = collection.Upsert(
		ctx,
		v2.WithIDs("doc-1", "doc-2", "doc-3"),
		v2.WithTexts(
			"Chroma stores vectors and metadata for semantic retrieval.",
			"Local persistence keeps data between application restarts.",
			"Upsert makes repeated runs idempotent for fixed document IDs.",
		),
	)
	if err != nil {
		log.Fatalf("Error upserting documents: %v", err)
	}

	countAfterUpsert, err := collection.Count(ctx)
	if err != nil {
		log.Fatalf("Error counting documents after upsert: %v", err)
	}
	fmt.Printf("Docs after upsert: %d\n", countAfterUpsert)

	queryResult, err := collection.Query(
		ctx,
		v2.WithQueryTexts("Which document explains local persistence?"),
		v2.WithNResults(1),
		v2.WithInclude(v2.IncludeDocuments),
	)
	if err != nil {
		log.Fatalf("Error querying collection: %v", err)
	}
	printTopResult(queryResult)

	fmt.Printf(
		"Tip: run `go run .` again. If \"Existing docs before upsert\" is > 0, local persistence is working.\n",
	)
}

func printTopResult(qr v2.QueryResult) {
	idGroups := qr.GetIDGroups()
	docGroups := qr.GetDocumentsGroups()
	if len(idGroups) == 0 || len(idGroups[0]) == 0 || len(docGroups) == 0 || len(docGroups[0]) == 0 {
		fmt.Println("Top query result: no results")
		return
	}

	fmt.Printf(
		"Top query result: id=%s, document=%q\n",
		idGroups[0][0],
		docGroups[0][0].ContentString(),
	)
}
