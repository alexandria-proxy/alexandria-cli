#!/usr/bin/env bash


#for build


set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MANIFEST="$ROOT/core/manifest.json"
REPO="XTLS/Xray-core"
SB_REPO="SagerNet/sing-box"

PLATFORMS="linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64 windows-arm64"

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

asset_for() {
	case "$1" in
		linux-amd64) echo "Xray-linux-64.zip" ;;
		linux-arm64) echo "Xray-linux-arm64-v8a.zip" ;;
		darwin-amd64) echo "Xray-macos-64.zip" ;;
		darwin-arm64) echo "Xray-macos-arm64-v8a.zip" ;;
		windows-amd64) echo "Xray-windows-64.zip" ;;
		windows-arm64) echo "Xray-windows-arm64-v8a.zip" ;;
		*) return 1 ;;
	esac
}

config_bin() {
	case "$(go env GOOS)" in
		darwin) echo "$HOME/Library/Application Support/alexandria/bin" ;;
		windows) echo "${APPDATA:-$HOME/AppData/Roaming}/alexandria/bin" ;;
		*) echo "${XDG_CONFIG_HOME:-$HOME/.config}/alexandria/bin" ;;
	esac
}

sha256_file() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | awk '{print $1}'
	else
		shasum -a 256 "$1" | awk '{print $1}'
	fi
}

sha256_from_dgst() {
	grep -iE '^sha2?-?256[ =:]' "$1" | grep -oiE '[0-9a-f]{64}' | head -n1
}

extract_core() {
	local zip="$1" dest="$2"
	mkdir -p "$dest"
	unzip -o -j "$zip" -d "$dest" >/dev/null
	find "$dest" -maxdepth 1 -type f ! -name 'xray' ! -name 'xray.exe' ! -name '*.dat' -delete
	if [ -f "$dest/xray" ]; then
		chmod +x "$dest/xray"
	fi
}

install_platform() {
	local plat="$1" dest="$2"
	local version file want
	version=$(jq -r '.version' "$MANIFEST")
	file=$(jq -r --arg p "$plat" '.assets[$p].file' "$MANIFEST")
	want=$(jq -r --arg p "$plat" '.assets[$p].sha256' "$MANIFEST" | tr 'A-Z' 'a-z')
	if [ "$file" = "null" ] || [ "$want" = "null" ] || [ -z "$want" ]; then
		echo "no manifest entry for $plat" >&2
		exit 1
	fi

	echo "downloading $file ..." >&2
	curl -fsSL -o "$WORK/$file" "https://github.com/$REPO/releases/download/v$version/$file"

	local got
	got=$(sha256_file "$WORK/$file" | tr 'A-Z' 'a-z')
	if [ "$got" != "$want" ]; then
		echo "checksum mismatch for $plat: got $got want $want" >&2
		exit 1
	fi

	extract_core "$WORK/$file" "$dest"
	echo "installed $plat -> $dest" >&2
}

sb_asset_for() {
	local v="$2"
	case "$1" in
		linux-amd64) echo "sing-box-$v-linux-amd64.tar.gz" ;;
		linux-arm64) echo "sing-box-$v-linux-arm64.tar.gz" ;;
		darwin-amd64) echo "sing-box-$v-darwin-amd64.tar.gz" ;;
		darwin-arm64) echo "sing-box-$v-darwin-arm64.tar.gz" ;;
		windows-amd64) echo "sing-box-$v-windows-amd64.zip" ;;
		windows-arm64) echo "sing-box-$v-windows-arm64.zip" ;;
		*) return 1 ;;
	esac
}

extract_singbox() {
	local archive="$1" dest="$2"
	mkdir -p "$dest"
	local tmp
	tmp="$(mktemp -d)"
	case "$archive" in
		*.zip) unzip -o "$archive" -d "$tmp" >/dev/null ;;
		*.tar.gz) tar -xzf "$archive" -C "$tmp" ;;
	esac
	local bin
	bin="$(find "$tmp" -type f \( -name 'sing-box' -o -name 'sing-box.exe' \) | head -n1)"
	if [ -z "$bin" ]; then
		echo "sing-box binary not found in $archive" >&2
		rm -rf "$tmp"
		exit 1
	fi
	cp "$bin" "$dest/$(basename "$bin")"
	chmod +x "$dest/$(basename "$bin")" 2>/dev/null || true
	rm -rf "$tmp"
}

