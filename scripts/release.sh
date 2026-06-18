#!/usr/bin/env bash
set -euo pipefail

# Bump the version and open a PR. When the PR merges, a GitHub Action
# detects the version change and tags the release automatically.
#
# Usage:
#   scripts/release.sh patch    # 0.3.1 → 0.3.2
#   scripts/release.sh minor    # 0.3.1 → 0.4.0
#   scripts/release.sh major    # 0.3.1 → 1.0.0
#   scripts/release.sh 1.2.3    # explicit version

cd "$(dirname "$0")/.."

bump="${1:-}"
if [[ -z "$bump" ]]; then
  echo "Usage: scripts/release.sh <patch|minor|major|x.y.z>" >&2
  exit 1
fi

current=$(grep 'var version' cmd/curlew/main.go | sed 's/.*"\(.*\)"/\1/')
IFS='.' read -r maj min pat <<< "$current"

case "$bump" in
  patch) new="$maj.$min.$((pat + 1))" ;;
  minor) new="$maj.$((min + 1)).0" ;;
  major) new="$((maj + 1)).0.0" ;;
  [0-9]*)
    if [[ ! "$bump" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
      echo "Invalid version: $bump (expected x.y.z)" >&2
      exit 1
    fi
    new="$bump"
    ;;
  *) echo "Unknown bump type: $bump" >&2; exit 1 ;;
esac

branch="release/v$new"

echo "Current version: $current"
echo "New version:     $new"
echo "Branch:          $branch"
echo ""

read -rp "Proceed? [y/N] " confirm
[[ "$confirm" == [yY] ]] || { echo "Aborted."; exit 0; }

# Ensure working tree is clean before switching branches
git diff --quiet && git diff --cached --quiet || { echo "Working tree not clean. Commit or stash first." >&2; exit 1; }

git checkout main
git pull origin main
git checkout -b "$branch"

sed -i.bak "s/var version = \".*\"/var version = \"$new\"/" cmd/curlew/main.go
rm -f cmd/curlew/main.go.bak

git add cmd/curlew/main.go
git commit -m "chore: bump version to $new"
git push -u origin "$branch"

gh pr create --title "chore: bump version to $new" --body "Release v$new. Merging this will automatically tag and release."

echo ""
echo "PR created. Merge it and the release will happen automatically."
