package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"entgo.io/ent/dialect"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentauditlog"
	"github.com/Wei-Shaw/sub2api/ent/promocode"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func subscriptionPromoCodeQuery(client *dbent.Client, code string) *dbent.PromoCodeQuery {
	query := client.PromoCode.Query().Where(promocode.CodeEqualFold(strings.TrimSpace(code)))
	// SQLite does not support SELECT ... FOR UPDATE. The max-use decision is
	// still protected by the conditional counter update below; PostgreSQL uses
	// the row lock to keep the subsequent audit write in the same transaction.
	if client != nil && client.Driver() != nil && client.Driver().Dialect() == dialect.Postgres {
		return query.ForUpdate()
	}
	return query
}

func claimSubscriptionPromoUse(ctx context.Context, tx *dbent.Tx, promoID int64) (bool, error) {
	if tx == nil || promoID <= 0 {
		return false, nil
	}
	updateSQL := "UPDATE promo_codes SET used_count = used_count + 1, updated_at = NOW() WHERE id = $1 AND (max_uses = 0 OR used_count < max_uses)"
	if tx.Client().Driver().Dialect() != dialect.Postgres {
		updateSQL = "UPDATE promo_codes SET used_count = used_count + 1, updated_at = CURRENT_TIMESTAMP WHERE id = $1 AND (max_uses = 0 OR used_count < max_uses)"
	}
	result, err := tx.ExecContext(ctx, updateSQL, promoID)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected > 0, err
}

const (
	subscriptionPromoReserved = "SUBSCRIPTION_PROMO_RESERVED"
	subscriptionPromoApplied  = "SUBSCRIPTION_PROMO_APPLIED"
	subscriptionPromoReleased = "SUBSCRIPTION_PROMO_RELEASED"
)

// reserveSubscriptionPromoCode claims one max-use slot in the same transaction
// that creates the payment order. Concurrent pending orders therefore cannot
// all observe and later consume the same remaining slot.
func (s *PaymentService) reserveSubscriptionPromoCode(ctx context.Context, tx *dbent.Tx, orderID int64, code string) error {
	if tx == nil || orderID <= 0 || strings.TrimSpace(code) == "" {
		return nil
	}
	txCtx := dbent.NewTxContext(ctx, tx)
	promo, err := subscriptionPromoCodeQuery(tx.Client(), code).Only(txCtx)
	if err != nil {
		return infraerrors.BadRequest("PROMO_CODE_INVALID", "promo code is invalid")
	}
	now := time.Now()
	if !promo.AppliesToSubscriptions || promo.Status != PromoCodeStatusActive ||
		(promo.StartsAt != nil && now.Before(*promo.StartsAt)) ||
		(promo.ExpiresAt != nil && now.After(*promo.ExpiresAt)) ||
		(promo.MaxUses > 0 && promo.UsedCount >= promo.MaxUses) {
		return infraerrors.BadRequest("PROMO_CODE_INVALID", "promo code is not available for this subscription")
	}
	claimed, err := claimSubscriptionPromoUse(txCtx, tx, promo.ID)
	if err != nil {
		return fmt.Errorf("reserve subscription promo code: %w", err)
	}
	if !claimed {
		return infraerrors.BadRequest("PROMO_CODE_INVALID", "promo code is not available for this subscription")
	}
	promo, err = tx.Client().PromoCode.Get(txCtx, promo.ID)
	if err != nil {
		return fmt.Errorf("reload subscription promo reservation: %w", err)
	}
	detail, _ := json.Marshal(map[string]any{
		"code":      strings.ToUpper(strings.TrimSpace(code)),
		"maxUses":   promo.MaxUses,
		"usedCount": promo.UsedCount,
	})
	if _, err := tx.Client().PaymentAuditLog.Create().
		SetOrderID(strconv.FormatInt(orderID, 10)).
		SetAction(subscriptionPromoReserved).
		SetDetail(string(detail)).
		SetOperator("system").
		Save(txCtx); err != nil {
		return fmt.Errorf("record subscription promo reservation: %w", err)
	}
	return nil
}

// releaseSubscriptionPromoReservation returns a reserved slot when a pending
// order is cancelled, expires, or fails before payment provider creation.
// Applied reservations are retained because the paid order consumed the
// discount.
func (s *PaymentService) releaseSubscriptionPromoReservation(ctx context.Context, order *dbent.PaymentOrder) error {
	if s == nil || s.entClient == nil || order == nil || order.ID <= 0 ||
		order.PromoCode == nil || strings.TrimSpace(*order.PromoCode) == "" {
		return nil
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)
	oid := strconv.FormatInt(order.ID, 10)
	reserved, err := tx.Client().PaymentAuditLog.Query().
		Where(paymentauditlog.OrderIDEQ(oid), paymentauditlog.ActionEQ(subscriptionPromoReserved)).
		Exist(txCtx)
	if err != nil || !reserved {
		if err != nil {
			return err
		}
		return tx.Commit()
	}
	applied, err := tx.Client().PaymentAuditLog.Query().
		Where(paymentauditlog.OrderIDEQ(oid), paymentauditlog.ActionEQ(subscriptionPromoApplied)).
		Exist(txCtx)
	if err != nil {
		return err
	}
	if applied {
		return tx.Commit()
	}
	released, err := tx.Client().PaymentAuditLog.Query().
		Where(paymentauditlog.OrderIDEQ(oid), paymentauditlog.ActionEQ(subscriptionPromoReleased)).
		Exist(txCtx)
	if err != nil {
		return err
	}
	if released {
		return tx.Commit()
	}
	promo, err := subscriptionPromoCodeQuery(tx.Client(), *order.PromoCode).Only(txCtx)
	if err != nil {
		// A deleted promo cannot be decremented, but the reservation still needs
		// a terminal audit so repeated cleanup is idempotent.
		if dbent.IsNotFound(err) {
			_, auditErr := tx.Client().PaymentAuditLog.Create().
				SetOrderID(oid).
				SetAction(subscriptionPromoReleased).
				SetDetail(`{"code":"deleted","decremented":false}`).
				SetOperator("system").
				Save(txCtx)
			if auditErr != nil {
				return auditErr
			}
			return tx.Commit()
		}
		return err
	}
	updateSQL := "UPDATE promo_codes SET used_count = used_count - 1, updated_at = NOW() WHERE id = $1 AND used_count > 0"
	if tx.Client().Driver().Dialect() != dialect.Postgres {
		updateSQL = "UPDATE promo_codes SET used_count = used_count - 1, updated_at = CURRENT_TIMESTAMP WHERE id = $1 AND used_count > 0"
	}
	result, err := tx.ExecContext(txCtx, updateSQL, promo.ID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	detail, _ := json.Marshal(map[string]any{
		"code":        strings.ToUpper(strings.TrimSpace(*order.PromoCode)),
		"decremented": affected > 0,
	})
	if _, err := tx.Client().PaymentAuditLog.Create().
		SetOrderID(oid).
		SetAction(subscriptionPromoReleased).
		SetDetail(string(detail)).
		SetOperator("system").
		Save(txCtx); err != nil {
		return err
	}
	return tx.Commit()
}
