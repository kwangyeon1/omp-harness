#!/bin/sh
set -eu
prefix=${HOME:?HOME is required}/.local/share/omp-harness
while [ "$#" -gt 0 ]; do
    case "$1" in
        --prefix) [ "$#" -ge 2 ] || { echo 'Missing --prefix value' >&2; exit 2; }; prefix=$2; shift 2 ;;
        --help) echo 'Usage: sh PACKAGE.run [--prefix /absolute/directory]'; exit 0 ;;
        *) echo "Unknown argument: $1" >&2; exit 2 ;;
    esac
done
case "$prefix" in /*) ;; *) echo 'Installation directory must be absolute' >&2; exit 2 ;; esac
case "$prefix" in /) echo 'Cannot install in /' >&2; exit 2 ;; esac
case "$(uname -s)" in Linux) ;; *) echo 'Linux is required' >&2; exit 1 ;; esac
case "$(uname -m)" in x86_64|amd64|aarch64|arm64) ;; *) echo 'x64 or ARM64 is required' >&2; exit 1 ;; esac
temp=$(mktemp -d)
trap 'rm -rf "$temp"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
tail -n +@PAYLOAD_LINE@ "$0" | tar -xz -C "$temp"
# Validate the complete payload before touching an existing installation.
for name in install-omp.sh install-omp-tui.sh uninstall.sh VERSION README.md; do
    [ -f "$temp/$name" ] || { echo "Missing payload: $name" >&2; exit 1; }
done
if [ -e "$prefix" ]; then
    [ -d "$prefix" ] && [ ! -L "$prefix" ] || { echo 'Destination must be a real directory' >&2; exit 1; }
    if [ ! -f "$prefix/.omp-harness-package" ] && [ -n "$(ls -A "$prefix")" ]; then
        echo 'Choose an empty directory or an existing OMP Harness Scripts installation' >&2
        exit 1
    fi
fi
mkdir -p "$prefix"
for name in install-omp.sh install-omp-tui.sh uninstall.sh VERSION README.md; do
    cp "$temp/$name" "$prefix/$name"
done
chmod 755 "$prefix/install-omp.sh" "$prefix/install-omp-tui.sh" "$prefix/uninstall.sh"
printf '%s\n' 'omp-harness-script-installer' > "$prefix/.omp-harness-package"
printf 'Installed OMP Harness Scripts in: %s\n' "$prefix"
printf 'Run the wizard: sh "%s/install-omp-tui.sh"\n' "$prefix"
printf 'Uninstall scripts: sh "%s/uninstall.sh"\n' "$prefix"
exit 0
__OMP_HARNESS_PAYLOAD__
