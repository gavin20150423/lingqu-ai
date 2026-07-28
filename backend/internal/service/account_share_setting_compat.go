package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

const (
	SettingKeyOpenAIAccountLevels              = "openai_account_levels"
	SettingKeyAccountShareCommentReviewEnabled = "account_share_comment_review_enabled"
	SettingKeyAccountShareCommentReviewURL     = "account_share_comment_review_url"
	SettingKeyAccountShareCommentReviewAPIKey  = "account_share_comment_review_api_key"
	SettingKeyAccountShareCommentReviewModel   = "account_share_comment_review_model"
)

func NormalizeRequiredAccountLevel(level string) string {
	normalized := NormalizeAccountLevel(level)
	if normalized == AccountLevelUnknown {
		return ""
	}
	return normalized
}

func (s *SettingService) GetOpenAIAccountLevelConfigs(ctx context.Context) ([]OpenAIAccountLevelConfig, error) {
	if s == nil || s.settingRepo == nil {
		return DefaultOpenAIAccountLevelConfigs(), nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyOpenAIAccountLevels)
	if errors.Is(err, ErrSettingNotFound) {
		return DefaultOpenAIAccountLevelConfigs(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("get openai account levels: %w", err)
	}
	if raw == "" {
		return DefaultOpenAIAccountLevelConfigs(), nil
	}
	var configs []OpenAIAccountLevelConfig
	if err := json.Unmarshal([]byte(raw), &configs); err != nil {
		return nil, err
	}
	return ValidateOpenAIAccountLevelConfigs(configs)
}
