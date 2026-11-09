package context

import (
	"errors"
	"strconv"
)

// ErrInvalidPathID deliberately excludes the submitted value: URI identifiers
// may be sensitive and should not appear in logs or public errors.
var ErrInvalidPathID = errors.New("invalid positive path identifier")

// PathParam provides the route parameter through Base rather than coupling
// API actions to Gin. Gitea exposes path accessors from the same layer.
func (b *Base) PathParam(key string) string {
	return b.source.PathParam(key)
}

// PositivePathID owns positive int64 parsing; the endpoint decides whether
// a malformed parameter is invalid input or an invisible domain resource.
func (b *Base) PositivePathID(key string) (int64, error) {
	id, err := strconv.ParseInt(b.PathParam(key), 10, 64)
	if err != nil || id <= 0 {
		return 0, ErrInvalidPathID
	}
	return id, nil
}
