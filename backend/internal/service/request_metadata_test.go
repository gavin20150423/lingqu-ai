package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

func TestSubPilotRetryDeadlineCannotBeExtendedByFailover(t *testing.T) {
	base := time.UnixMilli(1_000_000)
	firstBudget := 10 * time.Second
	ctx := withSubPilotAttemptTimeoutAt(context.Background(), &AccountSelectionResult{
		SubPilotAttemptTimeout:  8 * time.Second,
		SubPilotRemainingBudget: &firstBudget,
	}, base)

	remaining, ok := subPilotRetryRemainingAt(ctx, base.Add(3*time.Second))
	require.True(t, ok)
	require.Equal(t, 7*time.Second, remaining)

	// A fresh-looking budget from a later account must not restart the clock.
	secondBudget := 10 * time.Second
	ctx = withSubPilotAttemptTimeoutAt(ctx, &AccountSelectionResult{
		SubPilotAttemptTimeout:  6 * time.Second,
		SubPilotRemainingBudget: &secondBudget,
	}, base.Add(3*time.Second))
	remaining, ok = subPilotRetryRemainingAt(ctx, base.Add(3*time.Second))
	require.True(t, ok)
	require.Equal(t, 7*time.Second, remaining)
	require.Equal(t, 6*time.Second, SubPilotAttemptTimeoutFromContext(ctx))

	// A smaller authoritative remainder is allowed to tighten the deadline.
	shorterBudget := 2 * time.Second
	ctx = withSubPilotAttemptTimeoutAt(ctx, &AccountSelectionResult{
		SubPilotAttemptTimeout:  4 * time.Second,
		SubPilotRemainingBudget: &shorterBudget,
	}, base.Add(3*time.Second))
	remaining, ok = subPilotRetryRemainingAt(ctx, base.Add(3*time.Second))
	require.True(t, ok)
	require.Equal(t, 2*time.Second, remaining)
}

func TestSubPilotZeroRemainingBudgetIsExhausted(t *testing.T) {
	zero := time.Duration(0)
	ctx := WithSubPilotAttemptTimeout(context.Background(), &AccountSelectionResult{
		SubPilotAttemptTimeout:  30 * time.Second,
		SubPilotRemainingBudget: &zero,
	})

	require.True(t, SubPilotRetryBudgetExhausted(ctx))
}

func TestSubPilotRetryBudgetClampsOversizedValues(t *testing.T) {
	oversized := 24 * time.Hour
	ctx := withSubPilotAttemptTimeoutAt(context.Background(), &AccountSelectionResult{
		SubPilotRemainingBudget: &oversized,
	}, time.UnixMilli(1_000_000))

	remaining, ok := subPilotRetryRemainingAt(ctx, time.UnixMilli(1_000_000))
	require.True(t, ok)
	require.Equal(t, maxSubPilotRetryBudget, remaining)

	ctx = WithSubPilotRetryDirective(context.Background(), SubPilotRetryDirective{
		Available:         true,
		Action:            "retry_next",
		RemainingBudgetMS: int64((24 * time.Hour) / time.Millisecond),
	})
	remaining, ok = subPilotRetryRemainingAt(ctx, time.Now())
	require.True(t, ok)
	require.LessOrEqual(t, remaining, maxSubPilotRetryBudget)
	require.Greater(t, remaining, maxSubPilotRetryBudget-time.Second)
}

func TestCapSubPilotRetryTimeoutUsesRemainingBudget(t *testing.T) {
	base := time.UnixMilli(1_000_000)
	budget := 10 * time.Second
	ctx := withSubPilotAttemptTimeoutAt(context.Background(), &AccountSelectionResult{
		SubPilotRemainingBudget: &budget,
	}, base)

	require.Equal(t, 7*time.Second, capSubPilotRetryTimeoutAt(ctx, 7*time.Second, base.Add(3*time.Second)))
	require.Equal(t, 7*time.Second, capSubPilotRetryTimeoutAt(ctx, 30*time.Second, base.Add(3*time.Second)))
	require.Zero(t, capSubPilotRetryTimeoutAt(ctx, 30*time.Second, base.Add(11*time.Second)))
}

func TestWithoutSubPilotRetryDeadlineRetainsAttemptTimeout(t *testing.T) {
	budget := 10 * time.Second
	ctx := WithSubPilotAttemptTimeout(context.Background(), &AccountSelectionResult{
		SubPilotAttemptTimeout:  30 * time.Second,
		SubPilotRemainingBudget: &budget,
	})
	ctx = WithoutSubPilotRetryDeadline(ctx)

	_, hasDeadline := subPilotRetryRemainingAt(ctx, time.Now())
	require.False(t, hasDeadline)
	require.Equal(t, 30*time.Second, SubPilotAttemptTimeoutFromContext(ctx))
}

