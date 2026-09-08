package web

import (
	"embed"
	"io/fs"
)

//go:embed static
var staticFiles embed.FS

func StaticFS() fs.FS {
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic("static assets were not embedded correctly: " + err.Error())
	}

	return staticFS
}
