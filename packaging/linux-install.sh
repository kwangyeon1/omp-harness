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
# The extracted manager applies only package files; it never runs item scripts.
sh "$temp/manage.sh" bundle apply "$prefix"
printf 'Installed OMP Harness in: %s\n' "$prefix"
printf 'Open manager: sh "%s/manage.sh"\n' "$prefix"
printf 'Run OMP installer directly: sh "%s/install-omp-tui.sh"\n' "$prefix"
exit 0
__OMP_HARNESS_PAYLOAD__
