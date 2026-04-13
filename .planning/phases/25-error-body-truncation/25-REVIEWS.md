---
phase: 25
reviewers: [gemini, claude]
reviewed_at: 2026-04-13T04:34:21Z
plans_reviewed: [25-01-PLAN.md]
---

# Cross-AI Plan Review — Phase 25

## Gemini Review

This is a high-quality, comprehensive implementation plan for Phase 25. It correctly identifies the core issue—large raw HTTP bodies leaking into error messages—and provides a surgical, centralized solution that respects the project's architectural boundaries.

### Summary
The plan for Phase 25 is well-structured and focuses on centralization and safety. By introducing a shared `SanitizeErrorBody` utility in `pkg/commons/http`, the plan eliminates duplication and ensures a consistent error-reporting contract across all embedding providers. The emphasis on TDD and UTF-8 safe truncation by rune count (rather than byte count) demonstrates a mature understanding of Go library development. The plan's scope is appropriately limited to embedding providers, avoiding unnecessary refactoring of the core HTTP client or rerankers.

### Strengths
- **Centralization & Normalization:** Correctly identifies that `Perplexity` and `OpenRouter` have local helpers that should be migrated to the shared `chttp` utility to prevent future drift.
- **Separation of Concerns:** Explicitly preserves the distinction between transport-level safety (`ReadLimitedBody` / 200MB) and display-level safety (truncation for readability).
- **UTF-8 Safety:** The requirement to truncate by rune count is essential for a global SDK and is explicitly addressed in the plan and test strategy.
- **Comprehensive Audit:** The plan doesn't just rely on a simple grep; it includes a manual audit target list that catches edge cases like `Bedrock` (which uses a different variable name for the response body).
- **TDD-First Approach:** Starting with red tests for the new `[truncated]` suffix ensures that existing behaviors are effectively challenged and updated to the new standard.

### Concerns
- **Suffix Formatting (LOW):** The requirement specifies a `[truncated]` suffix. It should be clarified if a leading space is intended (e.g., `"message [truncated]"` vs `"message[truncated]"`) to ensure consistency across all providers.
- **Sanitization of Parsed Errors (LOW):** In `OpenRouter`, the plan mentions keeping the JSON parse but routing the final string through the sanitizer. If the parsed error message itself is excessively long, this is good. If it is short but the raw body is long, the sanitizer should be applied to the raw body fallback only. The plan seems to handle this, but care should be taken during implementation.

### Suggestions
- **Space before suffix:** I recommend the helper implementation appends ` [truncated]` (with a leading space) to ensure the suffix doesn't merge into the final word of the truncated body, improving readability in logs.
- **Helper Export:** While the plan suggests not exporting the 512-rune limit unless needed, exporting it as a constant in `pkg/commons/http` might be useful for providers that want to perform their own pre-sanitization logic while staying consistent with the SDK's standards.

### Risk Assessment
**LOW**
The plan is highly surgical and stays strictly within the requested scope. By pinning the shared helper with unit tests first, the risk of regressions in individual providers is minimized. The impact is limited to error message formatting, which has no effect on the primary data path of the SDK.

### Recommendation
**Proceed with 25-01-PLAN.md.** It is a solid, production-grade plan that fulfills all requirements and addresses the identified research pitfalls.

---

## Claude Review

Now I have everything I need for a thorough review. Let me compile the findings.

---

# Phase 25 Plan Review: Error Body Truncation (25-01-PLAN.md)

## Summary

This is a well-researched, well-structured plan for a breadth-heavy migration. The shared helper design is sound, the task ordering is correct (pin contract first, then migrate, then sweep), and the scope boundaries are clearly drawn. The plan correctly identifies all 19 in-scope providers and handles the three special cases (Cloudflare structured errors, OpenRouter JSON parsing, Twelve Labs structured parsing) explicitly. The main risks are minor: a few edge cases in specific providers that the plan handles implicitly but could call out more explicitly, and a TDD verification gap where the RED phase can't actually fail for the shared helper since it doesn't exist yet.

## Strengths

