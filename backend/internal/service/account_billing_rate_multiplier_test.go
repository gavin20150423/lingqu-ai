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

func TestAccount_VideoPreauthorizationAmountUsesConfiguredCeilingAndMultiplier(t *testing.T) {
	rate := 1.5
	a := Account{
		RateMultiplier: &rate,
		Credentials: map[string]any{
			OpenAIVideoPreauthorizationAmountCredentialKey: json.Number("12.25"),
		},
	}

	amount, ok := a.VideoPreauthorizationAmount()
	require.True(t, ok)
	require.InDelta(t, 18.375, amount, 0.00000001)
}

func TestAccount_VideoPreauthorizationAmountRequiresPositiveFiniteCeiling(t *testing.T) {
	for _, value := range []any{nil, 0, -1, "not-a-number"} {
		a := Account{Credentials: map[string]any{OpenAIVideoPreauthorizationAmountCredentialKey: value}}
		_, ok := a.VideoPreauthorizationAmount()
		require.False(t, ok)
	}
}

func TestAccount_VideoPreauthorizationAmountAllowsZeroBillingMultiplier(t *testing.T) {
	rate := 0.0
	a := Account{
		RateMultiplier: &rate,
		Credentials: map[string]any{
			OpenAIVideoPreauthorizationAmountCredentialKey: 10,
		},
	}

	amount, ok := a.VideoPreauthorizationAmount()
	require.True(t, ok)
	require.Zero(t, amount)
}
