# Release Procedure Setup Skill

**Category**: Project Setup, DevOps, Automation
**Languages**: Go (primary), adaptable to other languages
**Maturity**: Production-ready

## Description

This skill sets up a comprehensive, production-ready release procedure for software projects. It creates documentation, automation workflows, helper scripts, and establishes best practices for version management using Semantic Versioning and Conventional Commits.

## When to Use This Skill

Invoke this skill when:
- A project needs a formal release process
- Currently using manual or ad-hoc releases
- Want to implement automated release notes
- Need version management best practices
- Team wants standardized commit messages
- Moving from "just create a tag" to a robust release workflow

## What This Skill Creates

### Documentation
1. **RELEASE.md** - Comprehensive release checklist with:
   - Pre-release verification steps
   - Testing requirements
   - Version numbering guidelines
   - Step-by-step release process
   - Rollback procedures
   - Troubleshooting guide

2. **CONVENTIONAL_COMMITS.md** - Guide for standardized commits:
   - Commit message format and examples
   - Type definitions and version bump rules
   - Best practices and validation
   - Tool usage (Commitizen)

3. **CHANGELOG.md** - Template following Keep a Changelog format

### Automation
4. **Release GitHub Actions Workflow** - Automated pipeline that:
   - Triggers on version tag push
   - Runs quality checks (tests, linting)
   - Generates categorized release notes from commits
   - Creates GitHub releases automatically
   - Handles pre-releases
   - Triggers package registry updates

5. **Release Preparation Script** - Interactive bash script that:
   - Analyzes commit history
   - Suggests appropriate version bump (MAJOR/MINOR/PATCH)
   - Runs pre-release checks
   - Validates working directory state
   - Creates and pushes tags with confirmation

### Configuration
6. **Commitizen Config** (.cz.toml) - For interactive commits and versioning
7. **Release Issue Template** - GitHub issue template for tracking releases

## Usage Instructions

When the user invokes this skill, follow these steps:

### 1. Analyze Project Structure

First, gather information about the project:

```bash
# Check if it's a git repository
git status

# Identify project type
ls go.mod package.json Cargo.toml setup.py build.gradle pom.xml

# Check existing release infrastructure
ls .github/workflows/*release* RELEASE* CHANGELOG* 2>/dev/null || echo "No existing release files"

# Check for existing tags
git tag --sort=-v:refname | head -5

# Check recent commits for conventional commit usage
git log --oneline -20
```

### 2. Confirm Project Details

Ask the user to confirm:
- **Project type** (Go, Node.js, Rust, Python, etc.)
- **Current versioning approach** (if any)
- **Test command** (e.g., `make test`, `go test ./...`, `npm test`)
- **Lint command** (e.g., `make lint`, `golangci-lint run`)
- **Build command** (e.g., `make build`, `go build ./...`)
- **Package registry** (pkg.go.dev, npm, crates.io, PyPI, etc.)
- **CI/CD platform** (GitHub Actions, GitLab CI, CircleCI, etc.)

### 3. Create Release Documentation

Create `RELEASE.md` with sections:
- Prerequisites checklist
- Version numbering guide (Semantic Versioning)
- Pre-release checklist adapted to project:
  - Code quality verification (lint command)
  - Test suite execution (test command)
  - Documentation review
  - Dependency audit
  - Compatibility verification
- Step-by-step release process
- Post-release tasks
- Rollback procedures
- Troubleshooting

**Customization Notes**:
- Adapt test commands to project's test structure
- Include environment variable requirements if needed
- Add language-specific considerations (e.g., Go module proxy, npm registry)

### 4. Create Conventional Commits Guide

Create `docs/CONVENTIONAL_COMMITS.md` or `CONVENTIONAL_COMMITS.md` with:
- Commit message format explanation
- Type definitions (feat, fix, docs, etc.)
- Scope examples relevant to the project
- Breaking change notation
- Practical examples from the project domain
- Validation tools and setup

### 5. Create Automation Workflow

For **GitHub Actions** (most common):

Create `.github/workflows/release.yml` that:
- Triggers on tags matching version pattern (e.g., `v*`)
- Runs project-specific quality checks
- Generates release notes by parsing conventional commits
- Categorizes changes (Features, Bug Fixes, Breaking Changes, etc.)
- Creates GitHub release
- Marks as pre-release if version contains `-alpha`, `-beta`, `-rc`
- Triggers package registry updates

**Template Structure**:
```yaml
name: Release
on:
  push:
    tags: ['v*']
permissions:
  contents: write
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - Checkout with full history
      - Setup language environment
      - Run tests and lint
      - Extract version from tag
      - Generate categorized release notes
      - Create GitHub release
      - Trigger package registry update
```

For **other CI platforms**, provide equivalent configuration.

### 6. Create Helper Scripts

Create `scripts/prepare-release.sh` (or equivalent for Windows: `prepare-release.ps1`):

**Features**:
- Check git repository state
- Analyze commits since last release
- Count conventional commit types (feat, fix, BREAKING CHANGE)
- Suggest version bump based on Semantic Versioning
- Prompt for version confirmation
- Run pre-release checks:
  - Linting
  - Building
  - Testing (with skip option)
- Generate release summary
- Create annotated git tag
- Push to remote with confirmation

