---
phase: 260729-p70
verified: 2026-07-29T19:20:00Z
status: passed
score: 7/7 must-haves verified
overrides_applied: 0
---

# Quick Task 260729-p70: chroma-go-local v0.3.5 Bump Verification Report

**Task Goal:** Resolve GitHub issue #512 — `github.com/amikos-tech/chroma-go-local v0.3.4`
fails `go.sum` verification on any cold Go module cache, breaking `Smoke (windows-latest)`
and `Smoke (macos-latest)`.

**Verified:** 2026-07-29
**Commit under test:** `0187c56` on `fix/512-chroma-go-local-v0.3.5-bump`
**Status:** passed

## Pre-confirmed by Orchestrator (not re-checked)

- Annotated tag `v0.3.5` exists upstream, `.object.type == "tag"`, targets `ad3a5aa68f0434670bb8cc39811e69930f40ea70`
- `v0.3.4` ref still `41423e7d5d25eb80ecc86145c693469e2fa57639` (untouched)
- `sum.golang.org` notarized v0.3.5: `h1:F/vk7Nc6eC8tVFaj9O5XAkfBYGGQj5At6ccrlNxbzvU=`
- Root cold-cache `go test -run '^$' ./...` → exit 0
- Commit `0187c56` touches exactly 20 files, all `.mod`/`.sum`, 30 ins / 30 del
- `defaultLocalLibraryVersion` / `localLibraryCosignMainIdentityVersion` still `"v0.3.4"`
- No suppression tokens in the commit diff

## Independent Re-Verification (this pass)

### 1. Nine example modules — genuine cold-cache build

Ran with a freshly created, verified-empty `GOMODCACHE` (`entries before: 0`) and
`GOFLAGS=-mod=readonly`, using `go build -o /dev/null ./...` per instruction (avoids stray
binaries):

```
COLD=$(mktemp -d)   # entries before: 0
for d in basic array_metadata embedding_function_basic auth persistent_client \
         schema metadata_filters search tenant_and_db; do
  ( cd "examples/v2/$d" && GOMODCACHE="$COLD" GOFLAGS=-mod=readonly go build -o /dev/null ./... )
done
```

Result: exit 0 for all 9. The first module (`basic`) genuinely downloaded from the network
cold, including `go: downloading github.com/amikos-tech/chroma-go-local v0.3.5` — confirming
the go.sum entries verify against an unseeded module cache, the exact failure mode from
issue #512.

Pin confirmed in all 9 `go.mod` files:

```
basic: github.com/amikos-tech/chroma-go-local v0.3.5 // indirect
array_metadata: github.com/amikos-tech/chroma-go-local v0.3.5 // indirect
embedding_function_basic: github.com/amikos-tech/chroma-go-local v0.3.5 // indirect
auth: github.com/amikos-tech/chroma-go-local v0.3.5 // indirect
persistent_client: github.com/amikos-tech/chroma-go-local v0.3.5 // indirect
schema: github.com/amikos-tech/chroma-go-local v0.3.5 // indirect
metadata_filters: github.com/amikos-tech/chroma-go-local v0.3.5 // indirect
search: github.com/amikos-tech/chroma-go-local v0.3.5 // indirect
tenant_and_db: github.com/amikos-tech/chroma-go-local v0.3.5 // indirect
```

**Assessment:** The executor's self-reported shell breakage (`${PIPESTATUS[0]}` empty in zsh,
`noclobber` redirect failure) was a diagnostic artifact of *how* the check was run, not a
defect in the fix itself. Re-running the check cleanly from this independent process
reproduces exit 0 across all 9 modules with a verifiably empty cache. No gap.

Also independently re-ran the root module cold-cache check (already confirmed by
orchestrator, reproduced here): `GOMODCACHE=<empty> GOFLAGS=-mod=readonly go test -run '^$' ./...`
→ exit 0, 44 packages ok.

