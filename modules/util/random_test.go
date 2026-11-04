package util

import (
	"math"
	"testing"
)

func TestRandomIntBoundaries(t *testing.T) {
	for _, tc := range []struct {
		min int64
		max int64
	}{
		{0, 0},
		{-1, 1},
		{math.MaxInt64, math.MaxInt64},
		{math.MinInt64, math.MinInt64},
		{math.MinInt64, math.MaxInt64},
		{math.MinInt64, 0},
		{0, math.MaxInt64},
	} {
		for i := 0; i < 100; i++ {
			got := RandomInt(tc.min, tc.max)
			if got < tc.min || got > tc.max {
				t.Fatalf("RandomInt(%d, %d) = %d", tc.min, tc.max, got)
			}
		}
	}
}

func TestRandomIntRejectsInvertedRange(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("inverted range did not panic")
		}
	}()
	RandomInt(2, 1)
}
