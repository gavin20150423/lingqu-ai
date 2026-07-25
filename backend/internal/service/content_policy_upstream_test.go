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
