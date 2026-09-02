package service

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
)

type requestMetadataContextKey struct{}

var requestMetadataKey = requestMetadataContextKey{}

const maxSubPilotRetryBudget = 5 * time.Minute

type RequestMetadata struct {
	IsMaxTokensOneHaikuRequest  *bool
	ThinkingEnabled             *bool
	PrefetchedStickyAccountID   *int64
	PrefetchedStickyGroupID     *int64
	SingleAccountRetry          *bool
	AccountSwitchCount          *int
	SubPilotDisabled            *bool
	SubPilotAPIKeyID            *int64
	SubPilotRetryDirective      *SubPilotRetryDirective
	SubPilotAttemptTimeoutMS    int64
	SubPilotRetryDeadlineUnixMS int64
}

var (
	requestMetadataFallbackIsMaxTokensOneHaikuTotal atomic.Int64
	requestMetadataFallbackThinkingEnabledTotal     atomic.Int64
	requestMetadataFallbackPrefetchedStickyAccount  atomic.Int64
	requestMetadataFallbackPrefetchedStickyGroup    atomic.Int64
	requestMetadataFallbackSingleAccountRetryTotal  atomic.Int64
	requestMetadataFallbackAccountSwitchCountTotal  atomic.Int64
)

func RequestMetadataFallbackStats() (isMaxTokensOneHaiku, thinkingEnabled, prefetchedStickyAccount, prefetchedStickyGroup, singleAccountRetry, accountSwitchCount int64) {
	return requestMetadataFallbackIsMaxTokensOneHaikuTotal.Load(),
		requestMetadataFallbackThinkingEnabledTotal.Load(),
		requestMetadataFallbackPrefetchedStickyAccount.Load(),
		requestMetadataFallbackPrefetchedStickyGroup.Load(),
		requestMetadataFallbackSingleAccountRetryTotal.Load(),
		requestMetadataFallbackAccountSwitchCountTotal.Load()
}

func metadataFromContext(ctx context.Context) *RequestMetadata {
	if ctx == nil {
		return nil
	}
	md, _ := ctx.Value(requestMetadataKey).(*RequestMetadata)
	return md
}

func updateRequestMetadata(
	ctx context.Context,
	bridgeOldKeys bool,
	update func(md *RequestMetadata),
	legacyBridge func(ctx context.Context) context.Context,
) context.Context {
	if ctx == nil {
		return nil
	}
	current := metadataFromContext(ctx)
	next := &RequestMetadata{}
	if current != nil {
		*next = *current
	}
	update(next)
	ctx = context.WithValue(ctx, requestMetadataKey, next)
	if bridgeOldKeys && legacyBridge != nil {
		ctx = legacyBridge(ctx)
	}
	return ctx
}

func WithIsMaxTokensOneHaikuRequest(ctx context.Context, value bool, bridgeOldKeys bool) context.Context {
	return updateRequestMetadata(ctx, bridgeOldKeys, func(md *RequestMetadata) {
		v := value
		md.IsMaxTokensOneHaikuRequest = &v
	}, func(base context.Context) context.Context {
		return context.WithValue(base, ctxkey.IsMaxTokensOneHaikuRequest, value)
	})
}

func WithThinkingEnabled(ctx context.Context, value bool, bridgeOldKeys bool) context.Context {
	return updateRequestMetadata(ctx, bridgeOldKeys, func(md *RequestMetadata) {
		v := value
		md.ThinkingEnabled = &v
	}, func(base context.Context) context.Context {
		return context.WithValue(base, ctxkey.ThinkingEnabled, value)
	})
}

func WithPrefetchedStickySession(ctx context.Context, accountID, groupID int64, bridgeOldKeys bool) context.Context {
	return updateRequestMetadata(ctx, bridgeOldKeys, func(md *RequestMetadata) {
		account := accountID
		group := groupID
		md.PrefetchedStickyAccountID = &account
		md.PrefetchedStickyGroupID = &group
	}, func(base context.Context) context.Context {
		bridged := context.WithValue(base, ctxkey.PrefetchedStickyAccountID, accountID)
		return context.WithValue(bridged, ctxkey.PrefetchedStickyGroupID, groupID)
	})
}

func WithSingleAccountRetry(ctx context.Context, value bool, bridgeOldKeys bool) context.Context {
	return updateRequestMetadata(ctx, bridgeOldKeys, func(md *RequestMetadata) {
		v := value
		md.SingleAccountRetry = &v
	}, func(base context.Context) context.Context {
		return context.WithValue(base, ctxkey.SingleAccountRetry, value)
	})
}

func WithAccountSwitchCount(ctx context.Context, value int, bridgeOldKeys bool) context.Context {
	return updateRequestMetadata(ctx, bridgeOldKeys, func(md *RequestMetadata) {
		v := value
		md.AccountSwitchCount = &v
	}, func(base context.Context) context.Context {
		return context.WithValue(base, ctxkey.AccountSwitchCount, value)
	})
}

