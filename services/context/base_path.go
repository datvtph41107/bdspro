package context

// PathParam exposes router parameters through the shared Base contract,
// so API actions and assignments do not need to import Gin.
func (b *Base) PathParam(key string) string {
	return b.Param(key)
}
