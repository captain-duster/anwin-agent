#!/bin/bash
set -e

GITHUB_USER="captain-duster"
REPO="anwin-agent"
VERSION="v1.0.0"
BASE_URL="https://github.com/${GITHUB_USER}/${REPO}/releases/download/${VERSION}"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="anwin-agent"

echo ""
echo "  ┌─────────────────────────────────┐"
echo "  │     ANWIN Agent Installer       │"
echo "  │     Version ${VERSION}          |"
echo "  └─────────────────────────────────┘"
echo ""

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$OS" in
  linux)
    case "$ARCH" in
      x86_64)          FILE="anwin-agent-linux-amd64"  ;;
      aarch64|arm64)   FILE="anwin-agent-linux-arm64"  ;;
      i386|i686)       FILE="anwin-agent-linux-386"    ;;
      *)
        echo "  ERROR: Unsupported Linux architecture: $ARCH"
        exit 1
        ;;
    esac
    ;;
  darwin)
    case "$ARCH" in
      x86_64)  FILE="anwin-agent-mac-intel"          ;;
      arm64)   FILE="anwin-agent-mac-apple-silicon"  ;;
      *)
        echo "  ERROR: Unsupported Mac architecture: $ARCH"
        exit 1
        ;;
    esac
    ;;
  *)
    echo "  ERROR: Unsupported OS: $OS"
    echo "  On Windows use: irm https://downloads.anwin.ai/agent/install.ps1 | iex"
    exit 1
    ;;
esac

echo "  Detected: $OS / $ARCH"
echo "  Downloading: $FILE"
echo ""

TMP_FILE=$(mktemp)

if command -v curl &>/dev/null; then
  curl -fsSL "${BASE_URL}/${FILE}" -o "$TMP_FILE"
elif command -v wget &>/dev/null; then
  wget -q "${BASE_URL}/${FILE}" -O "$TMP_FILE"
else
  echo "  ERROR: Neither curl nor wget found. Please install one and retry."
  exit 1
fi

chmod +x "$TMP_FILE"

if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP_FILE" "${INSTALL_DIR}/${BINARY_NAME}"
else
  echo "  Requesting sudo to install to ${INSTALL_DIR}..."
  sudo mv "$TMP_FILE" "${INSTALL_DIR}/${BINARY_NAME}"
fi

if [ "$OS" = "darwin" ]; then
  xattr -d com.apple.quarantine "${INSTALL_DIR}/${BINARY_NAME}" 2>/dev/null || true
fi

echo "  ✓ ANWIN Agent installed successfully"
echo ""
echo "  Next step:"
echo "    anwin-agent setup"
echo ""