### 2. `make lint`

```
$ make lint
golangci-lint run
0 issues.
```
Clean.

### 3. No stray artifacts

`git status --porcelain` (before any check in this pass):
```
 M .planning/quick/260729-p70-fix-issue-512-chroma-go-local-v0-3-4-go-/260729-p70-PLAN.md
?? .DS_Store
?? .planning/quick/260729-p70-fix-issue-512-chroma-go-local-v0-3-4-go-/260729-p70-SUMMARY.md
```
Exactly the 3 expected entries — matches the focus-area expectation precisely.

`git status --porcelain --ignored` additionally lists only pre-existing gitignored noise
(`.env`, `.idea/`, `.venv/`, coverage/junit outputs, `artifacts/`, etc.) that predates this
task. Specifically checked `examples/v2/persistent_client/chroma_data_local_persistent/`
(mtime `Mar 2 2026`, months before this July 29 2026 task) — pre-existing, not a stray
artifact from this session's `go build` runs.

Searched all 9 example directories for stray executable binaries post cold-cache build
(the exact bug the executor self-reported and claimed to have cleaned up): none found.
`go build -o /dev/null ./...` used in this verification pass produced none by construction;
a `find ... -perm +111` sweep of the 9 dirs also confirms zero stray binaries remain from
the executor's earlier run.

### 4. Scope compliance

`git show --stat HEAD` confirms exactly 20 files changed, 30 insertions / 30 deletions —
`go.mod`, `go.sum`, and 9 example `{go.mod,go.sum}` pairs. No other files.

`git log -1 --name-only HEAD | grep -E 'custom_embedding_function|reranking_function_basic'`
→ no output — confirmed these two out-of-scope example dirs were not touched.

Confirmed still referencing `v0.3.4` (untouched, as required):
- `scripts/offline_bundle/main.go:31` → `defaultLocalShimVersion = "v0.3.4"`
- `scripts/fetch_runtime_deps.sh:10` → `LOCAL_SHIM_VERSION="${CHROMA_LOCAL_SHIM_VERSION:-v0.3.4}"`
- `docs/docs/client.md:107` → references `v0.3.4`
- `docs/go-examples/docs/run-chroma/persistent-client.md:100` → references `v0.3.4`

`pkg/api/v2/client_local_library_download.go` — `git show HEAD -- <file>` is empty (file not
touched by the commit); `defaultLocalLibraryVersion` and `localLibraryCosignMainIdentityVersion`
both still read `"v0.3.4"` in the working tree.

Full commit diff suppression-token scan: `git show HEAD | grep -Ei 'GONOSUMDB|GONOSUMCHECK|GOSUMDB|GOFLAGS|GOPRIVATE'`
→ no matches.

Residual v0.3.4 chroma-go-local references: `git grep -n 'chroma-go-local v0.3.4' -- '*.mod' '*.sum'`
→ no matches.

Root `go.mod`/`go.sum` content directly inspected:
```
go.mod: github.com/amikos-tech/chroma-go-local v0.3.5
go.sum: github.com/amikos-tech/chroma-go-local v0.3.5 h1:F/vk7Nc6eC8tVFaj9O5XAkfBYGGQj5At6ccrlNxbzvU=
go.sum: github.com/amikos-tech/chroma-go-local v0.3.5/go.mod h1:U37BtxfJldfcpiya6nelgy+SdMesYOVub9kcnRknTrk=
```
The `h1:` hash matches the sum.golang.org notarized value the orchestrator independently
confirmed.

### 5. Honesty audit of SUMMARY.md

All claims checked against direct reproduction; no overstated claims found:

- **Deviation #3 (zsh shell breakage)** — SUMMARY explicitly discloses the `${PIPESTATUS[0]}`
  and `noclobber` failures and states the reported exit 0 "comes from that clean run, not from
  an inferred result." This is corroborated: this verification pass independently reproduced
  exit 0 for both root and all 9 examples from a separate, freshly-created cold cache. No
  reason to doubt the claim.
