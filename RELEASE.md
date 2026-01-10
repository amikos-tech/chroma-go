# Release Procedure

This document outlines the comprehensive release procedure for chroma-go. Follow these steps carefully to ensure a successful release.

## Prerequisites

Before starting the release process, ensure you have:

- [ ] Write access to the repository
- [ ] Git configured with your credentials
- [ ] GitHub CLI (`gh`) installed and authenticated (optional but recommended)
- [ ] All PRs for the release are merged to `main`
- [ ] Understanding of [Semantic Versioning](https://semver.org/)

## Version Numbering

This project follows [Semantic Versioning 2.0.0](https://semver.org/):

- **MAJOR** version (x.0.0): Incompatible API changes
- **MINOR** version (0.x.0): New functionality in a backward compatible manner
- **PATCH** version (0.0.x): Backward compatible bug fixes

### Determining the Next Version

Review commits since the last release to determine version bump:

```bash
# View commits since last release (or from start if no releases)
git log --oneline --decorate

# If you have tags, view since last tag
git log $(git describe --tags --abbrev=0)..HEAD --oneline
```

Look for:
- **MAJOR**: Breaking changes, `BREAKING CHANGE:` in commit footer, or `!` after type/scope
- **MINOR**: `feat:` commits (new features)
- **PATCH**: `fix:` commits (bug fixes)

## Pre-Release Checklist

### 1. Code Quality Verification

- [ ] **Lint the codebase**: Ensure no linting errors
  ```bash
  make lint
  ```
  If issues found, fix them:
  ```bash
  make lint-fix
  ```

- [ ] **Build verification**: Ensure the code builds successfully
  ```bash
  make build
  ```

### 2. Comprehensive Testing

Run all test suites to ensure everything passes:

- [ ] **V1 API Tests**
  ```bash
  make test
  ```

- [ ] **V2 API Tests**
  ```bash
  make test-v2
  ```

- [ ] **Embedding Functions Tests** (requires API keys in environment)
  ```bash
  make test-ef
  ```

- [ ] **Reranking Functions Tests** (requires API keys in environment)
  ```bash
  make test-rf
  ```

- [ ] **Cloud Tests** (requires Chroma Cloud credentials)
  ```bash
  make test-cloud
  ```

**Note**: If you don't have credentials for cloud/embedding/reranking tests, ensure CI has passed for the latest commits.

### 3. Documentation Review

- [ ] **Update README.md** if needed:
  - [ ] Installation instructions are current
  - [ ] Examples are working and up-to-date
  - [ ] Supported Chroma versions are accurate
  - [ ] API documentation links are valid

- [ ] **Check documentation site** (if applicable):
  - [ ] Browse docs at the documentation URL
  - [ ] Verify all examples compile and run
  - [ ] Check for broken links

- [ ] **Review CLAUDE.md** for any needed updates:
  - [ ] Architecture changes
  - [ ] New build tags
  - [ ] Environment variables
  - [ ] Development guidelines

### 4. Dependency Audit

- [ ] **Review dependencies** for known vulnerabilities:
  ```bash
  go list -m all
  ```

- [ ] **Check for available updates**:
  ```bash
  go list -u -m all
  ```

- [ ] **If dependencies updated**, ensure tests still pass

### 5. Compatibility Verification

- [ ] **Verify Chroma version compatibility**:
  - Check `.github/workflows/go.yml` for tested Chroma versions
  - Current range: 0.4.8 to 1.3.3
  - Update if new Chroma versions should be supported

- [ ] **Go version compatibility**:
  - Check `go.mod` for minimum Go version (currently 1.24)
  - Ensure compatibility statement in README is accurate

### 6. Breaking Changes Assessment

If this is a MAJOR version release:

- [ ] **Document all breaking changes**:
  - List API changes
  - Provide migration guide
  - Update examples to reflect new API

- [ ] **Update V1/V2 API compatibility notes** if applicable

## Release Process

### 7. Create Release Branch (Optional for Major Releases)

For major releases or if you want to prepare release commits separately:

```bash
git checkout -b release/v<VERSION>
```

### 8. Update Version References

- [ ] **Check for hardcoded versions** in:
  - [ ] README.md
  - [ ] Documentation
  - [ ] Examples
  - [ ] Test fixtures

### 9. Final Verification

- [ ] **All tests passing on main branch**:
  - Check GitHub Actions workflows
  - Verify all jobs are green

- [ ] **No pending security alerts**:
  - Check GitHub Security tab
  - Review Dependabot alerts

- [ ] **All related issues closed or moved to next milestone**

### 10. Create and Push Tag

Create an annotated tag with release notes summary:

```bash
# Ensure you're on main and up to date
git checkout main
git pull origin main

# Create annotated tag (recommended)
git tag -a v<VERSION> -m "Release v<VERSION>

Summary of changes:
- Feature 1
- Feature 2
- Bug fix 1
"

# Push the tag
git push origin v<VERSION>
```

**Important**: Use semantic versioning format: `v<MAJOR>.<MINOR>.<PATCH>` (e.g., `v0.1.0`, `v1.2.3`)

### 11. Create GitHub Release

#### Option A: Manual (Current Method)

1. Go to [GitHub Releases](https://github.com/amikos-tech/chroma-go/releases)
2. Click "Draft a new release"
3. Select the tag you just created
4. Click "Generate release notes" to auto-generate from PRs
5. Edit the release notes:
   - Add a summary at the top
   - Organize changes by category:
     - 🚀 Features
     - 🐛 Bug Fixes
     - 📚 Documentation
     - 🔧 Maintenance
     - ⚠️ Breaking Changes (if any)
   - Highlight important changes
   - Add upgrade instructions if needed
6. If this is a pre-release, check "This is a pre-release"
7. Click "Publish release"

#### Option B: Automated (If workflow is configured)

The GitHub Actions workflow will automatically create a release when a tag is pushed.

### 12. Verify Release

- [ ] **GitHub Release is published**:
  - Visit the releases page
  - Verify release notes are correct
  - Check that the tag is properly linked

- [ ] **Go module is accessible**:
  ```bash
  # Test that the new version can be fetched
  go list -m github.com/amikos-tech/chroma-go@v<VERSION>
  ```

- [ ] **pkg.go.dev is updated**:
  - Visit https://pkg.go.dev/github.com/amikos-tech/chroma-go
  - Refresh the page or wait a few minutes
  - Verify the new version appears

### 13. Post-Release Tasks

- [ ] **Announce the release**:
  - [ ] Post in project Discord/Slack (if applicable)
  - [ ] Update social media (if applicable)
  - [ ] Notify key stakeholders

- [ ] **Create milestone for next release** (if using milestones)

- [ ] **Close the current milestone** (if using milestones)

- [ ] **Update project board** (if using project boards)

## Rollback Procedure

If critical issues are discovered after release:

### Option 1: Quick Patch Release

1. Fix the issue on main
2. Follow the release procedure with a PATCH version
3. Mark the problematic release as "Pre-release" on GitHub

### Option 2: Delete Tag (Use with caution)

**Warning**: Only do this immediately after release and if no one has started using it.

```bash
# Delete local tag
git tag -d v<VERSION>

# Delete remote tag
git push --delete origin v<VERSION>

# Delete GitHub release via UI
```

### Option 3: Yank Release

For Go modules, you can retract a version:

1. Add to `go.mod`:
   ```go
   retract v<VERSION> // Brief reason for retraction
   ```
2. Create a new PATCH release with the retraction

## Release Automation

This project uses GitHub Actions for automated release note generation. The workflow is triggered when a tag matching `v*` is pushed.

See `.github/workflows/release.yml` for the workflow configuration.

## Troubleshooting

### Tag already exists

```bash
# If you need to move a tag (use carefully)
git tag -d v<VERSION>
git push --delete origin v<VERSION>
# Then recreate the tag
```

### pkg.go.dev not updating

- Wait 15-30 minutes for indexing
- Manually request indexing at https://pkg.go.dev/github.com/amikos-tech/chroma-go?tab=versions

### GitHub release notes not generated

- Ensure commits follow conventional commit format
- Check that PRs have proper titles and descriptions
- Manually edit the release notes to add missing information

## Best Practices

1. **Release regularly**: Small, frequent releases are better than large, infrequent ones
2. **Follow conventional commits**: This enables automated changelog generation
3. **Test thoroughly**: Never skip the testing phase
4. **Document breaking changes**: Always provide migration guides
5. **Keep CLAUDE.md updated**: This helps future development
6. **Communicate clearly**: Good release notes help users understand changes

## Resources

- [Semantic Versioning](https://semver.org/)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [Go Modules Reference](https://go.dev/ref/mod)
- [GitHub Releases Documentation](https://docs.github.com/en/repositories/releasing-projects-on-github)

## Questions?

If you encounter issues or have questions about the release process, please:
1. Check the troubleshooting section
2. Review previous releases for examples
3. Ask in the team communication channel
4. Open an issue for documentation improvements
