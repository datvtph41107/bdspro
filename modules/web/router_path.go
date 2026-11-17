// Package web keeps route path adapters separate from route composition.
// Gitea also separates advanced path matching into router_path.go.
package web

import (
	"fmt"
	"strings"
)

// ginPattern converts simple Gitea/Chi-style {name} path parameters into
// Gin's :name syntax at the matcher boundary. Keep route declarations and
// routeDefinition paths in Gitea form; only the Gin adapter sees :name.
//
// Regex parameters, mixed literal/parameter segments, and wildcard syntax
// are intentionally not translated: do not silently change their semantics.
func ginPattern(pattern string) string {
	if !strings.ContainsAny(pattern, "{}") {
		return pattern // Existing Gin :name declarations remain compatible.
	}

	segments := strings.Split(pattern, "/")
	for i, segment := range segments {
		if !strings.ContainsAny(segment, "{}") {
			continue
		}

		if len(segment) < 3 || segment[0] != '{' || segment[len(segment)-1] != '}' {
			panic(fmt.Sprintf("web: unsupported route parameter segment %q", segment))
		}

		name := segment[1 : len(segment)-1]
		for j, char := range name {
			if !((char >= 'a' && char <= 'z') ||
				(char >= 'A' && char <= 'Z') ||
				(char == '_' || (j > 0 && char >= '0' && char <= '9'))) {
				panic(fmt.Sprintf("web: unsupported route parameter segment %q", segment))
			}
		}

		segments[i] = ":" + name
	}
	return strings.Join(segments, "/")
}

// matchesMountScope applies child BeforeRouting to the same path scope
// used by Gin for a mounted route. The check happens BEFORE route matching,
// so it cannot rely on Gin's path parameters. Parameters match one nonempty
// segment; static segments must match exactly.
func matchesMountScope(pattern, path string) bool {
	if pattern == "/" {
		return strings.HasPrefix(path, "/")
	}
	if !strings.HasPrefix(path, "/") {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(pattern, "/"), "/")
	actual := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(actual) < len(parts) {
		return false
	}
	for i, part := range parts {
		if part == "" || actual[i] == "" {
			return false
		}
		if (strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}")) ||
			(strings.HasPrefix(part, ":") && len(part) > 1) {
			continue
		}
		if part != actual[i] {
			return false
		}
	}
	return true
}
