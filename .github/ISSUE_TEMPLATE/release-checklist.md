---
name: Release Checklist
about: Checklist for preparing a new release
title: 'Release v[VERSION]'
labels: 'release'
assignees: ''
---

## Release v[VERSION]

**Target Release Date**: [DATE]

**Type**: [ ] Major [ ] Minor [ ] Patch

---

### Pre-Release Checklist

#### Code Quality
- [ ] `make lint` passes without errors
- [ ] `make build` completes successfully
- [ ] All linting issues fixed with `make lint-fix` if needed

#### Testing
- [ ] V1 API tests pass (`make test`)
- [ ] V2 API tests pass (`make test-v2`)
- [ ] Embedding function tests pass (`make test-ef`) or CI verified
- [ ] Reranking function tests pass (`make test-rf`) or CI verified
- [ ] Cloud tests pass (`make test-cloud`) or CI verified
- [ ] All GitHub Actions workflows are green on main

#### Documentation
- [ ] README.md updated (if needed)
  - [ ] Installation instructions current
  - [ ] Examples working and up-to-date
  - [ ] Supported versions accurate
- [ ] Documentation site reviewed (if applicable)
- [ ] CLAUDE.md updated (if architecture/guidelines changed)

#### Dependencies
- [ ] Dependencies reviewed for vulnerabilities
- [ ] Available dependency updates checked
- [ ] If dependencies updated, tests verified

#### Compatibility
- [ ] Chroma version compatibility verified
- [ ] Go version compatibility verified
- [ ] Compatibility statements in README accurate

#### Breaking Changes (Major Releases Only)
- [ ] All breaking changes documented
- [ ] Migration guide created
- [ ] Examples updated for new API

---

### Release Process

- [ ] Version references updated in code/docs
- [ ] All tests passing on main branch
- [ ] No pending security alerts
- [ ] Related issues closed or moved to next milestone
- [ ] Tag created and pushed: `v[VERSION]`
- [ ] GitHub Release created (manual or automated)
- [ ] Release notes reviewed and published

---

### Post-Release

- [ ] GitHub Release verified
- [ ] Go module accessible: `go list -m github.com/amikos-tech/chroma-go@v[VERSION]`
- [ ] pkg.go.dev updated (https://pkg.go.dev/github.com/amikos-tech/chroma-go)
- [ ] Release announced (Discord/Slack/social media)
- [ ] Milestone closed (if using milestones)
- [ ] Project board updated (if using project boards)

---

### Notes

<!-- Add any specific notes about this release -->

---

### Release Highlights

<!-- Summarize the key features, fixes, and changes in this release -->

**Features**:
-

**Bug Fixes**:
-

**Breaking Changes**:
-

---

See [RELEASE.md](../../RELEASE.md) for the complete release procedure.
