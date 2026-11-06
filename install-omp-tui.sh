#!/bin/sh
set -eu
base=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
ask() { printf '\n%s [%s]: ' "$1" "$2"; IFS= read -r answer || exit 1; answer=${answer:-$2}; }
yesno() { while :; do ask "$1 (y/n)" "$2"; case "$answer" in y|Y) return 0;; n|N) return 1;; esac; done; }
echo '=== Official OMP installation wizard ==='
echo 'Enter accepts the displayed default. Ctrl+C cancels.'
if command -v omp >/dev/null 2>&1; then
 command -v omp; omp --version || true
 yesno 'Replace/update this installation? No keeps it and exits' n || exit 0
fi
ask 'Release tag (latest or vVERSION; older tags downgrade)' latest
ref=$answer
[ "$ref" = latest ] || printf '%s\n' "$ref" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$' || exit 1
ask 'Scope: user or system (system requires running this wizard as root)' user
case "$answer" in user) dest="$HOME/.local/bin";; system) [ "$(id -u)" = 0 ] || { echo 'Re-run as root for system installation.'; exit 1; }; dest=/usr/local/bin;; *) exit 1;; esac
ask 'Installation directory (absolute path)' "$dest"; dest=$answer
case "$dest" in /*) ;; *) echo 'Absolute path required'; exit 1;; esac
if [ -e "$dest/omp" ]; then yesno 'Destination already exists. Replace it?' n || exit 0; fi
ask 'Proxy: keep, or HTTPS proxy URL (credentials should use environment variables)' keep
if [ "$answer" != keep ]; then export HTTPS_PROXY="$answer" https_proxy="$answer"; fi
ask 'CA bundle: keep, or absolute file path' keep
if [ "$answer" != keep ]; then [ -r "$answer" ] || exit 1; export CURL_CA_BUNDLE="$answer"; fi
echo 'Missing dependencies may require sudo/root: curl, CA certificates, coreutils, sed, grep; Alpine also libstdc++ and libgcc.'
echo 'Arch dependency installation runs pacman -Syu and can update the whole system.'
yesno 'Allow dependency installation and any required sudo/system update? No exits for manual preparation' y || exit 0
addpath=n; yesno 'Add installation directory to your ~/.profile PATH?' y && addpath=y
reset=n; yesno 'Back up and reset ~/.omp/agent/settings.json if present?' n && reset=y
ask 'Optional tools to install: none, git, python, both' none; extras=$answer
case "$extras" in none|git|python|both) ;; *) exit 1;; esac
ask 'Shell completion: none, bash, zsh, fish (saved under installation directory)' none; completion=$answer
case "$completion" in none|bash|zsh|fish) ;; *) exit 1;; esac
ask 'After installation: exit, run, login' exit; action=$answer
case "$action" in exit|run|login) ;; *) exit 1;; esac
printf '\nRelease: %s\nDirectory: %s\nPATH: %s; reset settings: %s; tools: %s; completion: %s\n' "$ref" "$dest" "$addpath" "$reset" "$extras" "$completion"
yesno 'Install now?' y || exit 0
if [ "$extras" != none ]; then
 packages=''; case "$extras" in git) packages=git;; python) packages=python3;; both) packages='git python3';; esac
 root() { if [ "$(id -u)" = 0 ]; then "$@"; else sudo "$@"; fi; }
 if command -v apt-get >/dev/null; then root apt-get update; root apt-get install -y $packages
 elif command -v apk >/dev/null; then root apk add $packages
 elif command -v dnf >/dev/null; then root dnf install -y $packages
 else echo 'Optional tools require manual installation on this distro.'; exit 1; fi
fi
export PI_INSTALL_DIR="$dest"
if [ "$ref" = latest ]; then sh "$base/install-omp.sh"; else sh "$base/install-omp.sh" "$ref"; fi
if [ "$reset" = y ] && [ -f "$HOME/.omp/agent/settings.json" ]; then
 backup=$(mktemp "$HOME/.omp/agent/settings.json.backup.XXXXXX")
 cp "$HOME/.omp/agent/settings.json" "$backup"
 rm "$HOME/.omp/agent/settings.json"
 echo "Settings backup: $backup"
fi
if [ "$addpath" = y ]; then
 # Single-quote arbitrary paths before writing shell code.
 quoted=$(printf '%s' "$dest" | sed "s/'/'\\\\''/g")
 line="export PATH='$quoted':\$PATH"
 grep -Fqx "$line" "$HOME/.profile" 2>/dev/null || printf '\n%s\n' "$line" >> "$HOME/.profile"
 echo 'PATH updated in ~/.profile; start a new login shell.'
fi
if [ "$completion" != none ]; then
 "$dest/omp" completions "$completion" > "$dest/omp-completion.$completion"
 echo "Completion saved: $dest/omp-completion.$completion (load it from your shell configuration)."
fi
if [ "$action" = login ]; then echo 'Use /login in OMP to authenticate.'; fi
[ "$action" = exit ] || "$dest/omp"
