package statik

import (
 "embed"
 "io/fs"
)

// Embedding the generated static snapshot makes the API reference available
// from one binary without relying on the process working directory.
//go:embed swagger/*
var embedded embed.FS

func Files() fs.FS {
 files, err := fs.Sub(embedded, "swagger")
 if err != nil { panic(err) }
 return files
}
