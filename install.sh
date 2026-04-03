#!/bin/bash
set -e

GITHUB_USER="captain-duster"
REPO="anwin-agent"
VERSION="v2.0.0"
BASE_URL="https://github.com/${GITHUB_USER}/${REPO}/releases/download/${VERSION}"
INSTALL_DIR="$HOME/.local/bin"
BINARY_NAME="anwin-agent"

echo ""
echo "  ╔══════════════════════════════════════════╗"
echo "  ║                                          ║"
echo '  ║                            _             ║'
echo '  ║   __ _  _ __  __      __  (_) _ __       ║'
echo '  ║  / _'"'"' | '"'"'_ \ \ \ /\ / /  | || '"'"'_ \     ║'
echo '  ║ | (_| | | | | \ V  V /   | || | | |     ║'
echo '  ║  \__,_|_| |_|  \_/\_/    |_||_| |_|     ║'
echo "  ║                                          ║"
echo "  ║         Local Code Sync Agent            ║"
echo "  ║             Version ${VERSION}               ║"
echo "  ║                                          ║"
echo "  ╚══════════════════════════════════════════╝"
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
        echo "  ✗ Unsupported architecture: $ARCH"
        exit 1
        ;;
    esac
    ;;
  darwin)
    case "$ARCH" in
      x86_64)  FILE="anwin-agent-mac-intel"          ;;
      arm64)   FILE="anwin-agent-mac-apple-silicon"  ;;
      *)
        echo "  ✗ Unsupported architecture: $ARCH"
        exit 1
        ;;
    esac
    ;;
  *)
    echo "  ✗ Unsupported OS: $OS"
    echo "  On Windows use: irm https://downloads.anwin.ai/agent/install.ps1 | iex"
    exit 1
    ;;
esac

echo "  ◆  Platform   →  $OS / $ARCH"
echo "  ◆  Binary     →  $FILE"
echo "  ◆  Installing →  $INSTALL_DIR/$BINARY_NAME"
echo ""

mkdir -p "$INSTALL_DIR"

TMP_FILE=$(mktemp)

if command -v curl &>/dev/null; then
  curl -f --progress-bar -L "${BASE_URL}/${FILE}" -o "$TMP_FILE"
elif command -v wget &>/dev/null; then
  wget --show-progress -q "${BASE_URL}/${FILE}" -O "$TMP_FILE"
else
  echo "  ✗ Neither curl nor wget found."
  exit 1
fi

chmod +x "$TMP_FILE"
mv "$TMP_FILE" "$INSTALL_DIR/$BINARY_NAME"

if [ "$OS" = "darwin" ]; then
  xattr -d com.apple.quarantine "$INSTALL_DIR/$BINARY_NAME" 2>/dev/null || true
fi

SHELL_RC=""
case "$SHELL" in
  */zsh)  SHELL_RC="$HOME/.zshrc" ;;
  */bash) SHELL_RC="$HOME/.bashrc" ;;
  *)      SHELL_RC="$HOME/.profile" ;;
esac

if ! echo "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_DIR"; then
  echo "" >> "$SHELL_RC"
  echo "export PATH=\"\$HOME/.local/bin:\$PATH\"" >> "$SHELL_RC"
  echo "  ◆  Added $INSTALL_DIR to PATH in $SHELL_RC"
fi

echo ""
echo "  ╔══════════════════════════════════════════╗"
echo "  ║   ✓  Installation complete!              ║"
echo "  ╠══════════════════════════════════════════╣"
echo "  ║                                          ║"
echo "  ║   Next steps:                            ║"
echo "  ║     1.  Restart your terminal            ║"
echo "  ║     2.  anwin-agent setup                ║"
echo "  ║     3.  anwin-agent start                ║"
echo "  ║                                          ║"
echo "  ║   Docs:  https://anwin.ai/docs/agent     ║"
echo "  ║                                          ║"
echo "  ╚══════════════════════════════════════════╝"
echo ""
