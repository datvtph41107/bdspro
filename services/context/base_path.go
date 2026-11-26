package context

import (
	"errors"
	"strconv"
)

var ErrInvalidPathID = errors.New("invalid positive path identifier")

func (b *Base) PathParam(key string) string {
	return b.source.PathParam(key)
}

func (b *Base) PositivePathID(key string) (int64, error) {
	id, err := strconv.ParseInt(b.PathParam(key), 10, 64)
	if err != nil || id <= 0 {
		return 0, ErrInvalidPathID
	}
	return id, nil
}
