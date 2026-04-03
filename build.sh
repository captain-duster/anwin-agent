#!/bin/bash
set -e

APP="anwin-agent"
VERSION="2.0.0"
OUT="./dist"

rm -rf "$OUT"
mkdir -p "$OUT"

echo ""
echo "  Building ANWIN Agent v$VERSION"
echo "  ──────────────────────────────────────────"

platforms=(
  "linux   amd64  ${APP}-linux-amd64"
  "linux   arm64  ${APP}-linux-arm64"
  "linux   386    ${APP}-linux-386"
  "darwin  amd64  ${APP}-mac-intel"
  "darwin  arm64  ${APP}-mac-apple-silicon"
  "windows amd64  ${APP}-windows-amd64.exe"
  "windows arm64  ${APP}-windows-arm64.exe"
  "windows 386    ${APP}-windows-386.exe"
)

for entry in "${platforms[@]}"; do
  read -r os arch file <<< "$entry"
  printf "  %-14s → %s " "$os/$arch" "$file"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -ldflags="-s -w" -o "$OUT/$file" .
  size=$(ls -lh "$OUT/$file" | awk '{print $5}')
  echo "($size)"
done

echo ""
echo "  Done. Files in $OUT/"
echo ""
ls -lh "$OUT/"
