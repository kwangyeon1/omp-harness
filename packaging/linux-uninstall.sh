#!/bin/sh
set -eu
base=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
[ "$(cat "$base/.omp-harness-package" 2>/dev/null)" = omp-harness-script-installer ] || {
    echo 'Not an OMP Harness Scripts installation' >&2; exit 1;
}
# Remove only files owned by this package. Preserve OMP and user data.
for name in install-omp.sh install-omp-tui.sh VERSION README.md .omp-harness-package uninstall.sh; do
    rm -f "$base/$name"
done
if ! rmdir "$base" 2>/dev/null; then
    printf 'Preserved directory containing other files: %s\n' "$base"
fi
echo 'OMP Harness Scripts removed. OMP and its settings were retained.'
