package service

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestReviewMembershipPersistsReviewForActivatedNonOwnerUse(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectExec(`INSERT INTO community_reviews`).
		WithArgs(int64(51), int64(17), 9.5, "稳定，速度正常").
		WillReturnResult(sqlmock.NewResult(1, 1))

	service := NewCommunityService(db, communityTestEncryptor{})
	err = service.ReviewMembership(context.Background(), 17, 51, 9.5, "稳定，速度正常")

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReviewMembershipRejectsUnownedOrNeverActivatedMembership(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectExec(`INSERT INTO community_reviews`).
		WithArgs(int64(51), int64(17), 8.0, "").
		WillReturnResult(sqlmock.NewResult(0, 0))

	service := NewCommunityService(db, communityTestEncryptor{})
	err = service.ReviewMembership(context.Background(), 17, 51, 8, "")

	require.ErrorIs(t, err, ErrCommunityForbidden)
	require.NoError(t, mock.ExpectationsWereMet())
}
