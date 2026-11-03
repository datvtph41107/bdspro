package templates

import "embed"

// Swagger holds the reviewed OpenAPI contract and the local Swagger UI assets.
// The embedding path is relative to this package, not to the process CWD.
//
//go:embed swagger
var Swagger embed.FS
