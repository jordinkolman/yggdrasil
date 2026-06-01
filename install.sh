#!/usr/bin/env bash
set -e

REPO="jordinkolman/yggdrasil"
# Dynamically fetch the latest release tag from GitHub API
VERSION=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$VERSION" ]; then
  echo "Failed to fetch latest version. Please check the repository URL."

  exit 1
fi


OS="$(uname -s)"
ARCH="$(uname -m)"

case "${OS}" in
    Linux*)     PLATFORM="linux" ;;
    Darwin*)    PLATFORM="darwin" ;;
    *)          echo "Unsupported OS: ${OS}"; exit 1 ;;
esac

case "${ARCH}" in
    x86_64)     ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *)          echo "Unsupported Architecture: ${ARCH}"; exit 1 ;;
esac

ARCHIVE_NAME="yggdrasil_${PLATFORM}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE_NAME}"

echo "Downloading Yggdrasil ${VERSION} for ${PLATFORM} ${ARCH}..."
TMP_DIR=$(mktemp -d)
cd "$TMP_DIR"

curl -sSL -o "$ARCHIVE_NAME" "$DOWNLOAD_URL"
tar -xzf "$ARCHIVE_NAME"


echo "Installing to /usr/local/bin (may require sudo)..."
sudo mv yggdrasil /usr/local/bin/

sudo chmod +x /usr/local/bin/yggdrasil

cd - > /dev/null
rm -rf "$TMP_DIR"

BASHRC="$HOME/.bashrc"

HOOK_MARKER="# Yggdrasil - Tmux Session Manager"

if ! grep -q "$HOOK_MARKER" "$BASHRC"; then
    echo "Injecting shell hook into $BASHRC..."
    cat << 'EOF' >> "$BASHRC"

# ---------------------------------------------------------
# Yggdrasil - Tmux Session Manager
# ---------------------------------------------------------
eval "$(yggdrasil init bash)"
if [ -z "$TMUX" ]; then
    ygg
fi
EOF
else
    echo "Shell hook already detected in $BASHRC. Skipping."
fi

echo "Installation complete! Run 'source ~/.bashrc' or restart your terminal."
