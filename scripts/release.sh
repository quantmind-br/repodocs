#!/usr/bin/env bash
#
# release.sh - Interactive release script for repodocs-go
#
# Usage:
#   ./scripts/release.sh              # Interactive mode
#   ./scripts/release.sh patch        # Auto patch release
#   ./scripts/release.sh minor        # Auto minor release
#   ./scripts/release.sh major        # Auto major release
#   ./scripts/release.sh v1.2.3       # Specific version
#

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

# Get current version info
get_version_info() {
    LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
    LATEST_VERSION="${LATEST_TAG#v}"

    IFS='.' read -r MAJOR MINOR PATCH <<< "$LATEST_VERSION"

    NEXT_PATCH="v${MAJOR}.${MINOR}.$((PATCH + 1))"
    NEXT_MINOR="v${MAJOR}.$((MINOR + 1)).0"
    NEXT_MAJOR="v$((MAJOR + 1)).0.0"
}

# Print header
print_header() {
    echo -e "${BLUE}"
    echo "╔══════════════════════════════════════════╗"
    echo "║         repodocs-go Release Tool         ║"
    echo "╚══════════════════════════════════════════╝"
    echo -e "${NC}"
}

# Print version info
print_version_info() {
    echo -e "Current version: ${GREEN}${LATEST_TAG}${NC}"
    echo ""
    echo "Available versions:"
    echo -e "  ${YELLOW}1)${NC} Patch  → ${GREEN}${NEXT_PATCH}${NC}"
    echo -e "  ${YELLOW}2)${NC} Minor  → ${GREEN}${NEXT_MINOR}${NC}"
    echo -e "  ${YELLOW}3)${NC} Major  → ${GREEN}${NEXT_MAJOR}${NC}"
    echo -e "  ${YELLOW}4)${NC} Custom version"
    echo -e "  ${YELLOW}q)${NC} Quit"
    echo ""
}

# Check prerequisites
check_prerequisites() {
    local errors=0

    # Check clean working directory
    if [[ -n "$(git status --porcelain)" ]]; then
        echo -e "${RED}✗ Working directory is not clean${NC}"
        git status --short
        errors=$((errors + 1))
    else
        echo -e "${GREEN}✓ Working directory is clean${NC}"
    fi

    # Check branch
    CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
    if [[ "$CURRENT_BRANCH" != "main" && "$CURRENT_BRANCH" != "master" ]]; then
        echo -e "${YELLOW}⚠ Not on main/master branch (current: ${CURRENT_BRANCH})${NC}"
    else
        echo -e "${GREEN}✓ On ${CURRENT_BRANCH} branch${NC}"
    fi

    # Check if up to date with remote
    git fetch origin --quiet 2>/dev/null || true
    LOCAL=$(git rev-parse HEAD 2>/dev/null)
    REMOTE=$(git rev-parse "origin/${CURRENT_BRANCH}" 2>/dev/null || echo "")

    if [[ -n "$REMOTE" && "$LOCAL" != "$REMOTE" ]]; then
        echo -e "${YELLOW}⚠ Local branch differs from remote${NC}"
    else
        echo -e "${GREEN}✓ Up to date with remote${NC}"
    fi

    # Check goreleaser config
    if [[ -f ".goreleaser.yaml" || -f ".goreleaser.yml" ]]; then
        echo -e "${GREEN}✓ GoReleaser config found${NC}"
    else
        echo -e "${YELLOW}⚠ No GoReleaser config found${NC}"
    fi

    echo ""
    return $errors
}

# Validate semver format
validate_version() {
    local version=$1
    if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        echo -e "${RED}Error: Version must match semver format (e.g., v1.0.0)${NC}"
        return 1
    fi

    if git rev-parse "$version" >/dev/null 2>&1; then
        echo -e "${RED}Error: Tag $version already exists${NC}"
        return 1
    fi

    return 0
}

# Create and push release
do_release() {
    local version=$1

    echo ""
    echo -e "Creating release ${GREEN}${version}${NC}..."
    echo ""

    # Create tag
    echo -e "${BLUE}→ Creating tag...${NC}"
    git tag -a "$version" -m "Release $version"
    echo -e "${GREEN}✓ Tag created${NC}"

    # Push tag
    echo -e "${BLUE}→ Pushing tag to origin...${NC}"
    git push origin "$version"
    echo -e "${GREEN}✓ Tag pushed${NC}"

    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║     Release ${version} completed!     ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════╝${NC}"
    echo ""
    echo "GitHub Actions will now build and publish the release."
    echo -e "Monitor: ${BLUE}https://github.com/quantmind-br/repodocs-go/actions${NC}"
}

# Interactive mode
interactive_mode() {
    print_header
    get_version_info

    echo "Running pre-release checks..."
    echo ""

    if ! check_prerequisites; then
        echo -e "${RED}Please fix the issues above before releasing.${NC}"
        exit 1
    fi

    print_version_info

    read -rp "Select option: " choice

    case $choice in
        1)
            RELEASE_VERSION="$NEXT_PATCH"
            ;;
        2)
            RELEASE_VERSION="$NEXT_MINOR"
            ;;
        3)
            RELEASE_VERSION="$NEXT_MAJOR"
            ;;
        4)
            read -rp "Enter version (e.g., v1.2.3): " RELEASE_VERSION
            ;;
        q|Q)
            echo "Aborted."
            exit 0
            ;;
        *)
            echo -e "${RED}Invalid option${NC}"
            exit 1
            ;;
    esac

    if ! validate_version "$RELEASE_VERSION"; then
        exit 1
    fi

    echo ""
    echo -e "You are about to release: ${GREEN}${RELEASE_VERSION}${NC}"
    read -rp "Confirm? [y/N] " confirm

    if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
        echo "Aborted."
        exit 0
    fi

    do_release "$RELEASE_VERSION"
}

# Main
main() {
    get_version_info

    case "${1:-}" in
        patch)
            RELEASE_VERSION="$NEXT_PATCH"
            ;;
        minor)
            RELEASE_VERSION="$NEXT_MINOR"
            ;;
        major)
            RELEASE_VERSION="$NEXT_MAJOR"
            ;;
        v*)
            RELEASE_VERSION="$1"
            ;;
        "")
            interactive_mode
            exit 0
            ;;
        *)
            echo "Usage: $0 [patch|minor|major|vX.Y.Z]"
            exit 1
            ;;
    esac

    # Non-interactive mode
    echo "Running pre-release checks..."
    if ! check_prerequisites; then
        echo -e "${RED}Please fix the issues above before releasing.${NC}"
        exit 1
    fi

    if ! validate_version "$RELEASE_VERSION"; then
        exit 1
    fi

    echo -e "Releasing: ${GREEN}${RELEASE_VERSION}${NC}"
    read -rp "Confirm? [y/N] " confirm

    if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
        echo "Aborted."
        exit 0
    fi

    do_release "$RELEASE_VERSION"
}

main "$@"
