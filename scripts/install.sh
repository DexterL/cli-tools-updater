#!/bin/sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
prefix=${PREFIX:-"$HOME/.local/bin"}
mkdir -p "$prefix"
install -m 755 "$script_dir/upcli" "$prefix/upcli"
printf 'Installed upcli to %s/upcli\n' "$prefix"
case ":${PATH}:" in
  *":$prefix:"*) ;;
  *) printf 'Add this to your shell profile if needed: export PATH="%s:$PATH"\n' "$prefix" ;;
esac
