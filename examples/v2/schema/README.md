# Schema Example

Concise copy/paste examples for schema support:

- `NewSchema` and `NewSchemaWithDefaults`
- `WithDefaultFtsIndex` (FTS)
- Metadata indexes (`WithStringIndex`, `WithIntIndex`, `WithFloatIndex`, `WithBoolIndex`)
- Disabling per-field indexes (`DisableStringIndex`)
- SPANN config for Chroma Cloud (`WithSpann`)

## Run

```bash
cd examples/v2/schema
go run .
```