func WithSubPilotDisabled(ctx context.Context, value bool) context.Context {
	return updateRequestMetadata(ctx, false, func(md *RequestMetadata) {
		v := value
		md.SubPilotDisabled = &v
	}, nil)
}

func WithSubPilotAPIKeyID(ctx context.Context, value int64) context.Context {
	return updateRequestMetadata(ctx, false, func(md *RequestMetadata) {
		v := value
		md.SubPilotAPIKeyID = &v
	}, nil)
}

func WithSubPilotRetryDirective(ctx context.Context, value SubPilotRetryDirective) context.Context {
	return updateRequestMetadata(ctx, false, func(md *RequestMetadata) {
		directive := value
		md.SubPilotRetryDirective = &directive
		// The failure response is authoritative for the budget left after the
		// attempt. Tighten the local deadline when it is present, but keep the
		// initial deadline when older SubPilot versions omit the header.
		if value.Available && value.RemainingBudgetMS > 0 {
			remainingBudgetMS := value.RemainingBudgetMS
			if remainingBudgetMS > maxSubPilotRetryBudget.Milliseconds() {
				remainingBudgetMS = maxSubPilotRetryBudget.Milliseconds()
			}
			candidateDeadline := time.Now().Add(time.Duration(remainingBudgetMS) * time.Millisecond).UnixMilli()
			if md.SubPilotRetryDeadlineUnixMS <= 0 || candidateDeadline < md.SubPilotRetryDeadlineUnixMS {
				md.SubPilotRetryDeadlineUnixMS = candidateDeadline
			}
		}
	}, nil)
}

func SubPilotRetryDirectiveFromContext(ctx context.Context) (SubPilotRetryDirective, bool) {
	if md := metadataFromContext(ctx); md != nil && md.SubPilotRetryDirective != nil {
		return *md.SubPilotRetryDirective, true
	}
	return SubPilotRetryDirective{}, false
}

func WithSubPilotAttemptTimeout(ctx context.Context, selection *AccountSelectionResult) context.Context {
	return withSubPilotAttemptTimeoutAt(ctx, selection, time.Now())
}

// withSubPilotAttemptTimeoutAt installs the selected account's attempt limit
// while preserving a request-level retry deadline. A later dispatch response
// can only reduce that deadline; it must never give a failover another full
// retry budget.
func withSubPilotAttemptTimeoutAt(ctx context.Context, selection *AccountSelectionResult, now time.Time) context.Context {
	if selection == nil {
		// Preserve the historical clearing semantics for callers that reset the
		// account-specific timeout between attempts; no request deadline is
		// created or changed when there is no new selection.
		return updateRequestMetadata(ctx, false, func(md *RequestMetadata) {
			md.SubPilotAttemptTimeoutMS = 0
		}, nil)
	}
	return updateRequestMetadata(ctx, false, func(md *RequestMetadata) {
		md.SubPilotAttemptTimeoutMS = selection.SubPilotAttemptTimeout.Milliseconds()
		if selection.SubPilotRemainingBudget != nil {
			remaining := *selection.SubPilotRemainingBudget
			if remaining < 0 {
				remaining = 0
			}
			if remaining > maxSubPilotRetryBudget {
				remaining = maxSubPilotRetryBudget
			}
			candidateDeadline := now.Add(remaining).UnixMilli()
			if md.SubPilotRetryDeadlineUnixMS <= 0 || candidateDeadline < md.SubPilotRetryDeadlineUnixMS {
				md.SubPilotRetryDeadlineUnixMS = candidateDeadline
			}
		}
	}, nil)
}

func SubPilotAttemptTimeoutFromContext(ctx context.Context) time.Duration {
	if md := metadataFromContext(ctx); md != nil && md.SubPilotAttemptTimeoutMS > 0 {
		return time.Duration(md.SubPilotAttemptTimeoutMS) * time.Millisecond
	}
	return 0
}

func subPilotRetryRemainingAt(ctx context.Context, now time.Time) (time.Duration, bool) {
	if md := metadataFromContext(ctx); md != nil && md.SubPilotRetryDeadlineUnixMS > 0 {
		if now.IsZero() {
			now = time.Now()
		}
		remaining := time.UnixMilli(md.SubPilotRetryDeadlineUnixMS).Sub(now)
		if remaining <= 0 {
			return 0, true
		}
		return remaining, true
	}
	return 0, false
}

func SubPilotRetryBudgetExhausted(ctx context.Context) bool {
	remaining, hasDeadline := subPilotRetryRemainingAt(ctx, time.Now())
	return hasDeadline && remaining <= 0
}

// CapSubPilotRetryTimeout limits a wait/attempt timeout to the request-level
// retry budget when one is present. It deliberately does not create a context
// deadline; long-lived transports still need their parent lifecycle context.
func CapSubPilotRetryTimeout(ctx context.Context, configured time.Duration) time.Duration {
	return capSubPilotRetryTimeoutAt(ctx, configured, time.Now())
}

