#!/bin/bash

# Release preparation script for chroma-go
# This script helps automate the pre-release verification steps

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}==>${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Parse command line arguments
VERSION=""
SKIP_TESTS=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        --skip-tests)
            SKIP_TESTS=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  -v, --version VERSION    Specify the version to release (e.g., v1.2.3)"
            echo "  --skip-tests             Skip running tests (not recommended)"
            echo "  -h, --help               Show this help message"
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            echo "Use -h or --help for usage information"
            exit 1
            ;;
    esac
done

# Header
echo ""
echo "╔═══════════════════════════════════════════════════════════╗"
echo "║         chroma-go Release Preparation Script             ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""

# Determine version if not provided
if [ -z "$VERSION" ]; then
    print_status "Determining next version..."

    # Get the last tag
    LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")

    if [ -z "$LAST_TAG" ]; then
        print_warning "No previous tags found. This appears to be the first release."
        read -p "Enter version for first release (e.g., v0.1.0): " VERSION
    else
        echo "Last release: $LAST_TAG"

        # Count commits by type since last tag
        COMMITS_SINCE=$(git log ${LAST_TAG}..HEAD --oneline 2>/dev/null || echo "")

        if [ -z "$COMMITS_SINCE" ]; then
            print_error "No commits since last release!"
            exit 1
        fi

        echo ""
        echo "Commits since last release:"
        echo "$COMMITS_SINCE"
        echo ""

        # Count conventional commit types
        BREAKING_CHANGES=$(echo "$COMMITS_SINCE" | grep -c "BREAKING CHANGE\|!" || true)
        FEATURES=$(echo "$COMMITS_SINCE" | grep -c "^feat" || true)
        FIXES=$(echo "$COMMITS_SINCE" | grep -c "^fix" || true)

        print_status "Change summary:"
        echo "  - Breaking changes: $BREAKING_CHANGES"
        echo "  - Features: $FEATURES"
        echo "  - Fixes: $FIXES"
        echo ""

        # Suggest version bump
        if [ $BREAKING_CHANGES -gt 0 ]; then
            print_warning "Breaking changes detected! This should be a MAJOR version bump."
        elif [ $FEATURES -gt 0 ]; then
            print_status "New features detected. This should be a MINOR version bump."
        elif [ $FIXES -gt 0 ]; then
            print_status "Bug fixes detected. This should be a PATCH version bump."
        fi

        echo ""
        read -p "Enter version for this release (e.g., v1.2.3): " VERSION
    fi
fi

# Validate version format
if ! [[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?$ ]]; then
    print_error "Invalid version format: $VERSION"
    print_error "Expected format: vX.Y.Z or vX.Y.Z-prerelease"
    exit 1
fi

print_success "Target version: $VERSION"
echo ""

# Check if we're on main branch
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" != "main" ]; then
    print_warning "You are on branch '$CURRENT_BRANCH', not 'main'"
    read -p "Continue anyway? (y/N): " CONTINUE
    if [ "$CONTINUE" != "y" ] && [ "$CONTINUE" != "Y" ]; then
        exit 1
    fi
fi

# Ensure working directory is clean
if ! git diff-index --quiet HEAD --; then
    print_error "Working directory is not clean. Please commit or stash changes."
    exit 1
fi
print_success "Working directory is clean"

# Update from remote
print_status "Updating from remote..."
git fetch origin
git pull origin main
print_success "Updated from remote"

# Check if tag already exists
if git rev-parse "$VERSION" >/dev/null 2>&1; then
    print_error "Tag $VERSION already exists!"
    exit 1
fi
print_success "Tag $VERSION does not exist yet"

echo ""
print_status "Running pre-release checks..."
echo ""

# 1. Lint
print_status "1/5 Running linter..."
if make lint >/dev/null 2>&1; then
    print_success "Linting passed"
else
    print_error "Linting failed"
    print_warning "Try running: make lint-fix"
    exit 1
fi

# 2. Build
print_status "2/5 Building..."
if make build >/dev/null 2>&1; then
    print_success "Build successful"
else
    print_error "Build failed"
    exit 1
fi

# 3. Tests
if [ "$SKIP_TESTS" = true ]; then
    print_warning "3/5 Skipping tests (--skip-tests flag provided)"
else
    print_status "3/5 Running tests..."

    print_status "  Running V1 API tests..."
    if make test >/dev/null 2>&1; then
        print_success "  V1 API tests passed"
    else
        print_error "  V1 API tests failed"
        exit 1
    fi

    print_status "  Running V2 API tests..."
    if make test-v2 >/dev/null 2>&1; then
        print_success "  V2 API tests passed"
    else
        print_error "  V2 API tests failed"
        exit 1
    fi

    print_success "All tests passed"
fi

# 4. Check for required tools
print_status "4/5 Checking for required tools..."
MISSING_TOOLS=0

if ! command_exists git; then
    print_error "git is not installed"
    MISSING_TOOLS=1
fi

if [ $MISSING_TOOLS -eq 1 ]; then
    print_error "Missing required tools. Please install them and try again."
    exit 1
fi
print_success "All required tools are available"

# 5. Generate release summary
print_status "5/5 Generating release summary..."

COMMITS_FOR_RELEASE=$(git log $(git describe --tags --abbrev=0 2>/dev/null || git rev-list --max-parents=0 HEAD)..HEAD --pretty=format:"- %s" 2>/dev/null || echo "")

echo ""
echo "╔═══════════════════════════════════════════════════════════╗"
echo "║                    Release Summary                        ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo ""
echo "Version: $VERSION"
echo "Branch: $CURRENT_BRANCH"
echo ""
echo "Changes in this release:"
echo "$COMMITS_FOR_RELEASE"
echo ""

# Final confirmation
print_status "All pre-release checks passed!"
echo ""
echo "Next steps:"
echo "1. Create and push the tag:"
echo "   git tag -a $VERSION -m \"Release $VERSION\""
echo "   git push origin $VERSION"
echo ""
echo "2. The GitHub Actions workflow will automatically create the release"
echo ""
echo "3. Review and publish the release on GitHub"
echo ""

read -p "Do you want to create and push the tag now? (y/N): " CREATE_TAG

if [ "$CREATE_TAG" = "y" ] || [ "$CREATE_TAG" = "Y" ]; then
    print_status "Creating tag $VERSION..."

    # Create tag message
    TAG_MESSAGE="Release $VERSION"

    git tag -a "$VERSION" -m "$TAG_MESSAGE"
    print_success "Tag created"

    print_status "Pushing tag to origin..."
    git push origin "$VERSION"
    print_success "Tag pushed"

    echo ""
    print_success "Release process initiated!"
    echo ""
    echo "Monitor the release workflow at:"
    echo "https://github.com/amikos-tech/chroma-go/actions"
    echo ""
else
    print_status "Tag not created. You can create it manually when ready:"
    echo "  git tag -a $VERSION -m \"Release $VERSION\""
    echo "  git push origin $VERSION"
fi

echo ""
print_success "Done!"
