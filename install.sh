#!/usr/bin/env bash
set -e

# ==============================================================================
# mncode 1-Line Universal Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/mncuchiinhuttt/mncode/main/install.sh | bash
# ==============================================================================

REPO="mncuchiinhuttt/mncode"
BINARY_NAME="mncode"

# Colors
BOLD="\033[1m"
GREEN="\033[1;32m"
CYAN="\033[1;36m"
PINK="\033[1;38;5;218m"
YELLOW="\033[1;33m"
RED="\033[1;31m"
RESET="\033[0m"

echo -e "${PINK}"
echo "  __  __ _  _  ____ ___  ____  ____ "
echo " (  \/  ( \( )/ ___/ _ \(    \(  __)  mncode CLI Installer"
echo "  )    ( )  (( (__ )(_) )) D ( ) _)   Claude Code Golang Clone"
echo " (_/\/\_(_)\_)\____\___/(____/(____) "
echo -e "${RESET}"

# 1. Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
  darwin) OS="darwin" ;;
  linux)  OS="linux" ;;
  *)
    echo -e "${RED}Error: Unsupported operating system: $OS${RESET}"
    exit 1
    ;;
esac

# 2. Detect Architecture
ARCH="$(uname -m | tr '[:upper:]' '[:lower:]')"
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)
    echo -e "${RED}Error: Unsupported architecture: $ARCH${RESET}"
    exit 1
    ;;
esac

echo -e "${CYAN}Detected Platform:${RESET} ${OS}-${ARCH}"

# 3. Determine Installation Destination
INSTALL_DIR=""
if [ -w "/usr/local/bin" ]; then
  INSTALL_DIR="/usr/local/bin"
elif [ -d "/usr/local/bin" ] && sudo -n true 2>/dev/null; then
  INSTALL_DIR="/usr/local/bin"
else
  INSTALL_DIR="$HOME/.local/bin"
  mkdir -p "$INSTALL_DIR"
fi

TARGET="$INSTALL_DIR/$BINARY_NAME"
TMP_DIR="$(mktemp -d)"
TMP_FILE="$TMP_DIR/$BINARY_NAME"

cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

# 4. Check if Go is installed (for building from source or downloading release)
if command -v go >/dev/null 2>&1; then
  echo -e "${CYAN}Building latest ${BINARY_NAME} from source using Go...${RESET}"
  git clone --depth 1 "https://github.com/${REPO}.git" "$TMP_DIR/src" 2>/dev/null || true
  if [ -d "$TMP_DIR/src" ]; then
    (cd "$TMP_DIR/src" && go build -ldflags="-s -w" -o "$TMP_FILE" ./cmd/mncode)
  fi
fi

# Fallback: Download pre-built release binary from GitHub
if [ ! -f "$TMP_FILE" ]; then
  TAG="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases" 2>/dev/null | grep -m1 '"tag_name":' | cut -d'"' -f4 || true)"
  if [ -z "$TAG" ]; then
    TAG="v0.1.2.6-beta"
  fi
  RELEASE_URL="https://github.com/${REPO}/releases/download/${TAG}/${BINARY_NAME}-${OS}-${ARCH}"
  echo -e "${CYAN}Downloading pre-built ${TAG} binary from GitHub Releases...${RESET}"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$RELEASE_URL" -o "$TMP_FILE" || true
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$TMP_FILE" "$RELEASE_URL" || true
  fi
fi

if [ ! -f "$TMP_FILE" ] || [ ! -s "$TMP_FILE" ]; then
  echo -e "${RED}Failed to build or download binary.${RESET}"
  echo -e "Please ensure you have access to https://github.com/${REPO} or Go installed."
  exit 1
fi

# 5. Install Binary
chmod +x "$TMP_FILE"
echo -e "${CYAN}Installing to ${TARGET}...${RESET}"

if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP_FILE" "$TARGET"
else
  echo -e "${YELLOW}Elevating permissions to install to ${INSTALL_DIR}...${RESET}"
  sudo mv "$TMP_FILE" "$TARGET"
fi

# 6. Ensure PATH is configured
SHELL_NAME="$(basename "$SHELL")"
RC_FILE=""

case "$SHELL_NAME" in
  zsh)  RC_FILE="$HOME/.zshrc" ;;
  bash) RC_FILE="$HOME/.bashrc" ;;
  *)    RC_FILE="$HOME/.profile" ;;
esac

if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
  echo -e "${YELLOW}Adding ${INSTALL_DIR} to PATH in ${RC_FILE}...${RESET}"
  echo "export PATH=\"\$PATH:${INSTALL_DIR}\"" >> "$RC_FILE"
  export PATH="$PATH:${INSTALL_DIR}"
fi

# 7. Initialize default config directory
mkdir -p "$HOME/.mncode"

echo
echo -e "${GREEN}✓ Successfully installed ${BOLD}mncode${RESET}${GREEN} to ${TARGET}!${RESET}"
echo
echo -e "Run ${BOLD}mncode${RESET} to start pair programming:"
echo -e "  ${CYAN}mncode${RESET}"
echo
