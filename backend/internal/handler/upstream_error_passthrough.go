package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Wei-Shaw/sub2api/internal/pkg/clienterror"
)

// writeRawUpstreamError writes an upstream error response to the client.
//
// ⚠️ 名字里的 raw 只指「沿用上游的 HTTP 状态码与结构化错误标识」，**不再透传上游文本**：
// 部分上游会在错误体里写加群/导流文案撬客户，因此这里统一走 clienterror.Sanitize，
// 自由文本一律丢弃，只保留白名单内的 type/code，文案由网关生成。
// 它仍然只用于未提交的非流式响应；一旦 SSE 已开始，必须用协议自己的终止帧结束。
func writeRawUpstreamError(c *gin.Context, status int, headers http.Header, body []byte) bool {
	if c == nil || c.Writer == nil || c.Writer.Written() || status <= 0 || len(body) == 0 {
		return false
	}
	if headers != nil {
		if retryAfter := strings.TrimSpace(headers.Get("Retry-After")); retryAfter != "" && !strings.ContainsAny(retryAfter, "\r\n") {
			c.Header("Retry-After", retryAfter)
		}
	}
	sanitized := clienterror.Sanitize(clienterror.ShapeOpenAI, status, body)
	c.Data(status, "application/json; charset=utf-8", sanitized)
	return true
}
