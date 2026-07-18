package service

import (
	"encoding/json"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

func TestCommunityStoreOrderIDFromSnapshot(t *testing.T) {
	id, ok := communityStoreOrderIDFromSnapshot(map[string]any{"community_store_order_id": float64(42)})
	require.True(t, ok)
	require.EqualValues(t, 42, id)

	id, ok = communityStoreOrderIDFromSnapshot(map[string]any{"community_store_order_id": json.Number("77")})
	require.True(t, ok)
	require.EqualValues(t, 77, id)

	_, ok = communityStoreOrderIDFromSnapshot(map[string]any{"community_store_order_id": "not-a-number"})
	require.False(t, ok)
}

func TestCommunityStorePaymentDoesNotGenerateAffiliateRechargeRebate(t *testing.T) {
	order := &dbent.PaymentOrder{OrderType: payment.OrderTypeStore, Amount: 12.50}
	require.Zero(t, affiliateRebateBaseAmount(order))
}
