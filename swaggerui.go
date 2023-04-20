package api

import (
	"embed"
	"io/fs"
)

//go:embed swaggerui/*
var swaggerUIInt embed.FS

var swaggerUI = mustSub(swaggerUIInt, "swaggerui")

func mustSub(fsys fs.FS, dir string) fs.FS {
	ret, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}

	return ret
}
