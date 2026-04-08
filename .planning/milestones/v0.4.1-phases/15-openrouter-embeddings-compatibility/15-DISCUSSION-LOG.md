# Phase 15: OpenRouter Embeddings Compatibility - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-03-30
**Phase:** 15-openrouter-embeddings-compatibility
**Areas discussed:** Model validation strategy, Provider preferences struct, Request field wiring, Config round-trip scope

---

## Architecture (pivotal decision)

User redirected the original approach of extending the OpenAI provider. Instead of adding OpenRouter fields to `pkg/embeddings/openai`, the decision is to create a standalone `pkg/embeddings/openrouter/` provider.

**User's rationale:** "I don't like adding this much OpenRouter specific things in the OpenAI provider. Let's just keep the OpenAI provider small and targeted to only OpenAI compatible interfaces."

This reshaped all subsequent gray areas.

---

## Model Validation Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Skip validation for custom base URL | Keep strict validation for default OpenAI URL, skip for custom base URLs. Add WithModelString bypass. | |
| Accept any string always | Remove all model validation from WithModel | |
| Add WithModelString bypass only | Keep WithModel strict, add new WithModelString(string) | |

After architecture pivot, question was reformulated:

| Option | Description | Selected |
|--------|-------------|----------|
| Leave OpenAI strict | No changes to OpenAI provider at all | |
| Still relax OpenAI model validation | Add WithModelString for OpenAI-compatible endpoints (Azure, LiteLLM, etc.) | ✓ |

**User's choice:** Add `WithModelString` to OpenAI provider for any OpenAI-compatible endpoint
**Notes:** Useful beyond just OpenRouter — Azure, LiteLLM, vLLM users also benefit

---

## Provider Preferences Struct

| Option | Description | Selected |
|--------|-------------|----------|
| Typed struct + extras map | Typed fields for documented prefs + Extras map for forward-compat | ✓ |
| Pure map[string]any | Maximum flexibility, zero type safety | |

**User's choice:** Typed struct with extras map
**Notes:** None

---

## Request Field Wiring

Resolved implicitly by the standalone provider decision — all OpenRouter fields (encoding_format, input_type, provider) are constructor-time `With*` options on the new OpenRouter provider.

---

## Config Round-Trip

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, full round-trip | Register as "openrouter" in dense registry with full GetConfig/FromConfig | ✓ |
| Constructor-only, no registry | No registry participation | |

**User's choice:** Full round-trip via registry
**Notes:** Consistent with how Gemini, Voyage, etc. work

---

## Claude's Discretion

- HTTP client setup and error handling patterns
- Test organization within the new package

## Deferred Ideas

None
