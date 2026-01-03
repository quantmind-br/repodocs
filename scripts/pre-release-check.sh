#!/usr/bin/env bash
#
# pre-release-check.sh - Run all pre-release checks
#
# Usage:
#   ./scripts/pre-release-check.sh
#
# Checks:
#   - Git status (clean working directory)
#   - Current branch
#   - Remote sync status
#   - Tests pass
#   - Linting passes
#   - Build succeeds
#

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

ERRORS=0
WARNINGS=0

print_header() {
    echo -e "${BLUE}"
    echo "╔══════════════════════════════════════════╗"
    echo "║       Pre-Release Checklist              ║"
    echo "╚══════════════════════════════════════════╝"
    echo -e "${NC}"
}

check_pass() {
    echo -e "${GREEN}✓${NC} $1"
}

check_fail() {
    echo -e "${RED}✗${NC} $1"
    ERRORS=$((ERRORS + 1))
}

check_warn() {
    echo -e "${YELLOW}⚠${NC} $1"
    WARNINGS=$((WARNINGS + 1))
}

check_skip() {
    echo -e "${BLUE}○${NC} $1 (skipped)"
}

# Main checks
print_header

echo -e "${BLUE}Git Status${NC}"
echo "─────────────────────────────────────────"

# Check clean working directory
if [[ -n "$(git status --porcelain)" ]]; then
    check_fail "Working directory has uncommitted changes"
    git status --short | head -10
    if [[ $(git status --porcelain | wc -l) -gt 10 ]]; then
        echo "  ... and more"
    fi
else
    check_pass "Working directory is clean"
fi

# Check branch
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [[ "$CURRENT_BRANCH" == "main" || "$CURRENT_BRANCH" == "master" ]]; then
    check_pass "On ${CURRENT_BRANCH} branch"
else
    check_warn "Not on main/master (current: ${CURRENT_BRANCH})"
fi

# Check remote sync
git fetch origin --quiet 2>/dev/null || true
LOCAL=$(git rev-parse HEAD 2>/dev/null)
REMOTE=$(git rev-parse "origin/${CURRENT_BRANCH}" 2>/dev/null || echo "")

if [[ -z "$REMOTE" ]]; then
    check_warn "No remote tracking branch"
elif [[ "$LOCAL" == "$REMOTE" ]]; then
    check_pass "Up to date with origin/${CURRENT_BRANCH}"
else
    AHEAD=$(git rev-list --count "origin/${CURRENT_BRANCH}..HEAD" 2>/dev/null || echo "0")
    BEHIND=$(git rev-list --count "HEAD..origin/${CURRENT_BRANCH}" 2>/dev/null || echo "0")
    if [[ "$AHEAD" -gt 0 && "$BEHIND" -gt 0 ]]; then
        check_warn "Diverged from remote (${AHEAD} ahead, ${BEHIND} behind)"
    elif [[ "$AHEAD" -gt 0 ]]; then
        check_warn "Ahead of remote by ${AHEAD} commits (need to push)"
    else
        check_fail "Behind remote by ${BEHIND} commits (need to pull)"
    fi
fi

# Check for existing tag
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
check_pass "Latest tag: ${LATEST_TAG}"

echo ""
echo -e "${BLUE}Code Quality${NC}"
echo "─────────────────────────────────────────"

# Run tests
echo -n "Running tests... "
if make test > /tmp/test-output.txt 2>&1; then
    check_pass "All tests pass"
else
    check_fail "Tests failed"
    tail -20 /tmp/test-output.txt
fi

# Run linting
echo -n "Running linter... "
if make lint > /tmp/lint-output.txt 2>&1; then
    check_pass "Linting passed"
else
    check_fail "Linting failed"
    tail -20 /tmp/lint-output.txt
fi

# Run vet
echo -n "Running go vet... "
if make vet > /tmp/vet-output.txt 2>&1; then
    check_pass "Go vet passed"
else
    check_fail "Go vet failed"
    tail -20 /tmp/vet-output.txt
fi

echo ""
echo -e "${BLUE}Build${NC}"
echo "─────────────────────────────────────────"

# Build
echo -n "Building binary... "
if make build > /tmp/build-output.txt 2>&1; then
    check_pass "Build succeeded"
    BINARY_SIZE=$(du -h build/repodocs 2>/dev/null | cut -f1 || echo "unknown")
    echo "  Binary size: ${BINARY_SIZE}"
else
    check_fail "Build failed"
    tail -20 /tmp/build-output.txt
fi

# Check goreleaser config
if [[ -f ".goreleaser.yaml" || -f ".goreleaser.yml" ]]; then
    echo -n "Validating goreleaser config... "
    if command -v goreleaser &> /dev/null; then
        if goreleaser check > /tmp/goreleaser-output.txt 2>&1; then
            check_pass "GoReleaser config valid"
        else
            check_fail "GoReleaser config invalid"
            cat /tmp/goreleaser-output.txt
        fi
    else
        check_skip "GoReleaser not installed"
    fi
else
    check_warn "No GoReleaser config found"
fi

echo ""
echo -e "${BLUE}Files${NC}"
echo "─────────────────────────────────────────"

# Check important files
for file in "README.md" "LICENSE" "CHANGELOG.md" ".goreleaser.yaml"; do
    if [[ -f "$file" ]]; then
        check_pass "${file} exists"
    else
        if [[ "$file" == "CHANGELOG.md" ]]; then
            check_warn "${file} not found (optional)"
        else
            check_warn "${file} not found"
        fi
    fi
done

# Summary
echo ""
echo "═════════════════════════════════════════"
if [[ $ERRORS -eq 0 && $WARNINGS -eq 0 ]]; then
    echo -e "${GREEN}All checks passed! Ready to release.${NC}"
    exit 0
elif [[ $ERRORS -eq 0 ]]; then
    echo -e "${YELLOW}Passed with ${WARNINGS} warning(s). Review before releasing.${NC}"
    exit 0
else
    echo -e "${RED}Failed with ${ERRORS} error(s) and ${WARNINGS} warning(s).${NC}"
    echo "Fix the issues above before releasing."
    exit 1
fi
