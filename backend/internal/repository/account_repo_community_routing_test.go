package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestResolveCommunityAccountIDUsesReservationOrderAndExclusions(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id,idle_expires_at FROM community_memberships`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "idle_expires_at"}))
	mock.ExpectQuery(`SELECT w\.id,w\.membership_id`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "membership_id", "payer_user_id", "owner_user_id", "active_seconds", "request_spend", "hourly_fee_precharged", "commission_rate", "hourly_minimum_spend"}))
	mock.ExpectQuery(`(?s)SELECT ca\.scheduler_account_id,m\.id,m\.status,m\.listing_id.*ORDER BY CASE`).
		WithArgs(int64(41), "gpt-5.4").
		WillReturnRows(sqlmock.NewRows([]string{"scheduler_account_id", "id", "status", "listing_id", "seat_limit", "idle_timeout_minutes"}).
			AddRow(int64(1001), int64(71), "active", int64(81), 2, 10).
			AddRow(int64(1002), int64(72), "reserved", int64(82), 2, 15))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM community_memberships`).
		WithArgs(int64(82), int64(72)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectExec(`UPDATE community_memberships SET status='reserved'`).
		WithArgs(int64(41), int64(72)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE community_memberships SET status='active'`).
		WithArgs(int64(72), 15).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT m\.member_user_id,l\.owner_user_id,l\.hourly_price`).
		WithArgs(int64(72)).
		WillReturnRows(sqlmock.NewRows([]string{"member_user_id", "owner_user_id", "hourly_price", "commission_rate", "activated_at", "billed_until"}).
			AddRow(int64(7), int64(9), 0.0, 10.0, time.Now(), nil))
	mock.ExpectCommit()

	repo := newAccountRepositoryWithSQL(nil, db, nil)
	accountID, found, err := repo.ResolveCommunityAccountID(context.Background(), 41, "gpt-5.4", map[int64]struct{}{1001: {}})

	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, int64(1002), accountID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestResolveCommunityAccountIDReturnsNoMatchWithoutMutation(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id,idle_expires_at FROM community_memberships`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "idle_expires_at"}))
	mock.ExpectQuery(`SELECT w\.id,w\.membership_id`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "membership_id", "payer_user_id", "owner_user_id", "active_seconds", "request_spend", "hourly_fee_precharged", "commission_rate", "hourly_minimum_spend"}))
	mock.ExpectQuery(`(?s)SELECT ca\.scheduler_account_id,m\.id,m\.status,m\.listing_id.*ORDER BY CASE`).
		WithArgs(int64(41), "claude-opus-4-6").
		WillReturnRows(sqlmock.NewRows([]string{"scheduler_account_id", "id", "status", "listing_id", "seat_limit", "idle_timeout_minutes"}))
	mock.ExpectCommit()

	repo := newAccountRepositoryWithSQL(nil, db, nil)
	accountID, found, err := repo.ResolveCommunityAccountID(context.Background(), 41, "claude-opus-4-6", nil)

	require.NoError(t, err)
	require.False(t, found)
	require.Zero(t, accountID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCommunityUsageMultiplierUsesActiveKeyBinding(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery(`SELECT l\.usage_multiplier`).
		WithArgs(int64(41), int64(1002)).
		WillReturnRows(sqlmock.NewRows([]string{"usage_multiplier"}).AddRow(0.35))

	repo := newAccountRepositoryWithSQL(nil, db, nil)
	multiplier, found, err := repo.CommunityUsageMultiplier(context.Background(), 41, 1002)

	require.NoError(t, err)
	require.True(t, found)
	require.InDelta(t, 0.35, multiplier, 1e-12)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRecordCommunityRequestSettlementCreditsOwnerOnce(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT m\.id,l\.id,m\.member_user_id,l\.owner_user_id,l\.commission_rate`).
		WithArgs(int64(41), int64(1002)).
		WillReturnRows(sqlmock.NewRows([]string{"membership_id", "listing_id", "payer_user_id", "owner_user_id", "commission_rate"}).
			AddRow(int64(72), int64(82), int64(7), int64(9), 10.0))
	mock.ExpectQuery(`INSERT INTO community_settlements`).
		WithArgs(int64(82), int64(72), int64(7), int64(9), 2.0, 10.0, 0.2, 1.8, "req-1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(91)))
	mock.ExpectExec(`UPDATE users SET balance=balance\+\$1`).
		WithArgs(1.8, int64(9)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE community_billing_windows SET request_spend=request_spend\+\$1`).
		WithArgs(2.0, int64(72)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := newAccountRepositoryWithSQL(nil, db, nil)
	applied, err := repo.RecordCommunityRequestSettlement(context.Background(), 41, 1002, "req-1", 2)

	require.NoError(t, err)
	require.True(t, applied)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRecordCommunityRequestSettlementDuplicateIsNoop(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT m\.id,l\.id,m\.member_user_id,l\.owner_user_id,l\.commission_rate`).
		WithArgs(int64(41), int64(1002)).
		WillReturnRows(sqlmock.NewRows([]string{"membership_id", "listing_id", "payer_user_id", "owner_user_id", "commission_rate"}).
			AddRow(int64(72), int64(82), int64(7), int64(9), 10.0))
	mock.ExpectQuery(`INSERT INTO community_settlements`).
		WithArgs(int64(82), int64(72), int64(7), int64(9), 2.0, 10.0, 0.2, 1.8, "req-1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectCommit()

	repo := newAccountRepositoryWithSQL(nil, db, nil)
	applied, err := repo.RecordCommunityRequestSettlement(context.Background(), 41, 1002, "req-1", 2)

	require.NoError(t, err)
	require.False(t, applied)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAccrueCommunityHourlyFeePrechargesByMinute(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	activatedAt := time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT m\.member_user_id,l\.owner_user_id,l\.hourly_price`).
		WithArgs(int64(72)).
		WillReturnRows(sqlmock.NewRows([]string{"member_user_id", "owner_user_id", "hourly_price", "commission_rate", "activated_at", "billed_until"}).
			AddRow(int64(7), int64(9), 0.60, 10.0, activatedAt, nil))
	mock.ExpectExec(`UPDATE users SET balance=balance-\$1`).
		WithArgs(0.01, int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO community_billing_windows`).
		WithArgs(int64(72), activatedAt, activatedAt.Add(time.Hour), int64(60), 0.01).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE community_memberships SET billed_until=\$2`).
		WithArgs(int64(72), activatedAt.Add(time.Minute)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	require.NoError(t, accrueCommunityHourlyFeeTx(context.Background(), tx, 72, activatedAt.Add(time.Minute)))
	require.NoError(t, tx.Commit())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSweepCommunityBillingRefundsHourlyFeeWhenProratedMinimumMet(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	now := time.Date(2026, 7, 18, 11, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id,idle_expires_at FROM community_memberships`).
		WithArgs(now).
		WillReturnRows(sqlmock.NewRows([]string{"id", "idle_expires_at"}))
	mock.ExpectQuery(`SELECT w\.id,w\.membership_id`).
		WithArgs(now).
		WillReturnRows(sqlmock.NewRows([]string{"id", "membership_id", "payer_user_id", "owner_user_id", "active_seconds", "request_spend", "hourly_fee_precharged", "commission_rate", "hourly_minimum_spend"}).
			AddRow(int64(101), int64(72), int64(7), int64(9), int64(300), 0.03, 0.05, 10.0, 0.30))
	mock.ExpectExec(`UPDATE users SET balance=balance\+\$1`).
		WithArgs(0.05, int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE community_billing_windows SET status='settled',hourly_fee_refunded=\$2`).
		WithArgs(int64(101), 0.05).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	require.NoError(t, sweepCommunityBillingTx(context.Background(), tx, now))
	require.NoError(t, tx.Commit())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSweepCommunityBillingExpiresInsufficientMemberAndContinues(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	now := time.Date(2026, 7, 18, 11, 0, 0, 0, time.UTC)
	cutoff := now.Add(-time.Minute)
	activatedAt := now.Add(-10 * time.Minute)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id,idle_expires_at FROM community_memberships`).
		WithArgs(now).
		WillReturnRows(sqlmock.NewRows([]string{"id", "idle_expires_at"}).AddRow(int64(72), cutoff))
	mock.ExpectQuery(`SELECT m\.member_user_id,l\.owner_user_id,l\.hourly_price`).
		WithArgs(int64(72)).
		WillReturnRows(sqlmock.NewRows([]string{"member_user_id", "owner_user_id", "hourly_price", "commission_rate", "activated_at", "billed_until"}).
			AddRow(int64(7), int64(9), 0.60, 10.0, activatedAt, nil))
	mock.ExpectExec(`UPDATE users SET balance=balance-\$1`).
		WithArgs(0.09, int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`UPDATE community_memberships SET status='expired',ended_at=COALESCE`).
		WithArgs(int64(72)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT w\.id,w\.membership_id`).
		WithArgs(now).
		WillReturnRows(sqlmock.NewRows([]string{"id", "membership_id", "payer_user_id", "owner_user_id", "active_seconds", "request_spend", "hourly_fee_precharged", "commission_rate", "hourly_minimum_spend"}))
	mock.ExpectCommit()

	tx, err := db.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	require.NoError(t, sweepCommunityBillingTx(context.Background(), tx, now))
	require.NoError(t, tx.Commit())
	require.NoError(t, mock.ExpectationsWereMet())
}
