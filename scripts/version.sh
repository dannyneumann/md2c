#!/bin/sh
# Print the version string baked into md2c (v0.x.y).
# Prefer an exact tag on HEAD, otherwise VERSION (so a missing fetch of
# v0.2.0 does not fall back to v0.1.2-4-g…).
set -e
root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
if git -C "$root" describe --tags --match 'v0.*' --exact-match >/dev/null 2>&1; then
	git -C "$root" describe --tags --match 'v0.*' --exact-match
	exit 0
fi
last_tag=$(git -C "$root" tag -l 'v0.*' --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | head -1 || true)
if [ -n "$last_tag" ]; then
	echo "$last_tag"
	exit 0
fi
if [ -f "$root/VERSION" ]; then
	printf 'v%s\n' "$(tr -d ' \t\n' <"$root/VERSION")"
	exit 0
fi
echo v0.0.0-dev
