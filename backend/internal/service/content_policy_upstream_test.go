package service

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestIsUpstreamContentPolicyBody(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   bool
	}{
		{"review refusal", http.StatusForbidden, `{"error":{"message":"This request has been flagged by the content review system and blocked according to the usage policy."}}`, true},
		{"structured marker", http.StatusForbidden, `{"type":"content_policy_violation"}`, true},
		{"upstream policy error", http.StatusForbidden, `{"error":{"message":"Request blocked by upstream content policy","type":"content_policy_error"}}`, true},
		{"permission failure", http.StatusForbidden, `{"error":{"message":"permission denied"}}`, false},
		{"wrong status", http.StatusBadRequest, `request blocked by content policy`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isUpstreamContentPolicyBody(tt.status, []byte(tt.body)); got != tt.want {
				t.Fatalf("isUpstreamContentPolicyBody() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsUpstreamRequestValidationBody(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   bool
	}{
		{"missing messages from Claude", http.StatusInternalServerError, "{\"error\":{\"message\":\"field messages is required (request id: req-1)\",\"type\":\"new_api_error\"},\"type\":\"error\"}", true},
		{"missing model from Claude", http.StatusInternalServerError, "{\"error\":{\"message\":\"field model is required\",\"type\":\"new_api_error\"}}", true},
		{"wrong error type", http.StatusInternalServerError, "{\"error\":{\"message\":\"field messages is required\",\"type\":\"invalid_request_error\"}}", false},
		{"ordinary upstream 500", http.StatusInternalServerError, "{\"error\":{\"message\":\"temporary failure\",\"type\":\"new_api_error\"}}", false},
		{"already valid client status", http.StatusBadRequest, "{\"error\":{\"message\":\"field messages is required\",\"type\":\"new_api_error\"}}", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isUpstreamRequestValidationBody(tt.status, []byte(tt.body)); got != tt.want {
				t.Fatalf("isUpstreamRequestValidationBody() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRequestValidationResponseSkipsRetryAndFailover(t *testing.T) {
	svc := &GatewayService{}
	account := &Account{Type: AccountTypeAPIKey}
	body := "{\"error\":{\"message\":\"field messages is required (request id: req-1)\",\"type\":\"new_api_error\"}}"

	retryResp := &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(body))}
	if svc.shouldRetryUpstreamResponse(account, retryResp) {
		t.Fatal("request validation error must not retry")
	}
	failoverResp := &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(body))}
	if svc.shouldFailoverUpstreamResponse(failoverResp) {
		t.Fatal("request validation error must not fail over")
	}
}

func TestHandleErrorResponseReturnsRequestValidationBodyAsBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	svc := &GatewayService{}
	account := &Account{ID: 1, Name: "test", Type: AccountTypeAPIKey, Platform: PlatformAnthropic}
	body := []byte("{\"error\":{\"message\":\"field messages is required (request id: req-1)\",\"type\":\"new_api_error\"},\"type\":\"error\"}")
	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(body)),
	}

	_, err := svc.handleErrorResponse(t.Context(), resp, c, account, "claude-opus-5")
	if err == nil {
		t.Fatal("expected upstream request validation error")
	}
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if strings.TrimSpace(recorder.Body.String()) != string(body) {
		t.Fatalf("response body = %s, want %s", recorder.Body.String(), body)
	}
}

func TestIsUpstreamContentPolicyResponseRestoresBody(t *testing.T) {
	body := `{"error":{"message":"request rejected by content policy"}}` + strings.Repeat("x", upstreamContentPolicyInspectLimit)
	resp := &http.Response{StatusCode: http.StatusForbidden, Body: io.NopCloser(strings.NewReader(body))}
	if !isUpstreamContentPolicyResponse(resp) {
		t.Fatal("expected content policy response")
	}
	restored, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(restored) != body {
		t.Fatalf("body was not restored: %q", restored)
	}
}

func TestContentPolicyResponseSkipsRetryAndFailover(t *testing.T) {
	svc := &GatewayService{}
	account := &Account{Type: AccountTypeOAuth}
	policyBody := `{"error":{"message":"This request has been flagged by the content review system and blocked according to the usage policy."}}`

	retryResp := &http.Response{StatusCode: http.StatusForbidden, Body: io.NopCloser(strings.NewReader(policyBody))}
	if svc.shouldRetryUpstreamResponse(account, retryResp) {
		t.Fatal("content policy refusal must not retry")
	}
	failoverResp := &http.Response{StatusCode: http.StatusForbidden, Body: io.NopCloser(strings.NewReader(policyBody))}
	if svc.shouldFailoverUpstreamResponse(failoverResp) {
		t.Fatal("content policy refusal must not fail over")
	}

	permissionBody := `{"error":{"message":"permission denied"}}`
	retryResp = &http.Response{StatusCode: http.StatusForbidden, Body: io.NopCloser(strings.NewReader(permissionBody))}
	if !svc.shouldRetryUpstreamResponse(account, retryResp) {
		t.Fatal("ordinary OAuth 403 must retain retry behavior")
	}
	failoverResp = &http.Response{StatusCode: http.StatusForbidden, Body: io.NopCloser(strings.NewReader(permissionBody))}
	if !svc.shouldFailoverUpstreamResponse(failoverResp) {
		t.Fatal("ordinary 403 must retain failover behavior")
	}
}

func TestHandleErrorResponseReturnsContentPolicy403(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	svc := &GatewayService{}
	account := &Account{ID: 1, Name: "test", Type: AccountTypeOAuth, Platform: PlatformAnthropic}
	body := []byte(`{"error":{"message":"This request has been flagged by the content review system and blocked according to the usage policy."}}`)
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(body)),
	}

	_, err := svc.handleErrorResponse(t.Context(), resp, c, account, "claude-sonnet-4-6")
	if err == nil {
		t.Fatal("expected upstream content policy error")
	}
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusForbidden, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"type":"content_policy_error"`) {
		t.Fatalf("unexpected response body: %s", recorder.Body.String())
	}
}
