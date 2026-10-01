#!/bin/sh
# Builds the browser (WASM + WebGL2) version. Desktop build is unchanged: go build -tags client
set -e
cd "$(dirname "$0")"
OUT=${1:-/Users/felix/Projects/fx-games/site/voxel-combat/play}

# Stage assets for go:embed. star_odyssey/*.png are only stat'ed (block texture names), so empty placeholders suffice.
rm -rf web/assets && mkdir -p web/assets
cp -R assets/fonts assets/gui assets/maps assets/models web/assets/
mkdir -p web/assets/textures/blocks/star_odyssey
cp -R assets/textures/skins assets/textures/decals web/assets/textures/
cp assets/textures/blocks/*.* web/assets/textures/blocks/
for f in assets/textures/blocks/star_odyssey/*.png; do : > "web/assets/textures/blocks/star_odyssey/$(basename "$f")"; done

# Swap go-gl/gl and glfw for the WebGL2/canvas shims in web/.
sed '/^go /a\
replace github.com/go-gl/gl => ./web/gl\
replace github.com/go-gl/glfw/v3.3/glfw => ./web/glfw
' go.mod > go.web.mod
cp go.sum go.web.sum

mkdir -p "$OUT"
GOOS=js GOARCH=wasm go build -modfile=go.web.mod -tags client -trimpath -ldflags="-s -w" -o "$OUT/voxel.wasm" .
gzip -9 -k -f "$OUT/voxel.wasm"
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" "$OUT/" 2>/dev/null || cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" "$OUT/"
ls -l "$OUT"
