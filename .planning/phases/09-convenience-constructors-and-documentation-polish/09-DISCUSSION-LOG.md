# Phase 9: Convenience Constructors and Documentation Polish - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-25
**Phase:** 09-convenience-constructors-and-documentation-polish
**Areas discussed:** Return type, Constructor scope, Options support, Doc presentation, Mixed-part helpers, Example updates, ContentOption reuse

---

## Return Type

| Option | Description | Selected |
|--------|-------------|----------|
| Content (Recommended) | Returns full Content{Parts: []Part{...}} — ready for EmbedContent. Maximum verbosity reduction. | ✓ |
| Part | Returns a Part — still needs Content{Parts: []Part{...}} wrapping. Less reduction but more composable. | |
| Both layers | Part-level shorthands AND Content-level shorthands. Two layers of convenience. | |

**User's choice:** Content
**Notes:** One-liner usage: `ef.EmbedContent(ctx, embeddings.NewImageFile("photo.png"))`

---

## Constructor Scope

| Option | Description | Selected |
|--------|-------------|----------|
| Minimum 6 + text | Roadmap set plus NewTextContent for symmetry. 7 total. | ✓ |
| Minimum + base64 | Add NewImageBase64, NewVideoBase64 etc. ~11 constructors. | |
| Full matrix | Every modality x source combination. 17 constructors. | |

**User's choice:** Minimum 6 + NewTextContent
**Notes:** Discussed tradeoffs in depth. Base64 is one stdlib call from bytes; bytes is one os.ReadFile from file path. The cases where you genuinely have raw bytes/base64 without a file path or URL are narrow. Ship minimal, expand later if demand warrants.

---

## Options Support

| Option | Description | Selected |
|--------|-------------|----------|
| Functional options | Constructors accept variadic ContentOption: NewImageFile(path, WithIntent(...), WithDimension(...)). | ✓ |
| Plain only | Constructors take just the source. Set Intent/Dimension on returned Content struct directly. | |
| Both (fluent) | Plain constructors plus Content.With() chaining. | |

**User's choice:** Functional options
**Notes:** None

---

## Doc Presentation

| Option | Description | Selected |
|--------|-------------|----------|
| Shorthand-first (Recommended) | Lead with convenience constructors. Verbose forms in reference link. | ✓ |
| Side-by-side | Show both forms together. | |
| Replace verbose entirely | Only show constructors in provider docs. | |

**User's choice:** Shorthand-first
**Notes:** None

---

## Mixed-Part Helpers

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, NewContent(...Part) | Add NewContent(parts []Part, opts ...ContentOption) that wraps parts into Content. | ✓ |
| No, use struct literal | Users compose mixed-part via Content{Parts: []Part{...}} as today. | |

**User's choice:** Yes, NewContent with slice signature
**Notes:** Discussed variadic vs slice vs marker interface approaches. Marker interface pattern is NOT present in the repo (checked). Slice approach chosen for consistency with existing codebase conventions. The `[]Part{}` wrapping is slightly clunky but idiomatic.

---

## Example Updates

| Option | Description | Selected |
|--------|-------------|----------|
| Rewrite existing (Recommended) | Update Phase 8 examples in-place to use convenience constructors. | ✓ |
| Keep existing, add new | Leave Phase 8 examples as-is, add new simplified examples. | |
| Rewrite + comment verbose | Rewrite but include commented-out verbose forms. | |

**User's choice:** Rewrite existing
**Notes:** None

---

## ContentOption Reuse

| Option | Description | Selected |
|--------|-------------|----------|
| Constructors only (Recommended) | ContentOption works only as constructor parameter. Manual Content{} uses direct field assignment. | ✓ |
| Universal via Apply | Add Apply(opts ...ContentOption) method on Content for universal use. | |

**User's choice:** Constructors only
**Notes:** None

---

## Claude's Discretion

- File placement for new constructors
- Exact ContentOption type definition
- WithProviderHints parameter design
- Test structure and assertion patterns

## Deferred Ideas

- Base64/Bytes convenience constructors — expand later if demand warrants
- Content.Apply() method — deferred to keep scope minimal
