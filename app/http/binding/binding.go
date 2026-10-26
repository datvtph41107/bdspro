package binding

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// JSON, Query and Path implement one bind/validate pass per input source.
// Route middleware must precede the controller; no hidden global input registry.
func JSON[T any](key string) gin.HandlerFunc {
	return bind[T](key, func(ctx *gin.Context, dst *T) error {
		return ctx.ShouldBindJSON(dst)
	})
}
func Query[T any](key string) gin.HandlerFunc {
	return bind[T](key, func(ctx *gin.Context, dst *T) error {
		return ctx.ShouldBindQuery(dst)
	})
}
func Path[T any](key string) gin.HandlerFunc {
	return bind[T](key, func(ctx *gin.Context, dst *T) error {
		return ctx.ShouldBindUri(dst)
	})
}
func Input[T any](ctx *gin.Context, key string) T {
	return ctx.MustGet(key).(T)
}
func bind[T any](key string, parse func(*gin.Context, *T) error) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var input T
		if err := parse(ctx, &input); err != nil {
			var size *http.MaxBytesError
			if errors.As(err, &size) {
				ctx.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request body too large"})
			} else {
				ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			}
			return
		}
		ctx.Set(key, input)
		ctx.Next()
	}
}
