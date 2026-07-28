package service

import "context"

func resolveAccountShareModeBillingBinding(
	ctx context.Context,
	accountShareModeService *AccountShareModeService,
	user *User,
	apiKey *APIKey,
	account *Account,
) (*AccountShareMembership, *AccountShareListing, error) {
	if accountShareModeService == nil || user == nil || apiKey == nil || apiKey.GroupID == nil || account == nil {
		return nil, nil, nil
	}
	membership, listing, err := accountShareModeService.ResolveActiveBindingForRequest(ctx, user.ID, apiKey.ID, *apiKey.GroupID)
	if err != nil {
		return nil, nil, err
	}
	if listing != nil && listing.AccountID != account.ID {
		return nil, nil, ErrNoAvailableAccounts
	}
	return membership, listing, nil
}

func accountShareModeBillingMultiplier(membership *AccountShareMembership, listing *AccountShareListing, fallback float64) float64 {
	if IsAccountShareModeOwnerSelfUse(membership, listing) {
		return AccountShareModeOwnerSelfUseMultiplier
	}
	if listing != nil {
		return listing.RateMultiplier
	}
	return fallback
}

func buildAccountShareModeSettlement(
	ctx context.Context,
	accountShareModeService *AccountShareModeService,
	account *Account,
	membership *AccountShareMembership,
	listing *AccountShareListing,
	cost *CostBreakdown,
	durationMs int,
) (*AccountShareModeBillingSnapshot, error) {
	if accountShareModeService == nil || account == nil || membership == nil || listing == nil || cost == nil {
		return nil, nil
	}
	policy, err := accountShareModeService.ResolvePolicy(ctx, account.Platform)
	if err != nil {
		return nil, err
	}
	return BuildAccountShareModeBillingSnapshot(membership, listing, policy, cost.ActualCost, 0, durationMs), nil
}
