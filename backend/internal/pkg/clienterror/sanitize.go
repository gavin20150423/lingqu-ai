// Package clienterror 负责把「上游返回的错误」重建成「可以给客户看的错误」。
//
// 背景：部分上游会在错误体里塞运营文案（例如引导用户加群、导流到别处），
// 一旦原样透传，客户就会被上游撬走。因此网关对客户侧的约束是绝对的：
//   - 上游的自由文本（message / detail / 任何未知字段）一律丢弃，绝不透传；
//   - 只保留白名单内的结构化标识（type / code / Google status），且不是白名单就丢掉；
//   - 文案由网关按状态码生成，不含任何上游内容。
//
// 原始错误体仍然完整记录在 ops（内部排障用），只是不给客户看。
package clienterror

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Shape 决定重建后的 JSON 结构：OpenAI/Anthropic 风格，还是 Google(Gemini) 风格。
type Shape int

const (
	// ShapeOpenAI 输出 {"error":{"type":..,"code":..,"message":..}}
	ShapeOpenAI Shape = iota
	// ShapeGoogle 输出 {"error":{"code":<int>,"status":"..","message":..}}
	ShapeGoogle
)

// 允许透给客户的结构化标识白名单（都是协议里的标准枚举值）。
// 不在表内的一律丢弃——上游完全可以把广告写进 type/code 字段。
var allowedOpenAITypes = map[string]struct{}{
	"invalid_request_error":      {},
	"authentication_error":       {},
	"permission_error":           {},
	"not_found_error":            {},
	"rate_limit_error":           {},
	"insufficient_quota":         {},
	"server_error":               {},
	"api_error":                  {},
	"overloaded_error":           {},
	"request_too_large":          {},
	"too_many_requests":          {},
	"content_policy_violation":   {},
	"billing_hard_limit_reached": {},
}

var allowedOpenAICodes = map[string]struct{}{
	"invalid_request_error":      {},
	"authentication_error":       {},
	"permission_error":           {},
	"not_found_error":            {},
	"rate_limit_error":           {},
	"insufficient_quota":         {},
	"server_error":               {},
	"api_error":                  {},
	"overloaded_error":           {},
	"request_too_large":          {},
	"too_many_requests":          {},
	"content_policy_violation":   {},
	"context_length_exceeded":    {},
	"model_not_found":            {},
	"invalid_api_key":            {},
	"billing_hard_limit_reached": {},
}

var allowedGoogleStatuses = map[string]struct{}{
	"INVALID_ARGUMENT":    {},
	"FAILED_PRECONDITION": {},
	"PERMISSION_DENIED":   {},
	"NOT_FOUND":           {},
	"RESOURCE_EXHAUSTED":  {},
	"INTERNAL":            {},
	"UNAVAILABLE":         {},
	"DEADLINE_EXCEEDED":   {},
	"UNAUTHENTICATED":     {},
	"ABORTED":             {},
	"OUT_OF_RANGE":        {},
	"UNIMPLEMENTED":       {},
}

// Sanitize 把上游错误体重建成可以给客户看的形式。
//
// statusCode 用于生成文案与兜底标识；body 只用于提取白名单内的结构化标识，
// 其自由文本一律丢弃。任何解析失败都退化为「纯网关文案」，保证不泄漏。
func Sanitize(shape Shape, statusCode int, body []byte) []byte {
	fallbackType, fallbackCode, fallbackGoogleStatus := classifyStatus(statusCode)

	upstreamType, upstreamCode, upstreamStatus := extractStructuredIDs(body)

	if shape == ShapeGoogle {
		status := ""
		if _, ok := allowedGoogleStatuses[upstreamStatus]; ok {
			status = upstreamStatus
		}
		if status == "" {
			status = fallbackGoogleStatus
		}
		payload := map[string]any{
			"error": map[string]any{
				"code":    statusCode,
				"message": messageForStatus(statusCode),
				"status":  status,
			},
		}
		return mustMarshal(payload)
	}

	outType := fallbackType
	if _, ok := allowedOpenAITypes[upstreamType]; ok {
		outType = upstreamType
	}
	// code 允许缺省：OpenAI 客户端按 type 分支，code 仅作细化。
	outCode := ""
	if _, ok := allowedOpenAICodes[upstreamCode]; ok {
		outCode = upstreamCode
	} else if _, ok := allowedOpenAICodes[fallbackCode]; ok {
		outCode = fallbackCode
	}

	errObj := map[string]any{
		"type":    outType,
		"message": messageForStatus(statusCode),
	}
	if outCode != "" {
		errObj["code"] = outCode
	}
	return mustMarshal(map[string]any{"error": errObj})
}

