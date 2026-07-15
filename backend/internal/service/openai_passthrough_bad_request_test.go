package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestIsOpenAIAccountSpecificBadRequest(t *testing.T) {
	tests := []struct {
		name string
		body string
		want bool
	}{
		{
			name: "model access",
			body: `{"error":{"message":"This account does not have access to model gpt-5.5"}}`,
			want: true,
		},
		{
			name: "model code",
			body: `{"error":{"code":"model_not_available","message":"model unavailable"}}`,
			want: true,
		},
		{
			name: "workspace deactivated",
			body: `{"response":{"error":{"code":"deactivated_workspace","message":"Workspace has been deactivated"}}}`,
			want: true,
		},
		{
			name: "client parameter error",
			body: `{"error":{"code":"invalid_request_error","message":"Missing required parameter: instructions"}}`,
			want: false,
		},
		{
			name: "context window",
			body: `{"error":{"message":"Your input exceeds the context window of this model"}}`,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := []byte(tt.body)
			require.Equal(t, tt.want, isOpenAIAccountSpecificBadRequest(http.StatusBadRequest, extractUpstreamErrorMessage(body), body))
		})
	}
}

func TestShouldFailoverOpenAIPassthroughResponseRecognizesAccountSpecificBadRequest(t *testing.T) {
	body := []byte(`{"error":{"message":"This account does not have access to model gpt-5.5"}}`)
	svc := &OpenAIGatewayService{}
	require.True(t, svc.shouldFailoverOpenAIPassthroughResponse(nil, http.StatusBadRequest, body))
}

func TestOpenAIPassthroughRecognized400TriggersFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(nil))
	c.Request.Header.Set("User-Agent", "codex_cli_rs/0.1.0")

	upstreamBody := `{"error":{"message":"This account does not have access to model gpt-5.5","code":"model_not_available"}}`
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(upstreamBody)),
	}}
	svc := &OpenAIGatewayService{
		cfg:          &config.Config{Gateway: config.GatewayConfig{}},
		httpUpstream: upstream,
	}
	account := &Account{
		ID:             123,
		Name:           "acc",
		Platform:       PlatformOpenAI,
		Type:           AccountTypeOAuth,
		Concurrency:    1,
		Credentials:    map[string]any{"access_token": "oauth-token", "chatgpt_account_id": "chatgpt-acc"},
		Extra:          map[string]any{"openai_passthrough": true},
		Status:         StatusActive,
		Schedulable:    true,
		RateMultiplier: f64p(1),
	}
	requestBody := []byte(`{"model":"gpt-5.5","stream":false,"instructions":"test","input":"hi"}`)

	_, err := svc.Forward(context.Background(), c, account, requestBody)
	require.Error(t, err)
	var failoverErr *UpstreamFailoverError
	require.True(t, errors.As(err, &failoverErr))
	require.Equal(t, http.StatusBadRequest, failoverErr.StatusCode)
	require.JSONEq(t, upstreamBody, string(failoverErr.ResponseBody))
	require.False(t, c.Writer.Written())
}

func TestExtractUpstreamErrorMessageResponsesWrapper(t *testing.T) {
	body := []byte(`{"response":{"error":{"message":"wrapped upstream detail"}}}`)
	require.Equal(t, "wrapped upstream detail", extractUpstreamErrorMessage(body))
}
