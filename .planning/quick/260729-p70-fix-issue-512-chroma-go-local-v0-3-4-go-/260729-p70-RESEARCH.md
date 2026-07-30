# Quick Task 260729-p70: fix issue #512 — chroma-go-local v0.3.4 go.sum mismatch — Research

**Researched:** 2026-07-29
**Domain:** Go module checksum/notarization, upstream release pipeline (GitHub Actions + cosign + R2)
**Confidence:** HIGH (all findings verified by file:line or live command output)

> **Superseded (commit `f801d35`, PR #514):** the "leave the version constants alone" conclusion
> below (Q1, and rows 5-6 in the constants table) was later overridden — `defaultLocalLibraryVersion`
> and `localLibraryCosignMainIdentityVersion` were intentionally bumped to `v0.3.5` (the latter is
> now a slice covering both versions). See `.planning/STATE.md` for the current, accurate state.

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Cut `v0.3.5` in `amikos-tech/chroma-go-local` at **exactly commit `ad3a5aa68f0434670bb8cc39811e69930f40ea70`**. Do NOT re-tag `v0.3.4`.
- Bump scope: root `go.mod` + `go.sum`, **and** all 9 `examples/v2/*/go.mod` + `go.sum` files.
- Prevention: **fix only**. No CI guards, no cold-cache job, no unrelated hardening in this repo.

### Claude's Discretion
- Exact mechanism for cutting the upstream release (workflow dispatch vs. manual tag push).
- Whether `go mod tidy` or a targeted `go get` regenerates hashes, provided the result is a legitimate re-resolution and not a suppression.

### Hard Constraints (non-negotiable)
- No `go.sum` line deletion, no `GONOSUMDB`/`GONOSUMCHECK`/`GOSUMDB=off`/`GOFLAGS=-mod=mod` anywhere in repo or CI.
</user_constraints>

---

## Q1 — Does bumping the Go module to v0.3.5 require published release ARTIFACTS at v0.3.5?

**Answer: NO — not if you leave the version constants alone. The runtime artifact version is pinned by a Go constant, not by the module version.**

This is the load-bearing finding. The `debug.ReadBuildInfo()` derivation exists in the code but is **unreachable through the public API**.

### The exact trace

`resolveLocalLibraryPath` decides the download version at `pkg/api/v2/client_local_library_download.go:92-108`:

```go
version, err := normalizeLocalLibraryTag(cfg.libraryVersion)   // :92
...
if version == "" {                                             // :96
    detectedVersion, detectErr := localDetectLibraryVersionFunc()  // :97  <- ReadBuildInfo path
    ...
}
if version == "" {                                             // :106
    version = defaultLocalLibraryVersion                       // :107
}
```

`cfg.libraryVersion` is **never empty** for any client built through the public constructor, because `defaultLocalClientConfig()` pre-populates it:

- `pkg/api/v2/client_local.go:123` — `libraryVersion: defaultLocalLibraryVersion,`
- `pkg/api/v2/client_local_library_download.go:29` — `defaultLocalLibraryVersion = "v0.3.4"`

Therefore the `:97` branch (`detectLocalLibraryVersion`, which reads `debug.ReadBuildInfo().Deps` looking for `localLibraryModulePath` at `:136-151`) is dead code in production. It is reachable only from tests that hand-construct `localClientConfig{}` with an empty `libraryVersion`, and from `WithPersistentLibraryVersion(...)` overrides (`client_local.go:397-412`) which set it explicitly anyway.

Confirmed by grep — the only non-test callers of `localDetectLibraryVersionFunc` are inside `client_local_library_download.go` itself.

### Consequences

| Change | Requires v0.3.5 release artifacts? |
|--------|-----------------------------------|
| Bump `go.mod` / `go.sum` to v0.3.5 only | **No.** Runtime keeps downloading `https://releases.amikos.tech/chroma-go-local/v0.3.4/...`, which is live (verified 200 below). |
| Also bump `defaultLocalLibraryVersion` to `"v0.3.5"` | **Yes.** Archives + `SHA256SUMS` + cosign bundle must exist at `v0.3.5` or every persistent client 404s at first use. |

**This de-risks the plan substantially.** A bare Git tag at `ad3a5aa` is sufficient to fix issue #512. The Go source at `v0.3.5` is byte-identical to `v0.3.4` (same commit), so the shim ABI it links against is identical — pinning the native artifact at `v0.3.4` while the module says `v0.3.5` is functionally correct, not a mismatch.

### Live R2 state (verified 2026-07-29)

```
https://releases.amikos.tech/chroma-go-local/v0.3.4/SHA256SUMS                  200
https://releases.amikos.tech/chroma-go-local/v0.3.4/SHA256SUMS.sigstore.json    200
https://releases.amikos.tech/chroma-go-local/v0.3.4/SHA256SUMS.sig              404
https://releases.amikos.tech/chroma-go-local/v0.3.4/SHA256SUMS.pem              404
https://releases.amikos.tech/chroma-go-local/v0.3.4/chroma-go-local-v0.3.4-darwin-arm64.tar.gz  200
https://releases.amikos.tech/chroma-go-local/v0.3.5/SHA256SUMS                  404
```

Because `.sig`/`.pem` are absent, `localPrepareLegacySignedChecksumsFromBase` (`:616`) fails for v0.3.4 and the code falls through to `localPrepareSigstoreBundleChecksumsFromBase` (`:655`). That fallback is wired at `:576-585`. This is the working path today.

### The cosign identity trap (HIGH severity for the planner)

I decoded the Fulcio cert embedded in the live `v0.3.4/SHA256SUMS.sigstore.json`:

```
X509v3 Subject Alternative Name: critical
    URI:https://github.com/amikos-tech/chroma-go-local/.github/workflows/release.yml@refs/heads/main
1.3.6.1.4.1.57264.1.2:  workflow_dispatch
1.3.6.1.4.1.57264.1.3:  a6830175d5b118796421b1da8f50d9a38355befd
```

The v0.3.4 artifacts were signed by a **`workflow_dispatch` run on `refs/heads/main`**, not by a tag push. That is precisely why `localLibraryCosignMainIdentityVersion = "v0.3.4"` exists at `client_local_library_download.go:42` — it is a narrowly-scoped back-compat allowance consumed by `localAllowedChecksumSignerIdentities` (`:685-691`).

That constant is a **single string, not a set**. Two implications:

1. **Do NOT change `localLibraryCosignMainIdentityVersion`.** Changing it to `"v0.3.5"` silently revokes acceptance of the v0.3.4 artifacts that every existing user is pulling.
2. If the planner ever does want v0.3.5 artifacts, they **must** be produced by a **tag push** (identity `…release.yml@refs/tags/v0.3.5`, matching `localLibraryCosignIdentityTemplate` at `:40`), never by `workflow_dispatch` from `main`.

**Recommendation:** minimal path. Bump `go.mod`/`go.sum` only. Leave `defaultLocalLibraryVersion`, `localLibraryCosignMainIdentityVersion`, `scripts/offline_bundle/main.go:31`, and `scripts/fetch_runtime_deps.sh:10` at `v0.3.4`. No v0.3.5 artifacts required.

---

## Q2 — What the upstream release process actually does

Source: `.github/workflows/release.yml` @ `ad3a5aa` (fetched via `gh api`, 423 lines; saved to scratchpad).

### Triggers (`:3-12`)
```yaml
on:
  push:
    tags: ["v*"]
  workflow_dispatch:
    inputs:
      release_tag: { required: true, type: string }
```
Both. A `git push origin v0.3.5` **will** trigger it. The workflow file that executes on a tag push is the one at the tagged commit — i.e. the `ad3a5aa` version.

### What it produces
- 3 jobs build `chroma-shim-{linux,macos,windows}-{amd64,arm64}.tar.gz`; `Normalize artifact names` (`:216-240`) renames them to `chroma-go-local-<TAG>-<os>-<arch>.tar.gz` (macos→darwin).
- `build-java-artifacts` produces 3 jars.
- `Build checksum manifest` (`:242-254`) → `sha256sum` over all `*.tar.gz *.jar` → `SHA256SUMS`.
- `Sign and verify artifacts` (`:262-308`) → cosign v3.0.5, `sign-blob --bundle <f>.sigstore.json --use-signing-config --output-signature <f>.sig --output-certificate <f>.pem`, identity `https://github.com/${{ github.workflow_ref }}`.
- `Upload artifacts to R2` (`:310-389`) → `aws s3 cp dist/ s3://releases/chroma-go-local/<TAG>/ --recursive`, plus `latest.json`, `releases.json` (+ signatures), then CDN purge.
- `Publish GitHub release` (`:411-423`) → `softprops/action-gh-release`, `overwrite_files: true`.

Note the `ad3a5aa` version **still emits `.sig`/`.pem`** and uploads them (`:284-285`, `:299-300`, `:422-423`).

### Does it exist at `ad3a5aa`? Yes. **Is it functional? Demonstrably NO.**

Run history for `release.yml` (`gh api .../workflows/release.yml/runs`):

```
2026-03-19T11:06:43Z  event=workflow_dispatch  branch=main     sha=a6830175  concl=success   <- what actually shipped v0.3.4
2026-03-19T10:02:08Z  event=push               branch=v0.3.4   sha=ad3a5aa6  concl=failure
2026-03-19T08:33:40Z  event=push               branch=v0.3.4   sha=5cdd82e4  concl=failure
2026-03-02T22:58:51Z  event=push               branch=v0.3.3   sha=165f8acd  concl=success
```

This is also the complete provenance of the re-tag: `v0.3.4` first pointed at `5cdd82e4` (release failed at `Sign and verify artifacts`), the fix commit `ad3a5aa` ("migrate release signing to cosign v3 bundles") landed, and the tag was **re-cut onto `ad3a5aa`** — which is exactly what burned the sumdb hash.

Step-level detail for the `ad3a5aa` run (id `23289602361`):

```
7  Install cosign                => success
8  Sign and verify artifacts     => success
9  Upload artifacts to R2        => failure     (annotation: "Process completed with exit code 255")
10 Purge release metadata        => skipped
11 Publish GitHub release        => skipped
```

Exit 255 is the AWS CLI's generic error code. Job logs are expired (HTTP 410), so the precise cause is unrecoverable. `scripts/build_releases_index.sh` **does** exist at `ad3a5aa`, so a missing-script explanation is ruled out.

The follow-up fix commit `a6830175` ("fix(ci): drop detached cosign outputs from releases") is **1 commit ahead of `ad3a5aa`** and touches only the `--output-signature`/`--output-certificate` flags and the `.sig`/`.pem` upload lines — **it changes nothing in the R2 step**. Since the very next `workflow_dispatch` run (on `a6830175`) succeeded at R2 with an unchanged R2 step, the most probable explanation is a transient/credential-config issue that was resolved between 10:02 and 11:06. `[MEDIUM confidence — inferred, logs expired]`

### Bottom line for Q2

Pushing tag `v0.3.5` at `ad3a5aa` will fire a release run using a workflow known to have failed once at the R2 step.

- Under the **Q1-recommended minimal plan, this does not matter** — no v0.3.5 artifacts are needed. A red workflow run is cosmetic noise. It can be cancelled immediately after the tag push if desired.
- If artifacts ARE wanted, tag push is still the **only** acceptable mechanism (identity `refs/tags/v0.3.5`). `workflow_dispatch` would sign as `refs/heads/main`, which the Go client rejects for anything but `v0.3.4` (see Q1 trap).

---

## Q3 — Exact mechanics of the bump in this repo

### Prerequisite: warm the module proxy first

Current proxy state (verified):
```
GET https://proxy.golang.org/github.com/amikos-tech/chroma-go-local/@v/list        -> lists v0.3.4
GET https://proxy.golang.org/github.com/amikos-tech/chroma-go-local/@v/v0.3.4.info -> 404
GET https://proxy.golang.org/github.com/amikos-tech/chroma-go-local/@latest         -> "not found: "
GET https://proxy.golang.org/github.com/amikos-tech/chroma-go-local/@v/v0.3.5.info -> 404
GET https://sum.golang.org/lookup/github.com/amikos-tech/chroma-go-local@v0.3.4    -> 200, h1:NUCaw0R+... (frozen, matches go.sum:60)
```

Note `@latest` is also broken — a secondary symptom worth confirming resolves after v0.3.5 lands.

Do NOT run `go get` in the repo before the proxy has indexed v0.3.5. If `proxy.golang.org` 404s, `GOPROXY=...,direct` falls through to a raw GitHub fetch and the resulting hash never round-trips through the notary in a controlled way.

Force indexing from **outside** the repo (this is a plain GET; the proxy fetches from origin on first miss):
```bash
curl -sS -o /dev/null -w '%{http_code}\n' \
  https://proxy.golang.org/github.com/amikos-tech/chroma-go-local/@v/v0.3.5.info
curl -sS https://proxy.golang.org/github.com/amikos-tech/chroma-go-local/@latest
```
First call typically returns 404 while the fetch is in flight. Poll until 200 — normally **seconds to ~2 minutes**; allow up to 10. Then confirm notarization:
```bash
curl -sS https://sum.golang.org/lookup/github.com/amikos-tech/chroma-go-local@v0.3.5
```
Only proceed once that returns 200 with `h1:` lines. **Once it does, `v0.3.5` is frozen forever — the tag must never be moved again.**

### Root module

```bash
go get github.com/amikos-tech/chroma-go-local@v0.3.5
go mod tidy
```

`go get` rewrites `go.mod:8` and appends the new `go.sum` entries but leaves the stale `v0.3.4` lines (`go.sum:60-61`). Those stale lines are harmless (go.sum is an allowlist) but should be pruned so the repo greps clean. `go mod tidy` prunes them.

**Is `go mod tidy` safe?** Yes. `go mod tidy` never upgrades anything — it adds missing requirements and removes unused ones at their currently-selected versions. The module graph here is already tidy (CI lints it). Expected diff is exactly: `go.mod` one line, `go.sum` two lines removed / two added. Anything wider is a red flag and should be reverted.

### The 9 example modules

Every example carries `replace github.com/amikos-tech/chroma-go => ../../../` (e.g. `examples/v2/basic/go.mod:5`; `examples/v2/auth/go.mod:30`). `chroma-go-local` reaches them **transitively through the replaced parent**, pinned `// indirect`.

Therefore, once the root `go.mod` is bumped, each example needs only:

```bash
for d in basic array_metadata embedding_function_basic auth persistent_client \
         schema metadata_filters search tenant_and_db; do
  ( cd "examples/v2/$d" && go mod tidy )
done
```

No `go get` in the example dirs — `go mod tidy` reads the new version straight from the replaced parent's `go.mod`. Order matters: root first.

Do **not** loop over `examples/v2/*/`. `custom_embedding_function` and `reranking_function_basic` do not depend on `chroma-go-local`; touching them is out of the locked scope.

There is **no `go.work`** at the repo root (verified), so each example resolves independently — which is exactly why they each need their own tidy.

### Verification (cold-cache reproduction of the CI failure, non-mutating)

```bash
COLD=$(mktemp -d)
GOMODCACHE="$COLD" GOFLAGS=-mod=readonly go test -run '^$' ./...
rm -rf "$COLD"
```
This is the honest reproduction of `Smoke (windows-latest)` / `Smoke (macos-latest)`, whose failing step is `Compile all packages` → `go test -run '^$' ./...` (`.github/workflows/go.yml:178-179`).

Note: examples are **not** built by any CI job (grep of `.github/workflows/` for `examples` returns nothing), so the example bumps are user-facing correctness, not CI-gating.

---

## Q4 — Collateral `v0.3.4` references

Exhaustive grep over `*.go`, `*.mod`, `*.sum`, `*.md`, `*.yml`, `*.yaml`, `*.json`, `*.sh`, `Makefile`. Beware false positives: `googleapis/enterprise-certificate-proxy v0.3.4`, `client9/misspell v0.3.4`, `golang.org/x/text v0.3.4` are unrelated modules that happen to share the version string.

| # | Location | Verdict | Reasoning |
|---|----------|---------|-----------|
| 1 | `go.mod:8` | **MUST** | The fix. |
| 2 | `go.sum:60-61` | **MUST** | The poisoned hashes. |
| 3 | `examples/v2/{basic,array_metadata,embedding_function_basic,auth,persistent_client,schema,metadata_filters,search,tenant_and_db}/go.mod` — `// indirect` line | **MUST** | Locked scope; each fails identically on cold cache. |
| 4 | Same 9 dirs, `go.sum:17-18` | **MUST** | Same poisoned hashes. |
| 5 | `pkg/api/v2/client_local_library_download.go:29` `defaultLocalLibraryVersion = "v0.3.4"` | **LEAVE** (see Q1) | Pins the native artifact version. Bumping it makes v0.3.5 release artifacts a hard runtime requirement. v0.3.4 artifacts are live and byte-compatible. |
| 6 | `pkg/api/v2/client_local_library_download.go:42` `localLibraryCosignMainIdentityVersion = "v0.3.4"` | **MUST NOT CHANGE** | Single-valued back-compat allowance for the `refs/heads/main`-signed v0.3.4 bundle. Changing it revokes verification of the artifacts users pull today. |
| 7 | `scripts/offline_bundle/main.go:31` `defaultLocalShimVersion = "v0.3.4"` | **LEAVE** | Native-artifact default; must stay consistent with #5. |
| 8 | `scripts/fetch_runtime_deps.sh:10` `LOCAL_SHIM_VERSION="${CHROMA_LOCAL_SHIM_VERSION:-v0.3.4}"` | **LEAVE** | Same; must stay consistent with #5 and #7. |
| 9 | `scripts/offline_bundle/main_test.go:339` R2 SHA256SUMS URL | **IRRELEVANT** | Test asserts `applyGitHubAuthHeader` adds no header for non-GitHub hosts. Only the hostname is load-bearing; the version is arbitrary string filler. |
| 10 | `pkg/api/v2/client_local_library_download_test.go:1479,1482` cosign identity for `v0.3.4` | **LEAVE** | Directly asserts the #6 special case. Must continue to assert `v0.3.4`. |
| 11 | `pkg/api/v2/client_local_crosslang_integration_test.go:146` comment `// chroma-go-local v0.3.4 supports Chroma 1.5.5.` | **SHOULD** | Stale after the module bump. One-word comment edit; zero behavioural impact. Purely cosmetic — planner may drop it to keep the diff minimal. |
| 12 | `docs/docs/client.md:107` `WithPersistentLibraryVersion("v0.3.4")` / "default `v0.3.4`" | **LEAVE** | Documents #5, which is not changing. Would become *wrong* if edited. |
| 13 | `docs/go-examples/docs/run-chroma/persistent-client.md:100` "default `v0.3.4`" | **LEAVE** | Same as #12. |
| 14 | `pkg/api/v2/client_local_test.go:174` | **IRRELEVANT** | References the constant symbolically (`defaultLocalLibraryVersion`), not the literal. |
| 15 | `.planning/**` (CONTEXT.md, milestone v0.4.1 phase 14 docs) | **IRRELEVANT** | Historical planning records. |

**No parity assertion exists** between the Go module version and `defaultLocalLibraryVersion` (verified by grep) — nothing breaks when they diverge.

---

## Assumptions Log

| # | Claim | Section | Risk if wrong |
|---|-------|---------|---------------|
| A1 | The `ad3a5aa` release run's R2 failure (exit 255) was transient/credential-config, since `a6830175` did not touch the R2 step yet succeeded 64 min later. | Q2 | Only matters if the plan needs v0.3.5 artifacts. Under the recommended minimal plan (Q1), a failed release run is cosmetic. Job logs are expired (410) so this cannot be verified. |
| A2 | `proxy.golang.org` will index `v0.3.5` within ~2 min of the tag push. | Q3 | Just a longer wait; the polling loop handles it. |

## Open Questions

1. **Should a v0.3.5 release also be published for artifact-version parity?**
   - Known: not required (Q1); v0.3.4 artifacts are live, signed, and content-identical.
   - Unclear: whether the maintainer wants module/artifact versions to track.
   - Recommendation: **no** — it converts a 2-line dependency fix into a release-pipeline exercise on a workflow with a known failure, and risks the `localLibraryCosignMainIdentityVersion` trap. Matches the locked "fix only" decision.

2. **`proxy.golang.org/@latest` currently returns "not found"** for `chroma-go-local`. Expected to self-heal once v0.3.5 is indexed; worth a post-fix confirmation but not a blocker.

## Sources

**HIGH confidence (direct command output / file:line)**
- `pkg/api/v2/client_local_library_download.go` (lines 29, 40, 42, 92-108, 126-153, 576-585, 616, 655, 685-691)
- `pkg/api/v2/client_local.go` (lines 102, 123, 397-412)
- `gh api repos/amikos-tech/chroma-go-local/{contents,releases,tags,actions/...}`
- OpenSSL decode of live `v0.3.4/SHA256SUMS.sigstore.json` Fulcio certificate
- `curl` HEAD probes against `releases.amikos.tech`, `proxy.golang.org`, `sum.golang.org`
- Repo-wide grep for `v0.3.4`

**MEDIUM confidence**
- Root cause of the `ad3a5aa` R2-step failure (inferred; GH job logs expired, HTTP 410)

**Environment:** `go version go1.26.5 darwin/arm64`; `GOPROXY=https://proxy.golang.org,direct`; `GOSUMDB=sum.golang.org`; `GOFLAGS` empty. No suppression env vars present.

**Research date:** 2026-07-29 · **Valid until:** 2026-08-28 (or until any upstream tag changes)
