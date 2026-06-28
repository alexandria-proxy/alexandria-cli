#!/bin/sh
set -eu

REPO="${ALEXANDRIA_REPO:-alexandria-proxy/alexandria-cli}"
APP_DIR="${ALEXANDRIA_HOME:-$HOME/.local/share/alexandria}"
BIN_DIR="$HOME/.local/bin"

force="${ALEXANDRIA_FORCE:-}"
for arg in "$@"; do
	case "$arg" in
		-f | --force) force=1 ;;
		*)
			echo "unknown option: $arg" >&2
			exit 1
			;;
	esac
done

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

config_dir() {
	case "$os" in
		darwin) printf '%s/Library/Application Support/alexandria' "$HOME" ;;
		*) printf '%s/alexandria' "${XDG_CONFIG_HOME:-$HOME/.config}" ;;
	esac
}

stop_daemon() {
	pidfile="$(config_dir)/control.pid"
	[ -f "$pidfile" ] || return 0
	pid="$(cat "$pidfile" 2>/dev/null || true)"
	[ -n "$pid" ] || return 0
	kill -0 "$pid" 2>/dev/null || return 0
	echo "stopping running daemon ..."
	kill "$pid" 2>/dev/null || true
	i=0
	while [ "$i" -lt 50 ]; do
		kill -0 "$pid" 2>/dev/null || return 0
		sleep 0.1
		i=$((i + 1))
	done
	kill -9 "$pid" 2>/dev/null || true
}

sync_tree() {
	src="$1"
	dst="$2"
	mkdir -p "$dst"
	( cd "$src" && find . -type f ) | while IFS= read -r f; do
		rel="${f#./}"
		if [ ! -f "$dst/$rel" ] || [ "$(sha256_of "$src/$rel")" != "$(sha256_of "$dst/$rel")" ]; then
			mkdir -p "$dst/$(dirname "$rel")"
			cp -f "$src/$rel" "$dst/$rel.new.$$"
			mv -f "$dst/$rel.new.$$" "$dst/$rel"
			echo "  updated $rel"
		fi
	done
	( cd "$dst" && find . -type f ) | while IFS= read -r f; do
		rel="${f#./}"
		[ -f "$src/$rel" ] || { rm -f "$dst/$rel" && echo "  removed $rel"; }
	done
}

version="${ALEXANDRIA_VERSION:-}"
if [ -z "$version" ]; then
	version="$(curl -fsSL "https://api.github.com/repos/$REPO/releases?per_page=1" | grep -m1 '"tag_name"' | sed -E 's/.*"v?([^"]+)".*/\1/')"
fi
if [ -z "$version" ]; then
	echo "could not find a release for $REPO" >&2
	exit 1
fi

if [ "$force" != "1" ] && [ -x "$APP_DIR/alexandria-cli" ]; then
	current="$("$APP_DIR/alexandria-cli" --version 2>/dev/null | awk '{print $2}')"
	if [ -n "$current" ] && [ "$current" = "$version" ]; then
		echo "alexandria-cli $version is already installed (use --force to reinstall)"
		exit 0
	fi
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

stage="$tmp/stage"
mkdir -p "$stage"
tar -xzf "$tmp/$archive" -C "$stage"

stop_daemon

echo "syncing into $APP_DIR ..."
sync_tree "$stage" "$APP_DIR"

chmod +x "$APP_DIR/alexandria-cli" 2>/dev/null || true
chmod +x "$APP_DIR/xray" 2>/dev/null || true
chmod +x "$APP_DIR/sing-box" 2>/dev/null || true

mkdir -p "$BIN_DIR"
ln -sf "$APP_DIR/alexandria-cli" "$BIN_DIR/alexandria-cli"

echo "installed alexandria-cli $version to $APP_DIR"
case ":$PATH:" in
	*":$BIN_DIR:"*) echo "run: alexandria-cli" ;;
	*) echo "add $BIN_DIR to your PATH, then run: alexandria-cli" ;;
esac
