// Code extracted from PIXEL-API/PixelAPI for account-sharing compatibility.
package repository

import (
	"context"
	"database/sql"

	"errors"

	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/shopspring/decimal"
)

func applyAccountShareModeSettlement(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand, usageLogID int64, result *service.UsageBillingApplyResult) error {
	if cmd == nil || cmd.AccountShareModeSettlement == nil {
		return nil
	}
	snapshot := cmd.AccountShareModeSettlement
	if snapshot.OwnerUserID <= 0 || snapshot.ConsumerUserID <= 0 || snapshot.OwnerUserID == snapshot.ConsumerUserID {
		return nil
	}
	totalCharge := decimalFromFloat(snapshot.TotalCharge)
	if totalCharge.IsZero() || totalCharge.IsNegative() {
		return nil
	}
	ownerRatio, platformRatio := accountShareModeSettlementRatios(snapshot.OwnerShareRatio, snapshot.PlatformShareRatio)
	ownerCredit := totalCharge.Mul(ownerRatio).Round(10)
	platformCredit := totalCharge.Mul(platformRatio).Round(10)
	periodStartedAt, periodEndedAt := accountShareModeUsageRequestPeriod(cmd, snapshot)
	inserted, err := insertAccountShareModeSettlement(ctx, tx, cmd, usageLogID, ownerCredit, platformCredit, periodStartedAt, periodEndedAt)
	if err != nil || !inserted {
		return err
	}
	if err := updateAccountShareWaiverProgressCache(ctx, tx, snapshot, totalCharge, periodStartedAt, periodEndedAt); err != nil {
		return err
	}
	if ownerCredit.IsZero() {
		return nil
	}
	newBalance, err := creditUsageBillingBalance(ctx, tx, snapshot.OwnerUserID, ownerCredit)
	if err != nil {
		return err
	}
	if err := insertUserBalanceLedger(ctx, tx, userBalanceLedgerInput{
		UserID:       snapshot.OwnerUserID,
		Direction:    "credit",
		Amount:       ownerCredit,
		Reason:       "account_share_mode_income",
		RefType:      "usage_log",
		RefID:        nullablePositiveInt64(usageLogID),
		BalanceAfter: decimalFromFloat(newBalance),
		Metadata: map[string]any{
			"request_id":       cmd.RequestID,
			"api_key_id":       snapshot.APIKeyID,
			"account_id":       snapshot.AccountID,
			"listing_id":       snapshot.ListingID,
			"membership_id":    snapshot.MembershipID,
			"consumer_user_id": snapshot.ConsumerUserID,
			"total_charge":     totalCharge.String(),
			"owner_ratio":      ownerRatio.String(),
			"platform_ratio":   platformRatio.String(),
		},
	}); err != nil {
		return err
	}
	appendUsageBillingCreditUser(result, snapshot.OwnerUserID)
	return nil
}

