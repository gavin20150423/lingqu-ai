package service

import (
	"context"
	"strconv"
)

func (s *GatewayService) subPilotClient() *subPilotClient {
	if s == nil {
		return nil
	}
	return newSubPilotClient(subPilotConfigFrom(s.cfg))
}

func (s *OpenAIGatewayService) subPilotClient() *subPilotClient {
	if s == nil {
		return nil
	}
	return newSubPilotClient(subPilotConfigFrom(s.cfg))
}

func (s *GatewayService) trySubPilotRecommend(ctx context.Context, groupID *int64, platform string, sessionKey string, requestedModel string, excludedIDs map[int64]struct{}, accounts []Account, useMixed bool) (*AccountSelectionResult, error) {
	if SubPilotDisabledFromContext(ctx) {
		return nil, nil
	}
	client := s.subPilotClient()
	if client == nil || groupID == nil || requestedModel == "" {
		return nil, nil
	}
	rec, err := client.recommendAccount(ctx, subPilotSelectRequest{
		RequestID:  subPilotRequestID(ctx),
		Platform:   platform,
		GroupID:    strconv.FormatInt(*groupID, 10),
		Model:      requestedModel,
		SessionKey: sessionKey,
	})
	if err != nil || rec == nil {
		return nil, err
	}
	if _, excluded := excludedIDs[rec.AccountID]; excluded {
		s.reportSubPilotRecommendationFailure(ctx, rec, nil, platform, *groupID, requestedModel, sessionKey, "excluded_account", "SubPilot recommended an excluded account")
		return nil, nil
	}
	account := s.validateSubPilotGatewayAccount(ctx, rec.AccountID, platform, requestedModel, accounts, useMixed)
	if account == nil {
		s.reportSubPilotRecommendationFailure(ctx, rec, nil, platform, *groupID, requestedModel, sessionKey, "invalid_account", "SubPilot recommended an account that failed local validation")
		return nil, nil
	}
	result, err := s.tryAcquireAccountSlot(ctx, account.ID, account.Concurrency)
	if err != nil || result == nil || !result.Acquired {
		s.reportSubPilotRecommendationFailure(ctx, rec, account, platform, *groupID, requestedModel, sessionKey, "account_slot_unavailable", errorMessageOrDefault(err, "Sub2API account concurrency slot unavailable"))
		return nil, err
	}
	if !s.checkAndRegisterSession(ctx, account, sessionKey) {
		result.ReleaseFunc()
		s.reportSubPilotRecommendationFailure(ctx, rec, account, platform, *groupID, requestedModel, sessionKey, "session_limit", "Sub2API session limit rejected the recommendation")
		return nil, nil
	}
	if sessionKey != "" && s.cache != nil {
		_ = s.cache.SetSessionAccountID(ctx, derefGroupID(groupID), sessionKey, account.ID, stickySessionTTL)
	}
	selection, err := s.newSelectionResult(ctx, account, true, result.ReleaseFunc, nil)
	if err != nil {
		return nil, err
	}
	selection.SubPilotLeaseID = rec.LeaseID
	selection.SubPilotRequestID = rec.RequestID
	return selection, nil
}

func (s *GatewayService) validateSubPilotGatewayAccount(ctx context.Context, accountID int64, platform string, requestedModel string, accounts []Account, useMixed bool) *Account {
	for i := range accounts {
		acc := &accounts[i]
		if acc.ID != accountID {
			continue
		}
		if !s.isAccountSchedulableForSelection(acc) {
			return nil
		}
		if !s.isAccountAllowedForPlatform(acc, platform, useMixed) {
			return nil
		}
		if requestedModel != "" && !s.isModelSupportedByAccountWithContext(ctx, acc, requestedModel) {
			return nil
		}
		if !s.isAccountSchedulableForModelSelection(ctx, acc, requestedModel) {
			return nil
		}
		if !s.isAccountSchedulableForQuota(acc) {
			return nil
		}
		if !s.isAccountSchedulableForWindowCost(ctx, acc, false) {
			return nil
		}
		if !s.isAccountSchedulableForRPM(ctx, acc, false) {
			return nil
		}
		return acc
	}
	return nil
}

