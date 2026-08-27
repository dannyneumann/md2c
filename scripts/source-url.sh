#!/bin/sh
# Print the public GitHub URL for this checkout (no .git suffix).
set -e
raw=${1:-}
if [ -z "$raw" ]; then
	raw=$(git config --get remote.origin.url 2>/dev/null || true)
fi
if [ -z "$raw" ]; then
	echo "https://github.com/dannyneumann/md2c"
	exit 0
fi
s=$raw
s=$(printf '%s' "$s" | sed -e 's#^git@github.com:#https://github.com/#' \
	-e 's#^ssh://git@github.com/#https://github.com/#' \
	-e 's#^git://github.com/#https://github.com/#')
s=$(printf '%s' "$s" | sed -e 's/[.]git$//' -e 's#/$##')
printf '%s\n' "$s"
