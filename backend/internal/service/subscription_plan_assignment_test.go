package service

import (
	"context"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestApplySubscriptionPlanCopiesAuthoritativeTierSnapshot(t *testing.T) {
	client := newPaymentConfigServiceTestClient(t)
	plan, err := client.SubscriptionPlan.Create().
		SetGroupID(142).
		SetName("Standard").
		SetPrice(299).
		SetValidityDays(1).
		SetValidityUnit("months").
		SetDailyLimitUsd(120).
		SetWeeklyLimitUsd(420).
		SetMonthlyLimitUsd(1500).
		SetEntitlements(map[string]interface{}{"gpt": float64(1500)}).
		Save(context.Background())
	require.NoError(t, err)

	planID := int64(plan.ID)
	input := &AssignSubscriptionInput{
		GroupID:         142,
		PlanID:          &planID,
		ValidityDays:    999,
		DailyLimitUSD:   float64Ptr(1),
		WeeklyLimitUSD:  float64Ptr(2),
		MonthlyLimitUSD: float64Ptr(3),
		Entitlements:    map[string]any{"gpt": float64(1)},
	}

	svc := &SubscriptionService{entClient: client}
	require.NoError(t, svc.applySubscriptionPlan(context.Background(), input))
	require.Equal(t, 30, input.ValidityDays)
	require.Equal(t, 120.0, *input.DailyLimitUSD)
	require.Equal(t, 420.0, *input.WeeklyLimitUSD)
	require.Equal(t, 1500.0, *input.MonthlyLimitUSD)
	require.Equal(t, map[string]any{"gpt": float64(1500)}, input.Entitlements)
}

func TestApplySubscriptionPlanRejectsGroupMismatch(t *testing.T) {
	client := newPaymentConfigServiceTestClient(t)
	plan, err := client.SubscriptionPlan.Create().
		SetGroupID(142).
		SetName("Basic").
		SetPrice(99).
		Save(context.Background())
	require.NoError(t, err)

	planID := int64(plan.ID)
	input := &AssignSubscriptionInput{GroupID: 143, PlanID: &planID}
	err = (&SubscriptionService{entClient: client}).applySubscriptionPlan(context.Background(), input)
	require.Error(t, err)
	require.Equal(t, infraerrors.Code(ErrSubscriptionPlanGroupMismatch), infraerrors.Code(err))
}

func TestApplySubscriptionPlanUsesGroupQuotaFallbackForLegacyPlan(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	group, err := client.Group.Create().
		SetName("legacy plan quota fallback").
		SetDailyLimitUsd(12).
		SetWeeklyLimitUsd(42).
		SetMonthlyLimitUsd(120).
		Save(ctx)
	require.NoError(t, err)

	plan, err := client.SubscriptionPlan.Create().
		SetGroupID(int64(group.ID)).
		SetName("Legacy").
		SetPrice(99).
		Save(ctx)
	require.NoError(t, err)

	planID := int64(plan.ID)
	input := &AssignSubscriptionInput{GroupID: int64(group.ID), PlanID: &planID}
	require.NoError(t, (&SubscriptionService{entClient: client}).applySubscriptionPlan(ctx, input))
	require.Equal(t, 12.0, *input.DailyLimitUSD)
	require.Equal(t, 42.0, *input.WeeklyLimitUSD)
	require.Equal(t, 120.0, *input.MonthlyLimitUSD)
}

func TestSubscriptionPlanValidityDays(t *testing.T) {
	require.Equal(t, 30, subscriptionPlanValidityDays(1, "month"))
	require.Equal(t, 14, subscriptionPlanValidityDays(2, "weeks"))
	require.Equal(t, 30, subscriptionPlanValidityDays(0, "days"))
	require.Equal(t, MaxValidityDays, subscriptionPlanValidityDays(MaxValidityDays, "months"))
}

func TestAdminAssignmentMigratesExistingSubscriptionToSelectedPlan(t *testing.T) {
	groupRepo := &subscriptionGroupRepoStub{
		group: &Group{ID: 142, SubscriptionType: SubscriptionTypeSubscription},
	}
	subRepo := newSubscriptionUserSubRepoStub()
	start := time.Now().Add(-time.Hour)
	subRepo.seed(&UserSubscription{
		ID:        900,
		UserID:    901,
		GroupID:   142,
		StartsAt:  start,
		ExpiresAt: start.AddDate(0, 0, 30),
		Status:    SubscriptionStatusActive,
		Notes:     "legacy admin assignment",
	})

	planID := int64(77)
	daily, weekly, monthly := 30.0, 105.0, 300.0
	svc := NewSubscriptionService(groupRepo, subRepo, nil, nil, nil)
	updated, reused, err := svc.assignSubscriptionWithReuse(context.Background(), &AssignSubscriptionInput{
		UserID:          901,
		GroupID:         142,
		PlanID:          &planID,
		DailyLimitUSD:   &daily,
		WeeklyLimitUSD:  &weekly,
		MonthlyLimitUSD: &monthly,
		ValidityDays:    30,
		Notes:           "legacy admin assignment",
		Entitlements:    map[string]any{"claude": float64(300)},
	})

	require.NoError(t, err)
	require.True(t, reused)
	require.Equal(t, int64(900), updated.ID)
	require.Equal(t, planID, *updated.PlanID)
	require.Equal(t, daily, *updated.DailyLimitUSD)
	require.Equal(t, weekly, *updated.WeeklyLimitUSD)
	require.Equal(t, monthly, *updated.MonthlyLimitUSD)
	require.Equal(t, map[string]any{"claude": float64(300)}, updated.Entitlements)
	require.Equal(t, start, updated.StartsAt)
	require.Equal(t, start.AddDate(0, 0, 30), updated.ExpiresAt)
}
