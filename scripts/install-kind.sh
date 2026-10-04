#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
test "$(uname -s)/$(uname -m)" = Darwin/arm64 || { echo 'Installer currently supports macOS arm64 only' >&2; exit 1; }
dir=.tools/kind-v0.33.0
mkdir -p "$dir"
curl --fail --location --proto '=https' --tlsv1.2 https://github.com/kubernetes-sigs/kind/releases/download/v0.33.0/kind-darwin-arm64 -o "$dir/kind"
expected=0c8c7dbe5e23594a198b786c4bc13dacc101fa6196b0cb0b23a1ca44e61f4b4f
actual=$(shasum -a 256 "$dir/kind" | awk '{print $1}')
test "$expected" = "$actual" || { echo 'kind checksum mismatch; refusing to execute' >&2; exit 1; }
chmod +x "$dir/kind"
"$dir/kind" version