- **Deviation #4 (9 stray binaries from `go build ./...`)** — plausible and consistent with
  Go's default binary-output behavior; verified none remain in the working tree today.
- **"Honest Reporting Notes" on `git status --porcelain` not being literally the 20 tracked
  files** — accurate; the 2 extra entries (`.DS_Store`, `PLAN.md`) match exactly what this
  verification pass also observed, correctly attributed as pre-existing/orchestrator-owned.
- **Suppression-scan false-positive explanation** (matches were inside uncommitted `PLAN.md`
  prose naming the forbidden env vars, not actual usage) — verified: `git show HEAD` (the
  actual committed diff) has zero matches for those tokens.
- No claim in SUMMARY.md was found to overstate what was actually done or verified.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Annotated tag v0.3.5 exists in amikos-tech/chroma-go-local at `ad3a5aa` | VERIFIED (orchestrator) | Pre-confirmed |
| 2 | sum.golang.org has notarized chroma-go-local@v0.3.5 | VERIFIED (orchestrator + this pass) | `h1:F/vk7Nc6eC8tVFaj9O5XAkfBYGGQj5At6ccrlNxbzvU=` matches go.sum |
| 3 | Root module compiles from a completely cold Go module cache, `-mod=readonly` | VERIFIED (reproduced) | `GOMODCACHE=<empty> go test -run '^$' ./...` → exit 0 |
| 4 | All 9 chroma-go-local-dependent example modules resolve and build from a cold module cache | VERIFIED (independently reproduced) | 9/9 `go build -o /dev/null ./...` → exit 0 on fresh empty cache; `basic` shows genuine cold download |
| 5 | No reference to chroma-go-local v0.3.4 remains in any go.mod/go.sum | VERIFIED | `git grep 'chroma-go-local v0.3.4' -- '*.mod' '*.sum'` → empty |
| 6 | No checksum-verification suppression exists anywhere in the repo or CI | VERIFIED | `git show HEAD` grep for suppression tokens → empty; `go env` clean |
| 7 | Native artifact version constants remain pinned at v0.3.4 | VERIFIED | `client_local_library_download.go` untouched by commit; constants still `"v0.3.4"` |

**Score:** 7/7 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `go.mod` | direct requirement on chroma-go-local v0.3.5 | VERIFIED | `github.com/amikos-tech/chroma-go-local v0.3.5` present |
| `go.sum` | notarized h1/go.mod hashes for v0.3.5 | VERIFIED | both hashes present, matching sumdb |
| `pkg/api/v2/client_local_library_download.go` | version constants unchanged | VERIFIED | file untouched by commit; constants at v0.3.4 |
| 9x `examples/v2/*/go.mod` | pin `chroma-go-local v0.3.5 // indirect` | VERIFIED | all 9 confirmed |

### Key Link Verification

| From | To | Via | Status |
|------|-----|-----|--------|
| go.sum | sum.golang.org | cold-cache `go test -run '^$' ./...` under `-mod=readonly` | WIRED — exit 0 |
| examples/v2/*/go.mod | root go.mod via replace directive | transitive indirect resolution | WIRED — all 9 pin v0.3.5 // indirect and build cold |

### Anti-Patterns Found

None. No debt markers, no suppression tokens, no stray artifacts, no out-of-scope files touched.

### Human Verification Required

None — all must-haves are programmatically verifiable and were independently reproduced.

## Gaps Summary

None. All 7 must-have truths hold under independent re-verification, including the two areas
flagged as most at-risk by the reported shell breakage (9-example cold-cache build, and lint).
The self-reported deviations in SUMMARY.md are consistent with what this pass reproduced and
do not indicate any unresolved defect.

---

_Verified: 2026-07-29_
_Verifier: Claude (gsd-verifier)_
