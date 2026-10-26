package api

import (
    "github.com/datvtph41107/bdspro/app/http/binding"
    "github.com/gin-gonic/gin"
)

// Compatibility entrypoint; the canonical binding mechanism lives in app/http/binding.
func preJSON[T any](key string) gin.HandlerFunc { return binding.JSON[T](key) }
