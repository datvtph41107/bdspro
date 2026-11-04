package cache

import (
	"context"
	"testing"
	"time"
)

func TestJSONCacheRoundTrip(t *testing.T) {
	c, err := NewMemoryStringCache(3)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	type record struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	if err := PutJSON(ctx, c, "u", record{7, "Alice"}, time.Minute); err != nil {
		t.Fatal(err)
	}
	var got record
	hit, err := GetJSON(ctx, c, "u", &got)
	if err != nil || !hit || got.ID != 7 || got.Name != "Alice" {
		t.Fatalf("JSON cache: hit=%v got=%+v err=%v", hit, got, err)
	}
	if err := c.Put(ctx, "invalid", "{", time.Minute); err != nil {
		t.Fatal(err)
	}
	if hit, err := GetJSON(ctx, c, "invalid", &got); hit || err == nil {
		t.Fatal("invalid cached JSON did not surface as an error")
	}
}
