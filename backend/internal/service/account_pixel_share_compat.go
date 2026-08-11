// Code extracted from PIXEL-API/PixelAPI for account-sharing compatibility.
package service

import (
	"context"

	"errors"
	"fmt"

	"sort"

	"strings"
	"time"
)

const (
	AccountLevelUnknown = "unknown"
	AccountLevelFree    = "free"
	AccountLevelPlus    = "plus"
	AccountLevelPro     = "pro"
	AccountLevelTeam    = "team"
	AccountLevelK12     = "k12"
)

const AccountShareModePrivate = "private"
const AccountShareModePublic = "public"
const AccountShareStatusPending = "pending"
const AccountShareStatusApproved = "approved"
const AccountShareStatusSuspended = "suspended"
const OAuthAccountDefaultConcurrency = 3
const OpenAIPlusDefaultConcurrency = 3
const OwnedPersonalDefaultLoadFactor = 10
const AccountMaxLoadFactor = 10000
const CodexQuotaDefaultLimitPercent = 100.0
const CodexQuotaMinLimitPercent = 1.0
const CodexQuotaMaxLimitPercent = 100.0
const AnthropicQuotaDefaultLimitPercent = CodexQuotaDefaultLimitPercent
const AnthropicQuotaMinLimitPercent = CodexQuotaMinLimitPercent
const AnthropicQuotaMaxLimitPercent = CodexQuotaMaxLimitPercent
const CodexQuotaWindow5h = "5h"
const CodexQuotaWindow7d = "7d"
const AnthropicQuotaWindow5h = CodexQuotaWindow5h
const AnthropicQuotaWindow7d = CodexQuotaWindow7d
const AccountListStatusRateLimited = "rate_limited"
const AccountListStatusTempUnschedulable = "temp_unschedulable"
const AccountListStatusUnschedulable = "unschedulable"
const AccountListStatusCodexQuotaProtected = "codex_quota_protected"

func NormalizeAccountLevel(level string) string {
	normalized := NormalizeAccountLevelKey(level)
	if normalized == "" {
		return AccountLevelUnknown
	}
	return normalized
}

