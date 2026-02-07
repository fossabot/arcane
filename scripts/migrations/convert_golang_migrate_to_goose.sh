#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 2 ]; then
  echo "Usage: $0 <source_dir> <dest_dir>" >&2
  exit 1
fi

src_dir="$1"
dest_dir="$2"

if [ ! -d "$src_dir" ]; then
  echo "Source directory not found: $src_dir" >&2
  exit 1
fi

rm -rf "$dest_dir"
mkdir -p "$dest_dir"

shopt -s nullglob
up_files=("$src_dir"/*_*.up.sql)
shopt -u nullglob

if [ "${#up_files[@]}" -eq 0 ]; then
  echo "No .up.sql files found in $src_dir" >&2
  exit 1
fi

for up_file in "${up_files[@]}"; do
  base_name=$(basename "$up_file")
  prefix=${base_name%%_*}
  rest=${base_name#*_}
  rest=${rest%.up.sql}

  if [ -z "$prefix" ] || [ "$prefix" = "$base_name" ]; then
    echo "Unable to parse migration prefix from $base_name" >&2
    exit 1
  fi

  number=$((10#$prefix))
  padded=$(printf "%05d" "$number")

  down_file="$src_dir/${prefix}_${rest}.down.sql"
  if [ ! -f "$down_file" ]; then
    echo "Missing down migration for $base_name" >&2
    exit 1
  fi

  out_file="$dest_dir/${padded}_${rest}.sql"

  {
    echo "-- +goose Up"
    cat "$up_file"
    echo ""
    echo "-- +goose Down"
    cat "$down_file"
  } > "$out_file"

done

echo "Converted ${#up_files[@]} migrations from $src_dir to $dest_dir"
