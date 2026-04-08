# Phase 20: GetOrCreateCollection contentEF Support - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-07
**Phase:** 20-getorcreatecollection-contentef-support
**Areas discussed:** Auto-wiring for existing collections, Embedded "get" path contentEF forwarding, CreateCollection contentEF state storage, Validation and conflict detection, Config persistence for contentEF, HTTP CreateCollection contentEF wiring

---

## Auto-wiring for existing collections

| Option | Description | Selected |
|--------|-------------|----------|
| User-provided only (Recommended) | Match Python/JS: no auto-wiring from server config. Only use the explicitly provided contentEF (or nil if none given). Consistent across all 3 SDKs. | ✓ |
| Auto-wire from config | Diverge from Python/JS by auto-wiring from server config for existing collections. More convenient but breaks cross-SDK behavioral consistency. | |
| You decide | Claude picks based on SDK consistency analysis. | |

**User's choice:** User-provided only
**Notes:** User requested SDK research before deciding. Parallel research into Python and JS Chroma clients confirmed neither auto-wires in getOrCreateCollection — only getCollection does.

---

## Embedded "get" path contentEF forwarding

| Option | Description | Selected |
|--------|-------------|----------|
| Forward contentEF same as denseEF (Recommended) | Add contentEF forwarding alongside existing denseEF forwarding. Keeps the embedded path internally consistent. | ✓ |
| Restructure to match Python/JS | Refactor embedded GetOrCreateCollection to NOT call GetCollection internally. Larger refactor, breaks existing embedded behavior. | |
| You decide | Claude picks based on minimal-change principle. | |

**User's choice:** Forward contentEF same as denseEF
**Notes:** None

---

## CreateCollection contentEF state storage

| Option | Description | Selected |
|--------|-------------|----------|
| Same as denseEF — ignore for existing (Recommended) | Set overrideContentEF=nil for existing collections, matching current denseEF behavior. | ✓ |
| Store user's contentEF in state | Override existing state with the user-provided contentEF even for existing collections. | |
| You decide | Claude picks based on consistency with denseEF pattern. | |

**User's choice:** Same as denseEF — ignore for existing
**Notes:** None

---

## Validation and conflict detection

| Option | Description | Selected |
|--------|-------------|----------|
| No validation (Recommended) | Match current Go client behavior: no EF conflict detection anywhere. | ✓ |
| Basic nil check only | Only validate that contentEF is non-nil when provided via the option. | |
| Full conflict detection | Validate contentEF against server-persisted config (like Python does). | |

**User's choice:** No validation for now
**Notes:** User requested a GH issue be created to track full conflict detection as a future cross-cutting concern, matching Python SDK's validation behavior.

---

## Config persistence for contentEF

| Option | Description | Selected |
|--------|-------------|----------|
| Persist contentEF config (Recommended) | Call SetContentEmbeddingFunction in PrepareAndValidateCollectionRequest when contentEF is provided. Enables future GetCollection auto-wiring. | ✓ |
| DenseEF config only | Only persist the denseEF config. ContentEF is in-memory only at create time. | |
| You decide | Claude picks based on round-trip consistency. | |

**User's choice:** Persist contentEF config
**Notes:** None

---

## HTTP CreateCollection contentEF wiring

| Option | Description | Selected |
|--------|-------------|----------|
| Mirror denseEF pattern (Recommended) | Add wrapContentEFCloseOnce(req.contentEmbeddingFunction) to CollectionImpl constructor. | ✓ |
| Conditional wiring | Only set contentEmbeddingFunction on CollectionImpl when non-nil. | |
| You decide | Claude picks the approach that matches the established pattern. | |

**User's choice:** Mirror denseEF pattern
**Notes:** None

---

## Claude's Discretion

- Test structure and file organization
- Whether to add contentEF wiring to embedded CreateCollection's isNewCreation=true path
- Internal helper decomposition

## Deferred Ideas

- Full EF conflict detection across all collection operations (create GH issue)
- Embedded GetOrCreateCollection restructuring to match Python/JS single-call pattern
