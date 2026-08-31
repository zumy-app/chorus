package handlers

import (
	"github.com/chorus/messenger/internal/middleware"
	"github.com/gin-gonic/gin"
)

// WriteError terminates the request with the shared, typed error envelope. All
// handlers route errors through here so the client receives a consistent
// {error:{kind,message,retryable}} contract and never sees a raw internal
// message. See middleware.WriteError for the exact wire format and kind list.
func WriteError(c *gin.Context, err error) {
	middleware.WriteError(c, err)
}
