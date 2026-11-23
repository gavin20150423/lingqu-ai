package service

import (
	"context"
	"log"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// SubscriptionResetCardStore 是订阅重置卡的存取端口。
//
// 由订阅仓储（internal/repository）实现；服务层通过类型断言获取，缺失时返回
// ErrResetCardStoreUnavailable，避免为了让测试替身一起实现而扩张主接口。
type SubscriptionResetCardStore interface {
	// GetResetCardBalance 返回剩余张数；无记录时按默认 1 张返回。
	GetResetCardBalance(ctx context.Context, userID int64) (int, error)
	// ConsumeResetCard 原子扣减一张，ok=false 表示余额不足（非错误）。
	ConsumeResetCard(ctx context.Context, userID, subscriptionID int64) (remaining int, ok bool, err error)
	// AdjustResetCards 发放/扣减/退回并写审计；balance 不会低于 0。
	AdjustResetCards(ctx context.Context, userID int64, delta int, action string, operatorID int64, note string) (int, error)
}

var (
	ErrResetCardStoreUnavailable     = infraerrors.InternalServer("RESET_CARD_STORE_UNAVAILABLE", "reset card store is not available")
	ErrResetCardInsufficient         = infraerrors.BadRequest("RESET_CARD_INSUFFICIENT", "no subscription reset cards remaining")
	ErrResetCardNothingToReset       = infraerrors.BadRequest("RESET_CARD_NOTHING_TO_RESET", "daily and weekly usage are already zero")
	ErrResetCardInvalidAmount        = infraerrors.BadRequest("RESET_CARD_INVALID_AMOUNT", "reset card adjustment must not be zero")
	ErrResetCardInvalidTarget        = infraerrors.BadRequest("RESET_CARD_INVALID_TARGET", "reset card target must be zero or greater")
	ErrResetCardSubscriptionMismatch = infraerrors.Forbidden("RESET_CARD_SUBSCRIPTION_MISMATCH", "subscription does not belong to this user")
)

func (s *SubscriptionService) resetCardStore() (SubscriptionResetCardStore, error) {
	store, ok := s.userSubRepo.(SubscriptionResetCardStore)
	if !ok {
		return nil, ErrResetCardStoreUnavailable
	}
	return store, nil
}

// GetResetCardBalance 返回用户剩余重置卡张数。
func (s *SubscriptionService) GetResetCardBalance(ctx context.Context, userID int64) (int, error) {
	store, err := s.resetCardStore()
	if err != nil {
		return 0, err
	}
	return store.GetResetCardBalance(ctx, userID)
}

// UseResetCard 消耗一张重置卡，把订阅的日、周用量清零。
//
// 语义与后台的「重置额度（日+周）」一致：日窗口锚点回到当天 0 点（保持每天 0 点
// 刷新的节奏），周窗口锚点重置为当前时刻；月度用量与月度窗口完全不动。
// 只有在日、周都还有用量时才允许消耗，避免误点浪费。
func (s *SubscriptionService) UseResetCard(ctx context.Context, userID, subscriptionID int64) (*UserSubscription, *SubscriptionProgress, int, error) {
	store, err := s.resetCardStore()
	if err != nil {
		return nil, nil, 0, err
	}

	sub, err := s.userSubRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, nil, 0, err
	}
	if sub.UserID != userID {
		return nil, nil, 0, ErrResetCardSubscriptionMismatch
	}

	now := s.now()
	if !sub.ExpiresAt.After(now) {
		return nil, nil, 0, ErrSubscriptionExpired
	}
	if sub.Status != SubscriptionStatusActive {
		return nil, nil, 0, ErrSubscriptionSuspended
	}
	// 日、周都已归零时没有可重置的内容 —— 不扣卡，直接告知。
	if sub.DailyUsageUSD <= 0 && sub.WeeklyUsageUSD <= 0 {
		return nil, nil, 0, ErrResetCardNothingToReset
	}

	remaining, ok, err := store.ConsumeResetCard(ctx, userID, subscriptionID)
	if err != nil {
		return nil, nil, 0, err
	}
	if !ok {
		return nil, nil, 0, ErrResetCardInsufficient
	}

	updated, err := s.AdminResetQuota(ctx, subscriptionID, true, true, false)
	if err != nil {
		// 扣卡成功但重置失败：必须退回，不能让用户白扣一张。
		if _, refundErr := store.AdjustResetCards(ctx, userID, 1, "refund", 0, "重置失败自动退回"); refundErr != nil {
			log.Printf("[SubscriptionResetCard] refund failed user_id=%d subscription_id=%d err=%v", userID, subscriptionID, refundErr)
		}
		return nil, nil, remaining, err
	}

	progress, progressErr := s.GetSubscriptionProgress(ctx, subscriptionID)
	if progressErr != nil {
		progress = nil
	} else {
		progress.ResetCardsRemaining = remaining
	}
	return updated, progress, remaining, nil
}

// AdjustResetCardsByAdmin 后台发放/扣减重置卡。
// setTo 非 nil 时表示直接设置到该值（内部换算成增量），否则按 delta 增减。
func (s *SubscriptionService) AdjustResetCardsByAdmin(
	ctx context.Context,
	userID int64,
	delta int,
	setTo *int,
	operatorID int64,
	note string,
) (int, error) {
	store, err := s.resetCardStore()
	if err != nil {
		return 0, err
	}

	if setTo != nil {
		if *setTo < 0 {
			return 0, ErrResetCardInvalidTarget
		}
		current, err := store.GetResetCardBalance(ctx, userID)
		if err != nil {
			return 0, err
		}
		if *setTo == current {
			return current, nil
		}
		return store.AdjustResetCards(ctx, userID, *setTo-current, "set", operatorID, note)
	}

	if delta == 0 {
		return 0, ErrResetCardInvalidAmount
	}
	action := "grant"
	if delta < 0 {
		action = "deduct"
	}
	return store.AdjustResetCards(ctx, userID, delta, action, operatorID, note)
}
