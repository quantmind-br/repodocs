#!/usr/bin/env bash
#
# version.sh - Version information utility
#
# Usage:
#   ./scripts/version.sh              # Show version info
#   ./scripts/version.sh current      # Show current version only
#   ./scripts/version.sh next         # Show next patch version
#   ./scripts/version.sh next minor   # Show next minor version
#   ./scripts/version.sh next major   # Show next major version
#   ./scripts/version.sh compare      # Compare with remote tags
#   ./scripts/version.sh history      # Show version history
#

set -euo pipefail

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

# Get version info
get_version_info() {
    LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
    LATEST_VERSION="${LATEST_TAG#v}"

    IFS='.' read -r MAJOR MINOR PATCH <<< "$LATEST_VERSION"

    NEXT_PATCH="v${MAJOR}.${MINOR}.$((PATCH + 1))"
    NEXT_MINOR="v${MAJOR}.$((MINOR + 1)).0"
    NEXT_MAJOR="v$((MAJOR + 1)).0.0"

    # Get commit info
    COMMITS_SINCE=$(git rev-list --count "${LATEST_TAG}..HEAD" 2>/dev/null || echo "0")
    CURRENT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
}

show_info() {
    get_version_info

    echo -e "${BLUE}Version Information${NC}"
    echo "─────────────────────────────────────────"
    echo ""
    echo -e "Current version:  ${GREEN}${LATEST_TAG}${NC}"
    echo -e "Current commit:   ${CURRENT_COMMIT}"
    echo -e "Commits since:    ${COMMITS_SINCE}"
    echo ""
    echo -e "${BLUE}Next Versions${NC}"
    echo "─────────────────────────────────────────"
    echo ""
    echo -e "Patch (bugfix):   ${GREEN}${NEXT_PATCH}${NC}"
    echo -e "Minor (feature):  ${GREEN}${NEXT_MINOR}${NC}"
    echo -e "Major (breaking): ${GREEN}${NEXT_MAJOR}${NC}"
    echo ""
    echo -e "${BLUE}Build Info${NC}"
    echo "─────────────────────────────────────────"
    echo ""
    echo "go version:       $(go version | awk '{print $3}')"
    echo "GOOS:             ${GOOS:-$(go env GOOS)}"
    echo "GOARCH:           ${GOARCH:-$(go env GOARCH)}"
}

show_current() {
    LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
    echo "$LATEST_TAG"
}

show_next() {
    get_version_info

    case "${1:-patch}" in
        patch)
            echo "$NEXT_PATCH"
            ;;
        minor)
            echo "$NEXT_MINOR"
            ;;
        major)
            echo "$NEXT_MAJOR"
            ;;
        *)
            echo "Usage: $0 next [patch|minor|major]"
            exit 1
            ;;
    esac
}

show_compare() {
    echo -e "${BLUE}Tag Comparison${NC}"
    echo "─────────────────────────────────────────"
    echo ""

    # Fetch remote tags
    git fetch --tags --quiet 2>/dev/null || true

    LOCAL_TAGS=$(git tag -l 'v*' | sort -V | tail -5)
    REMOTE_TAGS=$(git ls-remote --tags origin 2>/dev/null | grep -oP 'refs/tags/v[0-9.]+$' | sed 's/refs\/tags\///' | sort -V | tail -5)

    echo "Local tags (latest 5):"
    if [[ -n "$LOCAL_TAGS" ]]; then
        echo "$LOCAL_TAGS" | while read -r tag; do
            echo -e "  ${GREEN}${tag}${NC}"
        done
    else
        echo "  (none)"
    fi

    echo ""
    echo "Remote tags (latest 5):"
    if [[ -n "$REMOTE_TAGS" ]]; then
        echo "$REMOTE_TAGS" | while read -r tag; do
            echo -e "  ${GREEN}${tag}${NC}"
        done
    else
        echo "  (none)"
    fi

    # Check for unpushed tags
    echo ""
    UNPUSHED=$(comm -23 <(git tag -l 'v*' | sort) <(git ls-remote --tags origin 2>/dev/null | grep -oP 'refs/tags/v[0-9.]+$' | sed 's/refs\/tags\///' | sort) 2>/dev/null || echo "")

    if [[ -n "$UNPUSHED" ]]; then
        echo -e "${YELLOW}Unpushed tags:${NC}"
        echo "$UNPUSHED" | while read -r tag; do
            echo -e "  ${YELLOW}${tag}${NC}"
        done
    else
        echo -e "${GREEN}All tags are synced with remote${NC}"
    fi
}

show_history() {
    echo -e "${BLUE}Version History${NC}"
    echo "─────────────────────────────────────────"
    echo ""

    git tag -l 'v*' --sort=-version:refname | head -20 | while read -r tag; do
        DATE=$(git log -1 --format="%ci" "$tag" 2>/dev/null | cut -d' ' -f1 || echo "unknown")
        COMMITS=$(git rev-list --count "${tag}^..${tag}" 2>/dev/null || echo "?")
        echo -e "${GREEN}${tag}${NC}  (${DATE})"
    done

    TOTAL=$(git tag -l 'v*' | wc -l)
    if [[ $TOTAL -gt 20 ]]; then
        echo ""
        echo "... and $((TOTAL - 20)) more"
    fi
}

# Main
case "${1:-}" in
    current)
        show_current
        ;;
    next)
        show_next "${2:-patch}"
        ;;
    compare)
        show_compare
        ;;
    history)
        show_history
        ;;
    ""|info)
        show_info
        ;;
    *)
        echo "Usage: $0 [current|next|compare|history]"
        echo ""
        echo "Commands:"
        echo "  current           Show current version tag"
        echo "  next [type]       Show next version (patch|minor|major)"
        echo "  compare           Compare local and remote tags"
        echo "  history           Show version history"
        echo "  (none)            Show full version info"
        exit 1
        ;;
esac