// extractStructuredIDs 尽力从上游错误体里取出 type/code/status 三个标识，取值本身不做信任。
func extractStructuredIDs(body []byte) (errType, errCode, googleStatus string) {
	if len(body) == 0 {
		return "", "", ""
	}
	var parsed any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", "", ""
	}
	root, ok := parsed.(map[string]any)
	if !ok {
		return "", "", ""
	}
	errObj, ok := root["error"].(map[string]any)
	if !ok {
		// 有些上游把错误放在顶层，这里只取白名单字段，不取 message。
		if v, ok := root["type"].(string); ok {
			errType = strings.TrimSpace(v)
		}
		if v, ok := root["code"].(string); ok {
			errCode = strings.TrimSpace(v)
		}
		return errType, errCode, ""
	}
	if v, ok := errObj["type"].(string); ok {
		errType = strings.TrimSpace(v)
	}
	if v, ok := errObj["code"].(string); ok {
		errCode = strings.TrimSpace(v)
	}
	// Google 风格：error.status 是字符串枚举，error.code 是数字
	if v, ok := errObj["status"].(string); ok {
		googleStatus = strings.TrimSpace(v)
	}
	return errType, errCode, googleStatus
}

func classifyStatus(statusCode int) (errType, errCode, googleStatus string) {
	switch {
	case statusCode == http.StatusBadRequest:
		return "invalid_request_error", "invalid_request_error", "INVALID_ARGUMENT"
	case statusCode == http.StatusUnauthorized:
		return "authentication_error", "authentication_error", "UNAUTHENTICATED"
	case statusCode == http.StatusForbidden:
		return "permission_error", "permission_error", "PERMISSION_DENIED"
	case statusCode == http.StatusNotFound:
		return "not_found_error", "not_found_error", "NOT_FOUND"
	case statusCode == http.StatusRequestEntityTooLarge:
		return "request_too_large", "request_too_large", "INVALID_ARGUMENT"
	case statusCode == http.StatusTooManyRequests:
		return "rate_limit_error", "rate_limit_error", "RESOURCE_EXHAUSTED"
	case statusCode >= 500:
		return "server_error", "server_error", "UNAVAILABLE"
	default:
		return "api_error", "", "INTERNAL"
	}
}

// messageForStatus 是给客户看的文案，全部由网关自己生成，不含任何上游内容。
func messageForStatus(statusCode int) string {
	switch {
	case statusCode == http.StatusBadRequest:
		return "The request was rejected as invalid. Please check the parameters and try again."
	case statusCode == http.StatusUnauthorized:
		return "Authentication failed. Please check your credentials."
	case statusCode == http.StatusForbidden:
		return "You do not have permission to perform this request."
	case statusCode == http.StatusNotFound:
		return "The requested resource was not found."
	case statusCode == http.StatusRequestEntityTooLarge:
		return "The request payload is too large."
	case statusCode == http.StatusTooManyRequests:
		return "Rate limit reached. Please retry later."
	case statusCode >= 500:
		return "The upstream service is temporarily unavailable. Please retry later."
	default:
		return "The request failed. Please retry later."
	}
}

func mustMarshal(v any) []byte {
	buf, err := json.Marshal(v)
	if err != nil {
		// 理论上不可达（纯 map[string]any），兜底文案仍然不能带上游内容。
		return []byte(`{"error":{"type":"api_error","message":"The request failed. Please retry later."}}`)
	}
	return buf
}