func (s *OpenAIGatewayService) trySubPilotRecommend(ctx context.Context, groupID *int64, platform string, sessionKey string, requestedModel string, excludedIDs map[int64]struct{}, requireCompact bool, requiredCapability OpenAIEndpointCapability, accounts []Account) (*AccountSelectionResult, error) {
	if SubPilotDisabledFromContext(ctx) {
		return nil, nil
	}
	client := s.subPilotClient()
	if client == nil || groupID == nil || requestedModel == "" {
		return nil, nil
	}
	rec, err := client.recommendAccount(ctx, subPilotSelectRequest{
		RequestID:  subPilotRequestID(ctx),
		Platform:   normalizeOpenAICompatiblePlatform(platform),
		GroupID:    strconv.FormatInt(*groupID, 10),
		Model:      requestedModel,
		SessionKey: sessionKey,
	})
	if err != nil || rec == nil {
		return nil, err
	}
	if _, excluded := excludedIDs[rec.AccountID]; excluded {
		s.reportSubPilotRecommendationFailure(ctx, rec, nil, normalizeOpenAICompatiblePlatform(platform), *groupID, requestedModel, sessionKey, "excluded_account", "SubPilot recommended an excluded account")
		return nil, nil
	}
	account := s.validateSubPilotOpenAIAccount(ctx, rec.AccountID, platform, requestedModel, requireCompact, requiredCapability, accounts)
	if account == nil {
		s.reportSubPilotRecommendationFailure(ctx, rec, nil, normalizeOpenAICompatiblePlatform(platform), *groupID, requestedModel, sessionKey, "invalid_account", "SubPilot recommended an account that failed local validation")
		return nil, nil
	}
	result, err := s.tryAcquireAccountSlot(ctx, account.ID, account.Concurrency)
	if err != nil || result == nil || !result.Acquired {
		s.reportSubPilotRecommendationFailure(ctx, rec, account, normalizeOpenAICompatiblePlatform(platform), *groupID, requestedModel, sessionKey, "account_slot_unavailable", errorMessageOrDefault(err, "Sub2API account concurrency slot unavailable"))
		return nil, err
	}
	if sessionKey != "" {
		_ = s.setStickySessionAccountID(ctx, groupID, sessionKey, account.ID, openaiStickySessionTTL)
	}
	selection, err := s.newAcquiredSelectionResult(ctx, account, result.ReleaseFunc)
	if err != nil {
		return nil, err
	}
	selection.SubPilotLeaseID = rec.LeaseID
	selection.SubPilotRequestID = rec.RequestID
	return selection, nil
}

func (s *OpenAIGatewayService) validateSubPilotOpenAIAccount(ctx context.Context, accountID int64, platform string, requestedModel string, requireCompact bool, requiredCapability OpenAIEndpointCapability, accounts []Account) *Account {
	for i := range accounts {
		acc := &accounts[i]
		if acc.ID != accountID {
			continue
		}
		fresh := s.resolveFreshSchedulableOpenAIAccount(ctx, acc, platform, requestedModel, requireCompact, requiredCapability)
		if fresh == nil {
			return nil
		}
		fresh = s.recheckSelectedOpenAIAccountFromDB(ctx, fresh, platform, requestedModel, requireCompact, requiredCapability)
		if fresh == nil {
			return nil
		}
		return fresh
	}
	return nil
}

func subPilotGroupIDString(apiKey *APIKey) string {
	if apiKey == nil || apiKey.GroupID == nil {
		return ""
	}
	return strconv.FormatInt(*apiKey.GroupID, 10)
}

func subPilotAPIKeyIDString(apiKey *APIKey) string {
	if apiKey == nil {
		return ""
	}
	return strconv.FormatInt(apiKey.ID, 10)
}

