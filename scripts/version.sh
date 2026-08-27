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
if [ -f "$root/VERSION" ]; then
	printf 'v%s\n' "$(tr -d ' \t\n' <"$root/VERSION")"
	exit 0
fi
echo v0.0.0-dev
