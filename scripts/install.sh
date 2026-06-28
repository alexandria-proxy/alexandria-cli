#!/bin/sh
set -eu

REPO="${ALEXANDRIA_REPO:-alexandria-proxy/alexandria-cli}"
APP_DIR="${ALEXANDRIA_HOME:-$HOME/.local/share/alexandria}"
BIN_DIR="$HOME/.local/bin"

detect_os() {
	case "$(uname -s)" in
		Linux) echo linux ;;
		Darwin) echo darwin ;;
		*)
			echo "unsupported os: $(uname -s)" >&2
			exit 1
			;;
	esac
}

detect_arch() {
	case "$(uname -m)" in
		x86_64 | amd64) echo amd64 ;;
		aarch64 | arm64) echo arm64 ;;
		*)
			echo "unsupported arch: $(uname -m)" >&2
			exit 1
			;;
	esac
}

sha256_of() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | awk '{print $1}'
	else
		shasum -a 256 "$1" | awk '{print $1}'
	fi
}

os="$(detect_os)"
arch="$(detect_arch)"

version="${ALEXANDRIA_VERSION:-}"
if [ -z "$version" ]; then
	version="$(curl -fsSL "https://api.github.com/repos/$REPO/releases?per_page=1" | grep -m1 '"tag_name"' | sed -E 's/.*"v?([^"]+)".*/\1/')"
fi
if [ -z "$version" ]; then
	echo "could not find a release for $REPO" >&2
	exit 1
fi
base="https://github.com/$REPO/releases/download/v${version#v}"

archive="alexandria-$os-$arch.tar.gz"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "downloading $archive ..."
curl -fsSL -o "$tmp/$archive" "$base/$archive"
curl -fsSL -o "$tmp/checksums.txt" "$base/checksums.txt"

want="$(grep " $archive\$" "$tmp/checksums.txt" | awk '{print $1}')"
got="$(sha256_of "$tmp/$archive")"
if [ -z "$want" ] || [ "$want" != "$got" ]; then
	echo "checksum verification failed for $archive" >&2
	exit 1
fi

mkdir -p "$APP_DIR" "$BIN_DIR"
tar -xzf "$tmp/$archive" -C "$APP_DIR"
chmod +x "$APP_DIR/alexandria-cli" "$APP_DIR/xray" "$APP_DIR/sing-box" 2>/dev/null || true
ln -sf "$APP_DIR/alexandria-cli" "$BIN_DIR/alexandria-cli"

echo "installed alexandria-cli to $APP_DIR"
case ":$PATH:" in
	*":$BIN_DIR:"*) echo "run: alexandria-cli" ;;
	*) echo "add $BIN_DIR to your PATH, then run: alexandria-cli" ;;
esac
