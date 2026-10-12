// upstream_tool_mapping.go
//
// 上游「工具映射不支持」的语义识别。
//
// 背景（2026-10-12 事故）：Claude Code 新增了 `Read.pages` 工具，而 Kiro 上游
// （pool.chaozhiyuanai.com）尚未跟进等价实现，于是对带该工具的请求主动返回：
//
//	400 {"error":{"type":"invalid_request_error",
//	     "message":"工具映射不支持: Claude Code Read.pages has no Kiro read_file equivalent"}}
//
// 这类 400 与「请求本身非法」有本质区别：它是**账号能力不足**，换一个已支持该
// 工具的账号就能成功。但 apikey 账号的重试判定
// （shouldRetryUpstreamError = !account.ShouldHandleErrorCode(statusCode)）在未配置
// custom_error_codes 时对 400 恒为「不重试」，导致请求原样失败、客户侧全是报错。
//
// 因此这里单独识别该语义，让它进入 failover（换号）路径；其余 400 行为不变。
package service

import (
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/tidwall/gjson"
)

// upstreamToolMappingBodyLimit 限制了工具映射判定时读取的响应体大小，
// 与请求体校验检测器保持同一量级：这类错误文案很短，读多了纯属浪费。
const upstreamToolMappingBodyLimit = 8 << 10

var (
	// 中文文案（当前 Kiro 上游原文）
	upstreamToolMappingChinese = regexp.MustCompile(`工具映射不支持|没有等价的?\w*工具|不支持的工具`)

	// 英文文案，例如：
	//   "... has no Kiro read_file equivalent"
	//   "tool mapping for X is not supported"
	//   "unsupported tool: Read.pages"
	upstreamToolMappingEnglish = regexp.MustCompile(`(?i)has\s+no\s+[\w.]*\s*\w*\s+equivalent` +
		`|tool\s+mapping[^"]{0,80}?not\s+supported` +
		`|unsupported\s+tool`)
)

// isUpstreamToolMappingUnsupportedBody 判断响应体是否是「上游工具映射不支持」。
//
// 刻意收窄：只接受 400 + 明确工具映射语义，普通参数错误 400 不命中，
// 避免把真正的客户端错误也拖进换号重试（那会白撞 max_switches）。
func isUpstreamToolMappingUnsupportedBody(statusCode int, body []byte) bool {
	if statusCode != http.StatusBadRequest || len(body) == 0 {
		return false
	}
	msg := strings.TrimSpace(extractUpstreamErrorMessage(body))
	if msg == "" {
		msg = string(body)
	}
	if upstreamToolMappingChinese.MatchString(msg) {
		return true
	}
	return upstreamToolMappingEnglish.MatchString(msg)
}

// isUpstreamToolMappingUnsupported 读取响应体判定后会把Body 还原，
// 以便后续日志与错误处理仍能看到原始报文。
func isUpstreamToolMappingUnsupported(resp *http.Response) bool {
	if resp == nil || resp.Body == nil || resp.StatusCode != http.StatusBadRequest {
		return false
	}
	original := resp.Body
	body, err := io.ReadAll(io.LimitReader(original, upstreamToolMappingBodyLimit))
	resp.Body = &replayReadCloser{
		Reader: io.MultiReader(strings.NewReader(string(body)), original),
		closer: original,
	}
	if err != nil && len(body) == 0 {
		return false
	}
	// 同时看解出的 error.message 与原始报文：部分上游会把文案放在顶层而不是 error 里。
	return isUpstreamToolMappingUnsupportedBody(resp.StatusCode, body)
}

// upstreamToolMappingToolName 尽力从报文里提取缺失的工具名，仅用于日志可读性。
// 取不到返回空串，不影响判定。
func upstreamToolMappingToolName(resp *http.Response) string {
	if resp == nil || resp.Body == nil {
		return ""
	}
	original := resp.Body
	body, err := io.ReadAll(io.LimitReader(original, upstreamToolMappingBodyLimit))
	resp.Body = &replayReadCloser{
		Reader: io.MultiReader(strings.NewReader(string(body)), original),
		closer: original,
	}
	if err != nil && len(body) == 0 {
		return ""
	}
	msg := gjson.GetBytes(body, "error.message").String()
	if msg == "" {
		return ""
	}
	// "工具映射不支持: Claude Code Read.pages has no Kiro read_file equivalent"
	if idx := strings.Index(msg, ":"); idx >= 0 && idx+1 < len(msg) {
		rest := strings.TrimSpace(msg[idx+1:])
		if end := strings.Index(rest, " has no "); end > 0 {
			rest = rest[:end]
		}
		fields := strings.Fields(rest)
		if len(fields) > 0 {
			return fields[len(fields)-1]
		}
	}
	return ""
}