func NormalizeAccountLevelKey(level string) string {
	normalized := strings.ToLower(strings.TrimSpace(level))
	normalized = strings.NewReplacer(" ", "-", "_", "-").Replace(normalized)
	var b strings.Builder
	lastDash := false
	for _, r := range normalized {
		switch {
		case r >= 'a' && r <= 'z':
			_, _ = b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			_, _ = b.WriteRune(r)
			lastDash = false
		case r == '-':
			if b.Len() > 0 && !lastDash {
				_, _ = b.WriteRune(r)
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func IsConcreteAccountLevel(level string) bool {
	return NormalizeAccountLevel(level) != AccountLevelUnknown
}

func IsUserSelectableOpenAIAccountLevel(level string) bool {
	return IsUserSelectableOpenAIAccountLevelWithConfigs(level, DefaultOpenAIAccountLevelConfigs())
}

func RequiresUserOpenAIProxyLogin(level string) bool {
	return RequiresUserOpenAIProxyLoginWithConfigs(level, DefaultOpenAIAccountLevelConfigs())
}

func RequiresUserAccountOAuthProxy(platform, accountLevel string) bool {
	return RequiresUserAccountOAuthProxyWithConfigs(platform, accountLevel, DefaultOpenAIAccountLevelConfigs())
}

func RequiresUserAccountOAuthProxyWithConfigs(platform, accountLevel string, configs []OpenAIAccountLevelConfig) bool {
	switch platform {
	case PlatformAnthropic, PlatformGemini, PlatformAntigravity, PlatformGrok:
		return true
	case PlatformOpenAI:
		return RequiresUserOpenAIProxyLoginWithConfigs(accountLevel, configs)
	default:
		return false
	}
}

func IsUserSelectableOpenAIAccountLevelWithConfigs(level string, configs []OpenAIAccountLevelConfig) bool {
	normalized := NormalizeAccountLevel(level)
	for _, cfg := range NormalizeOpenAIAccountLevelConfigs(configs) {
		if cfg.Enabled && cfg.Key == normalized {
			return true
		}
	}
	return false
}

func RequiresUserOpenAIProxyLoginWithConfigs(level string, configs []OpenAIAccountLevelConfig) bool {
	normalized := NormalizeAccountLevel(level)
	for _, cfg := range NormalizeOpenAIAccountLevelConfigs(configs) {
		if cfg.Enabled && cfg.Key == normalized {
			return cfg.RequiresProxyLogin
		}
	}
	return false
}

func IsOpenAIPlusAccount(platform, accountLevel string) bool {
	return platform == PlatformOpenAI && NormalizeAccountLevel(accountLevel) == AccountLevelPlus
}

func NormalizeOpenAIAccountLevel(platform, accountLevel string, credentials, extra map[string]any) string {
	return NormalizeOpenAIAccountLevelWithConfigs(platform, accountLevel, credentials, extra, DefaultOpenAIAccountLevelConfigs())
}

func NormalizeOpenAIAccountLevelWithConfigs(platform, accountLevel string, credentials, extra map[string]any, configs []OpenAIAccountLevelConfig) string {
	level := NormalizeAccountLevel(accountLevel)
	if platform != PlatformOpenAI {
		return level
	}
	if OpenAIAccountLevelConfigByKeyIncludingDisabled(configs, level) != nil {
		return level
	}
	if inferred := InferOpenAIAccountLevelWithConfigs(credentials, extra, configs); OpenAIAccountLevelConfigByKeyIncludingDisabled(configs, inferred) != nil {
		return inferred
	}
	return level
}

func InferOpenAIAccountLevel(credentials, extra map[string]any) string {
	return InferOpenAIAccountLevelWithConfigs(credentials, extra, DefaultOpenAIAccountLevelConfigs())
}

func InferOpenAIAccountLevelWithConfigs(credentials, extra map[string]any, configs []OpenAIAccountLevelConfig) string {
	for _, values := range []map[string]any{credentials, extra} {
		for _, key := range []string{"plan_type", "chatgpt_plan_type", "subscription_plan"} {
			raw, ok := values[key].(string)
			if !ok {
				continue
			}
			if inferred := NormalizeOpenAIPlanAccountLevelWithConfigs(raw, configs); inferred != AccountLevelUnknown {
				return inferred
			}
		}
	}
	return AccountLevelUnknown
}

func OpenAIAccountPlanType(credentials, extra map[string]any) string {
	for _, values := range []map[string]any{credentials, extra} {
		for _, key := range []string{"plan_type", "chatgpt_plan_type", "subscription_plan"} {
			raw, ok := values[key].(string)
			if !ok {
				continue
			}
			if trimmed := strings.TrimSpace(raw); trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func NormalizeOpenAIPlanAccountLevel(planType string) string {
	return NormalizeOpenAIPlanAccountLevelWithConfigs(planType, DefaultOpenAIAccountLevelConfigs())
}

func NormalizeOpenAIPlanAccountLevelWithConfigs(planType string, configs []OpenAIAccountLevelConfig) string {
	token := normalizeOpenAIPlanAliasToken(planType)
	if token == "" {
		return AccountLevelUnknown
	}
	for _, cfg := range NormalizeOpenAIAccountLevelConfigs(configs) {
		if !cfg.Enabled {
			continue
		}
		for _, alias := range openAIAccountLevelAliasTokens(cfg) {
			if matchOpenAIPlanAliasToken(token, alias) {
				return cfg.Key
			}
		}
	}
	return AccountLevelUnknown
}

func NormalizeOpenAISharedPoolAccountLevel(level string) string {
	switch NormalizeAccountLevel(level) {
	case AccountLevelUnknown:
		return AccountLevelFree
	default:
		return NormalizeAccountLevel(level)
	}
}

func NormalizeOpenAISharedPoolRequiredLevel(level string) string {
	return NormalizeRequiredAccountLevel(level)
}

func OpenAISharedPoolLevelRank(level string) int {
	return OpenAISharedPoolLevelRankWithConfigs(level, DefaultOpenAIAccountLevelConfigs())
}

func OpenAISharedPoolLevelRankWithConfigs(level string, configs []OpenAIAccountLevelConfig) int {
	normalized := NormalizeOpenAISharedPoolAccountLevel(level)
	for index, cfg := range NormalizeOpenAIAccountLevelConfigs(configs) {
		if cfg.Enabled && cfg.Key == normalized {
			return index + 1
		}
	}
	return 0
}

func CanOpenAIAccountJoinSharedPool(accountLevel, requiredLevel string) bool {
	return CanOpenAIAccountJoinSharedPoolWithConfigs(accountLevel, requiredLevel, DefaultOpenAIAccountLevelConfigs())
}

func CanOpenAIAccountJoinSharedPoolWithConfigs(accountLevel, requiredLevel string, configs []OpenAIAccountLevelConfig) bool {
	required := NormalizeOpenAISharedPoolRequiredLevel(requiredLevel)
	if required == "" {
		return true
	}
	account := NormalizeOpenAISharedPoolAccountLevel(accountLevel)
	normalizedConfigs := NormalizeOpenAIAccountLevelConfigs(configs)
	accountRank := OpenAISharedPoolLevelRankWithConfigs(account, normalizedConfigs)
	requiredRank := OpenAISharedPoolLevelRankWithConfigs(required, normalizedConfigs)
	if accountRank > 0 || requiredRank > 0 {
		return accountRank > 0 && requiredRank > 0 && account == required
	}
	return account == required
}

func OpenAISharedPoolAllowedAccountLevels(requiredLevel string) []string {
	return OpenAISharedPoolAllowedAccountLevelsWithConfigs(requiredLevel, DefaultOpenAIAccountLevelConfigs())
}

func OpenAISharedPoolAllowedAccountLevelsWithConfigs(requiredLevel string, configs []OpenAIAccountLevelConfig) []string {
	required := NormalizeOpenAISharedPoolRequiredLevel(requiredLevel)
	if required == "" {
		return nil
	}
	normalizedConfigs := NormalizeOpenAIAccountLevelConfigs(configs)
	requiredRank := OpenAISharedPoolLevelRankWithConfigs(required, normalizedConfigs)
	if requiredRank == 0 {
		return []string{required}
	}
	levels := make([]string, 0, 6)
	if required == AccountLevelFree {
		levels = append(levels, AccountLevelUnknown)
	}
	for _, cfg := range normalizedConfigs {
		if cfg.Enabled && CanOpenAIAccountJoinSharedPoolWithConfigs(cfg.Key, required, normalizedConfigs) {
			levels = append(levels, cfg.Key)
		}
	}
	return levels
}

func ValidateConfiguredOpenAIAccountLevel(platform, level string, configs []OpenAIAccountLevelConfig) error {
	if platform != PlatformOpenAI {
		return nil
	}
	normalized := NormalizeAccountLevel(level)
	if normalized == AccountLevelUnknown {
		return nil
	}
	if OpenAIAccountLevelConfigByKeyIncludingDisabled(configs, normalized) == nil {
		return fmt.Errorf("invalid OpenAI account level: %s", normalized)
	}
	return nil
}

func DefaultOpenAIAccountLevelConfigs() []OpenAIAccountLevelConfig {
	return []OpenAIAccountLevelConfig{
		{Key: AccountLevelFree, Label: "Free", Aliases: []string{"free", "chatgptfree"}, SortOrder: 10, Enabled: true},
		{Key: AccountLevelPlus, Label: "Plus", Aliases: []string{"plus", "plus*", "chatgptplus"}, SortOrder: 20, Enabled: true},
		{Key: AccountLevelPro, Label: "Pro", Aliases: []string{"pro", "pro*", "chatgptpro", "chatgptpro*"}, SortOrder: 30, Enabled: true, RequiresProxyLogin: true},
		{Key: AccountLevelTeam, Label: "Team", Aliases: []string{"team", "team*", "chatgptteam"}, SortOrder: 40, Enabled: true},
		{Key: AccountLevelK12, Label: "K12", Aliases: []string{"k12", "chatgptk12", "chatgpt-k12"}, SortOrder: 50, Enabled: true},
	}
}

func NormalizeOpenAIAccountLevelConfigs(configs []OpenAIAccountLevelConfig) []OpenAIAccountLevelConfig {
	if len(configs) == 0 {
		configs = DefaultOpenAIAccountLevelConfigs()
	}
	out := make([]OpenAIAccountLevelConfig, 0, len(configs))
	seenKeys := make(map[string]struct{}, len(configs))
	seenAliases := make(map[string]string)
	for _, cfg := range configs {
		key := NormalizeAccountLevelKey(cfg.Key)
		if key == "" || key == AccountLevelUnknown {
			continue
		}
		if _, ok := seenKeys[key]; ok {
			continue
		}
		label := strings.TrimSpace(cfg.Label)
		if label == "" {
			label = key
		}
		aliases := make([]string, 0, len(cfg.Aliases)+1)
		for _, alias := range append([]string{key}, cfg.Aliases...) {
			normalizedAlias := normalizeOpenAIPlanAliasPattern(alias)
			if normalizedAlias == "" {
				continue
			}
			if owner, ok := seenAliases[normalizedAlias]; ok && owner != key {
				continue
			}
			seenAliases[normalizedAlias] = key
			if !containsString(aliases, normalizedAlias) {
				aliases = append(aliases, normalizedAlias)
			}
		}
		if len(aliases) == 0 {
			aliases = []string{key}
		}
		enabled := cfg.Enabled
		out = append(out, OpenAIAccountLevelConfig{
			Key:                key,
			Label:              label,
			Aliases:            aliases,
			SortOrder:          cfg.SortOrder,
			Enabled:            enabled,
			RequiresProxyLogin: cfg.RequiresProxyLogin,
		})
		seenKeys[key] = struct{}{}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].SortOrder == out[j].SortOrder {
			return out[i].Key < out[j].Key
		}
		return out[i].SortOrder < out[j].SortOrder
	})
	return out
}

func OpenAIAccountLevelConfigSelectable(configs []OpenAIAccountLevelConfig) []OpenAIAccountLevelConfig {
	normalized := NormalizeOpenAIAccountLevelConfigs(configs)
	out := make([]OpenAIAccountLevelConfig, 0, len(normalized))
	for _, cfg := range normalized {
		if cfg.Enabled {
			out = append(out, cfg)
		}
	}
	return out
}

func ValidateOpenAIAccountLevelConfigs(configs []OpenAIAccountLevelConfig) ([]OpenAIAccountLevelConfig, error) {
	if len(configs) == 0 {
		return nil, fmt.Errorf("openai_account_levels cannot be empty")
	}
	seenAliases := make(map[string]string)
	for _, cfg := range configs {
		key := NormalizeAccountLevelKey(cfg.Key)
		if key == "" || key == AccountLevelUnknown {
			continue
		}
		for _, alias := range append([]string{key}, cfg.Aliases...) {
			normalizedAlias := normalizeOpenAIPlanAliasPattern(alias)
			if normalizedAlias == "" {
				continue
			}
			if owner, ok := seenAliases[normalizedAlias]; ok && owner != key {
				return nil, fmt.Errorf("openai_account_levels alias %q is used by both %q and %q", normalizedAlias, owner, key)
			}
			seenAliases[normalizedAlias] = key
		}
	}
	normalized := NormalizeOpenAIAccountLevelConfigs(configs)
	if len(normalized) == 0 {
		return nil, fmt.Errorf("openai_account_levels must contain at least one valid level")
	}
	enabledCount := 0
	for _, cfg := range normalized {
		if cfg.Enabled {
			enabledCount++
		}
	}
	if enabledCount == 0 {
		return nil, fmt.Errorf("openai_account_levels must contain at least one enabled level")
	}
	return normalized, nil
}

func OpenAIAccountLevelConfigByKey(configs []OpenAIAccountLevelConfig, key string) *OpenAIAccountLevelConfig {
	return OpenAIAccountLevelConfigByKeyWithEnabled(configs, key, true)
}

func OpenAIAccountLevelConfigByKeyIncludingDisabled(configs []OpenAIAccountLevelConfig, key string) *OpenAIAccountLevelConfig {
	return OpenAIAccountLevelConfigByKeyWithEnabled(configs, key, false)
}

func OpenAIAccountLevelConfigByKeyWithEnabled(configs []OpenAIAccountLevelConfig, key string, requireEnabled bool) *OpenAIAccountLevelConfig {
	normalized := NormalizeAccountLevel(key)
	for _, cfg := range NormalizeOpenAIAccountLevelConfigs(configs) {
		if cfg.Key == normalized && (!requireEnabled || cfg.Enabled) {
			candidate := cfg
			return &candidate
		}
	}
	return nil
}

func openAIAccountLevelAliasTokens(cfg OpenAIAccountLevelConfig) []string {
	return NormalizeOpenAIAccountLevelConfigs([]OpenAIAccountLevelConfig{cfg})[0].Aliases
}

func normalizeOpenAIPlanAliasToken(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.NewReplacer(" ", "", "-", "", "_", "").Replace(normalized)
	return normalized
}

func normalizeOpenAIPlanAliasPattern(value string) string {
	trimmed := strings.TrimSpace(value)
	hasWildcard := strings.HasSuffix(trimmed, "*")
	if hasWildcard {
		trimmed = strings.TrimSuffix(trimmed, "*")
	}
	normalized := normalizeOpenAIPlanAliasToken(trimmed)
	if normalized == "" {
		return ""
	}
	if hasWildcard {
		return normalized + "*"
	}
	return normalized
}

func matchOpenAIPlanAliasToken(token, pattern string) bool {
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return prefix != "" && strings.HasPrefix(token, prefix)
	}
	return token == pattern
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func DefaultOAuthAccountConcurrencyForPlatform(platform string) int {
	if platform == PlatformOpenAI {
		return OpenAIPlusDefaultConcurrency
	}
	return OAuthAccountDefaultConcurrency
}

func NormalizeOpenAIPlusConcurrency(platform, accountLevel string, concurrency int) (int, error) {
	if !IsOpenAIPlusAccount(platform, accountLevel) {
		return concurrency, nil
	}
	if concurrency <= 0 {
		return OpenAIPlusDefaultConcurrency, nil
	}
	return concurrency, nil
}

func ValidateOpenAIPlusConcurrency(platform, accountLevel string, concurrency int) error {
	if !IsOpenAIPlusAccount(platform, accountLevel) {
		return nil
	}
	if concurrency <= 0 {
		return fmt.Errorf("openai plus account concurrency must be > 0")
	}
	return nil
}

func ValidateAccountLoadFactor(loadFactor *int) error {
	if loadFactor == nil || *loadFactor <= 0 {
		return nil
	}
	if *loadFactor > AccountMaxLoadFactor {
		return fmt.Errorf("load_factor must be <= %d", AccountMaxLoadFactor)
	}
	return nil
}

func NormalizeAccountShareMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case AccountShareModePublic:
		return AccountShareModePublic
	default:
		return AccountShareModePrivate
	}
}

func NormalizeAccountShareStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case AccountShareStatusPending:
		return AccountShareStatusPending
	case AccountShareStatusSuspended:
		return AccountShareStatusSuspended
	default:
		return AccountShareStatusApproved
	}
}

func (a *Account) IsPublicShareApproved() bool {
	return a != nil &&
		a.OwnerUserID != nil &&
		NormalizeAccountShareMode(a.ShareMode) == AccountShareModePublic &&
		NormalizeAccountShareStatus(a.ShareStatus) == AccountShareStatusApproved
}

func (a *Account) IsVisibleToConsumer(userID int64) bool {
	if a == nil {
		return false
	}
	if a.OwnerUserID == nil {
		return true
	}
	if userID > 0 && *a.OwnerUserID == userID {
		return true
	}
	return a.IsPublicShareApproved()
}

func (a *Account) IsSchedulableAt(now time.Time) bool {
	return a.isSchedulableAt(now, true)
}

func (a *Account) IsSchedulableWithoutCodexQuotaProtection() bool {
	return a.isSchedulableAt(time.Now(), false)
}

func (a *Account) isSchedulableAt(now time.Time, includeCodexQuotaProtection bool) bool {
	if !a.IsActive() || !a.Schedulable {
		return false
	}
	if a.AutoPauseOnExpired && a.ExpiresAt != nil && !now.Before(*a.ExpiresAt) {
		return false
	}
	if a.OverloadUntil != nil && now.Before(*a.OverloadUntil) {
		return false
	}
	if a.RateLimitResetAt != nil && now.Before(*a.RateLimitResetAt) {
		return false
	}
	if includeCodexQuotaProtection && (a.IsCodexQuotaProtectionActiveAt(now) || a.IsAnthropicQuotaProtectionActiveAt(now)) {
		return false
	}
	if a.TempUnschedulableUntil != nil && now.Before(*a.TempUnschedulableUntil) {
		return false
	}
	if paused, _ := shouldAutoPauseGrokAccountByQuota(a); paused {
		return false
	}
	if a.IsAPIKeyOrBedrock() && a.IsQuotaExceededAt(now) {
		return false
	}
	return true
}

func (a *Account) IsRateLimitedAt(now time.Time) bool {
	if a.RateLimitResetAt == nil {
		return false
	}
	return now.Before(*a.RateLimitResetAt)
}

func (a *Account) IsOverloadedAt(now time.Time) bool {
	if a.OverloadUntil == nil {
		return false
	}
	return now.Before(*a.OverloadUntil)
}

func (a *Account) GetClaudeOrgUUID() string {
	if v := strings.TrimSpace(a.GetExtraString("org_uuid")); v != "" {
		return v
	}
	return strings.TrimSpace(a.GetCredential("org_uuid"))
}

func (a *Account) GetClaudeAccountUUID() string {
	if v := strings.TrimSpace(a.GetExtraString("account_uuid")); v != "" {
		return v
	}
	return strings.TrimSpace(a.GetCredential("account_uuid"))
}

// IsOpenAIAgentIdentityCredentials reports whether credentials select the
// Codex Agent Identity authentication mode. Platform and account-type checks
// remain the caller's responsibility so this helper can also be used while a
// create/import request is still being validated.
func IsOpenAIAgentIdentityCredentials(credentials map[string]any) bool {
	if len(credentials) == 0 {
		return false
	}
	authMode, ok := credentials["auth_mode"].(string)
	return ok && strings.EqualFold(strings.TrimSpace(authMode), OpenAIAuthModeAgentIdentity)
}

type accountCredentialFieldExistenceChecker interface {
	ExistsByCredentialField(ctx context.Context, key, value string) (bool, error)
}

func credentialFieldExists(ctx context.Context, repository any, key, value string) (bool, error) {
	checker, ok := repository.(accountCredentialFieldExistenceChecker)
	if !ok {
		return false, errors.New("account repository does not support credential field lookup")
	}
	return checker.ExistsByCredentialField(ctx, key, value)
}

// OpenAIAgentIdentityRuntimeIDExists performs a narrow JSONB lookup.
// Repositories that cannot provide the lookup are rejected instead of falling
// back to loading and scanning every account.
func (s *AccountService) OpenAIAgentIdentityRuntimeIDExists(ctx context.Context, runtimeID string) (bool, error) {
	runtimeID = strings.TrimSpace(runtimeID)
	if runtimeID == "" {
		return false, errors.New("agent identity runtime id is required")
	}
	if s == nil || s.accountRepo == nil {
		return false, errors.New("account repository is required for Agent Identity duplicate detection")
	}
	return credentialFieldExists(ctx, s.accountRepo, "agent_runtime_id", runtimeID)
}

func (a *Account) GetCodex5hLimitPercent() float64 {
	return a.getCodexQuotaLimitPercent("codex_5h_limit_percent")
}

func (a *Account) GetCodex7dLimitPercent() float64 {
	return a.getCodexQuotaLimitPercent("codex_7d_limit_percent")
}

func (a *Account) GetAnthropic5hLimitPercent() float64 {
	return a.getAnthropicQuotaLimitPercent("anthropic_5h_limit_percent")
}

func (a *Account) GetAnthropic7dLimitPercent() float64 {
	return a.getAnthropicQuotaLimitPercent("anthropic_7d_limit_percent")
}

func (a *Account) getCodexQuotaLimitPercent(key string) float64 {
	if a == nil || a.Extra == nil {
		return CodexQuotaDefaultLimitPercent
	}
	raw, ok := a.Extra[key]
	if !ok || raw == nil {
		return CodexQuotaDefaultLimitPercent
	}
	limit := parseExtraFloat64(raw)
	if limit < CodexQuotaMinLimitPercent || limit > CodexQuotaMaxLimitPercent {
		return CodexQuotaDefaultLimitPercent
	}
	return limit
}

func (a *Account) getAnthropicQuotaLimitPercent(key string) float64 {
	if a == nil || a.Extra == nil {
		return AnthropicQuotaDefaultLimitPercent
	}
	raw, ok := a.Extra[key]
	if !ok || raw == nil {
		return AnthropicQuotaDefaultLimitPercent
	}
	limit := parseExtraFloat64(raw)
	if limit < AnthropicQuotaMinLimitPercent || limit > AnthropicQuotaMaxLimitPercent {
		return AnthropicQuotaDefaultLimitPercent
	}
	return limit
}

func (a *Account) IsCodexQuotaProtectionActiveAt(now time.Time) bool {
	return a.CodexQuotaProtectionReasonAt(now) != ""
}

func (a *Account) IsAnthropicQuotaProtectionActiveAt(now time.Time) bool {
	return a.AnthropicQuotaProtectionReasonAt(now) != ""
}

func (a *Account) CodexQuotaProtectionReasonAt(now time.Time) string {
	reason, _ := a.codexQuotaProtectionWindowAt(now)
	return reason
}

func (a *Account) AnthropicQuotaProtectionReasonAt(now time.Time) string {
	reason, _ := a.anthropicQuotaProtectionWindowAt(now)
	return reason
}

func (a *Account) CodexQuotaProtectionResetAt(now time.Time) *time.Time {
	_, resetAt := a.codexQuotaProtectionWindowAt(now)
	return resetAt
}

func (a *Account) AnthropicQuotaProtectionResetAt(now time.Time) *time.Time {
	_, resetAt := a.anthropicQuotaProtectionWindowAt(now)
	return resetAt
}

func (a *Account) CodexUsageProgress(window string, now time.Time) *UsageProgress {
	if a == nil || !a.IsOpenAIOAuth() {
		return nil
	}
	return buildCodexUsageProgressFromExtra(a.Extra, window, now)
}

func (a *Account) AnthropicUsageProgress(window string, now time.Time) *UsageProgress {
	if a == nil || !a.IsAnthropicOAuthOrSetupToken() {
		return nil
	}
	switch window {
	case AnthropicQuotaWindow5h:
		resetAt, _ := a.anthropic5hResetAt()
		return buildAnthropicUsageProgressFromExtra(a.Extra, "session_window_utilization", resetAt, now)
	case AnthropicQuotaWindow7d:
		resetAt, _ := anthropicQuotaResetAtFromExtra(a.Extra, "anthropic_7d_reset_at", "passive_usage_7d_reset")
		return buildAnthropicUsageProgressFromExtra(a.Extra, "passive_usage_7d_utilization", resetAt, now)
	default:
		return nil
	}
}

func (a *Account) CodexUsageUpdatedAt() *time.Time {
	if a == nil {
		return nil
	}
	updatedAt := a.getExtraTime("codex_usage_updated_at")
	if updatedAt.IsZero() {
		return nil
	}
	return &updatedAt
}

func (a *Account) AnthropicUsageUpdatedAt() *time.Time {
	if a == nil {
		return nil
	}
	updatedAt := a.getExtraTime("anthropic_usage_updated_at")
	if updatedAt.IsZero() {
		updatedAt = a.getExtraTime("passive_usage_sampled_at")
	}
	if updatedAt.IsZero() {
		return nil
	}
	return &updatedAt
}

func (a *Account) codexQuotaProtectionWindowAt(now time.Time) (string, *time.Time) {
	if a == nil || !a.IsOpenAIOAuth() || a.Extra == nil {
		return "", nil
	}
	reason, resetAt := "", time.Time{}
	if windowResetAt, ok := codexQuotaProtectedWindowResetAt(a.Extra, "codex_5h_used_percent", "codex_5h_reset_at", a.GetCodex5hLimitPercent(), now); ok {
		reason = CodexQuotaWindow5h
		resetAt = windowResetAt
	}
	if windowResetAt, ok := codexQuotaProtectedWindowResetAt(a.Extra, "codex_7d_used_percent", "codex_7d_reset_at", a.GetCodex7dLimitPercent(), now); ok {
		if reason == "" || windowResetAt.After(resetAt) {
			reason = CodexQuotaWindow7d
			resetAt = windowResetAt
		}
	}
	if reason == "" {
		return "", nil
	}
	return reason, &resetAt
}

func (a *Account) anthropicQuotaProtectionWindowAt(now time.Time) (string, *time.Time) {
	if a == nil || !a.IsAnthropicOAuthOrSetupToken() || a.Extra == nil {
		return "", nil
	}
	reason, resetAt := "", time.Time{}
	if windowResetAt, ok := a.anthropicQuotaProtectedWindowResetAt("session_window_utilization", "anthropic_5h_reset_at", a.GetAnthropic5hLimitPercent(), now); ok {
		reason = AnthropicQuotaWindow5h
		resetAt = windowResetAt
	}
	if windowResetAt, ok := a.anthropicQuotaProtectedWindowResetAt("passive_usage_7d_utilization", "anthropic_7d_reset_at", a.GetAnthropic7dLimitPercent(), now, "passive_usage_7d_reset"); ok {
		if reason == "" || windowResetAt.After(resetAt) {
			reason = AnthropicQuotaWindow7d
			resetAt = windowResetAt
		}
	}
	if reason == "" {
		return "", nil
	}
	return reason, &resetAt
}

func (a *Account) anthropicQuotaProtectedWindowResetAt(utilizationKey, resetKey string, limitPercent float64, now time.Time, resetFallbackKeys ...string) (time.Time, bool) {
	if limitPercent < AnthropicQuotaMinLimitPercent || limitPercent > AnthropicQuotaMaxLimitPercent {
		limitPercent = AnthropicQuotaDefaultLimitPercent
	}
	usedPercent, ok := anthropicUtilizationPercentFromExtra(a.Extra, utilizationKey)
	if !ok && utilizationKey == "session_window_utilization" {
		switch a.SessionWindowStatus {
		case "rejected":
			usedPercent = 100
			ok = true
		case "allowed_warning":
			usedPercent = 80
			ok = true
		}
	}
	if !ok || usedPercent < limitPercent {
		return time.Time{}, false
	}
	var resetAt time.Time
	var hasReset bool
	if utilizationKey == "session_window_utilization" {
		resetAt, hasReset = a.anthropic5hResetAt()
	}
	if !hasReset {
		keys := append([]string{resetKey}, resetFallbackKeys...)
		resetAt, hasReset = anthropicQuotaResetAtFromExtra(a.Extra, keys...)
	}
	if !hasReset || !now.Before(resetAt) {
		return time.Time{}, false
	}
	return resetAt, true
}

func (a *Account) anthropic5hResetAt() (time.Time, bool) {
	if a != nil && a.SessionWindowEnd != nil && !a.SessionWindowEnd.IsZero() {
		return *a.SessionWindowEnd, true
	}
	if a == nil {
		return time.Time{}, false
	}
	return anthropicQuotaResetAtFromExtra(a.Extra, "anthropic_5h_reset_at", "session_window_reset_at")
}

//nolint:unused // Retained for staged Codex quota compatibility handling.
func isCodexQuotaWindowProtected(extra map[string]any, usedPercentKey, resetAtKey string, limitPercent float64, now time.Time) bool {
	_, ok := codexQuotaProtectedWindowResetAt(extra, usedPercentKey, resetAtKey, limitPercent, now)
	return ok
}

func codexQuotaProtectedWindowResetAt(extra map[string]any, usedPercentKey, resetAtKey string, limitPercent float64, now time.Time) (time.Time, bool) {
	if limitPercent < CodexQuotaMinLimitPercent || limitPercent > CodexQuotaMaxLimitPercent {
		limitPercent = CodexQuotaDefaultLimitPercent
	}
	usedRaw, ok := extra[usedPercentKey]
	if !ok || parseExtraFloat64(usedRaw) < limitPercent {
		return time.Time{}, false
	}
	resetAt, ok := codexQuotaResetAtFromExtra(extra, resetAtKey)
	if !ok {
		return time.Time{}, false
	}
	if !now.Before(resetAt) {
		return time.Time{}, false
	}
	return resetAt, true
}

func codexQuotaResetAtFromExtra(extra map[string]any, key string) (time.Time, bool) {
	if extra == nil {
		return time.Time{}, false
	}
	raw, ok := extra[key]
	if !ok || raw == nil {
		return time.Time{}, false
	}
	value := strings.TrimSpace(fmt.Sprint(raw))
	if value == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return t, true
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, true
	}
	return time.Time{}, false
}

