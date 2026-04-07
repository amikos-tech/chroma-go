---
phase: 19
slug: embedded-client-ef-lifecycle-hardening
status: secured
threats_open: 0
asvs_level: 1
created: 2026-04-06
---

# Phase 19 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

No new trust boundaries introduced. This phase hardens internal resource lifecycle management (mutex usage, close ordering, error handling, structured logging) without adding new inputs, outputs, or authentication paths. The logger is an internal dependency injection point, not an external input.

---

## Threat Register

| Threat ID | Category | Component | Disposition | Mitigation | Status |
|-----------|----------|-----------|-------------|------------|--------|
| T-19-01 | D (Denial of Service) | embeddedLocalClient.Close() | mitigate | Close EFs outside mutex using copy-under-lock pattern (line 666-676 of client_local_embedded.go) | closed |
| T-19-02 | T (Tampering) | N/A | accept | No new inputs; all changes are internal lifecycle management. No external data flows affected. | closed |
| T-19-03 | I (Information Disclosure) | WithPersistentLogger | accept | Logger receives collection names and error messages only — no PII, no secrets. Collection names are already logged to stderr in current code. | closed |

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-19-01 | T-19-02 | No new inputs or external data flows — tampering not applicable to internal lifecycle changes | GSD | 2026-04-06 |
| AR-19-02 | T-19-03 | Logger receives only collection names (already public via stderr) and Go error messages — no sensitive data exposure | GSD | 2026-04-06 |

---

## Security Audit 2026-04-06

| Metric | Count |
|--------|-------|
| Threats found | 3 |
| Closed | 3 |
| Open | 0 |

### Verification Evidence

**T-19-01 (DoS — Close blocking):** Verified at `client_local_embedded.go:666-676`. Close() copies collectionState map under write lock, clears the map, releases the lock, then iterates the copy to close EFs outside the lock. Slow EF teardown cannot block concurrent GetCollection/DeleteCollection operations.

**T-19-02 (Tampering — accepted):** No new trust boundaries, inputs, or external data flows. All changes are internal mutex ordering, close-once wrapping, and error path improvements.

**T-19-03 (Info Disclosure — accepted):** WithPersistentLogger receives collection names (already logged to stderr) and Go error messages. No PII, credentials, or sensitive data reaches the logger. Standard observability pattern matching the HTTP client.
