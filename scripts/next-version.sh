#!/bin/sh
# Print the next v0.x.y tag. Ignores legacy npm tags (v1.*, v2.*).
set -e
if git describe --exact-match --match 'v0.*.*' HEAD >/dev/null 2>&1; then
	git describe --exact-match --match 'v0.*.*' HEAD
	exit 0
fi
last=$(git tag -l 'v0.*' --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | head -1 || true)
if [ -z "$last" ]; then
	echo v0.1.0
	exit 0
fi
echo "$last" | sed 's/^v//' | awk -F. '{printf "v%d.%d.%d\n", $1, $2, $3+1}'
