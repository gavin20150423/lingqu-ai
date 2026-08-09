package service

import (
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestValidateXiaoAPIAccountAcceptsDynamicUpstreamConfiguration(t *testing.T) {
	account := &Account{
		Platform: PlatformXiaoAPI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"base_url": "https://provider.example/custom/prefix/v1",
			"api_key":  "provider-secret",
			"model_mapping": map[string]any{
				"public-video": "provider-video-v2",
			},
			XiaoVideoPricingCredentialKey: []any{
				map[string]any{
					"model":                  "public-video",
					"resolution":             "1080p",
					"price_per_second":       0.8,
					"audio_price_per_second": 0.2,
					"default_resolution":     true,
					"default_duration":       6,
				},
			},
		},
	}

	require.NoError(t, validateXiaoAPIAccount(account))
}

func TestValidateXiaoAPIAccountRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Account)
		reason string
	}{
		{name: "oauth", mutate: func(account *Account) { account.Type = AccountTypeOAuth }, reason: "XIAOAPI_ACCOUNT_TYPE_INVALID"},
		{name: "missing api key", mutate: func(account *Account) { delete(account.Credentials, "api_key") }, reason: "XIAOAPI_API_KEY_REQUIRED"},
		{name: "invalid base url", mutate: func(account *Account) { account.Credentials["base_url"] = "provider.local/v1" }, reason: "XIAOAPI_BASE_URL_INVALID"},
		{name: "invalid video protocol", mutate: func(account *Account) { account.Credentials[XiaoVideoProtocolCredentialKey] = "unknown" }, reason: "XIAOAPI_VIDEO_PROTOCOL_INVALID"},
		{name: "missing pricing", mutate: func(account *Account) { delete(account.Credentials, XiaoVideoPricingCredentialKey) }, reason: "XIAOAPI_VIDEO_PRICING_INVALID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := &Account{
				Platform: PlatformXiaoAPI,
				Type:     AccountTypeAPIKey,
				Credentials: map[string]any{
					"base_url": "https://provider.example/v1",
					"api_key":  "provider-secret",
					XiaoVideoPricingCredentialKey: []any{
						map[string]any{"model": "video", "resolution": "720p", "price_per_second": 1, "default_duration": 4},
					},
				},
			}
			tt.mutate(account)
			err := validateXiaoAPIAccount(account)
			require.Error(t, err)
			require.Equal(t, tt.reason, infraerrors.Reason(err))
		})
	}
}
