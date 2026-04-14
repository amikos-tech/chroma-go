---
phase: 26
slug: twelve-labs-async-embedding
status: verified
threats_open: 0
asvs_level: 1
created: 2026-04-14
---

# Phase 26 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail for the Twelve Labs async embedding feature.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| caller → TwelveLabsClient | Public SDK surface; caller-supplied media URLs already validated upstream. | `WithAsyncPolling(maxWait time.Duration)` opts into async path. |
| Twelve Labs API → client | Server response bodies (JSON tasks API) are untrusted text flowing into error messages and logs. | `_id`, `status`, raw JSON bytes preserved in `TaskResponse.FailureDetail`. |
| caller's ctx → polling loop | Caller-controlled deadline / cancellation bounds the loop. | `context.Context`. |
| `maxWait` config → polling loop | Caller-tunable SDK-side bound, independent of ctx. | `time.Duration`. |
| saved config map → FromConfig | Persistent config can round-trip numerics through JSON (int64 → float64). | `async_polling: bool`, `async_max_wait_ms: int64`. |

---

## Threat Register

| Threat ID | Category | Component | Disposition | Mitigation | Status |
|-----------|----------|-----------|-------------|------------|--------|
| T-26-01 | I (Info Disclosure) | `doTaskPost` / `doTaskGet` error paths | mitigate | `chttp.SanitizeErrorBody` on every raw-body error path — `pkg/embeddings/twelvelabs/twelvelabs.go:293,295,336,338` | closed |
| T-26-02 | T (Tampering) | response body bloat | mitigate | `chttp.ReadLimitedBody` (200MB cap at `pkg/commons/http/utils.go:12,28`) — `twelvelabs.go:286,329` | closed |
| T-26-03 | D (Denial of Service) | unbounded response size | mitigate | Same `ReadLimitedBody` cap at `twelvelabs.go:286,329` | closed |
| T-26-04 | S (Spoofing) | task ID path injection | mitigate | `url.PathEscape(taskID)` at `twelvelabs.go:314` + empty-ID guard at `:311-313` | closed |
| T-26-05 | I (Info Disclosure) | API key in logs | accept | API key only set via `x-api-key` header at `twelvelabs.go:275,319`; never appears in error wrappers. See Accepted Risks AR-26-01. | closed |
| T-26-06 | I (Info Disclosure) | `pollTask` failed-status reason | mitigate | `chttp.SanitizeErrorBody(resp.FailureDetail)` at `twelvelabs_async.go:96`; raw body preserved at `twelvelabs.go:348` | closed |
| T-26-07 | D (Denial of Service) | unbounded polling loop | mitigate | Three independent bounds — `ctx.Done` at `twelvelabs_async.go:113`, `sdkMaxWaitDeadline` at `:62`/`:103-106`, `asyncPollCap=60s` at `twelvelabs.go:73` | closed |
| T-26-08 | D (Denial of Service) | timer leak | mitigate | `time.NewTimer` at `twelvelabs_async.go:111` + `timer.Stop()` on ctx branch at `:114`; no `time.After` in file | closed |
| T-26-09 | T (Tampering) | malformed status field | mitigate | D-16 default branch rejecting unknown status with `"unexpected status %q"` at `twelvelabs_async.go:99-101` | closed |
| T-26-10 | R (Repudiation) | indistinguishable timeout errors | mitigate | D-20 distinct messages — `"async polling maxWait %s exceeded"` at `:79,105`, `"async polling deadline exceeded"` at `:82,118`, `"async polling canceled"` at `:85,120`; `errors.Wrap` preserves `errors.Is` | closed |
| T-26-11 | T (Tampering) | negative maxWait | mitigate | `WithAsyncPolling` rejects `maxWait < 0` at `option.go:112-114` | closed |
| T-26-12 | D (Denial of Service) | missing maxWait | mitigate | `WithAsyncPolling(0)` → `30 * time.Minute` default at `option.go:116-118` | closed |
| T-26-13 | T (Tampering) | config type mismatch on reload | mitigate | `embeddings.ConfigInt` coerces int/int64/float64 at `twelvelabs.go:473`; broken-value path leaves async OFF (`:467-476`) | closed |
| T-26-14 | I (Info Disclosure) | config leakage | accept | `GetConfig` emits only `async_polling` (bool) + `async_max_wait_ms` (int64 ms) at `twelvelabs.go:438-441`; no secrets, no PII. See AR-26-02. | closed |
| T-26-15 | T (Tampering) | fixture drift `id` vs `_id` | mitigate | Test fixtures emit `_id` at `twelvelabs_test.go:365-374`; regression surfaces as loud 404 against `/tasks/` | closed |
| T-26-16 | D (Denial of Service) | slow tests | mitigate | `asyncPollInitial=1ms`, `asyncPollCap=10ms`, `asyncMaxWait=5s` in `newTestAsyncEF` at `twelvelabs_test.go:344,346` | closed |
| T-26-17 | R (Repudiation) | indistinguishable timeout errors in tests | mitigate | `TestTwelveLabsAsyncMaxWait` asserts `stderrors.Is(err, context.DeadlineExceeded) == false` at `twelvelabs_test.go:523`; `TestTwelveLabsAsyncBlockedHTTPMaxWait` mirrors at `:642` | closed |

*Status: open · closed*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-26-01 | T-26-05 | API key never appears in `errors.Errorf` / `errors.Wrap` messages on any of the four HTTP helpers (`doPost`, `doTaskPost`, `doTaskGet`, plus existing sync paths). The key only flows through the `x-api-key` request header. The convention is enforced by code review — adding a new helper that prints the key would be visible in PR diff. Mitigation cost (e.g., redacting the entire transport layer) outweighs the residual risk for a SDK that already requires the caller to handle their own credential storage. | gsd-security-auditor + author | 2026-04-14 |
| AR-26-02 | T-26-14 | The two emitted config keys are a boolean flag and an integer millisecond duration. Neither contains secrets, PII, URLs, model identifiers, or any other sensitive material. They are safe to persist in registries that round-trip through JSON. The accept disposition is the appropriate outcome — there is no information to disclose. | gsd-security-auditor + author | 2026-04-14 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-04-14 | 17 | 17 | 0 | gsd-security-auditor (sonnet) |

### Audit notes (2026-04-14)

- Bonus tests beyond the original register strengthen coverage: `TestTwelveLabsAsyncFailedReasonSanitized` (T-26-06), `TestTwelveLabsAsyncBlockedHTTPMaxWait` (T-26-07/T-26-17), `TestTwelveLabsAsyncTaskCreateError*` regression set (T-26-01), `TestTwelveLabsAsyncFusedRejected` (validation gate before any HTTP fires).
- T-26-13 was hardened during the WR-01/WR-02 review cycle: a malformed `async_max_wait_ms` value with `async_polling=true` now leaves async OFF rather than silently falling back to `WithAsyncPolling(0)` — registry corruption no longer triggers a hidden 30-minute default. Encoded at `twelvelabs.go:467-476`.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-04-14
