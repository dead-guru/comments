#!/usr/bin/env bash
set -euo pipefail

version="${1:?version is required}"
goos="${2:?goos is required}"
goarch="${3:?goarch is required}"

binary="deadcomments"
if [[ "$goos" == "windows" ]]; then
  binary="deadcomments.exe"
fi

package="deadcomments_${version}_${goos}_${goarch}"
out_dir="dist/${package}"

rm -rf "$out_dir"
mkdir -p "$out_dir"

CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
  go build -trimpath -ldflags="-s -w" -o "${out_dir}/${binary}" ./cmd/server

cp README.md "$out_dir/"
cp AGENTS.md "$out_dir/"
cp -R migrations "$out_dir/migrations"
mkdir -p "$out_dir/internal"
cp -R internal/templates "$out_dir/internal/templates"
cp -R internal/static "$out_dir/internal/static"
cp -R internal/widget "$out_dir/internal/widget"

tar -C dist -czf "dist/${package}.tar.gz" "$package"
rm -rf "$out_dir"