func buildAnthropicUsageProgressFromExtra(extra map[string]any, utilizationKey string, resetAt time.Time, now time.Time) *UsageProgress {
	utilization, hasUtilization := anthropicUtilizationPercentFromExtra(extra, utilizationKey)
	if !hasUtilization && resetAt.IsZero() {
		return nil
	}
	progress := &UsageProgress{Utilization: utilization}
	if !resetAt.IsZero() {
		progress.ResetsAt = &resetAt
		progress.RemainingSeconds = int(time.Until(resetAt).Seconds())
		if progress.RemainingSeconds < 0 {
			progress.RemainingSeconds = 0
		}
		if !now.Before(resetAt) {
			progress.Utilization = 0
		}
	}
	return progress
}

func anthropicUtilizationPercentFromExtra(extra map[string]any, key string) (float64, bool) {
	if extra == nil {
		return 0, false
	}
	raw, ok := extra[key]
	if !ok || raw == nil {
		return 0, false
	}
	value := parseExtraFloat64(raw)
	if value < 0 {
		value = 0
	}
	if value <= 1.5 {
		value *= 100
	}
	return value, true
}

func anthropicQuotaResetAtFromExtra(extra map[string]any, keys ...string) (time.Time, bool) {
	if extra == nil {
		return time.Time{}, false
	}
	for _, key := range keys {
		raw, ok := extra[key]
		if !ok || raw == nil {
			continue
		}
		if unix := int64(parseExtraFloat64(raw)); unix > 0 {
			return time.Unix(unix, 0), true
		}
		value := strings.TrimSpace(fmt.Sprint(raw))
		if value == "" {
			continue
		}
		if t, err := parseTime(value); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func (a *Account) isFixedDailyPeriodExpiredAt(periodStart time.Time, now time.Time) bool {
	if periodStart.IsZero() {
		return true
	}
	tz, err := time.LoadLocation(a.GetQuotaResetTimezone())
	if err != nil {
		tz = time.UTC
	}
	lastReset := lastFixedDailyReset(a.GetQuotaDailyResetHour(), tz, now)
	return periodStart.Before(lastReset)
}

func (a *Account) isFixedWeeklyPeriodExpiredAt(periodStart time.Time, now time.Time) bool {
	if periodStart.IsZero() {
		return true
	}
	tz, err := time.LoadLocation(a.GetQuotaResetTimezone())
	if err != nil {
		tz = time.UTC
	}
	lastReset := lastFixedWeeklyReset(a.GetQuotaWeeklyResetDay(), a.GetQuotaWeeklyResetHour(), tz, now)
	return periodStart.Before(lastReset)
}

func isPeriodExpiredAt(periodStart time.Time, dur time.Duration, now time.Time) bool {
	if periodStart.IsZero() {
		return true
	}
	return !now.Before(periodStart.Add(dur))
}

func (a *Account) IsDailyQuotaPeriodExpiredAt(now time.Time) bool {
	start := a.getExtraTime("quota_daily_start")
	if a.GetQuotaDailyResetMode() == "fixed" {
		return a.isFixedDailyPeriodExpiredAt(start, now)
	}
	return isPeriodExpiredAt(start, 24*time.Hour, now)
}

func (a *Account) IsWeeklyQuotaPeriodExpiredAt(now time.Time) bool {
	start := a.getExtraTime("quota_weekly_start")
	if a.GetQuotaWeeklyResetMode() == "fixed" {
		return a.isFixedWeeklyPeriodExpiredAt(start, now)
	}
	return isPeriodExpiredAt(start, 7*24*time.Hour, now)
}

func (a *Account) IsQuotaExceededAt(now time.Time) bool {

	if limit := a.GetQuotaLimit(); limit > 0 && a.GetQuotaUsed() >= limit {
		return true
	}

	if limit := a.GetQuotaDailyLimit(); limit > 0 {
		start := a.getExtraTime("quota_daily_start")
		var expired bool
		if a.GetQuotaDailyResetMode() == "fixed" {
			expired = a.isFixedDailyPeriodExpiredAt(start, now)
		} else {
			expired = isPeriodExpiredAt(start, 24*time.Hour, now)
		}
		if !expired && a.GetQuotaDailyUsed() >= limit {
			return true
		}
	}

	if limit := a.GetQuotaWeeklyLimit(); limit > 0 {
		start := a.getExtraTime("quota_weekly_start")
		var expired bool
		if a.GetQuotaWeeklyResetMode() == "fixed" {
			expired = a.isFixedWeeklyPeriodExpiredAt(start, now)
		} else {
			expired = isPeriodExpiredAt(start, 7*24*time.Hour, now)
		}
		if !expired && a.GetQuotaWeeklyUsed() >= limit {
			return true
		}
	}
	return false
}
