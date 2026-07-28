package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	SettingKeyWithdrawalManagementEnabled     = "withdrawal_management_enabled"
	SettingKeyWithdrawalRateLimitWindowDays   = "withdrawal_rate_limit_window_days"
	SettingKeyWithdrawalRateLimitMax          = "withdrawal_rate_limit_max"
	SettingKeyWithdrawalRateLimitExemptAmount = "withdrawal_rate_limit_exempt_amount"
)

func (s *SettingService) IsWithdrawalManagementEnabled(ctx context.Context) bool {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyWithdrawalManagementEnabled)
	if err != nil {
		return true
	}
	return !isFalseSettingValue(value)
}

func (s *SettingService) GetWithdrawalRateLimitConfig(ctx context.Context) (WithdrawalRateLimitConfig, error) {
	settings, err := s.settingRepo.GetMultiple(ctx, []string{
		SettingKeyWithdrawalRateLimitWindowDays,
		SettingKeyWithdrawalRateLimitMax,
		SettingKeyWithdrawalRateLimitExemptAmount,
	})
	if err != nil {
		return WithdrawalRateLimitConfig{}, fmt.Errorf("get withdrawal rate limit settings: %w", err)
	}
	return parseWithdrawalRateLimitConfig(settings)
}

func parseWithdrawalRateLimitConfig(settings map[string]string) (WithdrawalRateLimitConfig, error) {
	config := WithdrawalRateLimitConfig{
		WindowDays:   WithdrawalRateLimitWindowDaysDefault,
		MaxRequests:  WithdrawalRateLimitMaxDefault,
		ExemptAmount: WithdrawalRateLimitExemptAmountDefault,
	}
	if raw := strings.TrimSpace(settings[SettingKeyWithdrawalRateLimitWindowDays]); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return WithdrawalRateLimitConfig{}, infraerrors.BadRequest("WITHDRAWAL_RATE_LIMIT_CONFIG_INVALID", "withdrawal rate limit window days must be an integer")
		}
		config.WindowDays = value
	}
	if raw := strings.TrimSpace(settings[SettingKeyWithdrawalRateLimitMax]); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return WithdrawalRateLimitConfig{}, infraerrors.BadRequest("WITHDRAWAL_RATE_LIMIT_CONFIG_INVALID", "withdrawal rate limit max must be an integer")
		}
		config.MaxRequests = value
	}
	if raw := strings.TrimSpace(settings[SettingKeyWithdrawalRateLimitExemptAmount]); raw != "" {
		value, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return WithdrawalRateLimitConfig{}, infraerrors.BadRequest("WITHDRAWAL_RATE_LIMIT_CONFIG_INVALID", "withdrawal rate limit exempt amount must be a number")
		}
		config.ExemptAmount = value
	}
	if err := ValidateWithdrawalRateLimitConfig(config); err != nil {
		return WithdrawalRateLimitConfig{}, err
	}
	config.ExemptAmount, _ = normalizeWithdrawalAmount(config.ExemptAmount)
	return config, nil
}
