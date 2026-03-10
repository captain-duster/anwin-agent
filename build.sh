#!/bin/bash
set -e

APP="anwin-agent"
VERSION="1.0.0"
OUT="./dist"

mkdir -p "$OUT"

echo ""
echo "Building ANWIN Agent v${VERSION} for all platforms..."
echo "──────────────────────────────────────────────────────"

build() {
  local GOOS=$1
  local GOARCH=$2
  local SUFFIX=$3
  local OUT_FILE="${OUT}/${APP}-${SUFFIX}"
  printf "  %-40s" "${GOOS}/${GOARCH} → ${APP}-${SUFFIX}"
  GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=0 \
    go build \
      -ldflags="-s -w -X main.version=${VERSION}" \
      -trimpath \
      -o "$OUT_FILE" \
      . 2>&1
  SIZE=$(du -sh "$OUT_FILE" | cut -f1)
  echo "✓  ${SIZE}"
}

build linux   amd64  linux-amd64
build linux   arm64  linux-arm64
build linux   386    linux-386
build darwin  amd64  mac-intel
build darwin  arm64  mac-apple-silicon
build windows amd64  windows-amd64.exe
build windows 386    windows-386.exe
build windows arm64  windows-arm64.exe

echo ""
echo "All binaries written to: ${OUT}/"
echo ""
ls -lh "$OUT"/
echo ""