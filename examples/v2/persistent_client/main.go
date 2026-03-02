// Package main demonstrates local persistence with NewPersistentClient.
//
// Phase 1 writes documents to a fixed collection and closes the client.
// Phase 2 reopens the client from the same path and verifies the data persists.
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

var sampleData = []struct {
	id       string
	text     string
	topic    string
	priority int
}{
	{
		id:       "doc-1",
		text:     "Chroma stores vectors and metadata for semantic retrieval.",
		topic:    "overview",
		priority: 1,
	},
	{
		id:       "doc-2",
		text:     "Local persistence keeps data between application restarts.",
		topic:    "persistence",
		priority: 2,
	},
	{
		id:       "doc-3",
		text:     "Upsert makes repeated runs idempotent for fixed document IDs.",
		topic:    "operations",
		priority: 3,
	},
}

func main() {
	ctx := context.Background()

	fmt.Println("=== Phase 1: Write data to local persistent client ===")
	fmt.Printf("Persistence path: %s\n", persistPath)

	phaseOneClient, err := v2.NewPersistentClient(
		v2.WithPersistentPath(persistPath),
	)
	if err != nil {
		log.Fatalf("Error creating phase 1 client: %v", err)
	}

	// Use Default EF explicitly so text upsert/query is deterministic and reproducible.
	phaseOneEF, closePhaseOneEF, err := defaultef.NewDefaultEmbeddingFunction()
	if err != nil {
		_ = phaseOneClient.Close()
		log.Fatalf("Error creating phase 1 default embedding function: %v", err)
	}
	defer func() {
		if closeErr := closePhaseOneEF(); closeErr != nil {
			log.Fatalf("Error closing phase 1 default embedding function: %v", closeErr)
		}
	}()

	phaseOneCollection, err := phaseOneClient.GetOrCreateCollection(
		ctx,
		collectionName,
		v2.WithEmbeddingFunctionCreate(phaseOneEF),
	)
	if err != nil {
		_ = phaseOneClient.Close()
		log.Fatalf("Error creating/getting collection: %v", err)
	}

	err = phaseOneCollection.Upsert(
		ctx,
		v2.WithIDs("doc-1", "doc-2", "doc-3"),
		v2.WithTexts(sampleData[0].text, sampleData[1].text, sampleData[2].text),
		v2.WithMetadatas(
			v2.NewDocumentMetadata(
				v2.NewStringAttribute("topic", sampleData[0].topic),
				v2.NewIntAttribute("priority", int64(sampleData[0].priority)),
			),
			v2.NewDocumentMetadata(
				v2.NewStringAttribute("topic", sampleData[1].topic),
				v2.NewIntAttribute("priority", int64(sampleData[1].priority)),
			),
			v2.NewDocumentMetadata(
				v2.NewStringAttribute("topic", sampleData[2].topic),
				v2.NewIntAttribute("priority", int64(sampleData[2].priority)),
			),
		),
	)
	if err != nil {
		_ = phaseOneClient.Close()
		log.Fatalf("Error upserting documents: %v", err)
	}

	countAfterWrite, err := phaseOneCollection.Count(ctx)
	if err != nil {
		_ = phaseOneClient.Close()
		log.Fatalf("Error counting documents after write: %v", err)
	}
	fmt.Printf("Count after write: %d\n", countAfterWrite)

	phaseOneQuery, err := phaseOneCollection.Query(
		ctx,
		v2.WithQueryTexts("Which document explains local persistence?"),
		v2.WithNResults(1),
		v2.WithInclude(v2.IncludeDocuments, v2.IncludeMetadatas),
	)
	if err != nil {
		_ = phaseOneClient.Close()
		log.Fatalf("Error querying in phase 1: %v", err)
	}
	printTopResult("Phase 1 top result", phaseOneQuery)

	if err := phaseOneClient.Close(); err != nil {
		log.Fatalf("Error closing phase 1 client: %v", err)
	}
	fmt.Println("Phase 1 client closed.")

	fmt.Println()
	fmt.Println("=== Phase 2: Reopen client and verify persistence ===")
	fmt.Printf("Reopening from path: %s\n", persistPath)

	phaseTwoClient, err := v2.NewPersistentClient(
		v2.WithPersistentPath(persistPath),
	)
	if err != nil {
		log.Fatalf("Error creating phase 2 client: %v", err)
	}
	defer func() {
		if closeErr := phaseTwoClient.Close(); closeErr != nil {
			log.Fatalf("Error closing phase 2 client: %v", closeErr)
		}
	}()

	phaseTwoEF, closePhaseTwoEF, err := defaultef.NewDefaultEmbeddingFunction()
	if err != nil {
		log.Fatalf("Error creating phase 2 default embedding function: %v", err)
	}
	defer func() {
		if closeErr := closePhaseTwoEF(); closeErr != nil {
			log.Fatalf("Error closing phase 2 default embedding function: %v", closeErr)
		}
	}()

	phaseTwoCollection, err := phaseTwoClient.GetCollection(
		ctx,
		collectionName,
		v2.WithEmbeddingFunctionGet(phaseTwoEF),
	)
	if err != nil {
		log.Fatalf("Error getting collection after reopen: %v", err)
	}

	countAfterReopen, err := phaseTwoCollection.Count(ctx)
	if err != nil {
		log.Fatalf("Error counting documents after reopen: %v", err)
	}
	fmt.Printf("Count after reopen: %d\n", countAfterReopen)

	expectedMinimum := len(sampleData)
	if countAfterReopen == 0 || countAfterReopen < expectedMinimum || countAfterReopen < countAfterWrite {
		panic(fmt.Sprintf(
			"persistence check failed: count after reopen=%d, phase1 count=%d, expected minimum=%d",
			countAfterReopen,
			countAfterWrite,
			expectedMinimum,
		))
	}

	phaseTwoQuery, err := phaseTwoCollection.Query(
		ctx,
		v2.WithQueryTexts("Which document explains local persistence?"),
		v2.WithNResults(1),
		v2.WithInclude(v2.IncludeDocuments, v2.IncludeMetadatas),
	)
	if err != nil {
		log.Fatalf("Error querying in phase 2: %v", err)
	}
	printTopResult("Phase 2 top result", phaseTwoQuery)

	fmt.Printf(
		"Persistence verified: collection %q retained %d documents after client restart.\n",
		collectionName,
		countAfterReopen,
	)
}

func printTopResult(label string, qr v2.QueryResult) {
	idGroups := qr.GetIDGroups()
	docGroups := qr.GetDocumentsGroups()
	if len(idGroups) == 0 || len(idGroups[0]) == 0 || len(docGroups) == 0 || len(docGroups[0]) == 0 {
		fmt.Printf("%s: no results\n", label)
		return
	}

	fmt.Printf(
		"%s: id=%s, document=%q\n",
		label,
		idGroups[0][0],
		docGroups[0][0].ContentString(),
	)
}
