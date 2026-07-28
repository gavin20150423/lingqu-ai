package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/shopspring/decimal"
)

func normalizeLoadFactorPaidCeiling(value int) int {
	if value < service.OwnedPersonalDefaultLoadFactor {
		return service.OwnedPersonalDefaultLoadFactor
	}
	return value
}

func translateAccountPersistenceError(err error, notFound *infraerrors.ApplicationError) error {
	return translatePersistenceError(err, notFound, nil)
}

func decimalFromFloat(value float64) decimal.Decimal {
	if value <= 0 {
		return decimal.Zero
	}
	return decimal.NewFromFloat(value).Round(10)
}

func decimalFromSignedFloat(value float64) decimal.Decimal {
	return decimal.NewFromFloat(value).Round(10)
}

func nullablePositiveInt64(value int64) any {
	if value <= 0 {
		return nil
	}
	return value
}

func nullablePtrInt64(value *int64) any {
	if value == nil || *value <= 0 {
		return nil
	}
	return *value
}

func normalizeAccountShareModeRatio(value, fallback float64) decimal.Decimal {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		value = fallback
	}
	ratio := decimal.NewFromFloat(value)
	if ratio.IsNegative() {
		return decimal.Zero
	}
	if ratio.GreaterThan(decimal.NewFromInt(1)) {
		return decimal.NewFromInt(1)
	}
	return ratio
}

func accountShareModeSettlementRatios(ownerRaw, platformRaw float64) (decimal.Decimal, decimal.Decimal) {
	ownerRatio := normalizeAccountShareModeRatio(ownerRaw, service.AccountShareModeDefaultOwnerShareRatio)
	platformRatio := normalizeAccountShareModeRatio(platformRaw, service.AccountShareModeDefaultPlatformShareRatio)
	if ownerRatio.Add(platformRatio).GreaterThan(decimal.NewFromInt(1)) {
		platformRatio = decimal.NewFromInt(1).Sub(ownerRatio)
		if platformRatio.IsNegative() {
			platformRatio = decimal.Zero
		}
	}
	return ownerRatio, platformRatio
}

func creditUsageBillingBalance(ctx context.Context, tx *sql.Tx, userID int64, amount decimal.Decimal) (float64, error) {
	var balance float64
	err := tx.QueryRowContext(ctx, `
		UPDATE users
		SET balance = balance + $1::numeric, updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
		RETURNING balance
	`, amount.StringFixed(10), userID).Scan(&balance)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, service.ErrUserNotFound
	}
	return balance, err
}

type userBalanceLedgerInput struct {
	UserID          int64
	Direction       string
	Amount          decimal.Decimal
	Reason          string
	RefType         string
	RefID           any
	BalanceAfter    decimal.Decimal
	Metadata        map[string]any
	RequireInserted bool
}

func insertUserBalanceLedger(ctx context.Context, tx *sql.Tx, input userBalanceLedgerInput) error {
	if input.UserID <= 0 || input.Amount.IsNegative() {
		if input.RequireInserted {
			return fmt.Errorf("user balance ledger insert rejected: user_id=%d reason=%s", input.UserID, input.Reason)
		}
		return nil
	}
	metadata := input.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	rawMetadata, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO user_balance_ledger (
			user_id, direction, amount, reason, ref_type, ref_id, balance_after, metadata
		) VALUES ($1, $2, $3::numeric, $4, $5, $6, $7::numeric, $8::jsonb)
		ON CONFLICT DO NOTHING
	`, input.UserID, input.Direction, input.Amount.StringFixed(10), input.Reason, input.RefType, input.RefID, input.BalanceAfter.StringFixed(10), string(rawMetadata))
	if err != nil {
		return err
	}
	if !input.RequireInserted {
		return nil
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return fmt.Errorf("user balance ledger insert skipped: user_id=%d reason=%s", input.UserID, input.Reason)
	}
	return nil
}

func appendUsageBillingCreditUser(result *service.UsageBillingApplyResult, userID int64) {
	if result == nil || userID <= 0 {
		return
	}
	for _, existing := range result.BalanceCreditUserIDs {
		if existing == userID {
			return
		}
	}
	result.BalanceCreditUserIDs = append(result.BalanceCreditUserIDs, userID)
}