func insertAccountShareModeSettlement(ctx context.Context, tx *sql.Tx, cmd *service.UsageBillingCommand, usageLogID int64, ownerCredit, platformCredit decimal.Decimal, periodStartedAt, periodEndedAt time.Time) (bool, error) {
	var snapshot *service.AccountShareModeBillingSnapshot
	if cmd != nil {
		snapshot = cmd.AccountShareModeSettlement
	}
	if snapshot == nil {
		return false, nil
	}
	var id int64
	err := tx.QueryRowContext(ctx, `
		INSERT INTO account_share_mode_settlement_entries (
			usage_log_id,
			membership_id,
			listing_id,
			account_id,
			owner_user_id,
			consumer_user_id,
			api_key_id,
			base_charge,
			hourly_charge,
			total_charge,
			owner_credit,
			platform_credit,
			rate_multiplier_snapshot,
			hourly_rate_snapshot,
			owner_share_ratio_snapshot,
			platform_share_ratio_snapshot,
			duration_ms,
			period_started_at,
			period_ended_at,
			created_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17, $18, $19,
			NOW()
		)
		ON CONFLICT (usage_log_id) DO NOTHING
		RETURNING id
	`,
		nullablePositiveInt64(usageLogID),
		snapshot.MembershipID,
		snapshot.ListingID,
		snapshot.AccountID,
		snapshot.OwnerUserID,
		snapshot.ConsumerUserID,
		snapshot.APIKeyID,
		decimalFromFloat(snapshot.BaseCharge).StringFixed(10),
		decimalFromFloat(snapshot.HourlyCharge).StringFixed(10),
		decimalFromFloat(snapshot.TotalCharge).StringFixed(10),
		ownerCredit.StringFixed(10),
		platformCredit.StringFixed(10),
		decimalFromFloat(snapshot.RateMultiplier).StringFixed(4),
		decimalFromFloat(snapshot.HourlyRate).StringFixed(8),
		accountShareModeOwnerRatioString(snapshot),
		accountShareModePlatformRatioString(snapshot),
		snapshot.DurationMs,
		periodStartedAt,
		periodEndedAt,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return id > 0, nil
}

func updateAccountShareWaiverProgressCache(ctx context.Context, tx *sql.Tx, snapshot *service.AccountShareModeBillingSnapshot, totalCharge decimal.Decimal, periodStartedAt, periodEndedAt time.Time) error {
	if tx == nil || snapshot == nil || snapshot.MembershipID <= 0 || totalCharge.LessThanOrEqual(decimal.Zero) {
		return nil
	}
	periodStartedAt = periodStartedAt.UTC()
	periodEndedAt = periodEndedAt.UTC()
	if periodStartedAt.After(periodEndedAt) {
		periodStartedAt = periodEndedAt
	}

	var joinedAt time.Time
	err := tx.QueryRowContext(ctx, `
		SELECT joined_at
		FROM account_share_memberships
		WHERE id = $1
			AND status = $2
			AND deleted_at IS NULL
		FOR UPDATE
	`, snapshot.MembershipID, service.AccountShareMembershipStatusActive).Scan(&joinedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	windowStart := accountShareModeWaiverWindowStartAt(joinedAt, periodEndedAt)
	windowEnd := accountShareModeWaiverWindowEnd(windowStart)

	overlapCharge := accountShareModeWindowOverlapCharge(totalCharge, periodStartedAt, periodEndedAt, windowStart, windowEnd)
	if overlapCharge.LessThanOrEqual(decimal.Zero) {
		return nil
	}
	periodOccurredAt := periodEndedAt
	if periodOccurredAt.IsZero() {
		periodOccurredAt = time.Now().UTC()
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE account_share_memberships
		SET waiver_window_started_at = $2,
			waiver_window_usage_amount = CASE
				WHEN waiver_window_started_at IS DISTINCT FROM $2::timestamptz THEN $3::numeric
				ELSE waiver_window_usage_amount + $3::numeric
			END,
			waiver_window_request_count = CASE
				WHEN waiver_window_started_at IS DISTINCT FROM $2::timestamptz THEN 1
				ELSE waiver_window_request_count + 1
			END,
			waiver_window_last_request_at = CASE
				WHEN waiver_window_started_at IS DISTINCT FROM $2::timestamptz THEN $4::timestamptz
				ELSE GREATEST(COALESCE(waiver_window_last_request_at, $4::timestamptz), $4::timestamptz)
			END,
			updated_at = NOW()
		WHERE id = $1
			AND status = $5
			AND deleted_at IS NULL
	`, snapshot.MembershipID, windowStart, overlapCharge.StringFixed(10), periodOccurredAt, service.AccountShareMembershipStatusActive)
	return err
}

func accountShareModeWaiverWindowStartAt(joinedAt time.Time, at time.Time) time.Time {
	joinedAt = joinedAt.UTC()
	at = at.UTC()
	windowMax := service.AccountShareModeSeatWaiverWindowMax
	if windowMax <= 0 {
		windowMax = time.Hour
	}
	if at.Before(joinedAt) || !at.After(joinedAt) {
		return joinedAt
	}
	elapsed := at.Sub(joinedAt)
	windows := elapsed / windowMax
	return joinedAt.Add(windows * windowMax).UTC()
}

func accountShareModeWaiverWindowEnd(windowStart time.Time) time.Time {
	windowMax := service.AccountShareModeSeatWaiverWindowMax
	if windowMax <= 0 {
		windowMax = time.Hour
	}
	return windowStart.UTC().Add(windowMax).UTC()
}

func accountShareModeWindowOverlapCharge(totalCharge decimal.Decimal, periodStartedAt, periodEndedAt, windowStart, windowEnd time.Time) decimal.Decimal {
	if totalCharge.LessThanOrEqual(decimal.Zero) || !windowEnd.After(windowStart) {
		return decimal.Zero
	}
	periodStartedAt = periodStartedAt.UTC()
	periodEndedAt = periodEndedAt.UTC()
	windowStart = windowStart.UTC()
	windowEnd = windowEnd.UTC()
	if periodStartedAt.After(periodEndedAt) {
		periodStartedAt = periodEndedAt
	}
	if periodEndedAt.After(periodStartedAt) {
		overlapStart := accountShareModeMaxTime(periodStartedAt, windowStart)
		overlapEnd := accountShareModeMinTime(periodEndedAt, windowEnd)
		if !overlapEnd.After(overlapStart) {
			return decimal.Zero
		}
		totalNs := periodEndedAt.Sub(periodStartedAt).Nanoseconds()
		overlapNs := overlapEnd.Sub(overlapStart).Nanoseconds()
		if totalNs <= 0 || overlapNs <= 0 {
			return decimal.Zero
		}
		return totalCharge.Mul(decimal.NewFromInt(overlapNs)).Div(decimal.NewFromInt(totalNs)).Round(10)
	}
	if !periodEndedAt.Before(windowStart) && periodEndedAt.Before(windowEnd) {
		return totalCharge.Round(10)
	}
	return decimal.Zero
}

func accountShareModeMinTime(left, right time.Time) time.Time {
	if left.Before(right) {
		return left
	}
	return right
}

func accountShareModeMaxTime(left, right time.Time) time.Time {
	if left.After(right) {
		return left
	}
	return right
}

func accountShareModeUsageRequestPeriod(cmd *service.UsageBillingCommand, snapshot *service.AccountShareModeBillingSnapshot) (time.Time, time.Time) {
	endedAt := time.Now().UTC()
	if cmd != nil {
		if cmd.UsageLog != nil && !cmd.UsageLog.CreatedAt.IsZero() {
			endedAt = cmd.UsageLog.CreatedAt.UTC()
		} else if !cmd.UsageOccurredAt.IsZero() {
			endedAt = cmd.UsageOccurredAt.UTC()
		}
	}
	startedAt := endedAt
	if snapshot != nil && snapshot.DurationMs > 0 {
		startedAt = endedAt.Add(-time.Duration(snapshot.DurationMs) * time.Millisecond)
	}
	if startedAt.After(endedAt) {
		startedAt = endedAt
	}
	return startedAt, endedAt
}

func accountShareModeOwnerRatioString(snapshot *service.AccountShareModeBillingSnapshot) string {
	if snapshot == nil {
		return decimal.Zero.StringFixed(8)
	}
	ownerRatio, _ := accountShareModeSettlementRatios(snapshot.OwnerShareRatio, snapshot.PlatformShareRatio)
	return ownerRatio.StringFixed(8)
}

func accountShareModePlatformRatioString(snapshot *service.AccountShareModeBillingSnapshot) string {
	if snapshot == nil {
		return decimal.Zero.StringFixed(8)
	}
	_, platformRatio := accountShareModeSettlementRatios(snapshot.OwnerShareRatio, snapshot.PlatformShareRatio)
	return platformRatio.StringFixed(8)
}
