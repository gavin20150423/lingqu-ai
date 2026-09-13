//go:build unit

package handler

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckoutPlanIncludesForSaleState(t *testing.T) {
	raw, err := json.Marshal(checkoutPlan{ForSale: true})
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(raw, &payload))
	require.Equal(t, true, payload["for_sale"])
}
