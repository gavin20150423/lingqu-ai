package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccount_BillingRateMultiplier_DefaultsToOneWhenNil(t *testing.T) {
	var a Account
	require.NoError(t, json.Unmarshal([]byte(`{"id":1,"name":"acc","status":"active"}`), &a))
	require.Nil(t, a.RateMultiplier)
	require.Equal(t, 1.0, a.BillingRateMultiplier())
}

func TestAccount_BillingRateMultiplier_AllowsZero(t *testing.T) {
	v := 0.0
	a := Account{RateMultiplier: &v}
	require.Equal(t, 0.0, a.BillingRateMultiplier())
}

func TestAccount_BillingRateMultiplier_NegativeFallsBackToOne(t *testing.T) {
	v := -1.0
	a := Account{RateMultiplier: &v}
	require.Equal(t, 1.0, a.BillingRateMultiplier())
}

func TestAccount_XiaoVideoPriceUsesDynamicResolutionDurationAndAudioRates(t *testing.T) {
	a := Account{Credentials: map[string]any{
		XiaoVideoPricingCredentialKey: []any{
			map[string]any{
				"model":                  "video-public",
				"resolution":             "720p",
				"price_per_second":       0.75,
				"audio_price_per_second": 0.25,
				"default_resolution":     true,
				"default_duration":       8,
			},
		},
	}}

	amount, resolution, duration, ok := a.XiaoVideoPrice("video-public", "", 0, true)
	require.True(t, ok)
	require.Equal(t, "720p", resolution)
	require.Equal(t, 8, duration)
	require.InDelta(t, 8.0, amount, 0.00000001)

	amount, resolution, duration, ok = a.XiaoVideoPrice("video-public", "720p", 4, false)
	require.True(t, ok)
	require.Equal(t, "720p", resolution)
	require.Equal(t, 4, duration)
	require.InDelta(t, 3.0, amount, 0.00000001)
}

func TestAccount_XiaoVideoPricingRejectsInvalidAndAmbiguousRules(t *testing.T) {
	tests := []any{
		nil,
		[]any{},
		[]any{map[string]any{"model": "video-public", "resolution": "720p", "price_per_second": -1}},
		[]any{
			map[string]any{"model": "video-public", "resolution": "720p", "price_per_second": 1},
			map[string]any{"model": "video-public", "resolution": "720p", "price_per_second": 2},
		},
	}
	for _, pricing := range tests {
		a := Account{Credentials: map[string]any{XiaoVideoPricingCredentialKey: pricing}}
		_, err := a.XiaoVideoPricingRules()
		require.Error(t, err)
	}
}

func TestAccount_XiaoVideoPriceAllowsFreeConfiguredPrice(t *testing.T) {
	a := Account{Credentials: map[string]any{
		XiaoVideoPricingCredentialKey: []any{
			map[string]any{
				"model":              "video-public",
				"resolution":         "480p",
				"price_per_second":   0,
				"default_resolution": true,
				"default_duration":   4,
			},
		},
	}}

	amount, _, _, ok := a.XiaoVideoPrice("video-public", "", 0, false)
	require.True(t, ok)
	require.Zero(t, amount)
}