func subPilotPlatformFromAPIKey(apiKey *APIKey, fallback string) string {
	if apiKey != nil && apiKey.Group != nil && apiKey.Group.Platform != "" {
		return apiKey.Group.Platform
	}
	if fallback != "" {
		return fallback
	}
	return PlatformAnthropic
}

func subPilotReportModel(model string, upstreamModel string) string {
	if model != "" {
		return model
	}
	if upstreamModel != "" {
		return upstreamModel
	}
	return "unknown"
}

func subPilotReportRequestID(ctx context.Context, requestID string) string {
	if requestID != "" {
		return requestID
	}
	return subPilotRequestID(ctx)
}

func subPilotOfficialUSD(totalCost float64, accountRateMultiplier float64) float64 {
	if totalCost <= 0 || accountRateMultiplier <= 0 {
		return 0
	}
	return totalCost / accountRateMultiplier
}

func errorMessageOrDefault(err error, fallback string) string {
	if err == nil {
		return fallback
	}
	return err.Error()
}

func (s *GatewayService) reportSubPilotRecommendationFailure(ctx context.Context, rec *subPilotSelectResult, account *Account, platform string, groupID int64, model string, sessionKey string, code string, message string) {
	if s == nil || rec == nil || rec.LeaseID == "" {
		return
	}
	client := s.subPilotClient()
	if client == nil {
		return
	}
	accountID := rec.AccountID
	if account != nil {
		accountID = account.ID
	}
	client.reportFailure(ctx, subPilotReportFailureRequest{
		RequestID:    subPilotReportRequestID(ctx, rec.RequestID),
		LeaseID:      rec.LeaseID,
		AccountID:    strconv.FormatInt(accountID, 10),
		Platform:     platform,
		GroupID:      strconv.FormatInt(groupID, 10),
		Model:        subPilotReportModel(model, ""),
		SessionKey:   sessionKey,
		ErrorCode:    code,
		ErrorMessage: message,
		RequestType:  "selection",
	})
}

func (s *OpenAIGatewayService) reportSubPilotRecommendationFailure(ctx context.Context, rec *subPilotSelectResult, account *Account, platform string, groupID int64, model string, sessionKey string, code string, message string) {
	if s == nil || rec == nil || rec.LeaseID == "" {
		return
	}
	client := s.subPilotClient()
	if client == nil {
		return
	}
	accountID := rec.AccountID
	if account != nil {
		accountID = account.ID
	}
	client.reportFailure(ctx, subPilotReportFailureRequest{
		RequestID:    subPilotReportRequestID(ctx, rec.RequestID),
		LeaseID:      rec.LeaseID,
		AccountID:    strconv.FormatInt(accountID, 10),
		Platform:     platform,
		GroupID:      strconv.FormatInt(groupID, 10),
		Model:        subPilotReportModel(model, ""),
		SessionKey:   sessionKey,
		ErrorCode:    code,
		ErrorMessage: message,
		RequestType:  "selection",
	})
}

func (s *OpenAIGatewayService) reportSubPilotSelectionFailure(ctx context.Context, selection *AccountSelectionResult, platform string, groupID *int64, model string, sessionKey string, code string, message string) {
	if selection == nil || selection.SubPilotLeaseID == "" || selection.Account == nil {
		return
	}
	groupIDString := ""
	if groupID != nil {
		groupIDString = strconv.FormatInt(*groupID, 10)
	}
	s.ReportSubPilotFailure(ctx, SubPilotFailureInput{
		LeaseID:      selection.SubPilotLeaseID,
		Account:      selection.Account,
		RequestID:    selection.SubPilotRequestID,
		Platform:     normalizeOpenAICompatiblePlatform(platform),
		GroupID:      groupIDString,
		Model:        model,
		SessionKey:   sessionKey,
		ErrorCode:    code,
		ErrorMessage: message,
		RequestType:  "selection",
	})
}
