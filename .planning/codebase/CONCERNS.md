# Codebase Concerns

**Analysis Date:** 2026-03-18

## Tech Debt

**Embedding contract fragmentation:**
- Issue: dense, sparse, and multimodal contracts are split, while provider-specific task semantics and dimensionality controls are implemented inconsistently across providers
- Files: `pkg/embeddings/embedding.go`, `pkg/embeddings/registry.go`, `pkg/api/v2/configuration.go`, provider packages such as `pkg/embeddings/jina`, `pkg/embeddings/nomic`, `pkg/embeddings/cohere`, `pkg/embeddings/openai`, `pkg/embeddings/roboflow`
- Why: providers were added incrementally with provider-native models/options
- Impact: cross-provider multimodal/intent portability is difficult; config auto-wiring and future feature additions have a high compatibility burden
- Fix approach: introduce a provider-neutral shared contract and map providers to it explicitly

**Deprecated compatibility surface remains large:**
- Issue: multiple deprecated helpers and the legacy `default_ef` alias package remain in active use
- Files: `pkg/embeddings/default_ef/*`, `pkg/embeddings/ort/ort.go`, `pkg/api/v2/collection.go`, `pkg/api/v2/search.go`, `pkg/api/v2/rank.go`
- Why: preserving compatibility for earlier consumers and older docs/examples
- Impact: broader maintenance surface and more paths to regress
- Fix approach: keep compatibility tests, but continue consolidating new work onto the preferred APIs

**Runtime bootstrap logic is spread across packages and scripts:**
- Issue: local runtime download, verification, offline bundle generation, and tokenizer setup are distributed across several packages and scripts
- Files: `pkg/api/v2/client_local.go`, `pkg/api/v2/client_local_library_download.go`, `pkg/internal/downloadutil/*`, `pkg/internal/cosignutil/*`, `scripts/fetch_runtime_deps.sh`, `scripts/offline_bundle/main.go`
- Why: platform/runtime needs expanded over time
- Impact: onboarding and change risk are high for local-runtime work
- Fix approach: keep docs and tests aligned and avoid touching multiple flows without end-to-end validation

## Known Bugs

**Multimodal docs drift from code reality:**
- Symptoms: docs page says multimodal support is not yet available, but `roboflow` already implements text+image multimodal embeddings
- Files: `docs/go-examples/docs/embeddings/multimodal.md`, `pkg/embeddings/roboflow/roboflow.go`
- Trigger: provider support landed without docs fully catching up
- Workaround: read provider code/docs in `docs/docs/embeddings.md` instead of relying on the older example page
- Root cause: documentation and implementation evolved at different speeds

**Delete-heavy local perf path is still unstable enough to be opt-in:**
- Symptoms: delete+reinsert workload is disabled by default in perf docs/config
- Files: `README.md`, `docs/docs/performance-testing.md`, `pkg/api/v2/client_local_perf_helpers_test.go`
- Trigger: local persistent perf or soak runs that include delete-heavy write patterns
- Workaround: leave `CHROMA_PERF_ENABLE_DELETE_REINSERT` unset unless explicitly validating that path
- Root cause: local runtime stability under that workload is still being hardened

## Security Considerations

**Secret sprawl risk from provider-heavy env configuration:**
- Risk: the repo supports many providers, each with its own key/token env vars; docs/examples/scripts make secret handling pervasive
- Files: `pkg/embeddings/*`, `pkg/rerankings/*`, `docs/docs/embeddings.md`, `docs/docs/rerankers.md`, root `.env`
- Current mitigation: persisted configs usually store env-var names instead of secret values
- Recommendations: keep generated docs/maps free of actual credentials and avoid reading or copying `.env` values into committed artifacts

**Insecure transport escape hatches exist for some providers:**
- Risk: providers can allow non-HTTPS or insecure overrides for development/testing flows
- Files: provider packages such as `pkg/embeddings/roboflow/roboflow.go` and shared helpers in `pkg/embeddings/embedding.go`
- Current mitigation: secure defaults and validation in constructors
- Recommendations: call out insecure usage clearly in docs/tests and avoid enabling it in production paths

