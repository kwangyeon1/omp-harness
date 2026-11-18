#!/bin/sh
# Compatibility entry: the integrated repeatable menu owns no installation logic.
set -eu
base=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
exec sh "$base/manage.sh" "$@"