install_singbox() {
	local plat="$1" dest="$2"
	local version file want
	version=$(jq -r '.singbox.version' "$MANIFEST")
	file=$(jq -r --arg p "$plat" '.singbox.assets[$p].file' "$MANIFEST")
	want=$(jq -r --arg p "$plat" '.singbox.assets[$p].sha256' "$MANIFEST" | tr 'A-Z' 'a-z')
	if [ "$file" = "null" ] || [ "$want" = "null" ] || [ -z "$want" ]; then
		echo "no sing-box manifest entry for $plat — run: fetch-core.sh --update-singbox $version" >&2
		exit 1
	fi

	echo "downloading $file ..." >&2
	curl -fsSL -o "$WORK/$file" "https://github.com/$SB_REPO/releases/download/v$version/$file"

	local got
	got=$(sha256_file "$WORK/$file" | tr 'A-Z' 'a-z')
	if [ "$got" != "$want" ]; then
		echo "sing-box checksum mismatch for $plat: got $got want $want" >&2
		exit 1
	fi

	extract_singbox "$WORK/$file" "$dest"
	echo "installed sing-box $plat -> $dest" >&2
}

cmd_update() {
	local version="${1:-}"
	if [ -z "$version" ]; then
		echo "usage: fetch-core.sh --update <version>" >&2
		exit 1
	fi
	version="${version#v}"

	local manifest
	manifest=$(jq --arg v "$version" '.version = $v | .assets = {}' "$MANIFEST")

	for plat in $PLATFORMS; do
		local file url got want
		file=$(asset_for "$plat")
		url="https://github.com/$REPO/releases/download/v$version/$file"
		echo "fetching $plat ($file) ..." >&2
		curl -fsSL -o "$WORK/$file" "$url"
		curl -fsSL -o "$WORK/$file.dgst" "$url.dgst"

		got=$(sha256_file "$WORK/$file" | tr 'A-Z' 'a-z')
		want=$(sha256_from_dgst "$WORK/$file.dgst" | tr 'A-Z' 'a-z')
		if [ -z "$want" ]; then
			echo "no sha256 in dgst for $plat" >&2
			exit 1
		fi
		if [ "$got" != "$want" ]; then
			echo "XTLS digest mismatch for $plat: got $got want $want" >&2
			exit 1
		fi

		manifest=$(printf '%s' "$manifest" | jq --arg p "$plat" --arg f "$file" --arg s "$got" '.assets[$p] = {file: $f, sha256: $s}')
		echo "  ok $got" >&2
	done

	mkdir -p "$(dirname "$MANIFEST")"
	printf '%s\n' "$manifest" >"$MANIFEST"
	echo "wrote $MANIFEST (xray $version)" >&2
}

cmd_update_singbox() {
	local version="${1:-}"
	if [ -z "$version" ]; then
		echo "usage: fetch-core.sh --update-singbox <version>" >&2
		exit 1
	fi
	version="${version#v}"

	local manifest
	manifest=$(jq --arg v "$version" '.singbox.version = $v | .singbox.assets = {}' "$MANIFEST")

	for plat in $PLATFORMS; do
		local file url got
		file=$(sb_asset_for "$plat" "$version")
		url="https://github.com/$SB_REPO/releases/download/v$version/$file"
		echo "fetching sing-box $plat ($file) ..." >&2
		curl -fsSL -o "$WORK/$file" "$url"

		got=$(sha256_file "$WORK/$file" | tr 'A-Z' 'a-z')
		manifest=$(printf '%s' "$manifest" | jq --arg p "$plat" --arg f "$file" --arg s "$got" '.singbox.assets[$p] = {file: $f, sha256: $s}')
		echo "  ok $got" >&2
	done

	mkdir -p "$(dirname "$MANIFEST")"
	printf '%s\n' "$manifest" >"$MANIFEST"
	echo "wrote $MANIFEST (sing-box $version)" >&2
}

cmd_all() {
	for plat in $PLATFORMS; do
		install_platform "$plat" "$ROOT/.cores/$plat"
		install_singbox "$plat" "$ROOT/.cores/$plat"
	done
}

cmd_host() {
	local plat dest
	plat="$(go env GOOS)-$(go env GOARCH)"
	dest="$(config_bin)"
	install_platform "$plat" "$dest"
	install_singbox "$plat" "$dest"
}

case "${1:-}" in
	--update)
		shift
		cmd_update "${1:-}"
		;;
	--update-singbox)
		shift
		cmd_update_singbox "${1:-}"
		;;
	--all) cmd_all ;;
	"") cmd_host ;;
	*)
		echo "usage: fetch-core.sh [--all | --update <version> | --update-singbox <version>]" >&2
		exit 1
		;;
esac