1. **Correct task ordering.** Task 1 pins the contract and normalizes the two local precedents before Task 2 touches the remaining 17 providers. This prevents mixed-suffix drift during execution.

2. **Complete provider audit.** The plan lists all 19 HTTP-based embedding providers. My grep confirms the list is exhaustive — no provider is missing. The `gemini` provider uses the Google SDK (no raw HTTP), `bm25` and `ort` are local-only, and `default_ef` doesn't have embedding-level HTTP error paths. All correctly excluded.

3. **Scope discipline.** The plan explicitly excludes rerankings, `client_http.go`, and provider error wording redesign. The `<threat_model>` is proportionate to the actual risk level.

4. **Suffix contract change is explicit.** The plan calls out that `...(truncated)` → `[truncated]` is a semantics change, not just centralization. Test updates are included in Task 1, not deferred.

5. **Representative test strategy is appropriate.** Adding full regression tests to all 19 providers would be scope creep. Shared helper tests + 2 representative provider regressions (OpenAI, Baseten) is the right coverage shape for a display-layer change.

6. **`read_first` gates are thorough.** Each task lists the specific files the executor must read before editing, including the research doc and validation doc.

7. **All providers already import `chttp`.** My audit confirms every in-scope provider already has the `chttp` import alias, so the migration is purely replacing `string(respData)` with `chttp.SanitizeErrorBody(respData)` — no new imports needed.

## Concerns

### HIGH

None.

### MEDIUM

**M1: TDD RED phase for the shared helper is structurally impossible.**
Task 1 is marked `tdd="true"` and Step 1 says "make the contract fail against the current code." But `SanitizeErrorBody` doesn't exist yet — the tests will fail with a **compile error**, not a behavior-RED. This is technically still "RED" in TDD parlance (tests fail), but the executor might be confused about whether to commit a compile-broken state. The plan should clarify that RED here means "tests are written and fail to compile because the function doesn't exist yet" — which is fine, just not the classic "test compiles but assertion fails" RED.

**M2: Twelve Labs has a structured-then-raw pattern the plan doesn't call out explicitly.**
Lines 183-186 of `twelvelabs.go` show: if JSON parsing succeeds and `apiErr.Message` is non-empty, the structured message is used (line 184); otherwise raw `string(respData)` is the fallback (line 186). The plan's Task 2 action says to replace `string(respData)` patterns, which would correctly catch line 186's fallback. But line 184's `apiErr.Message` is also body-derived text — if a malicious API returns a 10KB `message` field, it would bypass sanitization. The plan should decide: sanitize `apiErr.Message` too, or accept that structured error messages are trusted.

**M3: Cloudflare's error format appends raw body after structured `embeddings.Errors`.**
The plan correctly notes this ("preserve `embeddings.Errors` exactly as-is and sanitize only the raw body segment"). But the current code uses `\n%v` between the structured errors and the raw body (line 157). After migration it would be `\n` + `chttp.SanitizeErrorBody(respData)`. This changes the error message format slightly — the truncated suffix would now appear at the end of the Cloudflare error. Any existing tests asserting on Cloudflare error message format should be checked. No Cloudflare-specific test update is planned.

### LOW

**L1: The display limit (512 runes) is not exported.**
The plan says "do not export or expose it unless tests truly require that." This is fine for now, but if a downstream caller ever needs to know the limit (e.g., for log formatting decisions), they'd need a code change. Acceptable for this phase — flagging only for awareness.

**L2: OpenRouter's `parseAPIError` structured path also returns body-derived text.**
When JSON parsing succeeds, `apiErr.Error.Message` is returned directly (line 197-198 of openrouter.go). Like the Twelve Labs concern (M2), this structured message is body-derived but not sanitized. The plan says "keep the JSON parse if `apiErr.Error.Message` is present" — this is a reasonable design choice (structured errors are typically short), but it's worth noting that it creates a code path that bypasses `SanitizeErrorBody`.

**L3: No explicit handling of `nil` body in the plan's `<interfaces>` block.**
The recommended helper shape says "return `\"\"` for nil/empty input" which is correct. But the plan's TDD test table says "nil or empty body returns empty string" — the executor should make sure `SanitizeErrorBody(nil)` doesn't panic, which is trivial in Go (`string(nil)` → `""`), but worth a test case.

