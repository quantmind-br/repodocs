#!/usr/bin/env bash
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

cd "$(dirname "${BASH_SOURCE[0]}")/.."

LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
VERSION="${LATEST_TAG#v}"
IFS='.' read -r MAJOR MINOR PATCH <<< "$VERSION"

NEXT_PATCH="v${MAJOR}.${MINOR}.$((PATCH + 1))"
NEXT_MINOR="v${MAJOR}.$((MINOR + 1)).0"
NEXT_MAJOR="v$((MAJOR + 1)).0.0"

echo ""
echo -e "Current version: ${GREEN}${LATEST_TAG}${NC}"
echo ""
echo -e "  ${CYAN}1)${NC} patch  → ${GREEN}${NEXT_PATCH}${NC}"
echo -e "  ${CYAN}2)${NC} minor  → ${GREEN}${NEXT_MINOR}${NC}"
echo -e "  ${CYAN}3)${NC} major  → ${GREEN}${NEXT_MAJOR}${NC}"
echo -e "  ${CYAN}4)${NC} custom"
echo -e "  ${CYAN}q)${NC} quit"
echo ""

read -rp "Select [1-4/q]: " choice

case $choice in
    1) NEW_VERSION="$NEXT_PATCH" ;;
    2) NEW_VERSION="$NEXT_MINOR" ;;
    3) NEW_VERSION="$NEXT_MAJOR" ;;
    4) read -rp "Version (e.g., v1.2.3): " NEW_VERSION ;;
    q|Q) echo "Aborted."; exit 0 ;;
    *) echo -e "${RED}Invalid option${NC}"; exit 1 ;;
esac

if [[ ! "$NEW_VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo -e "${RED}Error: Invalid semver format${NC}"
    exit 1
fi

if git rev-parse "$NEW_VERSION" >/dev/null 2>&1; then
    echo -e "${RED}Error: Tag $NEW_VERSION already exists${NC}"
    exit 1
fi

if [[ -n "$(git status --porcelain)" ]]; then
    echo -e "${RED}Error: Uncommitted changes. Commit first.${NC}"
    exit 1
fi

echo ""
echo -e "Release: ${YELLOW}${LATEST_TAG}${NC} → ${GREEN}${NEW_VERSION}${NC}"
read -rp "Confirm? [y/N]: " confirm

if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
    echo "Aborted."
    exit 0
fi

echo ""
echo -e "${CYAN}Creating tag...${NC}"
git tag -a "$NEW_VERSION" -m "Release $NEW_VERSION"

echo -e "${CYAN}Pushing to origin...${NC}"
git push origin "$NEW_VERSION"

echo ""
echo -e "${GREEN}✓ Release $NEW_VERSION created!${NC}"
echo ""
echo "GitHub Actions will build and publish the release."
echo -e "Monitor: ${CYAN}https://github.com/quantmind-br/repodocs-go/actions${NC}"
