package service

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestValidPayoutQRCodeAcceptsSupportedImages(t *testing.T) {
	images := map[string][]byte{
		"image/png":  {'\x89', 'P', 'N', 'G', '\r', '\n', '\x1a', '\n'},
		"image/jpeg": {'\xff', '\xd8', '\xff', '\xe0', 0, 16, 'J', 'F', 'I', 'F', 0},
		"image/webp": {'R', 'I', 'F', 'F', 4, 0, 0, 0, 'W', 'E', 'B', 'P', 'V', 'P', '8', ' '},
	}
	for mime, image := range images {
		payload := base64.StdEncoding.EncodeToString(image)
		require.True(t, validPayoutQRCode("data:"+mime+";base64,"+payload), mime)
	}
}

func TestValidPayoutQRCodeRejectsUnsafeOrOversizedValues(t *testing.T) {
	require.False(t, validPayoutQRCode("https://attacker.example/collect"))
	require.False(t, validPayoutQRCode("javascript:alert(1)"))
	require.False(t, validPayoutQRCode("data:image/svg+xml;base64,"+base64.StdEncoding.EncodeToString([]byte("<svg/>"))))
	require.False(t, validPayoutQRCode("data:image/png;base64,not-base64"))
	require.False(t, validPayoutQRCode("data:image/png;base64,"+base64.StdEncoding.EncodeToString([]byte("not a png"))))
	require.False(t, validPayoutQRCode("data:image/png;base64,"+base64.StdEncoding.EncodeToString(make([]byte, 2*1024*1024+1))))
}

func TestReviewWithdrawalCannotSkipApproval(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT user_id,amount,fee,status FROM withdrawals`).
		WithArgs(int64(81)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "amount", "fee", "status"}).
			AddRow(int64(17), 10.0, 0.1, "pending"))
	mock.ExpectRollback()

	service := NewCommunityService(db, communityTestEncryptor{})
	err = service.ReviewWithdrawal(context.Background(), 1, 81, "paid", "", "offline-001")

	require.ErrorIs(t, err, ErrCommunityConflict)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReviewWithdrawalRequiresOfflinePaymentReference(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	service := NewCommunityService(db, communityTestEncryptor{})
	err = service.ReviewWithdrawal(context.Background(), 1, 81, "paid", "", "")

	require.ErrorIs(t, err, ErrCommunityInvalid)
	require.NoError(t, mock.ExpectationsWereMet())
}
