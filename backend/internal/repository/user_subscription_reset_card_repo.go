package repository

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// defaultSubscriptionResetCards 是「有订阅的用户」首次读取时的默认张数。
// 没有订阅的用户默认 0 张（不能凭空白拿），管理员仍可单独发放。
const defaultSubscriptionResetCards = 1

// 订阅重置卡存取挂在现有订阅仓储上。
//
// 刻意用小接口 + 运行期断言，而不是扩张 service.UserSubscriptionRepository：
// 后者被大量测试替身实现，加方法会让所有替身一起编译失败；重置卡属于新增能力，
// 生产仓储实现它即可，替身缺失时服务层返回明确错误而不是 panic。
var _ service.SubscriptionResetCardStore = (*userSubscriptionRepository)(nil)

func (r *userSubscriptionRepository) resetCardExecutor(ctx context.Context) (sqlQueryExecutor, error) {
	exec := sqlExecutorFromEntClient(clientFromContext(ctx, r.client))
	if exec == nil {
		return nil, fmt.Errorf("reset card sql executor is not configured")
	}
	return exec, nil
}

// ensureResetCardRow 在缺失时补一行：有订阅 = 默认赠送的 1 张，否则 0 张。
// 用 INSERT ... SELECT 让「是否赠送」与插入在同一条语句里完成，避免并发下
// 读到半截状态。
func ensureResetCardRow(ctx context.Context, exec sqlQueryExecutor, userID int64) error {
	if _, err := exec.ExecContext(ctx, `
		INSERT INTO user_subscription_reset_cards (user_id, balance)
		SELECT $1, CASE WHEN EXISTS (
			SELECT 1 FROM user_subscriptions s
			WHERE s.user_id = $1 AND s.deleted_at IS NULL
		) THEN $2 ELSE 0 END
		ON CONFLICT (user_id) DO NOTHING`, userID, defaultSubscriptionResetCards); err != nil {
		return fmt.Errorf("ensure reset card row: %w", err)
	}
	return nil
}

// GetResetCardBalance 返回剩余重置卡张数。
// 没有行时按「是否有订阅」给出默认值：有订阅 1 张，没有订阅 0 张（只读，不写行）。
func (r *userSubscriptionRepository) GetResetCardBalance(ctx context.Context, userID int64) (int, error) {
	exec, err := r.resetCardExecutor(ctx)
	if err != nil {
		return 0, err
	}
	rows, err := exec.QueryContext(ctx,
		`SELECT balance FROM user_subscription_reset_cards WHERE user_id = $1`, userID)
	if err != nil {
		return 0, fmt.Errorf("query reset card balance: %w", err)
	}
	if rows.Next() {
		var balance int
		scanErr := rows.Scan(&balance)
		iterErr := rows.Err()
		_ = rows.Close()
		if scanErr != nil {
			return 0, fmt.Errorf("scan reset card balance: %w", scanErr)
		}
		if iterErr != nil {
			return 0, fmt.Errorf("iterate reset card balance: %w", iterErr)
		}
		return balance, nil
	}
	iterErr := rows.Err()
	_ = rows.Close()
	if iterErr != nil {
		return 0, fmt.Errorf("iterate reset card balance: %w", iterErr)
	}

	defaultRows, err := exec.QueryContext(ctx, `
		SELECT CASE WHEN EXISTS (
			SELECT 1 FROM user_subscriptions s
			WHERE s.user_id = $1 AND s.deleted_at IS NULL
		) THEN $2 ELSE 0 END`, userID, defaultSubscriptionResetCards)
	if err != nil {
		return 0, fmt.Errorf("query reset card default: %w", err)
	}
	defer func() { _ = defaultRows.Close() }()
	if defaultRows.Next() {
		var balance int
		if scanErr := defaultRows.Scan(&balance); scanErr != nil {
			return 0, fmt.Errorf("scan reset card default: %w", scanErr)
		}
		return balance, nil
	}
	if err := defaultRows.Err(); err != nil {
		return 0, fmt.Errorf("iterate reset card default: %w", err)
	}
	return 0, nil
}

