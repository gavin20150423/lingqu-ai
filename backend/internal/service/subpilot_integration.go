package service

import (
	"context"
	"sort"
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

func (s *GatewayService) trySubPilotRecommend(ctx context.Context, groupID *int64, platform string, sessionKey string, requestedModel string, excludedIDs map[int64]struct{}, accounts []Account, useMixed bool) (*AccountSelectionResult, bool, error) {
	if SubPilotDisabledFromContext(ctx) {
		return nil, false, nil
	}
	client := s.subPilotClient()
	if client == nil || groupID == nil || requestedModel == "" {
		return nil, false, nil
	}
	localExcluded := cloneSubPilotExcludedIDs(excludedIDs)
	for {
		req := newSubPilotSelectRequest(ctx, platform, *groupID, requestedModel, sessionKey)
		req.ExcludedAccountIDs = subPilotExcludedAccountIDs(localExcluded)
		rec, handled, err := client.recommendAccountWithOwnership(ctx, req)
		if err != nil || !handled {
			return nil, handled, err
		}
		if rec == nil {
			return nil, true, ErrNoAvailableAccounts
		}
		if _, excluded := localExcluded[rec.AccountID]; excluded {
			releaseSubPilotRecommendation(client, rec)
			return nil, true, ErrNoAvailableAccounts
		}
		account := s.validateSubPilotGatewayAccount(ctx, rec.AccountID, platform, requestedModel, accounts, useMixed)
		if account == nil {
			releaseSubPilotRecommendation(client, rec)
			localExcluded[rec.AccountID] = struct{}{}
			continue
		}
		result, acquireErr := s.tryAcquireAccountSlot(ctx, account.ID, account.Concurrency)
		if acquireErr != nil {
			releaseSubPilotRecommendation(client, rec)
			return nil, true, acquireErr
		}
		if result == nil || !result.Acquired {
			releaseSubPilotRecommendation(client, rec)
			localExcluded[rec.AccountID] = struct{}{}
			continue
		}
		if !s.checkAndRegisterSession(ctx, account, sessionKey) {
			result.ReleaseFunc()
			releaseSubPilotRecommendation(client, rec)
			localExcluded[rec.AccountID] = struct{}{}
			continue
		}
		if sessionKey != "" && s.cache != nil {
			_ = s.cache.SetSessionAccountID(ctx, derefGroupID(groupID), sessionKey, account.ID, stickySessionTTL)
		}
		selection, selectionErr := s.newSelectionResult(ctx, account, true, result.ReleaseFunc, nil)
		if selectionErr != nil {
			result.ReleaseFunc()
			releaseSubPilotRecommendation(client, rec)
			return nil, true, selectionErr
		}
		selection.SubPilotLeaseID = rec.LeaseID
		selection.SubPilotRequestID = rec.RequestID
		return selection, true, nil
	}
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

func (s *OpenAIGatewayService) trySubPilotRecommend(ctx context.Context, groupID *int64, platform string, sessionKey string, requestedModel string, excludedIDs map[int64]struct{}, requireCompact bool, requiredCapability OpenAIEndpointCapability, requiredTransport OpenAIUpstreamTransport, requiredImageCapability OpenAIImagesCapability, accounts []Account) (*AccountSelectionResult, bool, error) {
	if SubPilotDisabledFromContext(ctx) {
		return nil, false, nil
	}
	client := s.subPilotClient()
	if client == nil || groupID == nil || requestedModel == "" {
		return nil, false, nil
	}
	localExcluded := cloneSubPilotExcludedIDs(excludedIDs)
	for {
		req := newSubPilotSelectRequest(ctx, normalizeOpenAICompatiblePlatform(platform), *groupID, requestedModel, sessionKey)
		req.ExcludedAccountIDs = subPilotExcludedAccountIDs(localExcluded)
		rec, handled, err := client.recommendAccountWithOwnership(ctx, req)
		if err != nil || !handled {
			return nil, handled, err
		}
		if rec == nil {
			return nil, true, ErrNoAvailableAccounts
		}
		if _, excluded := localExcluded[rec.AccountID]; excluded {
			releaseSubPilotRecommendation(client, rec)
			return nil, true, ErrNoAvailableAccounts
		}
		account := s.validateSubPilotOpenAIAccount(ctx, rec.AccountID, groupID, platform, requestedModel, requireCompact, requiredCapability, requiredTransport, requiredImageCapability, accounts)
		if account == nil {
			releaseSubPilotRecommendation(client, rec)
			localExcluded[rec.AccountID] = struct{}{}
			continue
		}
		result, acquireErr := s.tryAcquireAccountSlot(ctx, account.ID, account.Concurrency)
		if acquireErr != nil {
			releaseSubPilotRecommendation(client, rec)
			return nil, true, acquireErr
		}
		if result == nil || !result.Acquired {
			releaseSubPilotRecommendation(client, rec)
			localExcluded[rec.AccountID] = struct{}{}
			continue
		}
		if sessionKey != "" {
			_ = s.setStickySessionAccountID(ctx, groupID, sessionKey, account.ID, openaiStickySessionTTL)
		}
		selection, selectionErr := s.newAcquiredSelectionResult(ctx, account, result.ReleaseFunc)
		if selectionErr != nil {
			result.ReleaseFunc()
			releaseSubPilotRecommendation(client, rec)
			return nil, true, selectionErr
		}
		selection.SubPilotLeaseID = rec.LeaseID
		selection.SubPilotRequestID = rec.RequestID
		return selection, true, nil
	}
}

func (s *OpenAIGatewayService) validateSubPilotOpenAIAccount(ctx context.Context, accountID int64, groupID *int64, platform string, requestedModel string, requireCompact bool, requiredCapability OpenAIEndpointCapability, requiredTransport OpenAIUpstreamTransport, requiredImageCapability OpenAIImagesCapability, accounts []Account) *Account {
	for i := range accounts {
		acc := &accounts[i]
		if acc.ID != accountID {
			continue
		}
		fresh := s.resolveFreshSchedulableOpenAIAccount(ctx, acc, platform, requestedModel, requireCompact, requiredCapability)
		if fresh == nil {
			return nil
		}
		fresh = s.recheckSelectedOpenAIAccountFromDB(ctx, fresh, groupID, platform, requestedModel, requireCompact, requiredCapability)
		if fresh == nil {
			return nil
		}
		if !s.isOpenAIAccountTransportCompatible(fresh, requiredTransport) || !accountSupportsOpenAICapabilities(fresh, requiredCapability, requiredImageCapability) {
			return nil
		}
		return fresh
	}
	return nil
}

func cloneSubPilotExcludedIDs(excludedIDs map[int64]struct{}) map[int64]struct{} {
	cloned := make(map[int64]struct{}, len(excludedIDs))
	for accountID := range excludedIDs {
		cloned[accountID] = struct{}{}
	}
	return cloned
}

func subPilotExcludedAccountIDs(excludedIDs map[int64]struct{}) []string {
	ids := make([]int64, 0, len(excludedIDs))
	for accountID := range excludedIDs {
		if accountID > 0 {
			ids = append(ids, accountID)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	result := make([]string, 0, len(ids))
	for _, accountID := range ids {
		result = append(result, strconv.FormatInt(accountID, 10))
	}
	return result
}

func releaseSubPilotRecommendation(client *subPilotClient, rec *subPilotSelectResult) {
	if client == nil || rec == nil || rec.AccountID <= 0 || rec.LeaseID == "" {
		return
	}
	client.releaseLease(context.Background(), subPilotReleaseLeaseRequest{
		RequestID: rec.RequestID,
		LeaseID:   rec.LeaseID,
		AccountID: strconv.FormatInt(rec.AccountID, 10),
	})
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

func subPilotAPIKeyIDFromRequestContext(ctx context.Context) string {
	apiKeyID, ok := SubPilotAPIKeyIDFromContext(ctx)
	if !ok || apiKeyID <= 0 {
		return ""
	}
	return strconv.FormatInt(apiKeyID, 10)
}

func newSubPilotSelectRequest(ctx context.Context, platform string, groupID int64, model string, sessionKey string) subPilotSelectRequest {
	return subPilotSelectRequest{
		RequestID:  subPilotRequestID(ctx),
		APIKeyID:   subPilotAPIKeyIDFromRequestContext(ctx),
		Platform:   platform,
		GroupID:    strconv.FormatInt(groupID, 10),
		Model:      model,
		SessionKey: sessionKey,
	}
}

func subPilotReportAPIKeyID(ctx context.Context, apiKey *APIKey) string {
	if apiKeyID := subPilotAPIKeyIDString(apiKey); apiKeyID != "" {
		return apiKeyID
	}
	return subPilotAPIKeyIDFromRequestContext(ctx)
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
