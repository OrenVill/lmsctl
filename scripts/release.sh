#!/usr/bin/env bash
# Builds Linux binaries (amd64 + arm64) for lmsctl, tags the release, and
# publishes it as a GitHub release via `gh`.
#
# Usage: scripts/release.sh vX.Y.Z
set -euo pipefail

VERSION="${1:-}"
if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "usage: scripts/release.sh vX.Y.Z" >&2
  exit 1
fi

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

BRANCH="$(git rev-parse --abbrev-ref HEAD)"
if [[ "$BRANCH" != "master" ]]; then
  echo "error: releases are built from master, currently on $BRANCH" >&2
  exit 1
fi

if [[ -n "$(git status --porcelain)" ]]; then
  echo "error: working tree isn't clean; commit or stash changes first" >&2
  exit 1
fi

git fetch origin master --quiet
if [[ "$(git rev-parse HEAD)" != "$(git rev-parse origin/master)" ]]; then
  echo "error: local master doesn't match origin/master; push or pull first" >&2
  exit 1
fi

if git rev-parse "$VERSION" >/dev/null 2>&1; then
  echo "error: tag $VERSION already exists" >&2
  exit 1
fi

echo "Running checks..."
UNFORMATTED="$(gofmt -l .)"
if [[ -n "$UNFORMATTED" ]]; then
  echo "error: gofmt found unformatted files:" >&2
  echo "$UNFORMATTED" >&2
  exit 1
fi
go vet ./...
go test ./... -count=1

DIST="$REPO_ROOT/dist"
rm -rf "$DIST"
mkdir -p "$DIST"

for ARCH in amd64 arm64; do
  echo "Building linux/$ARCH..."
  PKG_DIR="lmsctl-${VERSION}-linux-${ARCH}"
  mkdir -p "$DIST/$PKG_DIR"
  GOOS=linux GOARCH="$ARCH" go build -trimpath \
    -ldflags "-s -w -X lmsctl/cmd.version=${VERSION}" \
    -o "$DIST/$PKG_DIR/lmsctl" .
  cp README.md "$DIST/$PKG_DIR/"
  tar -C "$DIST" -czf "$DIST/${PKG_DIR}.tar.gz" "$PKG_DIR"
  rm -rf "$DIST/$PKG_DIR"
done

(cd "$DIST" && sha256sum ./*.tar.gz > SHA256SUMS)

echo "Tagging $VERSION..."
git tag -a "$VERSION" -m "lmsctl $VERSION"
git push origin "$VERSION"

echo "Creating GitHub release..."
gh release create "$VERSION" "$DIST"/*.tar.gz "$DIST/SHA256SUMS" \
  --title "$VERSION" \
  --generate-notes

echo "Published: https://github.com/OrenVill/lmsctl/releases/tag/$VERSION"
