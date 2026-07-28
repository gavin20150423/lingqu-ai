package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"entgo.io/ent/dialect"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

func parseBoolWithDefault(raw string, fallback bool) bool {
	value, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	return value
}

type serviceSQLQueryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type serviceSQLExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type pointsAdjustmentInput struct {
	UserID         int64
	Delta          float64
	Reason         string
	RefType        string
	RefID          int64
	OperatorUserID int64
	Metadata       map[string]any
	ClampZero      bool
}

func applyPointsAdjustmentInTx(ctx context.Context, tx *dbent.Tx, in pointsAdjustmentInput) error {
	if tx == nil {
		return errors.New("points adjustment requires transaction")
	}
	queryer, queryOK := tx.Driver().(serviceSQLQueryer)
	execer, execOK := tx.Driver().(serviceSQLExecer)
	if !queryOK || !execOK {
		return errors.New("points adjustment requires SQL driver access")
	}
	balanceBefore, err := currentPointsBalanceWithQueryer(ctx, queryer, in.UserID, tx.Driver().Dialect() == dialect.Postgres)
	if err != nil {
		return err
	}
	delta := in.Delta
	if in.ClampZero && delta < 0 && balanceBefore+delta < 0 {
		delta = -balanceBefore
	}
	balanceAfter := balanceBefore + delta
	if balanceAfter < -1e-9 {
		return infraerrors.BadRequest("POINTS_BALANCE_NEGATIVE", "points balance cannot be negative")
	}
	if balanceAfter < 0 {
		balanceAfter = 0
	}

	dialectName := tx.Driver().Dialect()
	amount := delta
	direction := "credit"
	if amount < 0 {
		direction = "debit"
		amount = -amount
	}
	amountValue := decimal.NewFromFloat(amount).Round(10).StringFixed(10)
	beforeValue := decimal.NewFromFloat(balanceBefore).Round(10).StringFixed(10)
	afterValue := decimal.NewFromFloat(balanceAfter).Round(10).StringFixed(10)
	updateQuery := `UPDATE users SET points_balance = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2 AND deleted_at IS NULL`
	if dialectName == dialect.Postgres {
		updateQuery = `UPDATE users SET points_balance = $1::numeric, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL`
	}
	if _, err := execer.ExecContext(ctx, updateQuery, afterValue, in.UserID); err != nil {
		return err
	}
	if amount == 0 {
		return nil
	}
	metadata, err := json.Marshal(in.Metadata)
	if err != nil {
		return err
	}
	var refID, operatorID any
	if in.RefID > 0 {
		refID = in.RefID
	}
	if in.OperatorUserID > 0 {
		operatorID = in.OperatorUserID
	}
	insertQuery := `INSERT INTO points_ledger (user_id, direction, amount, reason, ref_type, ref_id, balance_before, balance_after, operator_user_id, metadata) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT DO NOTHING`
	if dialectName == dialect.Postgres {
		insertQuery = `INSERT INTO points_ledger (user_id, direction, amount, reason, ref_type, ref_id, balance_before, balance_after, operator_user_id, metadata) VALUES ($1,$2,$3::numeric,$4,$5,$6,$7::numeric,$8::numeric,$9,$10::jsonb) ON CONFLICT DO NOTHING`
	}
	_, err = execer.ExecContext(ctx, insertQuery, in.UserID, direction, amountValue, strings.TrimSpace(in.Reason), strings.TrimSpace(in.RefType), refID, beforeValue, afterValue, operatorID, string(metadata))
	return err
}

func currentPointsBalanceWithQueryer(ctx context.Context, queryer serviceSQLQueryer, userID int64, forUpdate bool) (float64, error) {
	query := `SELECT points_balance FROM users WHERE id = $1 AND deleted_at IS NULL`
	if forUpdate {
		query += " FOR UPDATE"
	}
	rows, err := queryer.QueryContext(ctx, query, userID)
	if err != nil {
		return 0, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return 0, ErrUserNotFound
	}
	var balance float64
	if err := rows.Scan(&balance); err != nil {
		return 0, err
	}
	return balance, rows.Err()
}

type loadFactorCreditsAdjustmentInput struct {
	UserID         int64
	Delta          int
	Reason         string
	RefType        string
	RefID          int64
	BalanceBefore  int
	BalanceAfter   int
	OperatorUserID int64
	Metadata       map[string]any
}

func currentLoadFactorCreditsBalanceInTx(ctx context.Context, tx *dbent.Tx, userID int64) (int, error) {
	if tx == nil {
		return 0, errors.New("load factor credits lookup requires transaction")
	}
	queryer, ok := tx.Driver().(serviceSQLQueryer)
	if !ok {
		return 0, errors.New("load factor credits lookup requires SQL driver access")
	}
	query := `SELECT load_factor_credits_balance FROM users WHERE id = $1 AND deleted_at IS NULL`
	if tx.Driver().Dialect() == dialect.Postgres {
		query += " FOR UPDATE"
	}
	rows, err := queryer.QueryContext(ctx, query, userID)
	if err != nil {
		return 0, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return 0, ErrUserNotFound
	}
	var balance int
	if err := rows.Scan(&balance); err != nil {
		return 0, err
	}
	return balance, rows.Err()
}

func applyLoadFactorCreditsAdjustmentInTx(ctx context.Context, tx *dbent.Tx, in loadFactorCreditsAdjustmentInput) error {
	if tx == nil {
		return errors.New("load factor credits adjustment requires transaction")
	}
	execer, ok := tx.Driver().(serviceSQLExecer)
	if !ok {
		return errors.New("load factor credits adjustment requires SQL driver access")
	}
	dialectName := tx.Driver().Dialect()
	updateQuery := `UPDATE users SET load_factor_credits_balance = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2 AND deleted_at IS NULL`
	if dialectName == dialect.Postgres {
		updateQuery = `UPDATE users SET load_factor_credits_balance = $1, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL`
	}
	if _, err := execer.ExecContext(ctx, updateQuery, in.BalanceAfter, in.UserID); err != nil {
		return err
	}
	direction, amount := "credit", in.Delta
	if amount < 0 {
		direction, amount = "debit", -amount
	}
	metadata, err := json.Marshal(in.Metadata)
	if err != nil {
		return err
	}
	var refID, operatorID any
	if in.RefID > 0 {
		refID = in.RefID
	}
	if in.OperatorUserID > 0 {
		operatorID = in.OperatorUserID
	}
	insertQuery := `INSERT INTO user_load_factor_ledger (user_id, account_id, direction, amount, reason, ref_type, ref_id, balance_before, balance_after, operator_user_id, metadata) VALUES ($1,NULL,$2,$3,$4,$5,$6,$7,$8,$9,$10)`
	if dialectName == dialect.Postgres {
		insertQuery = `INSERT INTO user_load_factor_ledger (user_id, account_id, direction, amount, reason, ref_type, ref_id, balance_before, balance_after, operator_user_id, metadata) VALUES ($1,NULL,$2,$3,$4,$5,$6,$7,$8,$9,$10::jsonb)`
	}
	_, err = execer.ExecContext(ctx, insertQuery, in.UserID, direction, amount, strings.TrimSpace(in.Reason), strings.TrimSpace(in.RefType), refID, in.BalanceBefore, in.BalanceAfter, operatorID, string(metadata))
	return err
}
