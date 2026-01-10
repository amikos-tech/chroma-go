# Conventional Commits Guide

This project uses [Conventional Commits](https://www.conventionalcommits.org/) specification for commit messages. This enables automatic changelog generation and semantic versioning.

## Why Conventional Commits?

- **Automated releases**: Automatically determine version bumps and generate changelogs
- **Clear history**: Easy to understand what changes were made
- **Better collaboration**: Standardized format makes it easier for others to understand your changes

## Commit Message Format

```
<type>[optional scope][optional !]: <description>

[optional body]

[optional footer(s)]
```

### Examples

```
feat: add support for OpenAI embeddings
```

```
fix(client): resolve connection timeout issue
```

```
feat(api)!: change authentication method to OAuth

BREAKING CHANGE: The basic auth method has been removed.
Use OAuth authentication instead.
```

```
docs: update installation instructions

Added section about environment variables and
clarified the minimum Go version requirement.

Closes #123
```

## Commit Types

| Type | Description | Version Bump |
|------|-------------|--------------|
| `feat` | A new feature | MINOR |
| `fix` | A bug fix | PATCH |
| `docs` | Documentation only changes | - |
| `style` | Code style changes (formatting, missing semi-colons, etc.) | - |
| `refactor` | Code change that neither fixes a bug nor adds a feature | PATCH |
| `perf` | Code change that improves performance | PATCH |
| `test` | Adding missing tests or correcting existing tests | - |
| `build` | Changes to build system or external dependencies | - |
| `ci` | Changes to CI configuration files and scripts | - |
| `chore` | Other changes that don't modify src or test files | - |
| `revert` | Reverts a previous commit | - |

## Scopes

Scopes are optional and provide additional context about what part of the codebase is affected:

- `api` - Changes to the API layer
- `client` - Changes to client implementation
- `embeddings` - Changes to embedding functions
- `rerankings` - Changes to reranking functions
- `metadata` - Changes to metadata handling
- `auth` - Changes to authentication
- `docs` - Documentation changes
- `tests` - Test-related changes

### Examples with Scopes

```
feat(embeddings): add Mistral embedding support
fix(client): handle connection retry properly
docs(api): update V2 API documentation
test(rerankings): add tests for Cohere reranker
```

## Breaking Changes

Breaking changes should be indicated in two ways:

1. Add `!` after the type/scope: `feat(api)!: change return type`
2. Include `BREAKING CHANGE:` in the footer with a description

### Example

```
feat(api)!: change Collection.Query to accept context

BREAKING CHANGE: All Collection methods now require a context.Context
as the first parameter. Update all calls to include context.

Migration example:
  - Before: collection.Query(params)
  + After:  collection.Query(ctx, params)

Closes #456
```

## Using Commitizen (Optional)

You can use [Commitizen](https://github.com/commitizen-tools/commitizen) to help write conventional commits interactively.

### Installation

**Python version** (recommended):
```bash
pip install commitizen
```

**Go version**:
```bash
go install github.com/shipengqi/commitizen/cmd/cz@latest
```

### Usage

Instead of `git commit`, use:
```bash
cz commit
```

or with Python version:
```bash
cz c
```

This will guide you through creating a properly formatted commit message.

### Configuration

This repository includes a `.cz.toml` configuration file that defines:
- Available commit types
- Message format
- Version bump rules
- Changelog generation settings

## Commit Message Best Practices

### Do:

✅ **Use imperative mood**: "add feature" not "added feature"
```
feat: add caching support
```

✅ **Be concise in the subject**: Keep it under 72 characters
```
fix: resolve race condition in query execution
```

✅ **Provide details in the body**: Explain what and why, not how
```
feat: add request retry mechanism

Implements exponential backoff retry for failed API requests.
This improves reliability when dealing with transient network issues.

Closes #789
```

✅ **Reference issues**: Link related issues in the footer
```
fix: correct metadata filtering logic

Fixes #123
```

### Don't:

❌ **Don't use vague messages**
```
fix: fix bug
chore: update stuff
```

❌ **Don't mix multiple changes**
```
feat: add feature X, fix bug Y, update docs
```
*Instead, make separate commits for each change*

❌ **Don't include file names** (unless providing context)
```
fix: update client.go
```
*Instead, describe what was fixed*

## Advanced Examples

### Multi-line Body

```
feat(client): implement connection pooling

Add connection pool management to improve performance
when making multiple concurrent requests. The pool size
is configurable via ClientOptions.

Default pool size: 10 connections
Maximum pool size: 100 connections

Related to #456
```

### Multiple Issues

```
fix(api): resolve authentication edge cases

Fixes several authentication-related issues:
- Handle expired tokens correctly
- Refresh tokens before they expire
- Clear cached credentials on logout

Fixes #123, #124, #125
```

### Revert Example

```
revert: feat(api): add experimental feature X

This reverts commit abc123def456.

Reason: The feature caused performance issues in production.
Will be reimplemented with better performance in the next iteration.
```

## Validation

### Pre-commit Hook (Optional)

You can add a pre-commit hook to validate commit messages:

```bash
#!/bin/bash
# .git/hooks/commit-msg

commit_msg_file=$1
commit_msg=$(cat "$commit_msg_file")

# Conventional commit pattern
pattern="^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\(.+\))?(!)?: .{1,72}"

if ! echo "$commit_msg" | grep -qE "$pattern"; then
    echo "❌ Invalid commit message format!"
    echo ""
    echo "Commit message must follow Conventional Commits specification:"
    echo "<type>[optional scope][optional !]: <description>"
    echo ""
    echo "Examples:"
    echo "  feat: add new feature"
    echo "  fix(client): resolve bug"
    echo "  docs: update README"
    echo ""
    echo "See docs/CONVENTIONAL_COMMITS.md for more information"
    exit 1
fi
```

### CI Validation

The repository can be configured to validate commit messages in CI using tools like:
- [commitlint](https://commitlint.js.org/)
- [gitlint](https://jorisroovers.com/gitlint/)

## Quick Reference

**Feature**:
```
feat: add support for vector database X
```

**Bug Fix**:
```
fix: resolve connection timeout
```

**Documentation**:
```
docs: update API reference
```

**Performance**:
```
perf: optimize query execution
```

**Tests**:
```
test: add integration tests for embeddings
```

**Breaking Change**:
```
feat!: redesign client API

BREAKING CHANGE: Client constructor signature changed.
See migration guide in docs/MIGRATION.md
```

## Resources

- [Conventional Commits Specification](https://www.conventionalcommits.org/)
- [Semantic Versioning](https://semver.org/)
- [Keep a Changelog](https://keepachangelog.com/)
- [Commitizen](https://github.com/commitizen-tools/commitizen)
- [Commitizen for Go](https://github.com/shipengqi/commitizen)

## Questions?

If you have questions about commit message formatting:
1. Check the [examples](#advanced-examples) above
2. Look at recent commits in the repository
3. Ask in the team communication channel
4. Open an issue for documentation improvements
