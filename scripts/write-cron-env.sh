#!/bin/bash
set -euo pipefail

if [ "$#" -ne 1 ]; then
  echo "Usage: $0 output-file" >&2
  exit 2
fi

umask 077
unset BASH_ENV
export -p > "$1"
chmod 600 "$1"
