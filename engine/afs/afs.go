// Package afs is the read-only asset filesystem: the working directory on desktop, embedded files in the browser.
package afs

import (
	"io/fs"
	"os"
	"path"
)

var FS fs.FS = os.DirFS(".")

func Open(name string) (fs.File, error)          { return FS.Open(path.Clean(name)) }
func ReadFile(name string) ([]byte, error)       { return fs.ReadFile(FS, path.Clean(name)) }
func Stat(name string) (fs.FileInfo, error)      { return fs.Stat(FS, path.Clean(name)) }
func ReadDir(name string) ([]fs.DirEntry, error) { return fs.ReadDir(FS, path.Clean(name)) }
