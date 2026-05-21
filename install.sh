#!/bin/sh
# install.sh — install clibo, the online_clipboard CLI.
#
# Usage:
#   curl -sSL https://clipboard.lab.rm-info.fr/install.sh | sh
#
# Overrides (env vars):
#   CLIBO_BASE   Source server (default: https://clipboard.lab.rm-info.fr).
#                Point this at a self-hosted instance to install from a fork.
#   CLIBO_BIN    Target binary path (default: /usr/local/bin/clibo).
#                Use ~/.local/bin/clibo to avoid sudo.

set -eu

CLIBO_BASE="${CLIBO_BASE:-https://clipboard.lab.rm-info.fr}"
CLIBO_BIN="${CLIBO_BIN:-/usr/local/bin/clibo}"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
    x86_64|amd64) arch=amd64 ;;
    aarch64|arm64) arch=arm64 ;;
    *)
        echo "clibo: unsupported architecture '$arch'" >&2
        echo "       see $CLIBO_BASE/cli for manual downloads" >&2
        exit 1 ;;
esac
case "$os" in
    linux|darwin) ;;
    *)
        echo "clibo: unsupported OS '$os'" >&2
        echo "       see $CLIBO_BASE/cli for manual downloads" >&2
        exit 1 ;;
esac

target="${os}-${arch}"
url="${CLIBO_BASE}/cli/${target}"

if ! command -v curl >/dev/null 2>&1; then
    echo "clibo: curl is required" >&2
    exit 1
fi

tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT

printf 'clibo: downloading %s\n' "$url"
if ! curl -fL --progress-bar -o "$tmp" "$url"; then
    echo "clibo: download failed" >&2
    exit 1
fi
chmod +x "$tmp"

target_dir=$(dirname "$CLIBO_BIN")
mkdir -p "$target_dir" 2>/dev/null || true
if [ -w "$target_dir" ]; then
    mv "$tmp" "$CLIBO_BIN"
else
    printf 'clibo: %s is not writable, escalating with sudo\n' "$target_dir"
    sudo mv "$tmp" "$CLIBO_BIN"
fi
trap - EXIT

printf 'clibo: installed %s\n' "$CLIBO_BIN"
if ! "$CLIBO_BIN" --version 2>/dev/null; then
    echo "clibo: installed binary did not respond to --version (is $CLIBO_BIN in PATH?)"
fi
