package service

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func toolMappingResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewReader([]byte(body))),
		Header:     http.Header{},
	}
}

func TestIsUpstreamToolMappingUnsupportedBody(t *testing.T) {
	cases := []struct {
		name string
		code int
		body string
		want bool
	}{
		{
			name: "Kiro 中文原文（本次事故）",
			code: 400,
			body: `{"error":{"type":"invalid_request_error","message":"工具映射不支持: Claude Code Read.pages has no Kiro read_file equivalent"}}`,
			want: true,
		},
		{
			name: "英文 has no ... equivalent",
			code: 400,
			body: `{"error":{"message":"tool error: Claude Code WebSearch has no Kiro web_search equivalent"}}`,
			want: true,
		},
		{
			name: "tool mapping not supported",
			code: 400,
			body: `{"error":{"message":"tool mapping for Read.pages is not supported by this upstream"}}`,
			want: true,
		},
		{
			name: "unsupported tool",
			code: 400,
			body: `{"error":{"message":"unsupported tool: Read.pages"}}`,
			want: true,
		},
		{
			name: "普通参数错误 400 不应命中",
			code: 400,
			body: `{"error":{"message":"messages: field required"}}`,
			want: false,
		},
		{
			name: "上下文超长 400 不应命中",
			code: 400,
			body: `{"error":{"message":"prompt is too long: 250000 tokens"}}`,
			want: false,
		},
		{
			name: "内容策略 400 不应命中",
			code: 400,
			body: `{"error":{"message":"Your request was rejected by our content policy"}}`,
			want: false,
		},
		{
			name: "非 400 不应命中",
			code: 500,
			body: `{"error":{"message":"工具映射不支持: Claude Code Read.pages"}}`,
			want: false,
		},
		{
			name: "空 body 不应命中",
			code: 400,
			body: "",
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, isUpstreamToolMappingUnsupportedBody(tc.code, []byte(tc.body)))
		})
	}
}

// 判定器会读取并还原 Body，必须保证下游仍能读到完整报文。
func TestIsUpstreamToolMappingUnsupportedRestoresBody(t *testing.T) {
	const body = `{"error":{"type":"invalid_request_error","message":"工具映射不支持: Claude Code Read.pages has no Kiro read_file equivalent"}}`
	resp := toolMappingResponse(400, body)

	require.True(t, isUpstreamToolMappingUnsupported(resp))

	again, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, body, string(again))
}

// 连续多次判定不应互相消费 Body。
func TestIsUpstreamToolMappingUnsupportedRepeatable(t *testing.T) {
	const body = `{"error":{"message":"tool mapping for Read.pages is not supported"}}`
	resp := toolMappingResponse(400, body)

	for i := 0; i < 3; i++ {
		require.True(t, isUpstreamToolMappingUnsupported(resp), "第 %d 次判定应命中", i)
	}
	got, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, body, string(got))
}

func TestUpstreamToolMappingToolName(t *testing.T) {
	resp := toolMappingResponse(400, `{"error":{"message":"工具映射不支持: Claude Code Read.pages has no Kiro read_file equivalent"}}`)
	require.Equal(t, "Read.pages", upstreamToolMappingToolName(resp))
}

// 核心行为：工具映射 400 必须放行 failover，普通 400 不放行。
func TestShouldFailoverUpstreamResponse_ToolMapping(t *testing.T) {
	svc := &GatewayService{}

	toolResp := toolMappingResponse(400, `{"error":{"message":"工具映射不支持: Claude Code Read.pages has no Kiro read_file equivalent"}}`)
	require.True(t, svc.shouldFailoverUpstreamResponse(toolResp), "工具映射 400 应触发换号")

	paramResp := toolMappingResponse(400, `{"error":{"message":"messages: field required"}}`)
	require.False(t, svc.shouldFailoverUpstreamResponse(paramResp), "普通参数 400 不应换号")
}

func TestShouldRetryUpstreamResponse_ToolMapping(t *testing.T) {
	svc := &GatewayService{}
	// apikey 账号、未配 custom_error_codes：普通 400 恒不重试。
	account := &Account{Type: AccountTypeAPIKey}

	toolResp := toolMappingResponse(400, `{"error":{"message":"工具映射不支持: Claude Code Read.pages has no Kiro read_file equivalent"}}`)
	require.True(t, svc.shouldRetryUpstreamResponse(account, toolResp), "工具映射 400 应触发重试")

	paramResp := toolMappingResponse(400, `{"error":{"message":"messages: field required"}}`)
	require.False(t, svc.shouldRetryUpstreamResponse(account, paramResp), "普通参数 400 维持不重试")
}

// 工具名提取失败不应影响判定本身。
func TestIsUpstreamToolMappingUnsupported_BodyStillReadableAfterNameExtraction(t *testing.T) {
	const body = `{"error":{"message":"工具映射不支持: Claude Code Read.pages has no Kiro read_file equivalent"}}`
	resp := toolMappingResponse(400, body)

	_ = upstreamToolMappingToolName(resp)
	got, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, body, string(got))
	require.True(t, strings.Contains(string(got), "Read.pages"))
}

// 同账号重试闸门：400 一律不进入同账号重试循环。
//
// 2026-10-16 事故回归：v0.2.15-lingqu.2 曾把「工具映射不支持」的 400 放行进入同账号
// 重试循环，导致单请求在同一账号上白撞满 5 次（~7–10s）才换号。此用例锁死该不变量。
func TestSameAccountRetryEligible(t *testing.T) {
	cases := []struct {
		name string
		code int
		want bool
	}{
		{name: "工具映射 400 不得同账号重试", code: 400, want: false},
		{name: "参数错误 400 不得同账号重试", code: 400, want: false},
		{name: "401 可同账号重试", code: 401, want: true},
		{name: "403 可同账号重试", code: 403, want: true},
		{name: "429 可同账号重试", code: 429, want: true},
		{name: "500 可同账号重试", code: 500, want: true},
		{name: "503 可同账号重试", code: 503, want: true},
		{name: "2xx 不适用", code: 200, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, sameAccountRetryEligible(tc.code))
		})
	}
}

// 组合不变量：工具映射 400 必须「不同账号重试」但「换号」。
func TestToolMapping400_SkipsSameAccountRetryButFailsOver(t *testing.T) {
	svc := &GatewayService{}
	account := &Account{Type: AccountTypeAPIKey}
	resp := toolMappingResponse(400,
		`{"error":{"message":"工具映射不支持: Claude Code Read.pages has no Kiro read_file equivalent"}}`)

	require.False(t, sameAccountRetryEligible(resp.StatusCode), "工具映射 400 不应进入同账号重试循环")
	require.True(t, svc.shouldRetryUpstreamResponse(account, resp), "工具映射 400 应放行到重试耗尽分支")
	require.True(t, svc.shouldFailoverUpstreamResponse(resp), "工具映射 400 应触发换号")
}
