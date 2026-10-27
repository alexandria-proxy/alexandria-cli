#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."
rm -rf completions
mkdir -p completions
bin="$(mktemp)"
trap 'rm -f "$bin"' EXIT
go build -o "$bin" .
"$bin" completion bash > completions/alexandria-cli.bash
"$bin" completion zsh > completions/_alexandria-cli
"$bin" completion fish > completions/alexandria-cli.fish
"$bin" completion powershell > completions/alexandria-cli.ps1