func TestRequestMetadataWriteAndRead_NoBridge(t *testing.T) {
	ctx := context.Background()
	ctx = WithIsMaxTokensOneHaikuRequest(ctx, true, false)
	ctx = WithThinkingEnabled(ctx, true, false)
	ctx = WithPrefetchedStickySession(ctx, 123, 456, false)
	ctx = WithSingleAccountRetry(ctx, true, false)
	ctx = WithAccountSwitchCount(ctx, 2, false)

	isHaiku, ok := IsMaxTokensOneHaikuRequestFromContext(ctx)
	require.True(t, ok)
	require.True(t, isHaiku)

	thinking, ok := ThinkingEnabledFromContext(ctx)
	require.True(t, ok)
	require.True(t, thinking)

	accountID, ok := PrefetchedStickyAccountIDFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, int64(123), accountID)

	groupID, ok := PrefetchedStickyGroupIDFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, int64(456), groupID)

	singleRetry, ok := SingleAccountRetryFromContext(ctx)
	require.True(t, ok)
	require.True(t, singleRetry)

	switchCount, ok := AccountSwitchCountFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, 2, switchCount)

	require.Nil(t, ctx.Value(ctxkey.IsMaxTokensOneHaikuRequest))
	require.Nil(t, ctx.Value(ctxkey.ThinkingEnabled))
	require.Nil(t, ctx.Value(ctxkey.PrefetchedStickyAccountID))
	require.Nil(t, ctx.Value(ctxkey.PrefetchedStickyGroupID))
	require.Nil(t, ctx.Value(ctxkey.SingleAccountRetry))
	require.Nil(t, ctx.Value(ctxkey.AccountSwitchCount))
}

func TestRequestMetadataWrite_BridgeLegacyKeys(t *testing.T) {
	ctx := context.Background()
	ctx = WithIsMaxTokensOneHaikuRequest(ctx, true, true)
	ctx = WithThinkingEnabled(ctx, true, true)
	ctx = WithPrefetchedStickySession(ctx, 123, 456, true)
	ctx = WithSingleAccountRetry(ctx, true, true)
	ctx = WithAccountSwitchCount(ctx, 2, true)

	require.Equal(t, true, ctx.Value(ctxkey.IsMaxTokensOneHaikuRequest))
	require.Equal(t, true, ctx.Value(ctxkey.ThinkingEnabled))
	require.Equal(t, int64(123), ctx.Value(ctxkey.PrefetchedStickyAccountID))
	require.Equal(t, int64(456), ctx.Value(ctxkey.PrefetchedStickyGroupID))
	require.Equal(t, true, ctx.Value(ctxkey.SingleAccountRetry))
	require.Equal(t, 2, ctx.Value(ctxkey.AccountSwitchCount))
}

func TestRequestMetadataRead_LegacyFallbackAndStats(t *testing.T) {
	beforeHaiku, beforeThinking, beforeAccount, beforeGroup, beforeSingleRetry, beforeSwitchCount := RequestMetadataFallbackStats()

	ctx := context.Background()
	ctx = context.WithValue(ctx, ctxkey.IsMaxTokensOneHaikuRequest, true)
	ctx = context.WithValue(ctx, ctxkey.ThinkingEnabled, true)
	ctx = context.WithValue(ctx, ctxkey.PrefetchedStickyAccountID, int64(321))
	ctx = context.WithValue(ctx, ctxkey.PrefetchedStickyGroupID, int64(654))
	ctx = context.WithValue(ctx, ctxkey.SingleAccountRetry, true)
	ctx = context.WithValue(ctx, ctxkey.AccountSwitchCount, int64(3))

	isHaiku, ok := IsMaxTokensOneHaikuRequestFromContext(ctx)
	require.True(t, ok)
	require.True(t, isHaiku)

	thinking, ok := ThinkingEnabledFromContext(ctx)
	require.True(t, ok)
	require.True(t, thinking)

	accountID, ok := PrefetchedStickyAccountIDFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, int64(321), accountID)

	groupID, ok := PrefetchedStickyGroupIDFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, int64(654), groupID)

	singleRetry, ok := SingleAccountRetryFromContext(ctx)
	require.True(t, ok)
	require.True(t, singleRetry)

	switchCount, ok := AccountSwitchCountFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, 3, switchCount)

	afterHaiku, afterThinking, afterAccount, afterGroup, afterSingleRetry, afterSwitchCount := RequestMetadataFallbackStats()
	require.Equal(t, beforeHaiku+1, afterHaiku)
	require.Equal(t, beforeThinking+1, afterThinking)
	require.Equal(t, beforeAccount+1, afterAccount)
	require.Equal(t, beforeGroup+1, afterGroup)
	require.Equal(t, beforeSingleRetry+1, afterSingleRetry)
	require.Equal(t, beforeSwitchCount+1, afterSwitchCount)
}

func TestRequestMetadataRead_PreferMetadataOverLegacy(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxkey.ThinkingEnabled, false)
	ctx = WithThinkingEnabled(ctx, true, false)

	thinking, ok := ThinkingEnabledFromContext(ctx)
	require.True(t, ok)
	require.True(t, thinking)
	require.Equal(t, false, ctx.Value(ctxkey.ThinkingEnabled))
}