## Suggestions

1. **Clarify TDD RED semantics in Task 1.** Add a note: "RED means the new test functions are written but `SanitizeErrorBody` does not exist yet, so compilation fails. This is the expected RED state — commit the test updates to Perplexity and OpenRouter (which do compile against existing local helpers with the wrong suffix), then proceed to GREEN."

2. **Decide on structured-message sanitization for Twelve Labs and OpenRouter.** Add a one-line note to Task 2's action: "Structured error messages from JSON parsing (e.g., `apiErr.Message` in Twelve Labs, `apiErr.Error.Message` in OpenRouter) are not sanitized — they are trusted as short, provider-formatted messages. Only raw-body fallback paths use `SanitizeErrorBody`." This makes the design choice explicit rather than implicit.

3. **Add Cloudflare to the representative test check.** The Cloudflare error format is unique (`embeddings.Errors` + `\n` + raw body). If there's an existing Cloudflare test asserting on error format, it should be checked after migration. If there isn't one, no action needed — but the executor should verify.

4. **Consider adding `MaxErrorDisplayRunes` as an exported constant.** Not strictly needed, but it would let the Perplexity and OpenRouter tests reference the shared limit instead of hardcoding `512` in test assertions. This avoids silent test breakage if the limit ever changes.

## Risk Assessment

**Overall Risk: LOW**

Justification:
- The change is purely at the display layer — no transport, authentication, or query semantics are affected
- All providers already import the target package (`chttp`)
- The migration is mechanical: replace `string(respData)` with `chttp.SanitizeErrorBody(respData)` in each provider
- The two most complex cases (OpenRouter and Perplexity) are handled first in Task 1, de-risking the rest
- The plan has clear scope boundaries and doesn't attempt to redesign provider error formats
- Test coverage is proportionate — shared helper suite + 2 representative providers + suffix migration in existing tests
- The only semantic change (`...(truncated)` → `[truncated]`) is explicitly called out and tested

The plan is ready for execution. The medium concerns (M1-M3) are worth noting in execution but don't require plan revision — they're implementation details the executor can handle in-line.

---

## Consensus Summary

Both reviewers consider `25-01-PLAN.md` execution-ready and overall low risk. They agree the plan is intentionally narrow, keeps the change at the display layer, and uses a sensible shared-helper-first migration strategy. Neither reviewer asked for a plan rewrite before execution.

### Agreed Strengths

- Centralizing error-body sanitization in `pkg/commons/http` is the right design and correctly removes local drift between Perplexity and OpenRouter.
- The plan preserves the transport/display split by leaving `ReadLimitedBody(...)` and `MaxResponseBodySize` alone while adding a display-layer sanitizer.
- The provider audit, UTF-8-safe truncation requirement, and representative test strategy are proportionate to the scope and keep the migration focused.

### Agreed Concerns

- Decide explicitly whether structured JSON error-message fields are trusted as-is or should also be passed through `SanitizeErrorBody(...)`. Both reviewers flagged this for OpenRouter, and Claude identified the same shape in Twelve Labs.
- Lock down the exact sanitization contract before implementation so execution does not drift on details such as suffix formatting and the RED/green sequencing around introducing a brand-new helper.
- Verify special-case provider error formatting after migration, especially flows that combine parsed errors with raw-body fallbacks, so the shared sanitizer does not introduce unnoticed message-shape regressions.

### Divergent Views

- Gemini recommends appending ` [truncated]` with a leading space and is mildly in favor of exporting the display-limit constant for reuse.
- Claude treats constant export as optional and focuses instead on clarifying the compile-fail RED step, checking Cloudflare's mixed structured/raw formatting, and documenting the structured-message policy explicitly.

### Consensus Concerns

- Structured parsed error messages need an explicit policy: sanitize them too, or document that only raw-body fallback paths are sanitized.
- The sanitization contract should be stated precisely enough that every provider lands on the same suffix and sequencing behavior.
- Provider-specific message shapes such as Cloudflare's combined structured/raw errors should be spot-checked after migration.
