package service

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// SubscriptionEntitlementDebiter is implemented by the subscription repository
// to atomically consume a model/platform gift balance after a request succeeds.
// It is deliberately optional on UserSubscriptionRepository so lightweight
// service test doubles and older adapters remain source-compatible.
type SubscriptionEntitlementDebiter interface {
	DeductSubscriptionEntitlement(ctx context.Context, subscriptionID int64, entitlementKey string, amountUSD float64) (bool, error)
}

// SubscriptionEntitlementReader lets the billing preflight refresh just the
// independent balances without expanding UserSubscriptionRepository's port.
type SubscriptionEntitlementReader interface {
	GetSubscriptionEntitlements(ctx context.Context, subscriptionID int64) (map[string]any, error)
}

// ErrSubscriptionEntitlementExhausted is returned before forwarding when the
// requested platform has a configured gift pool with no remaining balance.
var ErrSubscriptionEntitlementExhausted = infraerrors.TooManyRequests(
	"SUBSCRIPTION_ENTITLEMENT_EXHAUSTED",
	"The included model or platform allowance has been exhausted.",
)

// SubscriptionEntitlementBalance resolves the configured gift pool for a
// request. Keys are intentionally flexible so administrators can configure
// familiar product names (gpt, ccmax, kiro) or the concrete gateway platform
// name (openai, anthropic, antigravity). The returned balance is in USD.
func SubscriptionEntitlementBalance(subscription *UserSubscription, group *Group, platform string) (key string, balance float64, ok bool) {
	if subscription == nil || len(subscription.Entitlements) == 0 {
		return "", 0, false
	}

	values := make(map[string]float64, len(subscription.Entitlements))
	originalKeys := make(map[string]string, len(subscription.Entitlements))
	for rawKey, rawValue := range subscription.Entitlements {
		key := normalizeEntitlementKey(rawKey)
		if key == "" {
			continue
		}
		value, valid := entitlementNumber(rawValue)
		if !valid || math.IsNaN(value) || math.IsInf(value, 0) {
			continue
		}
		values[key] = value
		originalKeys[key] = strings.TrimSpace(rawKey)
	}

	for _, candidate := range subscriptionEntitlementCandidates(group, platform) {
		candidate = normalizeEntitlementKey(candidate)
		if candidate == "" {
			continue
		}
		if value, exists := values[candidate]; exists {
			return originalKeys[candidate], value, true
		}
	}
	return "", 0, false
}

func subscriptionEntitlementCandidates(group *Group, platform string) []string {
	seen := make(map[string]struct{}, 12)
	var candidates []string
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		key := normalizeEntitlementKey(value)
		if key == "" {
			return
		}
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		candidates = append(candidates, value)
	}
	add(platform)
	if group != nil {
		add(group.Platform)
	}

	// Product aliases keep the admin configuration readable while still using
	// the resolved target platform for composite groups and forced routes.
	for _, source := range append([]string{platform}, groupPlatformValues(group)...) {
		switch normalizeEntitlementKey(source) {
		case "openai", "openai_compatible", "openai_compat", "gpt":
			add("gpt")
			add("openai")
		case "anthropic", "claude", "claude_code", "ccmax":
			add("ccmax")
			add("claude")
			add("anthropic")
		case "antigravity", "kiro":
			add("kiro")
			add("antigravity")
		}
	}
	// A group name is the final fallback. This ordering is important for a
	// composite group named "ccmax" that also gifts a separate "gpt" pool:
	// OpenAI traffic must select gpt before falling back to the group name.
	if group != nil {
		add(group.Name)
		switch normalizeEntitlementKey(group.Name) {
		case "openai", "openai_compatible", "openai_compat", "gpt":
			add("gpt")
			add("openai")
		case "anthropic", "claude", "claude_code", "ccmax":
			add("ccmax")
			add("claude")
			add("anthropic")
		case "antigravity", "kiro":
			add("kiro")
			add("antigravity")
		}
	}
	return candidates
}

func groupPlatformValues(group *Group) []string {
	if group == nil {
		return nil
	}
	return []string{group.Platform}
}

func normalizeEntitlementKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func entitlementNumber(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int8:
		return float64(typed), true
	case int16:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case uint:
		return float64(typed), true
	case uint8:
		return float64(typed), true
	case uint16:
		return float64(typed), true
	case uint32:
		return float64(typed), true
	case uint64:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}
