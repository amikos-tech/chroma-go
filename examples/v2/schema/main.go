package main

import (
	"encoding/json"
	"fmt"
	"log"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func mustSchema(opts ...chroma.SchemaOption) *chroma.Schema {
	schema, err := chroma.NewSchema(opts...)
	if err != nil {
		log.Fatalf("schema creation failed: %v", err)
	}
	return schema
}

func printSchema(title string, schema *chroma.Schema) {
	payload, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		log.Fatalf("schema marshal failed: %v", err)
	}
	fmt.Printf("== %s ==\n%s\n\n", title, payload)
}

func main() {
	defaults, err := chroma.NewSchemaWithDefaults()
	if err != nil {
		log.Fatalf("schema defaults failed: %v", err)
	}
	printSchema("NewSchemaWithDefaults", defaults)

	withFTS := mustSchema(
		chroma.WithDefaultVectorIndex(chroma.NewVectorIndexConfig(
			chroma.WithSpace(chroma.SpaceL2),
		)),
		chroma.WithDefaultFtsIndex(&chroma.FtsIndexConfig{}),
	)
	printSchema("Default vector + FTS", withFTS)

	withMetadata := mustSchema(
		chroma.WithDefaultVectorIndex(chroma.NewVectorIndexConfig(
			chroma.WithSpace(chroma.SpaceCosine),
		)),
		chroma.WithStringIndex("category"),
		chroma.WithIntIndex("year"),
		chroma.WithFloatIndex("rating"),
		chroma.WithBoolIndex("published"),
	)
	printSchema("Metadata indexes", withMetadata)

	withDisabledField := mustSchema(
		chroma.WithDefaultVectorIndex(chroma.NewVectorIndexConfig(
			chroma.WithSpace(chroma.SpaceCosine),
		)),
		chroma.DisableStringIndex("large_text_field"),
	)
	printSchema("Disable index for one field", withDisabledField)

	withSpann := mustSchema(
		chroma.WithDefaultVectorIndex(chroma.NewVectorIndexConfig(
			chroma.WithSpace(chroma.SpaceCosine),
			chroma.WithSpann(chroma.NewSpannConfig(
				chroma.WithSpannSearchNprobe(64),
				chroma.WithSpannEfConstruction(200),
			)),
		)),
	)
	printSchema("SPANN (Chroma Cloud)", withSpann)
}