// ConsumeResetCard 原子扣减一张卡：仅当 balance > 0 时成功，返回扣减后余额。
// ok=false 表示卡不足（不是错误）。
func (r *userSubscriptionRepository) ConsumeResetCard(ctx context.Context, userID, subscriptionID int64) (int, bool, error) {
	exec, err := r.resetCardExecutor(ctx)
	if err != nil {
		return 0, false, err
	}
	if err := ensureResetCardRow(ctx, exec, userID); err != nil {
		return 0, false, err
	}

	rows, err := exec.QueryContext(ctx, `
		UPDATE user_subscription_reset_cards
		SET balance = balance - 1,
		    updated_at = NOW()
		WHERE user_id = $1 AND balance > 0
		RETURNING balance`, userID)
	if err != nil {
		return 0, false, fmt.Errorf("consume reset card: %w", err)
	}
	var remaining int
	hasRow := false
	if rows.Next() {
		hasRow = true
		if scanErr := rows.Scan(&remaining); scanErr != nil {
			_ = rows.Close()
			return 0, false, fmt.Errorf("scan consumed reset card: %w", scanErr)
		}
	}
	iterErr := rows.Err()
	_ = rows.Close()
	if iterErr != nil {
		return 0, false, fmt.Errorf("iterate consumed reset card: %w", iterErr)
	}
	if !hasRow {
		return 0, false, nil
	}

	if _, err := exec.ExecContext(ctx, `
		INSERT INTO user_subscription_reset_card_logs
			(user_id, subscription_id, action, delta, balance_after, note)
		VALUES ($1, $2, 'consume', -1, $3, NULLIF($4, ''))`,
		userID, subscriptionID, remaining, "用户自助重置（日+周）"); err != nil {
		return remaining, true, fmt.Errorf("log reset card consume: %w", err)
	}
	return remaining, true, nil
}

// AdjustResetCards 发放 / 扣减重置卡，并写审计日志。
// action 取值 grant / set / refund（由服务层决定语义），balance 不会低于 0。
func (r *userSubscriptionRepository) AdjustResetCards(
	ctx context.Context,
	userID int64,
	delta int,
	action string,
	operatorID int64,
	note string,
) (int, error) {
	exec, err := r.resetCardExecutor(ctx)
	if err != nil {
		return 0, err
	}
	if err := ensureResetCardRow(ctx, exec, userID); err != nil {
		return 0, err
	}

	rows, err := exec.QueryContext(ctx, `
		UPDATE user_subscription_reset_cards
		SET balance = GREATEST(balance + $2, 0),
		    updated_at = NOW()
		WHERE user_id = $1
		RETURNING balance`, userID, delta)
	if err != nil {
		return 0, fmt.Errorf("adjust reset cards: %w", err)
	}
	var balance int
	hasRow := false
	if rows.Next() {
		hasRow = true
		if scanErr := rows.Scan(&balance); scanErr != nil {
			_ = rows.Close()
			return 0, fmt.Errorf("scan adjusted reset cards: %w", scanErr)
		}
	}
	iterErr := rows.Err()
	_ = rows.Close()
	if iterErr != nil {
		return 0, fmt.Errorf("iterate adjusted reset cards: %w", iterErr)
	}
	if !hasRow {
		return 0, fmt.Errorf("reset card row for user %d not found", userID)
	}

	if _, err := exec.ExecContext(ctx, `
		INSERT INTO user_subscription_reset_card_logs
			(user_id, subscription_id, action, delta, balance_after, operator_id, note)
		VALUES ($1, NULL, $2, $3, $4, NULLIF($5, 0), NULLIF($6, ''))`,
		userID, action, delta, balance, operatorID, note); err != nil {
		return balance, fmt.Errorf("log reset card adjust: %w", err)
	}
	return balance, nil
}
