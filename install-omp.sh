#!/bin/sh
# Install the unmodified upstream binary, including bootstrap dependencies.
set -eu
fail() { printf 'Error: %s\n' "$*" >&2; exit 1; }
[ "$(uname -s)" = Linux ] || fail 'This script supports Linux only.'
case "$(uname -m)" in x86_64|amd64|aarch64|arm64) ;; *) fail 'Only x64 and ARM64 are supported.' ;; esac
ref=${1:-}
[ "$#" -le 1 ] || fail 'Usage: sh install-omp.sh [vVERSION]'
if [ -n "$ref" ]; then
    printf '%s\n' "$ref" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$' || fail 'Expected a release tag such as v18.8.9.'
fi
as_root() {
    if [ "$(id -u)" -eq 0 ]; then "$@"
    elif command -v sudo >/dev/null 2>&1; then sudo "$@"
    else fail 'Missing system dependencies: run as root or install sudo, then retry.'
    fi
}
need_packages=false
for tool in curl sha256sum mktemp sed grep cut tr chmod mv mkdir rm; do
    command -v "$tool" >/dev/null 2>&1 || need_packages=true
done
# curl needs a system trust store even when it is already present.
if [ ! -s /etc/ssl/certs/ca-certificates.crt ] && [ ! -s /etc/pki/tls/certs/ca-bundle.crt ]; then
    need_packages=true
fi
if command -v apk >/dev/null 2>&1; then
    apk info -e libstdc++ libgcc >/dev/null 2>&1 || need_packages=true
fi
if [ "$need_packages" = true ]; then
    echo 'Installing missing bootstrap dependencies (system package manager)...'
    if command -v apt-get >/dev/null 2>&1; then
        as_root apt-get update
        as_root env DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends ca-certificates curl coreutils sed grep
    elif command -v apk >/dev/null 2>&1; then
        as_root apk add --no-cache ca-certificates curl coreutils sed grep libstdc++ libgcc
    elif command -v dnf >/dev/null 2>&1; then
        as_root dnf install -y ca-certificates curl coreutils sed grep
    elif command -v yum >/dev/null 2>&1; then
        as_root yum install -y ca-certificates curl coreutils sed grep
    elif command -v zypper >/dev/null 2>&1; then
        as_root zypper --non-interactive install ca-certificates curl coreutils sed grep
    elif command -v pacman >/dev/null 2>&1; then
        as_root pacman -Syu --noconfirm --needed ca-certificates curl coreutils sed grep
    else
        fail 'Install ca-certificates, curl, coreutils, sed and grep with your package manager, then retry.'
    fi
fi
: "${HOME:?HOME must identify the installation user}"
export PI_INSTALL_DIR="${PI_INSTALL_DIR:-$HOME/.local/bin}"
# Always use the official service, regardless of any inherited override.
export PI_BUILD_URL=https://build.stencil.so
installer=$(mktemp)
trap 'rm -f "$installer"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
curl --fail --show-error --silent --location --proto '=https' --proto-redir '=https' --retry 3 --connect-timeout 20 --max-time 120 https://omp.sh/install -o "$installer"
if [ -n "$ref" ]; then
    sh "$installer" --binary --ref "$ref"
else
    sh "$installer" --binary
fi
"$PI_INSTALL_DIR/omp" --version
printf '\nInstalled official OMP: %s/omp\nAdd this directory to PATH or invoke the full path.\n' "$PI_INSTALL_DIR"
