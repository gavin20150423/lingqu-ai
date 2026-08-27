package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// writeRawUpstreamError writes the response already produced by an upstream
// provider. It is intentionally limited to an uncommitted, non-streaming
// response; once an SSE stream has started the handler must terminate it using
// that protocol rather than append a second HTTP body.
func writeRawUpstreamError(c *gin.Context, status int, headers http.Header, body []byte) bool {
	if c == nil || c.Writer == nil || c.Writer.Written() || status <= 0 || len(body) == 0 {
		return false
	}
	if headers != nil {
		if contentType := strings.TrimSpace(headers.Get("Content-Type")); contentType != "" {
			c.Header("Content-Type", contentType)
		}
		if retryAfter := strings.TrimSpace(headers.Get("Retry-After")); retryAfter != "" && !strings.ContainsAny(retryAfter, "\r\n") {
			c.Header("Retry-After", retryAfter)
		}
	}
	contentType := strings.TrimSpace(c.Writer.Header().Get("Content-Type"))
	if contentType == "" {
		contentType = "application/json"
	}
	c.Data(status, contentType, body)
	return true
}
