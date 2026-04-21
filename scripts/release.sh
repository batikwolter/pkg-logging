#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<USAGE
Usage: scripts/release.sh <version> [--no-push] [--dry-run]

Examples:
  scripts/release.sh v0.1.1
  scripts/release.sh v0.1.1 --no-push
  scripts/release.sh v0.1.1 --dry-run
USAGE
}

if [[ $# -lt 1 ]]; then
  usage
  exit 1
fi

VERSION="$1"
shift || true

PUSH=true
DRY_RUN=false

for arg in "$@"; do
  case "$arg" in
    --no-push)
      PUSH=false
      ;;
    --dry-run)
      DRY_RUN=true
      ;;
    *)
      echo "unknown argument: $arg"
      usage
      exit 1
      ;;
  esac
done

if ! [[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "invalid version format: $VERSION (expected vX.Y.Z)"
  exit 1
fi

if ! git diff --quiet || ! git diff --cached --quiet; then
  echo "working tree not clean. commit/stash changes first."
  exit 1
fi

if git rev-parse "$VERSION" >/dev/null 2>&1; then
  echo "tag already exists: $VERSION"
  exit 1
fi

NUM_VERSION="${VERSION#v}"
TODAY="$(date +%F)"

if ! grep -Eq "^## \[$NUM_VERSION\] - [0-9]{4}-[0-9]{2}-[0-9]{2}$" CHANGELOG.md; then
  echo "CHANGELOG.md missing release section for $NUM_VERSION"
  echo "expected header format: ## [$NUM_VERSION] - $TODAY"
  exit 1
fi

echo "Running tests..."
go test ./...

echo "Release checks passed for $VERSION"

if [[ "$DRY_RUN" == "true" ]]; then
  echo "dry-run mode: no commit/tag/push performed"
  exit 0
fi

git add -A
git commit -m "release: $VERSION"
git tag -a "$VERSION" -m "$VERSION"

if [[ "$PUSH" == "true" ]]; then
  git push origin main
  git push origin "$VERSION"
  echo "pushed main and $VERSION"
else
  echo "created commit + tag locally (no push)"
fi
