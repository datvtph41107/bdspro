package doc

import "embed"

// Swagger embeds the versioned OpenAPI contract and UI distribution.
//
//go:embed swagger
var Swagger embed.FS
