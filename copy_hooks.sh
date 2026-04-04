#!/bin/sh
set -e
src_dir="$(git rev-parse --show-toplevel)/hooks"
dest_dir="$(git rev-parse --show-toplevel)/.git/hooks"
mkdir -p "$dest_dir"
for hook in "$src_dir"/*; do
    cp "$hook" "$dest_dir/"
done
echo "Hooks installed."