**Customization**:
- Use project-specific test/lint/build commands
- Add language-specific checks
- Include documentation generation if applicable

### 7. Create Configuration Files

**Commitizen Configuration**:
Create `.cz.toml` (Python Commitizen) or `commitizen.config.js` (Node.js):
- Define commit types
- Set version bump rules
- Configure changelog generation
- Add project-specific scopes

**GitHub Issue Template**:
Create `.github/ISSUE_TEMPLATE/release-checklist.md`:
- Pre-filled release checklist
- Version and date placeholders
- Category checkboxes
- Links to release documentation

### 8. Create Changelog Template

Create `CHANGELOG.md` following Keep a Changelog format:
- Version sections with dates
- Categories: Added, Changed, Deprecated, Removed, Fixed, Security
- Unreleased section for ongoing work

### 9. Update Existing Documentation

If `README.md` or `CONTRIBUTING.md` exist:
- Add link to RELEASE.md in README
- Reference conventional commits in CONTRIBUTING.md
- Update installation instructions if needed

### 10. Validate and Test

Before completing:
```bash
# Validate workflow syntax
yamllint .github/workflows/release.yml || echo "Install yamllint to validate"

# Test script is executable
chmod +x scripts/prepare-release.sh
./scripts/prepare-release.sh --help

# Verify all files created
ls -la RELEASE.md CHANGELOG.md .cz.toml .github/workflows/release.yml
```

## Language-Specific Adaptations

### Go Projects
- Use `go.mod` for version detection
- Reference pkg.go.dev in docs
- Include `go list -m` verification
- Test across multiple Go versions if applicable

### Node.js Projects
- Update `package.json` version
- Use `npm version` integration
- Reference npm registry
- Include `npm publish` workflow steps

### Rust Projects
- Update `Cargo.toml` version
- Reference crates.io
- Include `cargo publish` steps
- Document `cargo release` tool

### Python Projects
- Update `setup.py` or `pyproject.toml` version
- Reference PyPI
- Include `twine upload` steps
- Document `bump2version` tool

## Best Practices to Follow

1. **Version Numbering**: Always use Semantic Versioning (MAJOR.MINOR.PATCH)
2. **Conventional Commits**: Explain benefits clearly (automation, clarity)
3. **Testing**: Never skip tests in the checklist
4. **Rollback**: Always include rollback procedures
5. **Automation**: Prefer automated workflows over manual steps
6. **Documentation**: Keep it concise but comprehensive
7. **Accessibility**: Make the process accessible to all team members

## Common Pitfalls to Avoid

- Don't make the process too complex for small projects
- Don't require tools that aren't commonly available
- Don't skip the "why" - explain the benefits
- Don't forget platform-specific instructions (Windows vs Unix)
- Don't hardcode repository-specific values - use placeholders

## User Interaction

After creating all files:

1. **Summarize what was created** with file paths
2. **Explain the workflow**:
   - How to make releases going forward
   - How to use the preparation script
   - How automation works
3. **Provide quick start**:
   ```bash
   # Test the new release process
   ./scripts/prepare-release.sh --help

   # For next release
   ./scripts/prepare-release.sh
   ```
4. **Recommend next steps**:
   - Install Commitizen (optional)
   - Update README to link release docs
   - Create a test release (if appropriate)
   - Train team on conventional commits

## Example Invocation

User: "Can you set up a release procedure for this project?"

Claude response:
1. Analyzes project structure
2. Asks for confirmation of project details
3. Creates all necessary files
4. Commits the changes
5. Provides usage instructions

## Output Example

```
✅ Release Procedure Setup Complete!

Created:
- RELEASE.md (comprehensive release checklist)
- docs/CONVENTIONAL_COMMITS.md (commit message guide)
- CHANGELOG.md (changelog template)
- .github/workflows/release.yml (automated releases)
- scripts/prepare-release.sh (release preparation script)
- .cz.toml (Commitizen configuration)
- .github/ISSUE_TEMPLATE/release-checklist.md (issue template)

How it works:
1. Make commits using conventional format: feat: add feature
2. When ready to release: ./scripts/prepare-release.sh
3. Script analyzes commits and suggests version
4. Push tag: git push origin v1.2.3
5. GitHub Actions automatically creates release with notes

Try it:
  ./scripts/prepare-release.sh --help

Next steps:
- Install Commitizen: pip install commitizen
- Use: cz commit (for guided commits)
- Review RELEASE.md for complete process
```

## Testing This Skill

To test this skill on a new project:
1. Clone a project without a release procedure
2. Invoke this skill
3. Verify all files are created correctly
4. Run the preparation script with `--help`
5. Check workflow syntax is valid
6. Ensure documentation is clear and actionable

## Maintenance

This skill should be updated when:
- New release automation best practices emerge
- GitHub Actions syntax changes
- New conventional commit tools become popular
- Semantic Versioning specification updates

## Related Skills

- **ci-cd-setup**: Setting up continuous integration
- **documentation**: Project documentation structure
- **testing-setup**: Test infrastructure setup

## License Considerations

All generated files use standard formats and best practices that are not subject to licensing. The skill can be freely used for any project.

## References

- [Semantic Versioning](https://semver.org/)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [Keep a Changelog](https://keepachangelog.com/)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Commitizen](https://github.com/commitizen-tools/commitizen)
