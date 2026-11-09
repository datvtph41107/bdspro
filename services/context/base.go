package context

import (
	"time"

	"github.com/gin-gonic/gin"
)

// Base is the shared request-scoped HTTP contract. Gin is retained only as
// an adapter for legacy HTTP actions; domain services never import it.
// Like Gitea's Base, request-local data belongs to exactly one request.
type Base struct {
	*gin.Context
	Data map[string]any
}

func NewBaseContext(c *gin.Context) *Base {
	return &Base{Context: c, Data: make(map[string]any)}
}

func (b *Base) GetData() map[string]any {
	return b.Data
}

// Delegate to the current request context rather than retaining the initial
// one: authentication/transaction middleware may replace c.Request later.
func (b *Base) Deadline() (time.Time, bool) {
	return b.Request.Context().Deadline()
}

func (b *Base) Done() <-chan struct{} {
	return b.Request.Context().Done()
}

func (b *Base) Err() error {
	return b.Request.Context().Err()
}

func (b *Base) Value(key any) any {
	return b.Request.Context().Value(key)
}
