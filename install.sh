#!/bin/sh
# Cuck Code installer for macOS and Linux.
#
#   curl -fsSL https://raw.githubusercontent.com/ruskaruma/cuck-code/main/install.sh | sh
#
# Environment:
#   CUCK_VERSION      release tag to install (default: latest)
#   CUCK_INSTALL_DIR  where to put the binary (default: ~/.local/bin)
#   CUCK_SKIP_SETUP   set to 1 to skip the "get cucked?" question
set -eu

repo="ruskaruma/cuck-code"
dir="${CUCK_INSTALL_DIR:-$HOME/.local/bin}"

case "$(uname -s)" in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  *) echo "cuck: unsupported OS $(uname -s); on Windows use install.ps1" >&2; exit 1 ;;
esac
case "$(uname -m)" in
  x86_64 | amd64) arch=amd64 ;;
  aarch64 | arm64) arch=arm64 ;;
  *) echo "cuck: unsupported CPU $(uname -m)" >&2; exit 1 ;;
esac

if [ -n "${CUCK_VERSION:-}" ]; then
  base="https://github.com/$repo/releases/download/$CUCK_VERSION"
else
  base="https://github.com/$repo/releases/latest/download"
fi
asset="cuck-$os-$arch"

fetch() {
  if command -v curl >/dev/null 2>&1; then curl -fsSL "$1" -o "$2"
  elif command -v wget >/dev/null 2>&1; then wget -qO "$2" "$1"
  else echo "cuck: need curl or wget" >&2; exit 1
  fi
}

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT INT TERM
echo "Downloading $asset..."
fetch "$base/$asset" "$tmp/cuck"
fetch "$base/checksums.txt" "$tmp/checksums.txt"

want="$(grep " $asset\$" "$tmp/checksums.txt" | cut -d' ' -f1)"
if command -v sha256sum >/dev/null 2>&1; then got="$(sha256sum "$tmp/cuck" | cut -d' ' -f1)"
else got="$(shasum -a 256 "$tmp/cuck" | cut -d' ' -f1)"
fi
if [ -z "$want" ] || [ "$want" != "$got" ]; then
  echo "cuck: checksum mismatch for $asset; aborting" >&2
  exit 1
fi

mkdir -p "$dir"
chmod 755 "$tmp/cuck"
mv "$tmp/cuck" "$dir/cuck"
echo "Installed $("$dir/cuck" --version) to $dir/cuck"

case ":$PATH:" in
  *":$dir:"*) ;;
  *) echo "Note: $dir is not on your PATH. Add it, e.g.: echo 'export PATH=\"$dir:\$PATH\"' >> ~/.profile" ;;
esac

if [ -z "${CUCK_SKIP_SETUP:-}" ]; then
  # setup asks on /dev/tty, so this works even though stdin is the pipe from curl.
  PATH="$dir:$PATH" "$dir/cuck" setup </dev/null || true
fi
