#!/bin/bash
set -e

GITHUB_USER="captain-duster"
REPO="anwin-agent"
VERSION="v1.0.0"
BASE_URL="https://github.com/${GITHUB_USER}/${REPO}/releases/download/${VERSION}"
INSTALL_DIR="$HOME/.local/bin"
BINARY_NAME="anwin-agent"

echo ""
echo "  ╔═════════════════════════════════════════════════════════════════╗"
echo "  ║     █████╗ ███╗   ██╗██╗    ██╗██╗███╗   ██╗    █████╗ ██╗      ║"
echo "  ║     ██╔══██╗████╗  ██║██║    ██║██║████╗  ██║   ██╔══██╗██║     ║"
echo "  ║     ███████║██╔██╗ ██║██║ █╗ ██║██║██╔██╗ ██║   ███████║██║     ║"
echo "  ║     ██╔══██║██║╚██╗██║██║███╗██║██║██║╚██╗██║   ██╔══██║██║     ║"
echo "  ║     ██║  ██║██║ ╚████║╚███╔███╔╝██║██║ ╚████║   ██║  ██║██║     ║"
echo "  ║     ╚═╝  ╚═╝╚═╝  ╚═══╝ ╚══╝╚══╝ ╚═╝╚═╝  ╚═══╝  ╚═╝  ╚═╝╚═╝      ║"
echo "  ║                                                                 ║"
echo "  ║                            anwin.ai                             ║"
echo "  ║                                                                 ║"
echo "  ║                local code sync agent  ·  ${VERSION}                 ║"
echo "  ╚═════════════════════════════════════════════════════════════════╝"
echo ""

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$OS" in
  linux)
    case "$ARCH" in
      x86_64)        FILE="anwin-agent-linux-amd64" ;;
      aarch64|arm64) FILE="anwin-agent-linux-arm64" ;;
      i386|i686)     FILE="anwin-agent-linux-386"   ;;
      *)
        echo "  ✗  unsupported linux architecture: $ARCH"
        exit 1
        ;;
    esac
    ;;
  darwin)
    case "$ARCH" in
      x86_64) FILE="anwin-agent-mac-intel"         ;;
      arm64)  FILE="anwin-agent-mac-apple-silicon" ;;
      *)
        echo "  ✗  unsupported mac architecture: $ARCH"
        exit 1
        ;;
    esac
    ;;
  *)
    echo "  ✗  unsupported os: $OS"
    echo "  on windows use: irm https://downloads.anwin.ai/agent/install.ps1 | iex"
    exit 1
    ;;
esac

echo "  ◆  platform   →  $OS / $ARCH"
echo "  ◆  binary     →  $FILE"
echo "  ◆  installing →  ${INSTALL_DIR}/${BINARY_NAME}"
echo ""

mkdir -p "$INSTALL_DIR"

TMP_FILE=$(mktemp)

if command -v curl &>/dev/null; then
  curl -f -L --progress-bar "${BASE_URL}/${FILE}" -o "$TMP_FILE" 2>&1 < /dev/null
elif command -v wget &>/dev/null; then
  wget -q --show-progress "${BASE_URL}/${FILE}" -O "$TMP_FILE" < /dev/null
else
  echo "  ✗  neither curl nor wget found. please install one and retry."
  exit 1
fi

chmod +x "$TMP_FILE"
mv "$TMP_FILE" "${INSTALL_DIR}/${BINARY_NAME}"

if [ "$OS" = "darwin" ]; then
  xattr -d com.apple.quarantine "${INSTALL_DIR}/${BINARY_NAME}" 2>/dev/null || true
fi

SHELL_RC=""
case "$SHELL" in
  */zsh)  SHELL_RC="$HOME/.zshrc"   ;;
  */bash) SHELL_RC="$HOME/.bashrc"  ;;
  *)      SHELL_RC="$HOME/.profile" ;;
esac

if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
  echo "" >> "$SHELL_RC"
  echo "export PATH=\"\$HOME/.local/bin:\$PATH\"" >> "$SHELL_RC"
  export PATH="$INSTALL_DIR:$PATH"
fi

echo ""
echo "  ╔═════════════════════════════════════════════════════════════════╗"
echo "  ║   ✓  installation complete!                                    ║"
echo "  ╠═════════════════════════════════════════════════════════════════╣"
echo "  ║                                                                 ║"
echo "  ║   next steps:                                                   ║"
echo "  ║     1.  anwin-agent setup                                       ║"
echo "  ║     2.  anwin-agent start                                       ║"
echo "  ║                                                                 ║"
echo "  ║   docs:  https://anwin.ai/docs/agent                           ║"
echo "  ║                                                                 ║"
echo "  ╚═════════════════════════════════════════════════════════════════╝"
echo ""