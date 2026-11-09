package context

import (
	"errors"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPositivePathIDUsesOneParsingContract(t *testing.T) {
	cases := []struct {
		raw string
		want int64
		valid bool
	}{
		{"42", 42, true},
		{"9223372036854775807", 9223372036854775807, true},
		{"", 0, false},
		{"0", 0, false},
		{"-1", 0, false},
		{"hello", 0, false},
		{"9223372036854775808", 0, false},
	}
	for _, tt := range cases {
		t.Run(tt.raw, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			c := &gin.Context{Params: gin.Params{{Key: "id", Value: tt.raw}}}
			value, err := NewBaseContext(c).PositivePathID("id")
			if (err == nil) != tt.valid || value != tt.want {
				t.Fatalf("raw=%q value=%d err=%v", tt.raw, value, err)
			}
			if !tt.valid && !errors.Is(err, ErrInvalidPathID) {
				t.Fatalf("invalid ID did not use shared sentinel: %v", err)
			}
		})
	}
}
