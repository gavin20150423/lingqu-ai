// Code extracted from PIXEL-API/PixelAPI for account-sharing compatibility.
package service

import (
	"context"

	"time"
)

func (s *OpenAIGatewayService) selectAccountShareModeBoundAccount(
	ctx context.Context,
	groupID *int64,
	requestedModel string,
	excludedIDs map[int64]struct{},
	requiredTransport OpenAIUpstreamTransport,
	requiredImageCapability OpenAIImagesCapability,
	requiredEndpointCapability OpenAIEndpointCapability,
	requireCompact bool,
) (*AccountSelectionResult, OpenAIAccountScheduleDecision, bool, error) {
	decision := OpenAIAccountScheduleDecision{Layer: openAIAccountScheduleLayerAccountShareMode}
	if s == nil || s.accountShareModeService == nil || groupID == nil || *groupID <= 0 {
		return nil, decision, false, nil
	}
	if !s.accountShareModeService.IsModeGroup(ctx, *groupID) {
		return nil, decision, false, nil
	}
	boundImageCapability := requiredImageCapability
	if boundImageCapability == OpenAIImagesCapabilityNative {

		boundImageCapability = OpenAIImagesCapabilityBasic
	}
	reqCtx, ok := AccountShareModeRequestFromContext(ctx)
	if !ok {
		return nil, decision, true, ErrAccountShareModeGroupUnbound
	}
	var membership *AccountShareMembership
	var listing *AccountShareListing
	var account *Account
	var lastErr error
	for attempt := 0; attempt < AccountShareModeQueueMaxItems; attempt++ {
		var err error
		membership, listing, err = s.accountShareModeService.ResolveActiveBindingForRequest(ctx, reqCtx.UserID, reqCtx.APIKeyID, *groupID)
		if err != nil {
			return nil, decision, true, err
		}
		if membership == nil || listing == nil {
			return nil, decision, true, ErrAccountShareModeGroupUnbound
		}
		accountID := membership.AccountID
		if accountID <= 0 {
			return nil, decision, true, ErrNoAvailableAccounts
		}
		decision.CandidateCount = 1
		decision.SelectedAccountID = accountID
		retryCurrentMembership := false
		if excludedIDs != nil {
			if _, excluded := excludedIDs[accountID]; excluded {
				lastErr = ErrNoAvailableAccounts
				retryCurrentMembership = true
			}
		}
		if !retryCurrentMembership && s.userRepo != nil {
			user, err := s.userRepo.GetByID(ctx, reqCtx.UserID)
			if err != nil {
				return nil, decision, true, err
			}
			if user.Balance < listing.MinBalanceRequired {
				lastErr = ErrAccountShareBalanceBelowMinimum
				retryCurrentMembership = true
			}
		}
		if !retryCurrentMembership {
			account, err = s.accountRepo.GetByID(ctx, accountID)
			if err != nil {
				return nil, decision, true, err
			}
			if account == nil {
				lastErr = ErrNoAvailableAccounts
				retryCurrentMembership = true
			}
		}
		if !retryCurrentMembership {
			decision.SelectedAccountType = account.Type
			if account.ID != accountID || !account.IsOpenAICompatible() || !account.IsSchedulable() {
				lastErr = ErrNoAvailableAccounts
				retryCurrentMembership = true
			}
		}
		if !retryCurrentMembership && requestedModel != "" && !account.IsModelSupported(requestedModel) {
			return nil, decision, true, accountShareModeUnsupportedModelError(requestedModel)
		}
		if !retryCurrentMembership && (!isOpenAIAccountEligibleForRequest(account, requestedModel, requireCompact) || s.isOpenAIAccountRequestRuntimeBlocked(account, requestedModel)) {
			lastErr = ErrNoAvailableAccounts
			retryCurrentMembership = true
		}
		if !retryCurrentMembership && !accountSupportsRequestedOpenAIImageCapability(account, boundImageCapability) {
			lastErr = ErrNoAvailableAccounts
			retryCurrentMembership = true
		}
		if !retryCurrentMembership && !account.SupportsOpenAIEndpointCapability(requiredEndpointCapability) {
			lastErr = ErrNoAvailableAccounts
			retryCurrentMembership = true
		}
		if !retryCurrentMembership && !s.isOpenAIAccountTransportCompatible(account, requiredTransport) {
			lastErr = ErrNoAvailableAccounts
			retryCurrentMembership = true
		}
		if !retryCurrentMembership && s.needsUpstreamChannelRestrictionCheck(ctx, groupID) && s.isUpstreamModelRestrictedByChannel(ctx, *groupID, account, requestedModel, requireCompact) {
			lastErr = ErrNoAvailableAccounts
			retryCurrentMembership = true
		}
		if retryCurrentMembership {
			now := time.Now().UTC()
			deferred, err := s.accountShareModeService.deferMembershipForDispatchRetry(ctx, reqCtx, membership, now)
			if err != nil {
				return nil, decision, true, err
			}
			if !deferred {
				if lastErr != nil {
					return nil, decision, true, lastErr
				}
				return nil, decision, true, ErrNoAvailableAccounts
			}
			membership = nil
			listing = nil
			account = nil
			continue
		}
		break
	}
	if membership == nil || listing == nil || account == nil {
		if lastErr != nil {
			return nil, decision, true, lastErr
		}
		return nil, decision, true, ErrNoAvailableAccounts
	}

	membershipSlot, err := s.accountShareModeService.AcquireMembershipSlot(ctx, membership.ID, listing.PerUserConcurrency)
	if err != nil {
		return nil, decision, true, err
	}
	if membershipSlot == nil || !membershipSlot.Acquired {
		return nil, decision, true, ErrAccountSharePerUserConcurrencyExceeded
	}
	accountSlot, err := s.tryAcquireAccountSlot(ctx, account.ID, account.Concurrency)
	if err != nil {
		if membershipSlot.ReleaseFunc != nil {
			membershipSlot.ReleaseFunc()
		}
		return nil, decision, true, err
	}
	if accountSlot == nil || !accountSlot.Acquired {
		if membershipSlot.ReleaseFunc != nil {
			membershipSlot.ReleaseFunc()
		}
		return nil, decision, true, ErrNoAvailableAccounts
	}

	release := func() {
		if accountSlot.ReleaseFunc != nil {
			accountSlot.ReleaseFunc()
		}
		if membershipSlot.ReleaseFunc != nil {
			membershipSlot.ReleaseFunc()
		}
	}
	return newAccountShareModeSelectionResult(account, true, release, nil), decision, true, nil
}
