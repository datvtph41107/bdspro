//go:build tools

// Package tools keeps the generated Swagger/Statik runtime dependency in the
// module graph even before "make swagger" creates doc/statik/statik.go.
// When Statik generation is removed, remove this package and the dependency.
package tools

import _ "github.com/rakyll/statik/fs"
