//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// resetCardUserSubRepoStub 在 noop 替身基础上实现重置卡用例需要的方法：
// GetByID / ResetUsageWindows，以及 SubscriptionResetCardStore 的三个方法。
type resetCardUserSubRepoStub struct {
	userSubRepoNoop

	sub     *UserSubscription
	balance int

	consumed int
	refunds  int

	resetDaily   bool
	resetWeekly  bool
	resetMonthly bool

	resetErr error
}

func (r *resetCardUserSubRepoStub) GetByID(_ context.Context, id int64) (*UserSubscription, error) {
	if r.sub == nil || r.sub.ID != id {
		return nil, ErrSubscriptionNotFound
	}
	cp := *r.sub
	return &cp, nil
}

func (r *resetCardUserSubRepoStub) ResetUsageWindows(
	_ context.Context,
	_ int64,
	resetDaily, resetWeekly, resetMonthly bool,
	dailyStart, periodicStart time.Time,
) error {
	r.resetDaily, r.resetWeekly, r.resetMonthly = resetDaily, resetWeekly, resetMonthly
	if r.resetErr != nil {
		return r.resetErr
	}
	if r.sub != nil {
		if resetDaily {
			r.sub.DailyUsageUSD = 0
			r.sub.DailyWindowStart = &dailyStart
		}
		if resetWeekly {
			r.sub.WeeklyUsageUSD = 0
			r.sub.WeeklyWindowStart = &periodicStart
		}
		if resetMonthly {
			r.sub.MonthlyUsageUSD = 0
			r.sub.MonthlyWindowStart = &periodicStart
		}
	}
	return nil
}

func (r *resetCardUserSubRepoStub) GetResetCardBalance(_ context.Context, _ int64) (int, error) {
	return r.balance, nil
}

func (r *resetCardUserSubRepoStub) ConsumeResetCard(_ context.Context, _, _ int64) (int, bool, error) {
	if r.balance <= 0 {
		return 0, false, nil
	}
	r.balance--
	r.consumed++
	return r.balance, true, nil
}

func (r *resetCardUserSubRepoStub) AdjustResetCards(
	_ context.Context,
	_ int64,
	delta int,
	_ string,
	_ int64,
	_ string,
) (int, error) {
	r.balance += delta
	if delta > 0 {
		r.refunds++
	}
	return r.balance, nil
}

func newResetCardService(stub *resetCardUserSubRepoStub, now time.Time) *SubscriptionService {
	return &SubscriptionService{
		userSubRepo: stub,
		now:         func() time.Time { return now },
	}
}

func newResetCardSub(now time.Time) *UserSubscription {
	return &UserSubscription{
		ID:                 7,
		UserID:             40,
		GroupID:            138,
		Status:             SubscriptionStatusActive,
		ExpiresAt:          now.Add(24 * time.Hour),
		DailyUsageUSD:      12.5,
		WeeklyUsageUSD:     30,
		MonthlyUsageUSD:    99,
		DailyWindowStart:   resetCardPtrTime(now.Add(-2 * time.Hour)),
		WeeklyWindowStart:  resetCardPtrTime(now.Add(-48 * time.Hour)),
		MonthlyWindowStart: resetCardPtrTime(now.Add(-240 * time.Hour)),
		Group:              &Group{ID: 138, Name: "test-group"},
	}
}

func resetCardPtrTime(t time.Time) *time.Time { return &t }

// 正常消耗：扣 1 张、日周归零、月度不变，且明确只重置日 + 周。
func TestUseResetCard_ResetsDailyAndWeeklyOnly(t *testing.T) {
	now := time.Date(2026, time.November, 23, 12, 0, 0, 0, time.UTC)
	stub := &resetCardUserSubRepoStub{sub: newResetCardSub(now), balance: 3}
	svc := newResetCardService(stub, now)

	updated, progress, remaining, err := svc.UseResetCard(context.Background(), 40, 7)

	require.NoError(t, err)
	require.Equal(t, 2, remaining)
	require.Equal(t, 1, stub.consumed)
	require.True(t, stub.resetDaily, "daily window must be reset")
	require.True(t, stub.resetWeekly, "weekly window must be reset")
	require.False(t, stub.resetMonthly, "monthly window must NOT be reset")
	require.InDelta(t, 0, updated.DailyUsageUSD, 1e-9)
	require.InDelta(t, 0, updated.WeeklyUsageUSD, 1e-9)
	require.InDelta(t, 99, updated.MonthlyUsageUSD, 1e-9, "monthly usage must stay untouched")
	require.NotNil(t, progress)
	require.Equal(t, 2, progress.ResetCardsRemaining)
}

