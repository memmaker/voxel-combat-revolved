//go:build js

package main

import (
	"embed"
	"io/fs"

	"github.com/memmaker/battleground/engine/afs"
)

// web/assets is staged by build_web.sh (a trimmed copy of ./assets).
//
//go:embed all:web/assets
var webAssets embed.FS

func init() {
	sub, _ := fs.Sub(webAssets, "web")
	afs.FS = sub
}
