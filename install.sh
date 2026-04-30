#!/bin/sh
# docksmith installer — downloads a release binary into INSTALL_DIR (default: /usr/local/bin).
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/permanu/Docksmith/main/install.sh | sh
#   curl -fsSL https://raw.githubusercontent.com/permanu/Docksmith/main/install.sh | sh -s -- v0.4.2
#   INSTALL_DIR=$HOME/.local/bin curl -fsSL ... | sh
#
# Source:    https://github.com/permanu/Docksmith
# Releases:  https://github.com/permanu/Docksmith/releases

set -eu

REPO="permanu/Docksmith"
BINARY="docksmith"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION="${1:-latest}"

err() { printf 'docksmith-install: %s\n' "$*" >&2; exit 1; }
info() { printf 'docksmith-install: %s\n' "$*"; }

need() { command -v "$1" >/dev/null 2>&1 || err "missing required tool: $1"; }
need uname
need tar
if command -v curl >/dev/null 2>&1; then
  fetch() { curl -fsSL "$1" -o "$2"; }
  fetch_stdout() { curl -fsSL "$1"; }
elif command -v wget >/dev/null 2>&1; then
  fetch() { wget -qO "$2" "$1"; }
  fetch_stdout() { wget -qO - "$1"; }
else
  err "need curl or wget"
fi

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
  linux|darwin) ;;
  *) err "unsupported OS: $os (supported: linux, darwin). Use 'go install github.com/permanu/docksmith/cmd/docksmith@latest' on Windows." ;;
esac

arch_raw=$(uname -m)
case "$arch_raw" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) err "unsupported architecture: $arch_raw (supported: amd64, arm64)" ;;
esac

if [ "$VERSION" = "latest" ]; then
  VERSION=$(fetch_stdout "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep -E '"tag_name"' | head -1 | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
  [ -n "$VERSION" ] || err "could not resolve latest release"
fi

archive="${BINARY}_${os}_${arch}.tar.gz"
url="https://github.com/${REPO}/releases/download/${VERSION}/${archive}"
checksums_url="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

info "downloading ${archive} (${VERSION})"
fetch "$url" "${tmp}/${archive}" || err "download failed: ${url}"

# Verify checksum if checksums.txt is reachable. Don't fail if it's not (older
# releases may not have one); do fail if it's there and doesn't match.
if fetch "$checksums_url" "${tmp}/checksums.txt" 2>/dev/null; then
  expected=$(grep " ${archive}\$" "${tmp}/checksums.txt" | awk '{print $1}')
  if [ -n "$expected" ]; then
    if command -v sha256sum >/dev/null 2>&1; then
      actual=$(sha256sum "${tmp}/${archive}" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
      actual=$(shasum -a 256 "${tmp}/${archive}" | awk '{print $1}')
    else
      actual=""
    fi
    if [ -n "$actual" ] && [ "$actual" != "$expected" ]; then
      err "checksum mismatch for ${archive}: expected ${expected}, got ${actual}"
    fi
    [ -n "$actual" ] && info "checksum verified"
  fi
fi

tar -xzf "${tmp}/${archive}" -C "$tmp"
[ -f "${tmp}/${BINARY}" ] || err "archive did not contain ${BINARY}"

dest="${INSTALL_DIR}/${BINARY}"
if [ -w "$INSTALL_DIR" ] || ([ ! -e "$INSTALL_DIR" ] && mkdir -p "$INSTALL_DIR" 2>/dev/null); then
  install -m 0755 "${tmp}/${BINARY}" "$dest"
elif command -v sudo >/dev/null 2>&1; then
  info "INSTALL_DIR ${INSTALL_DIR} not writable, escalating with sudo"
  sudo install -m 0755 "${tmp}/${BINARY}" "$dest"
else
  err "INSTALL_DIR ${INSTALL_DIR} not writable and sudo unavailable. Re-run with INSTALL_DIR=\$HOME/.local/bin"
fi

info "installed ${BINARY} ${VERSION} to ${dest}"
case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) ;;
  *) info "note: ${INSTALL_DIR} is not on \$PATH. Add it to your shell profile to use 'docksmith' directly." ;;
esac
