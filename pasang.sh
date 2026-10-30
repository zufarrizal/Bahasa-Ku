#!/usr/bin/env bash
# Skrip Pemasang Otomatis Bahasa Ku untuk Linux & macOS
set -e

echo "=================================================="
echo "    Pemasang Otomatis Bahasa Ku (Linux / macOS)   "
echo "=================================================="

INSTALL_DIR="$HOME/.bahasa-ku"
mkdir -p "$INSTALL_DIR"

OS="$(uname -s)"
ARCH="$(uname -m)"

if [ "$OS" = "Linux" ]; then
    URL="https://github.com/zufarrizal/Bahasa-Ku/releases/download/v0.3.1/ku-linux-amd64"
elif [ "$OS" = "Darwin" ]; then
    URL="https://github.com/zufarrizal/Bahasa-Ku/releases/download/v0.3.1/ku-darwin-arm64"
else
    echo "OS tidak didukung oleh skrip ini: $OS"
    exit 1
fi

echo "[1/3] Mengunduh alat Bahasa Ku..."
curl -sL "$URL" -o "$INSTALL_DIR/ku"
chmod +x "$INSTALL_DIR/ku"

echo "[2/3] Mendaftarkan 'ku' ke PATH Anda..."
PROFILE="$HOME/.bashrc"
if [ -f "$HOME/.zshrc" ]; then
    PROFILE="$HOME/.zshrc"
fi

if ! grep -q "$INSTALL_DIR" "$PROFILE"; then
    echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$PROFILE"
    echo "      -> Berhasil ditambahkan ke $PROFILE"
fi

echo "[3/3] Memeriksa instalasi VS Code..."
if command -v code &> /dev/null; then
    echo "      -> Memasang ekstensi VS Code otomatis..."
    VSIX_URL="https://github.com/zufarrizal/Bahasa-Ku/releases/download/v0.3.1/bahasa-ku-vscode-0.2.0.vsix"
    curl -sL "$VSIX_URL" -o "$INSTALL_DIR/bahasa-ku.vsix"
    code --install-extension "$INSTALL_DIR/bahasa-ku.vsix" --force > /dev/null 2>&1 || true
    rm -f "$INSTALL_DIR/bahasa-ku.vsix"
    echo "      -> Ekstensi VS Code Bahasa Ku berhasil terpasang!"
fi

echo ""
echo "=================================================="
echo "  Pemasangan Selesai 100%! Selamat Menggunakan Ku! 🎉"
echo "=================================================="
echo "Buka terminal baru atau ketik: source $PROFILE"
echo "Lalu coba: ku"
