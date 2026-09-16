package handler

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	upstreamPoolExhaustedStatus  = http.StatusServiceUnavailable
	upstreamPoolExhaustedCode    = "upstream_pool_exhausted"
	upstreamPoolExhaustedMessage = "Upstream account pool is temporarily unavailable"
)

// An exhausted internal account pool is not an aggregate rate limit. Returning
// the last account's 429 would make a downstream gateway cool this whole service
// as one account even though another internal account may recover at any time.
func isUpstreamPoolRateLimitExhausted(failoverErr *service.UpstreamFailoverError) bool {
	return failoverErr != nil && failoverErr.StatusCode == http.StatusTooManyRequests
}

func clearUpstreamPoolRetryAfter(c *gin.Context) {
	if c != nil {
		c.Writer.Header().Del("Retry-After")
	}
}
