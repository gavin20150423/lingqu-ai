//go:build unit

package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/Wei-Shaw/sub2api/internal/pkg/clienterror"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

const testAd = "请加群 987654321 购买更便宜的额度"

func newSanitizeRouter(shape clienterror.Shape, limit int, handler gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SanitizeUpstreamErrorBodyWithLimit(shape, limit))
	r.POST("/t", handler)
	return r
}

func doPost(r *gin.Engine) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/t", nil))
	return rec
}

// 上游错误（带广告）→ 客户只看到网关文案。
func TestSanitizeUpstreamErrorBody_RewritesUpstreamErrorText(t *testing.T) {
	r := newSanitizeRouter(clienterror.ShapeOpenAI, 0, func(c *gin.Context) {
		body := []byte(`{"error":{"type":"invalid_request_error","message":"` + testAd + `"}}`)
		service.SetOpsUpstreamError(c, http.StatusBadRequest, testAd, "")
		c.Data(http.StatusBadRequest, "application/json", body)
	})

	rec := doPost(r)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.NotContains(t, rec.Body.String(), "987654321")
	require.NotContains(t, rec.Body.String(), "加群")
	require.Contains(t, rec.Body.String(), "invalid_request_error")
}

// 网关自己的错误（未发生上游错误）→ 文案原样保留。
func TestSanitizeUpstreamErrorBody_KeepsGatewayOwnErrorText(t *testing.T) {
	const ownText = "余额不足，请先充值"
	r := newSanitizeRouter(clienterror.ShapeOpenAI, 0, func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "invalid_request_error", "message": ownText}})
	})

	rec := doPost(r)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), ownText)
}

// 成功响应一律不动。
func TestSanitizeUpstreamErrorBody_LeavesSuccessUntouched(t *testing.T) {
	const payload = `{"data":[{"b64_json":"aGVsbG8="}]}`
	r := newSanitizeRouter(clienterror.ShapeOpenAI, 0, func(c *gin.Context) {
		c.Data(http.StatusOK, "application/json", []byte(payload))
	})

	rec := doPost(r)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, payload, rec.Body.String())
}

// SSE 一律不动（流式错误由协议自身的终止帧处理）。
func TestSanitizeUpstreamErrorBody_LeavesEventStreamUntouched(t *testing.T) {
	const sse = "event: response.failed\ndata: {\"type\":\"response.failed\"}\n\n"
	r := newSanitizeRouter(clienterror.ShapeOpenAI, 0, func(c *gin.Context) {
		service.SetOpsUpstreamError(c, http.StatusBadGateway, "upstream", "")
		c.Header("Content-Type", "text/event-stream")
		c.Status(http.StatusBadGateway)
		_, _ = c.Writer.Write([]byte(sse))
	})

	rec := doPost(r)

	require.Equal(t, http.StatusBadGateway, rec.Code)
	require.Equal(t, sse, rec.Body.String())
}

// 非 JSON 的上游错误体（纯文本广告）也不能漏出去。
func TestSanitizeUpstreamErrorBody_RewritesNonJSONUpstreamBody(t *testing.T) {
	r := newSanitizeRouter(clienterror.ShapeOpenAI, 0, func(c *gin.Context) {
		service.SetOpsUpstreamError(c, http.StatusInternalServerError, testAd, "")
		c.Data(http.StatusInternalServerError, "text/plain", []byte(testAd))
	})

	rec := doPost(r)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.NotContains(t, rec.Body.String(), "987654321")
	require.Contains(t, rec.Body.String(), "server_error")
}

// 超过缓冲上限：宁可丢内容也不漏原文。
func TestSanitizeUpstreamErrorBody_OverflowNeverLeaks(t *testing.T) {
	big := strings.Repeat(testAd, 200) // 远超 64 字节上限
	r := newSanitizeRouter(clienterror.ShapeOpenAI, 64, func(c *gin.Context) {
		service.SetOpsUpstreamError(c, http.StatusBadGateway, testAd, "")
		c.Data(http.StatusBadGateway, "application/json", []byte(`{"error":{"message":"`+big+`"}}`))
	})

	rec := doPost(r)

	require.Equal(t, http.StatusBadGateway, rec.Code)
	require.NotContains(t, rec.Body.String(), "987654321")
	require.Contains(t, rec.Body.String(), "server_error")
}

// Google 结构按 Google 形状重建。
func TestSanitizeUpstreamErrorBody_GoogleShape(t *testing.T) {
	r := newSanitizeRouter(clienterror.ShapeGoogle, 0, func(c *gin.Context) {
		service.SetOpsUpstreamError(c, http.StatusTooManyRequests, testAd, "")
		c.Data(http.StatusTooManyRequests, "application/json",
			[]byte(`{"error":{"code":429,"message":"`+testAd+`","status":"RESOURCE_EXHAUSTED"}}`))
	})

	rec := doPost(r)

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.NotContains(t, rec.Body.String(), "987654321")
	require.Contains(t, rec.Body.String(), `"status":"RESOURCE_EXHAUSTED"`)
	require.Contains(t, rec.Body.String(), `"code":429`)
}
