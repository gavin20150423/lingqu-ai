//go:build unit

package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionPlanCurrencyDefaultsToCNY(t *testing.T) {
	require.Equal(t, payment.DefaultPaymentCurrency, subscriptionPlanCurrency(""))
	require.Equal(t, payment.DefaultPaymentCurrency, subscriptionPlanCurrency("   "))
	require.Equal(t, payment.DefaultPaymentCurrency, subscriptionPlanCurrency("not-a-currency"))
	require.Equal(t, "USD", subscriptionPlanCurrency("USD"))
	require.Equal(t, payment.DefaultPaymentCurrency, subscriptionPlanCurrency("RMB"))
}