// 卡不足：不扣卡、不重置。
func TestUseResetCard_InsufficientCards(t *testing.T) {
	now := time.Date(2026, time.November, 23, 12, 0, 0, 0, time.UTC)
	stub := &resetCardUserSubRepoStub{sub: newResetCardSub(now), balance: 0}
	svc := newResetCardService(stub, now)

	_, _, _, err := svc.UseResetCard(context.Background(), 40, 7)

	require.ErrorIs(t, err, ErrResetCardInsufficient)
	require.Equal(t, 0, stub.consumed)
	require.False(t, stub.resetDaily)
	require.False(t, stub.resetWeekly)
}

// 日、周都已是 0：拒绝并提示，不浪费卡。
func TestUseResetCard_NothingToReset(t *testing.T) {
	now := time.Date(2026, time.November, 23, 12, 0, 0, 0, time.UTC)
	sub := newResetCardSub(now)
	sub.DailyUsageUSD = 0
	sub.WeeklyUsageUSD = 0
	stub := &resetCardUserSubRepoStub{sub: sub, balance: 5}
	svc := newResetCardService(stub, now)

	_, _, _, err := svc.UseResetCard(context.Background(), 40, 7)

	require.ErrorIs(t, err, ErrResetCardNothingToReset)
	require.Equal(t, 0, stub.consumed)
	require.False(t, stub.resetDaily)
}

// 订阅不属于该用户：越权拦截，不扣卡。
func TestUseResetCard_RejectsForeignSubscription(t *testing.T) {
	now := time.Date(2026, time.November, 23, 12, 0, 0, 0, time.UTC)
	stub := &resetCardUserSubRepoStub{sub: newResetCardSub(now), balance: 1}
	svc := newResetCardService(stub, now)

	_, _, _, err := svc.UseResetCard(context.Background(), 999, 7)

	require.ErrorIs(t, err, ErrResetCardSubscriptionMismatch)
	require.Equal(t, 0, stub.consumed)
}

// 重置失败必须退回卡，用户不能白扣。
func TestUseResetCard_RefundsOnResetFailure(t *testing.T) {
	now := time.Date(2026, time.November, 23, 12, 0, 0, 0, time.UTC)
	stub := &resetCardUserSubRepoStub{sub: newResetCardSub(now), balance: 1}
	stub.resetErr = errors.New("db down")
	svc := newResetCardService(stub, now)

	_, _, _, err := svc.UseResetCard(context.Background(), 40, 7)

	require.Error(t, err)
	require.Equal(t, 1, stub.consumed)
	require.Equal(t, 1, stub.refunds, "failed reset must refund the consumed card")
	require.Equal(t, 1, stub.balance)
}

// 过期订阅不允许用卡。
func TestUseResetCard_RejectsExpiredSubscription(t *testing.T) {
	now := time.Date(2026, time.November, 23, 12, 0, 0, 0, time.UTC)
	sub := newResetCardSub(now)
	sub.ExpiresAt = now.Add(-time.Minute)
	stub := &resetCardUserSubRepoStub{sub: sub, balance: 3}
	svc := newResetCardService(stub, now)

	_, _, _, err := svc.UseResetCard(context.Background(), 40, 7)

	require.ErrorIs(t, err, ErrSubscriptionExpired)
	require.Equal(t, 0, stub.consumed)
}

// 后台发放：delta 为正 -> grant；set 到目标值 -> 换算增量；0 增量报错。
func TestAdjustResetCardsByAdmin(t *testing.T) {
	ctx := context.Background()

	grantStub := &resetCardUserSubRepoStub{sub: newResetCardSub(time.Now()), balance: 1}
	grantSvc := newResetCardService(grantStub, time.Now())
	balance, err := grantSvc.AdjustResetCardsByAdmin(ctx, 40, 3, nil, 1, "补偿")
	require.NoError(t, err)
	require.Equal(t, 4, balance)

	setStub := &resetCardUserSubRepoStub{sub: newResetCardSub(time.Now()), balance: 1}
	setSvc := newResetCardService(setStub, time.Now())
	target := 10
	balance, err = setSvc.AdjustResetCardsByAdmin(ctx, 40, 0, &target, 1, "")
	require.NoError(t, err)
	require.Equal(t, 10, balance)

	zeroStub := &resetCardUserSubRepoStub{sub: newResetCardSub(time.Now()), balance: 1}
	zeroSvc := newResetCardService(zeroStub, time.Now())
	_, err = zeroSvc.AdjustResetCardsByAdmin(ctx, 40, 0, nil, 1, "")
	require.ErrorIs(t, err, ErrResetCardInvalidAmount)

	negative := -1
	_, err = zeroSvc.AdjustResetCardsByAdmin(ctx, 40, 0, &negative, 1, "")
	require.ErrorIs(t, err, ErrResetCardInvalidTarget)
}
