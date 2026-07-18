package service

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestJoinListingAllowsOwnerSelfUseWithoutMinimumBalance(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	const userID int64 = 17
	const listingID int64 = 31
	const keyID int64 = 41

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT l\.owner_user_id,a\.provider,l\.minimum_balance`).
		WithArgs(listingID).
		WillReturnRows(sqlmock.NewRows([]string{"owner_user_id", "provider", "minimum_balance"}).
			AddRow(userID, "anthropic", 100.0))
	mock.ExpectQuery(`SELECT u\.balance,k\.account_mode_platform`).
		WithArgs(keyID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"balance", "account_mode_platform"}).
			AddRow(0.0, "anthropic"))
	mock.ExpectQuery(`SELECT COALESCE\(MIN\(slot\),0\) FROM generate_series`).
		WithArgs(keyID).
		WillReturnRows(sqlmock.NewRows([]string{"reservation_order"}).AddRow(1))
	mock.ExpectQuery(`INSERT INTO community_memberships`).
		WithArgs(listingID, userID, keyID, 1, 10).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(51)))
	mock.ExpectCommit()

	service := NewCommunityService(db, communityTestEncryptor{})
	membershipID, err := service.JoinListing(context.Background(), userID, listingID, keyID, 10)

	require.NoError(t, err)
	require.Equal(t, int64(51), membershipID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestJoinListingUsesFirstAvailableReservationOrder(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	const userID int64 = 17
	const listingID int64 = 31
	const keyID int64 = 41

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT l\.owner_user_id,a\.provider,l\.minimum_balance`).
		WithArgs(listingID).
		WillReturnRows(sqlmock.NewRows([]string{"owner_user_id", "provider", "minimum_balance"}).
			AddRow(int64(99), "openai", 0.0))
	mock.ExpectQuery(`SELECT u\.balance,k\.account_mode_platform`).
		WithArgs(keyID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"balance", "account_mode_platform"}).
			AddRow(10.0, "openai"))
	mock.ExpectQuery(`SELECT COALESCE\(MIN\(slot\),0\) FROM generate_series`).
		WithArgs(keyID).
		WillReturnRows(sqlmock.NewRows([]string{"reservation_order"}).AddRow(2))
	mock.ExpectQuery(`INSERT INTO community_memberships`).
		WithArgs(listingID, userID, keyID, 2, 10).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(52)))
	mock.ExpectCommit()

	service := NewCommunityService(db, communityTestEncryptor{})
	membershipID, err := service.JoinListing(context.Background(), userID, listingID, keyID, 10)

	require.NoError(t, err)
	require.Equal(t, int64(52), membershipID)
	require.NoError(t, mock.ExpectationsWereMet())
}
