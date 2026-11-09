package context

import "github.com/gin-gonic/gin"

// Legacy Gin admission and public authentication routes use the exact same
// validation classification as typed API actions. No parallel JSON writer.
func FieldErrors(c *gin.Context, issues map[string][]FieldIssue) {
	GetAPIContext(c).FieldErrors(issues)
}

func FieldError(c *gin.Context, field, code, message string) {
	GetAPIContext(c).FieldError(field, code, message)
}

func Validation(c *gin.Context, err error, request any) {
	GetAPIContext(c).Validation(err, request)
}
