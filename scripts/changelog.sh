#!/usr/bin/env bash
#
# changelog.sh - Generate changelog from git commits
#
# Usage:
#   ./scripts/changelog.sh              # Changes since last tag
#   ./scripts/changelog.sh v1.0.0       # Changes since specific tag
#   ./scripts/changelog.sh v1.0.0 HEAD  # Changes between versions
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

# Get version range
FROM_TAG="${1:-$(git describe --tags --abbrev=0 2>/dev/null || echo "")}"
TO_REF="${2:-HEAD}"

if [[ -z "$FROM_TAG" ]]; then
    echo "No tags found. Showing all commits."
    FROM_TAG=$(git rev-list --max-parents=0 HEAD)
fi

# Get next version for header
if [[ "$TO_REF" == "HEAD" ]]; then
    LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
    LATEST_VERSION="${LATEST_TAG#v}"
    IFS='.' read -r MAJOR MINOR PATCH <<< "$LATEST_VERSION"
    NEXT_VERSION="v${MAJOR}.${MINOR}.$((PATCH + 1))"
    HEADER="## ${NEXT_VERSION} (Unreleased)"
else
    HEADER="## ${TO_REF}"
fi

echo -e "${BLUE}${HEADER}${NC}"
echo ""

# Categorize commits
declare -A categories=(
    ["feat"]="Features"
    ["fix"]="Bug Fixes"
    ["docs"]="Documentation"
    ["style"]="Styles"
    ["refactor"]="Refactoring"
    ["perf"]="Performance"
    ["test"]="Tests"
    ["build"]="Build"
    ["ci"]="CI/CD"
    ["chore"]="Chores"
)

# Get commits
COMMITS=$(git log "${FROM_TAG}..${TO_REF}" --pretty=format:"%s|%h|%an" --reverse 2>/dev/null || echo "")

if [[ -z "$COMMITS" ]]; then
    echo "No changes since ${FROM_TAG}"
    exit 0
fi

# Parse and categorize
declare -A categorized_commits

while IFS='|' read -r message hash author; do
    # Extract conventional commit type
    if [[ "$message" =~ ^([a-z]+)(\(.+\))?!?:\ (.+)$ ]]; then
        type="${BASH_REMATCH[1]}"
        scope="${BASH_REMATCH[2]}"
        desc="${BASH_REMATCH[3]}"

        # Clean scope
        scope="${scope#(}"
        scope="${scope%)}"

        if [[ -n "${categories[$type]:-}" ]]; then
            category="${categories[$type]}"
        else
            category="Other"
        fi

        if [[ -n "$scope" ]]; then
            entry="- **${scope}**: ${desc} (\`${hash}\`)"
        else
            entry="- ${desc} (\`${hash}\`)"
        fi

        categorized_commits["$category"]+="${entry}"$'\n'
    else
        # Non-conventional commit
        entry="- ${message} (\`${hash}\`)"
        categorized_commits["Other"]+="${entry}"$'\n'
    fi
done <<< "$COMMITS"

# Print categorized commits
for category in "Features" "Bug Fixes" "Performance" "Refactoring" "Documentation" "Tests" "Build" "CI/CD" "Chores" "Styles" "Other"; do
    if [[ -n "${categorized_commits[$category]:-}" ]]; then
        echo -e "${GREEN}### ${category}${NC}"
        echo ""
        echo -e "${categorized_commits[$category]}"
    fi
done

# Stats
echo -e "${YELLOW}### Stats${NC}"
echo ""
COMMIT_COUNT=$(git rev-list --count "${FROM_TAG}..${TO_REF}" 2>/dev/null || echo "0")
CONTRIBUTORS=$(git log "${FROM_TAG}..${TO_REF}" --pretty=format:"%an" 2>/dev/null | sort -u | wc -l)
FILES_CHANGED=$(git diff --stat "${FROM_TAG}..${TO_REF}" 2>/dev/null | tail -1 || echo "0 files changed")

echo "- Commits: ${COMMIT_COUNT}"
echo "- Contributors: ${CONTRIBUTORS}"
echo "- ${FILES_CHANGED}"
