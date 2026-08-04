package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type xiaoAccountTestRepo struct {
	AccountRepository
	account *Account
}

func (r *xiaoAccountTestRepo) GetByID(context.Context, int64) (*Account, error) {
	return r.account, nil
}

type xiaoAccountTestUpstream struct {
	request *http.Request
	resp    *http.Response
}

func (u *xiaoAccountTestUpstream) Do(*http.Request, string, int64, int) (*http.Response, error) {
	return nil, nil
}

func (u *xiaoAccountTestUpstream) DoWithTLS(req *http.Request, _ string, _ int64, _ int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	u.request = req
	return u.resp, nil
}

func TestAccountTestServiceXiaoAPIUsesReadOnlyModelsProbe(t *testing.T) {
	gin.SetMode(gin.TestMode)
	account := &Account{
		ID:          42,
		Platform:    PlatformXiaoAPI,
		Type:        AccountTypeAPIKey,
		Concurrency: 2,
		Credentials: map[string]any{
			"base_url": "https://provider.example/custom/v1",
			"api_key":  "provider-secret",
			XiaoVideoPricingCredentialKey: []any{
				map[string]any{"model": "public-video", "resolution": "720p", "price_per_second": 1, "default_duration": 4},
			},
		},
	}
	upstream := &xiaoAccountTestUpstream{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(`{"object":"list","data":[{"id":"provider-video"}]}`)),
	}}
	svc := NewAccountTestService(
		&xiaoAccountTestRepo{account: account}, nil, nil, nil, nil, upstream,
		&config.Config{}, nil,
	)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/42/test", nil)

	require.NoError(t, svc.TestAccountConnection(c, account.ID, "", "", AccountTestModeDefault))
	require.NotNil(t, upstream.request)
	require.Equal(t, http.MethodGet, upstream.request.Method)
	require.Equal(t, "https://provider.example/custom/v1/models", upstream.request.URL.String())
	require.Equal(t, "Bearer provider-secret", upstream.request.Header.Get("Authorization"))
	require.Contains(t, recorder.Body.String(), `"type":"test_complete"`)
	require.Contains(t, recorder.Body.String(), `"success":true`)
}
