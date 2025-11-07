# Release Procedure Setup Skill

A Claude Code skill that sets up comprehensive, production-ready release procedures for software projects.

## Quick Start

Invoke this skill when you need to establish a formal release process:

```
Set up a release procedure for this project
```

or

```
/skill release-procedure
```

## What It Does

This skill automates the creation of:

1. **Release Documentation** (RELEASE.md) - Complete checklist and guidelines
2. **Conventional Commits Guide** - Standardized commit message format
3. **Changelog Template** - Following Keep a Changelog format
4. **Automated Release Workflow** - GitHub Actions (or other CI/CD)
5. **Release Preparation Script** - Interactive helper for releases
6. **Commitizen Configuration** - Tool for guided commit messages
7. **Issue Templates** - GitHub issue template for tracking releases

## When to Use

- Moving from ad-hoc to systematic releases
- Need automated release notes generation
- Want to implement Semantic Versioning
- Team needs commit message standards
- Currently just creating tags without process

## Supported Project Types

- **Go** (primary support with Go modules, pkg.go.dev)
- **Node.js** (npm, package.json)
- **Rust** (Cargo, crates.io)
- **Python** (PyPI, setup.py/pyproject.toml)
- **Other languages** (customizable templates)

## Example Output

After running this skill, you'll have:

```
project/
├── RELEASE.md                                    # Release process documentation
├── CHANGELOG.md                                  # Changelog template
├── .cz.toml                                      # Commitizen config
├── docs/
│   └── CONVENTIONAL_COMMITS.md                   # Commit message guide
├── .github/
│   ├── workflows/
│   │   └── release.yml                          # Automated release workflow
│   └── ISSUE_TEMPLATE/
│       └── release-checklist.md                 # Release tracking template
└── scripts/
    └── prepare-release.sh                       # Interactive release script
```

## Workflow After Setup

1. **Make commits** using conventional format:
   ```bash
   git commit -m "feat: add new feature"
   git commit -m "fix: resolve bug"
   ```

2. **Prepare release**:
   ```bash
   ./scripts/prepare-release.sh
   ```
   - Analyzes commits
   - Suggests version bump
   - Runs tests and linting
   - Creates and pushes tag

3. **Automated release**:
   - GitHub Actions triggers on tag push
   - Generates categorized release notes
   - Creates GitHub release
   - Updates package registries

## Customization

The skill adapts to your project by:
- Detecting project type (Go, Node.js, Rust, Python, etc.)
- Using your existing test/lint/build commands
- Configuring appropriate package registry updates
- Adding language-specific release steps

## Requirements

- Git repository
- GitHub (or other Git hosting with CI/CD)
- Test and lint commands configured

## Benefits

✅ **Automated**: Tag push triggers full release pipeline
✅ **Consistent**: Standardized process every time
✅ **Quality**: Pre-release checks prevent issues
✅ **Clarity**: Clear commit history and release notes
✅ **Versioning**: Semantic versioning enforced

## Related Documentation

- [Semantic Versioning](https://semver.org/)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [Keep a Changelog](https://keepachangelog.com/)

## Maintenance

The skill creates maintainable, standard-format files that:
- Don't require updates for minor changes
- Use well-documented formats
- Follow industry best practices
- Are easily customizable for your needs
