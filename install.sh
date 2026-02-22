#!/usr/bin/env bash
# Claude TELEX — Installer for macOS/Linux
# Usage: curl -fsSL https://raw.githubusercontent.com/nguyenhx2/claude-telex/main/install.sh | bash

set -euo pipefail

REPO="nguyenhx2/claude-telex"
INSTALL_DIR="${HOME}/.local/bin"
BIN_NAME="claude-telex"

echo ""
echo "  ╔══════════════════════════════════════╗"
echo "  ║    Claude TELEX — Vietnamese IME Fix    ║"
echo "  ╚══════════════════════════════════════╝"
echo ""

# Detect OS and arch
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

echo "[1/3] Fetching latest release..."
RELEASE_URL="https://api.github.com/repos/${REPO}/releases/latest"
RELEASE_JSON=$(curl -fsSL "$RELEASE_URL")
VERSION=$(echo "$RELEASE_JSON" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
echo "  Latest version: $VERSION"

# Find download URL
ASSET_PATTERN="${BIN_NAME}_.*_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL=$(echo "$RELEASE_JSON" | grep "browser_download_url" | grep -i "${OS}_${ARCH}" | head -1 | sed 's/.*"\(https[^"]*\)".*/\1/')

if [ -z "$DOWNLOAD_URL" ]; then
  echo "  Error: No matching asset for ${OS}/${ARCH}"
  exit 1
fi

echo "[2/3] Downloading..."
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

curl -fsSL "$DOWNLOAD_URL" -o "${TMP_DIR}/claude-telex.tar.gz"
tar xzf "${TMP_DIR}/claude-telex.tar.gz" -C "$TMP_DIR"

echo "[3/3] Installing to ${INSTALL_DIR}..."
mkdir -p "$INSTALL_DIR"
mv "${TMP_DIR}/${BIN_NAME}" "${INSTALL_DIR}/${BIN_NAME}"
chmod +x "${INSTALL_DIR}/${BIN_NAME}"

# Check PATH
if ! echo "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_DIR"; then
  SHELL_RC=""
  case "$SHELL" in
    */zsh)  SHELL_RC="$HOME/.zshrc" ;;
    */bash) SHELL_RC="$HOME/.bashrc" ;;
    *)      SHELL_RC="$HOME/.profile" ;;
  esac
  echo "export PATH=\"\$PATH:${INSTALL_DIR}\"" >> "$SHELL_RC"
  echo "  Added ${INSTALL_DIR} to PATH in ${SHELL_RC}"
  echo "  Run: source ${SHELL_RC}"
fi

echo ""
echo "  ✓ Claude TELEX ${VERSION} installed!"
echo ""
echo "  Run:  ${BIN_NAME}"
echo "  Path: ${INSTALL_DIR}/${BIN_NAME}"
echo ""