func capSubPilotRetryTimeoutAt(ctx context.Context, configured time.Duration, now time.Time) time.Duration {
	if configured <= 0 {
		return configured
	}
	remaining, hasDeadline := subPilotRetryRemainingAt(ctx, now)
	if !hasDeadline || remaining >= configured {
		return configured
	}
	if remaining <= 0 {
		return 0
	}
	return remaining
}

// WithoutSubPilotRetryDeadline removes only the dispatch-sequence deadline,
// retaining the selected account's per-attempt timeout. Long-lived transports
// such as WebSockets call this after the initial account/credential admission;
// subsequent response.create turns are independent requests.
func WithoutSubPilotRetryDeadline(ctx context.Context) context.Context {
	return updateRequestMetadata(ctx, false, func(md *RequestMetadata) {
		md.SubPilotRetryDeadlineUnixMS = 0
	}, nil)
}

func SubPilotAPIKeyIDFromContext(ctx context.Context) (int64, bool) {
	if md := metadataFromContext(ctx); md != nil && md.SubPilotAPIKeyID != nil {
		return *md.SubPilotAPIKeyID, true
	}
	return 0, false
}

func IsMaxTokensOneHaikuRequestFromContext(ctx context.Context) (bool, bool) {
	if md := metadataFromContext(ctx); md != nil && md.IsMaxTokensOneHaikuRequest != nil {
		return *md.IsMaxTokensOneHaikuRequest, true
	}
	if ctx == nil {
		return false, false
	}
	if value, ok := ctx.Value(ctxkey.IsMaxTokensOneHaikuRequest).(bool); ok {
		requestMetadataFallbackIsMaxTokensOneHaikuTotal.Add(1)
		return value, true
	}
	return false, false
}

func ThinkingEnabledFromContext(ctx context.Context) (bool, bool) {
	if md := metadataFromContext(ctx); md != nil && md.ThinkingEnabled != nil {
		return *md.ThinkingEnabled, true
	}
	if ctx == nil {
		return false, false
	}
	if value, ok := ctx.Value(ctxkey.ThinkingEnabled).(bool); ok {
		requestMetadataFallbackThinkingEnabledTotal.Add(1)
		return value, true
	}
	return false, false
}

func PrefetchedStickyGroupIDFromContext(ctx context.Context) (int64, bool) {
	if md := metadataFromContext(ctx); md != nil && md.PrefetchedStickyGroupID != nil {
		return *md.PrefetchedStickyGroupID, true
	}
	if ctx == nil {
		return 0, false
	}
	v := ctx.Value(ctxkey.PrefetchedStickyGroupID)
	switch t := v.(type) {
	case int64:
		requestMetadataFallbackPrefetchedStickyGroup.Add(1)
		return t, true
	case int:
		requestMetadataFallbackPrefetchedStickyGroup.Add(1)
		return int64(t), true
	}
	return 0, false
}

func PrefetchedStickyAccountIDFromContext(ctx context.Context) (int64, bool) {
	if md := metadataFromContext(ctx); md != nil && md.PrefetchedStickyAccountID != nil {
		return *md.PrefetchedStickyAccountID, true
	}
	if ctx == nil {
		return 0, false
	}
	v := ctx.Value(ctxkey.PrefetchedStickyAccountID)
	switch t := v.(type) {
	case int64:
		requestMetadataFallbackPrefetchedStickyAccount.Add(1)
		return t, true
	case int:
		requestMetadataFallbackPrefetchedStickyAccount.Add(1)
		return int64(t), true
	}
	return 0, false
}

func SingleAccountRetryFromContext(ctx context.Context) (bool, bool) {
	if md := metadataFromContext(ctx); md != nil && md.SingleAccountRetry != nil {
		return *md.SingleAccountRetry, true
	}
	if ctx == nil {
		return false, false
	}
	if value, ok := ctx.Value(ctxkey.SingleAccountRetry).(bool); ok {
		requestMetadataFallbackSingleAccountRetryTotal.Add(1)
		return value, true
	}
	return false, false
}

func AccountSwitchCountFromContext(ctx context.Context) (int, bool) {
	if md := metadataFromContext(ctx); md != nil && md.AccountSwitchCount != nil {
		return *md.AccountSwitchCount, true
	}
	if ctx == nil {
		return 0, false
	}
	v := ctx.Value(ctxkey.AccountSwitchCount)
	switch t := v.(type) {
	case int:
		requestMetadataFallbackAccountSwitchCountTotal.Add(1)
		return t, true
	case int64:
		requestMetadataFallbackAccountSwitchCountTotal.Add(1)
		return int(t), true
	}
	return 0, false
}

func SubPilotDisabledFromContext(ctx context.Context) bool {
	md := metadataFromContext(ctx)
	return md != nil && md.SubPilotDisabled != nil && *md.SubPilotDisabled
}
