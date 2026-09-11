package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSubscriptionEntitlementBalanceUsesProductAliases(t *testing.T) {
	subscription := &UserSubscription{
		Entitlements: map[string]any{
			"ccmax": 12.5,
			"kiro":  8,
			"gpt":   3.25,
		},
	}

	key, balance, ok := SubscriptionEntitlementBalance(subscription, &Group{Platform: PlatformAnthropic}, PlatformAnthropic)
	require.True(t, ok)
	require.Equal(t, "ccmax", key)
	require.Equal(t, 12.5, balance)

	key, balance, ok = SubscriptionEntitlementBalance(subscription, &Group{Platform: PlatformAntigravity}, PlatformAntigravity)
	require.True(t, ok)
	require.Equal(t, "kiro", key)
	require.Equal(t, float64(8), balance)

	key, balance, ok = SubscriptionEntitlementBalance(subscription, &Group{Platform: PlatformOpenAI}, PlatformOpenAI)
	require.True(t, ok)
	require.Equal(t, "gpt", key)
	require.Equal(t, 3.25, balance)
}

func TestSubscriptionEntitlementBalancePoolsRemainIndependent(t *testing.T) {
	subscription := &UserSubscription{
		Entitlements: map[string]any{"gpt": 0, "ccmax": 9},
	}

	_, balance, ok := SubscriptionEntitlementBalance(subscription, &Group{Platform: PlatformOpenAI}, PlatformOpenAI)
	require.True(t, ok)
	require.Zero(t, balance)

	_, balance, ok = SubscriptionEntitlementBalance(subscription, &Group{Platform: PlatformAnthropic}, PlatformAnthropic)
	require.True(t, ok)
	require.Equal(t, float64(9), balance)
}
