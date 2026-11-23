//go:build unit

package clienterror

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const adText = "请加入我们的官方群 123456789 获取更便宜的额度"

func TestSanitize_StripsUpstreamAdFromMessage(t *testing.T) {
	body := []byte(`{"error":{"type":"invalid_request_error","code":"invalid_request_error","message":"` + adText + `"}}`)

	out := string(Sanitize(ShapeOpenAI, http.StatusBadRequest, body))

	require.NotContains(t, out, "123456789")
	require.NotContains(t, out, "官方群")
	require.Contains(t, out, "invalid_request_error")
}

func TestSanitize_DropsNonWhitelistedCodeCarryingAd(t *testing.T) {
	// 上游把广告塞进 code 字段
	body := []byte(`{"error":{"type":"server_error","code":"` + adText + `","message":"boom"}}`)

	out := string(Sanitize(ShapeOpenAI, http.StatusBadGateway, body))

	require.NotContains(t, out, "123456789")
	require.NotContains(t, out, "官方群")
	// server_error 在白名单内，保留
	require.Contains(t, out, "server_error")
}

func TestSanitize_UnknownTypeFallsBackToStatus(t *testing.T) {
	body := []byte(`{"error":{"type":"` + adText + `","message":"x"}}`)

	out := string(Sanitize(ShapeOpenAI, http.StatusTooManyRequests, body))

	require.NotContains(t, out, "123456789")
	require.Contains(t, out, "rate_limit_error")
}

func TestSanitize_NonJSONBodyBecomesCanonicalError(t *testing.T) {
	out := string(Sanitize(ShapeOpenAI, http.StatusBadRequest, []byte("错误："+adText)))

	require.NotContains(t, out, "123456789")
	require.NotContains(t, out, "官方群")
	require.Contains(t, out, `"type":"invalid_request_error"`)
}

func TestSanitize_GoogleShapeKeepsStatusAndDropsAd(t *testing.T) {
	body := []byte(`{"error":{"code":400,"message":"` + adText + `","status":"INVALID_ARGUMENT"}}`)

	out := string(Sanitize(ShapeGoogle, http.StatusBadRequest, body))

	require.NotContains(t, out, "123456789")
	require.Contains(t, out, `"status":"INVALID_ARGUMENT"`)
	require.Contains(t, out, `"code":400`)
}

func TestSanitize_GoogleShapeRejectsUnknownStatus(t *testing.T) {
	body := []byte(`{"error":{"code":503,"message":"x","status":"` + adText + `"}}`)

	out := string(Sanitize(ShapeGoogle, http.StatusServiceUnavailable, body))

	require.NotContains(t, out, "123456789")
	require.Contains(t, out, `"status":"UNAVAILABLE"`)
}

func TestSanitize_AlwaysProducesJSONObject(t *testing.T) {
	for _, shape := range []Shape{ShapeOpenAI, ShapeGoogle} {
		for _, status := range []int{400, 401, 403, 404, 413, 429, 500, 502, 503, 504, 418} {
			out := Sanitize(shape, status, []byte(adText))
			require.True(t, strings.HasPrefix(string(out), "{"), "shape=%d status=%d", shape, status)
			require.NotContains(t, string(out), "123456789")
		}
	}
}
