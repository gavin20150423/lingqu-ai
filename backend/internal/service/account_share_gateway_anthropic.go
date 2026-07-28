// Code extracted from PIXEL-API/PixelAPI for account-sharing compatibility.
package service

import (
	"context"

	"time"
)

func (s *GatewayService) selectAccountShareModeBoundAccount(ctx context.Context, groupID *int64, sessionHash string, requestedModel string, excludedIDs map[int64]struct{}) (*AccountSelectionResult, bool, error) {
	if s == nil || s.accountShareModeService == nil || s.accountRepo == nil || groupID == nil || *groupID <= 0 {
		return nil, false, nil
	}
	if !s.accountShareModeService.IsModeGroup(ctx, *groupID) {
		return nil, false, nil
	}
	reqCtx, ok := AccountShareModeRequestFromContext(ctx)
	if !ok || reqCtx.UserID <= 0 || reqCtx.APIKeyID <= 0 {
		return nil, true, ErrAccountShareModeGroupUnbound
	}
	var membership *AccountShareMembership
	var listing *AccountShareListing
	var account *Account
	var lastErr error
	for attempt := 0; attempt < AccountShareModeQueueMaxItems; attempt++ {
		var err error
		membership, listing, err = s.accountShareModeService.ResolveActiveBindingForRequest(ctx, reqCtx.UserID, reqCtx.APIKeyID, *groupID)
		if err != nil {
			return nil, true, err
		}
		if membership == nil || listing == nil {
			return nil, true, ErrAccountShareModeGroupUnbound
		}
		accountID := membership.AccountID
		if accountID <= 0 {
			return nil, true, ErrNoAvailableAccounts
		}
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
				return nil, true, err
			}
			if user.Balance < listing.MinBalanceRequired {
				lastErr = ErrAccountShareBalanceBelowMinimum
				retryCurrentMembership = true
			}
		}
		if !retryCurrentMembership {
			account, err = s.accountRepo.GetByID(ctx, accountID)
			if err != nil {
				return nil, true, err
			}
			if account == nil || account.ID != accountID || account.Platform != PlatformAnthropic || account.Type != AccountTypeOAuth || !s.isAccountSchedulableForSelection(account) {
				lastErr = ErrNoAvailableAccounts
				retryCurrentMembership = true
			}
		}
		if !retryCurrentMembership && requestedModel != "" && !s.isModelSupportedByAccountWithContext(ctx, account, requestedModel) {
			return nil, true, accountShareModeUnsupportedModelError(requestedModel)
		}
		if !retryCurrentMembership && !s.isAccountSchedulableForModelSelection(ctx, account, requestedModel) {
			lastErr = ErrNoAvailableAccounts
			retryCurrentMembership = true
		}
		if !retryCurrentMembership && !s.isAccountSchedulableForQuota(account) {
			lastErr = ErrNoAvailableAccounts
			retryCurrentMembership = true
		}
		if !retryCurrentMembership && !s.isAccountSchedulableForWindowCost(ctx, account, false) {
			lastErr = ErrNoAvailableAccounts
			retryCurrentMembership = true
		}
		if !retryCurrentMembership && !s.isAccountSchedulableForRPM(ctx, account, false) {
			lastErr = ErrNoAvailableAccounts
			retryCurrentMembership = true
		}
		if retryCurrentMembership {
			now := time.Now().UTC()
			deferred, err := s.accountShareModeService.deferMembershipForDispatchRetry(ctx, reqCtx, membership, now)
			if err != nil {
				return nil, true, err
			}
			if !deferred {
				if lastErr != nil {
					return nil, true, lastErr
				}
				return nil, true, ErrNoAvailableAccounts
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
			return nil, true, lastErr
		}
		return nil, true, ErrNoAvailableAccounts
	}

	membershipSlot, err := s.accountShareModeService.AcquireMembershipSlot(ctx, membership.ID, listing.PerUserConcurrency)
	if err != nil {
		return nil, true, err
	}
	if membershipSlot == nil || !membershipSlot.Acquired {
		return nil, true, ErrAccountSharePerUserConcurrencyExceeded
	}
	accountSlot, err := s.tryAcquireAccountSlot(ctx, account.ID, account.Concurrency)
	if err != nil {
		if membershipSlot.ReleaseFunc != nil {
			membershipSlot.ReleaseFunc()
		}
		return nil, true, err
	}
	if accountSlot == nil || !accountSlot.Acquired {
		if membershipSlot.ReleaseFunc != nil {
			membershipSlot.ReleaseFunc()
		}
		return nil, true, ErrNoAvailableAccounts
	}
	release := func() {
		if accountSlot.ReleaseFunc != nil {
			accountSlot.ReleaseFunc()
		}
		if membershipSlot.ReleaseFunc != nil {
			membershipSlot.ReleaseFunc()
		}
	}
	if !s.checkAndRegisterSession(ctx, account, sessionHash) {
		release()
		return nil, true, ErrNoAvailableAccounts
	}
	if sessionHash != "" && s.cache != nil {
		_ = s.cache.SetSessionAccountID(ctx, derefGroupID(groupID), sessionHash, account.ID, stickySessionTTL)
	}
	return newAccountShareModeSelectionResult(account, true, release, nil), true, nil
}
