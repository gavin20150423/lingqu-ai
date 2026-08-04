package service

import (
	"context"
	"testing"
)

func TestContentModerationAccountShareScope(t *testing.T) {
	groupID := int64(20)
	resolver := &contentModerationAccountShareScopeResolverStub{
		modeGroupID: groupID,
		membership: &AccountShareMembership{
			ID:             301,
			ListingID:      101,
			AccountID:      201,
			OwnerUserID:    401,
			ConsumerUserID: 501,
			APIKeyID:       601,
		},
		listing: &AccountShareListing{
			ID:          101,
			AccountID:   201,
			OwnerUserID: 401,
		},
	}
	svc := NewContentModerationService(nil, nil, nil, nil, nil, nil, nil, nil)
	svc.SetAccountShareModeResolver(resolver)
	input := ContentModerationCheckInput{
		UserID:   501,
		APIKeyID: 601,
		GroupID:  &groupID,
	}

	cfg := defaultContentModerationConfig()
	cfg.AllGroups = false
	cfg.GroupIDs = []int64{groupID}
	cfg.AccountShareModeScope = ContentModerationAccountShareModeScopeConfig{Enabled: false}
	cfg.normalize()
	inScope, _ := svc.resolveScope(context.Background(), cfg, input)
	if inScope {
		t.Fatalf("account mode group_ids must not enable moderation when account_share_mode_scope is disabled")
	}

	cfg.AccountShareModeScope = ContentModerationAccountShareModeScopeConfig{
		Enabled:    true,
		All:        false,
		Platforms:  []string{AccountShareModeGroupPlatformOpenAI},
		ListingIDs: []int64{101},
	}
	cfg.normalize()
	inScope, scope := svc.resolveScope(context.Background(), cfg, input)
	if !inScope || scope.ScopeType != contentModerationScopeTypeAccountShareMode {
		t.Fatalf("expected account share listing scope hit, inScope=%v scope=%#v", inScope, scope)
	}
	if scope.AccountShareListingID == nil || *scope.AccountShareListingID != 101 || scope.ConsumerUserID == nil || *scope.ConsumerUserID != 501 {
		t.Fatalf("unexpected scope context: %#v", scope)
	}

	resolver.err = ErrAccountShareModeGroupUnbound
	inScope, _ = svc.resolveScope(context.Background(), cfg, input)
	if inScope {
		t.Fatalf("unbound account mode group should be skipped by moderation")
	}
}

type contentModerationAccountShareScopeResolverStub struct {
	modeGroupID int64
	membership  *AccountShareMembership
	listing     *AccountShareListing
	err         error
}

func (s *contentModerationAccountShareScopeResolverStub) IsModeGroup(_ context.Context, groupID int64) bool {
	return s != nil && s.modeGroupID == groupID
}

func (s *contentModerationAccountShareScopeResolverStub) ResolveActiveBindingForRequest(context.Context, int64, int64, int64) (*AccountShareMembership, *AccountShareListing, error) {
	if s == nil {
		return nil, nil, nil
	}
	return s.membership, s.listing, s.err
}
