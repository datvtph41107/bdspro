package context

import (
	"errors"
	"strconv"
)

var ErrInvalidPathID = errors.New("invalid positive path identifier")

// PathParam reads a value set by the HTTP router's matched-route boundary.
func (b *Base) PathParam(key string) string {
	return b.Req.PathValue(key)
}

func (b *Base) PositivePathID(key string) (int64, error) {
	id, err := strconv.ParseInt(b.PathParam(key), 10, 64)
	if err != nil || id <= 0 {
		return 0, ErrInvalidPathID
	}
	return id, nil
}
