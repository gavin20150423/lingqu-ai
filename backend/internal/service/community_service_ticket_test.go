package service

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestCreateTicketRejectsUnknownCategoryAndPriorityBeforeWriting(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	service := NewCommunityService(db, communityTestEncryptor{})

	_, err = service.CreateTicket(context.Background(), 17, "主题", "unknown", "normal", "内容")
	require.ErrorIs(t, err, ErrCommunityInvalid)
	_, err = service.CreateTicket(context.Background(), 17, "主题", "support", "critical", "内容")
	require.ErrorIs(t, err, ErrCommunityInvalid)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReplyTicketRejectsClosedTicket(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT user_id,status FROM support_tickets`).
		WithArgs(int64(81), int64(17)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "status"}).AddRow(int64(17), "closed"))
	mock.ExpectRollback()

	service := NewCommunityService(db, communityTestEncryptor{})
	err = service.ReplyTicket(context.Background(), 17, 81, false, "继续回复")
	require.ErrorIs(t, err, ErrCommunityConflict)
	require.NoError(t, mock.ExpectationsWereMet())
}
