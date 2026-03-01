#!/usr/bin/env sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
TARGET_DIR="$ROOT_DIR/docs/franken-ui/contexts"
TMP_DIR=$(mktemp -d)

cleanup() {
	rm -rf "$TMP_DIR"
}
trap cleanup EXIT

echo "Cloning franken-ui/contexts..."
git clone --depth 1 https://github.com/franken-ui/contexts "$TMP_DIR/contexts" >/dev/null 2>&1

echo "Refreshing markdown files..."
mkdir -p "$TARGET_DIR"
rm -f "$TARGET_DIR"/*.md
cp "$TMP_DIR/contexts"/*.md "$TARGET_DIR"/

SHA=$(git -C "$TMP_DIR/contexts" rev-parse HEAD)

echo "Snapshot updated"
echo "Commit: $SHA"
echo "Path: $TARGET_DIR"
