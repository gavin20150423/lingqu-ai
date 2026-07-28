package service

import (
	"context"
	"fmt"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const ownedPersonalDefaultPriority = 1

const (
	GroupScopePublic      = "public"
	GroupScopeUserPrivate = "user_private"
)

func NormalizeGroupScope(scope string) string {
	if strings.EqualFold(strings.TrimSpace(scope), GroupScopeUserPrivate) {
		return GroupScopeUserPrivate
	}
	return GroupScopePublic
}

type AccountShareAPIKeyBindingChecker interface {
	HasActiveOrQueuedMembershipForAPIKey(ctx context.Context, consumerUserID, apiKeyID int64) (bool, error)
}

var (
	ErrCodexQuotaLimitPercentInvalid     = infraerrors.BadRequest("CODEX_QUOTA_LIMIT_PERCENT_INVALID", "Codex quota limit percent must be between 1 and 100")
	ErrAPIKeyInactive                    = infraerrors.Unauthorized("API_KEY_INACTIVE", "api key is not active")
	ErrOwnedAccountTypeNotAllowed        = infraerrors.BadRequest("OWNED_ACCOUNT_TYPE_NOT_ALLOWED", "user accounts only support official OAuth accounts")
	ErrOwnedAccountCredentialsNotAllowed = infraerrors.BadRequest("OWNED_ACCOUNT_CREDENTIALS_NOT_ALLOWED", "user accounts cannot include API keys, custom URLs, upstream endpoints, cookies or manual session credentials")
)

func ProxyAccountLimitExceededError(proxyID, current, limit, additional int64) error {
	return infraerrors.Conflict(
		"PROXY_ACCOUNT_LIMIT_EXCEEDED",
		fmt.Sprintf("proxy %d account binding limit exceeded: %d/%d accounts would be bound", proxyID, current+additional, limit),
	)
}

func validateOwnedAccountSourceForPlatform(_ string, accountType string, credentials, extra map[string]any) error {
	if strings.ToLower(strings.TrimSpace(accountType)) != AccountTypeOAuth {
		return ErrOwnedAccountTypeNotAllowed
	}
	if !hasNonEmptyStringField(credentials, "access_token") {
		return ErrOwnedAccountCredentialsInvalid
	}
	if field, blocked := findDisallowedOwnedAccountField(credentials); blocked {
		return ErrOwnedAccountCredentialsNotAllowed.WithMetadata(map[string]string{"section": "credentials", "field": field})
	}
	if field, blocked := findDisallowedOwnedAccountField(extra); blocked {
		return ErrOwnedAccountCredentialsNotAllowed.WithMetadata(map[string]string{"section": "extra", "field": field})
	}
	return nil
}

func hasNonEmptyStringField(values map[string]any, key string) bool {
	value, ok := values[key].(string)
	return ok && strings.TrimSpace(value) != ""
}

func findDisallowedOwnedAccountField(values map[string]any) (string, bool) {
	return findDisallowedCredentialContent(values, credentialSafetyOptions{
		AllowOAuthTokenValues:  true,
		AllowOAuthMetadataURLs: true,
	})
}

func NormalizeCodexQuotaLimitExtra(platform, accountType string, extra map[string]any) (map[string]any, error) {
	if len(extra) == 0 {
		return extra, nil
	}
	if platform != PlatformOpenAI || accountType != AccountTypeOAuth {
		delete(extra, "codex_5h_limit_percent")
		delete(extra, "codex_7d_limit_percent")
		return extra, nil
	}
	for _, key := range []string{"codex_5h_limit_percent", "codex_7d_limit_percent"} {
		value, ok, err := normalizeCodexQuotaLimitPercentValue(extra[key])
		if err != nil {
			return nil, err
		}
		if !ok || value == CodexQuotaDefaultLimitPercent {
			delete(extra, key)
			continue
		}
		extra[key] = value
	}
	return extra, nil
}

func normalizeCodexQuotaLimitPercentValue(raw any) (float64, bool, error) {
	if raw == nil {
		return 0, false, nil
	}
	if value, ok := raw.(string); ok && strings.TrimSpace(value) == "" {
		return 0, false, nil
	}
	value := parseExtraFloat64(raw)
	if value < CodexQuotaMinLimitPercent || value > CodexQuotaMaxLimitPercent {
		return 0, false, ErrCodexQuotaLimitPercentInvalid
	}
	return value, true, nil
}
