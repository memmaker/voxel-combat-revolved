#!/usr/bin/env bash
# Build the browser version and publish it to https://ruzzoli.de/games/voxel-combat/play/
set -euo pipefail
cd "$(dirname "$0")"
./build_web.sh
rsync -az --delete /Users/felix/Projects/fx-games/site/voxel-combat/play/ ruzzoli.de:/var/www/ruzzoli.de/games/voxel-combat/play/
echo "published: https://ruzzoli.de/games/voxel-combat/play/"