## Performance Bottlenecks

**Per-item multimodal/provider batching gaps:**
- Problem: some provider implementations still loop item-by-item instead of using a true batch API path
- Files: `pkg/embeddings/roboflow/roboflow.go` (`EmbedDocuments`, `EmbedImages`)
- Cause: simplest provider integration path favored correctness first
- Improvement path: add true batch request support where providers allow it

**Runtime/bootstrap latency for local default embeddings:**
- Problem: first-run local embedding/runtime setup can require downloads, verification, and library discovery
- Files: `pkg/embeddings/default_ef/*`, `pkg/api/v2/client_local_library_download.go`, `scripts/fetch_runtime_deps.sh`
- Cause: native/runtime dependencies are resolved lazily or via bootstrap tooling
- Improvement path: lean on offline bundle flow and cache reuse in CI/dev workflows

## Fragile Areas

**Embedding auto-wiring across config + schema + client modes:**
- Why fragile: behavior crosses `configuration.go`, schema resolution, registry lookup, and multiple client backends
- Common failures: provider config mismatch, env-var naming issues, unsupported provider registrations, version-specific server behavior
- Safe modification: change shared contracts and config tests together; validate HTTP, cloud, and local flows
- Test coverage: decent config tests exist, but cross-provider semantic consistency is still a risk

**Embedded local client state and dimension tracking:**
- Why fragile: embedded runtime must track collection dimension/state while staying compatible with remote collection semantics
- Files: `pkg/api/v2/client_local_embedded.go`, `pkg/api/v2/client_local_embedded_test.go`
- Common failures: dimension mismatch, runtime init/download issues, state drift between operations
- Safe modification: run focused local/client/perf tests and keep fallback paths intact

## Scaling Limits

**CI and provider surface area scale maintenance cost quickly:**
- Current capacity: broad but manageable provider matrix
- Limit: each new provider or shared embedding contract change multiplies config, docs, example, and test obligations
- Symptoms at limit: doc drift, inconsistent task semantics, growing compatibility branches
- Scaling path: centralize provider-neutral contracts and reduce bespoke per-provider branching

## Dependencies at Risk

**External provider APIs and models:**
- Risk: remote embedding/reranking APIs evolve independently, especially around tasks, models, and response formats
- Impact: provider packages and config persistence can drift or break silently
- Migration plan: keep provider packages isolated and cover config round-trip/live behavior where credentials are available

**Deprecated default embedding alias package:**
- Risk: `pkg/embeddings/default_ef` remains intentionally supported but deprecated in favor of `pkg/embeddings/ort`
- Impact: duplicate maintenance and user confusion
- Migration plan: continue steering new code toward `ort` while preserving tests for legacy paths

## Missing Critical Features

**Provider-neutral multimodal foundation:**
- Problem: the shared interface still treats multimodal as “dense + image methods” rather than a generalized modality/intention model
- Current workaround: rely on provider-specific options and the single current multimodal implementation (`roboflow`)
- Blocks: portable multimodal support across providers and cleaner config/capability introspection
- Implementation complexity: medium-high, because it touches shared contracts, registries, persistence, docs, and tests

## Test Coverage Gaps

**Live-provider coverage is opportunistic:**
- What's not tested: many provider behaviors in CI unless secrets are available
- Risk: regressions in task mapping, auth handling, or API compatibility can slip through
- Priority: High for shared embedding contract changes
- Difficulty to test: requires external credentials and provider availability

**Docs/example consistency is lightly enforced:**
- What's not tested: that docs and examples reflect current supported embedding capabilities
- Risk: user confusion and planning mistakes from stale guidance
- Priority: Medium
- Difficulty to test: mostly documentation process rather than code complexity

---

*Concerns audit: 2026-03-18*
*Update as issues are fixed or new risk areas emerge*
