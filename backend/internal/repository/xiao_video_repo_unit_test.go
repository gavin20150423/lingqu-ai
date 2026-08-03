package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestVideoRepository_CreateMediaPersistsOwnershipAndAccount(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	media := &service.VideoMedia{
		MediaID:         "vidmedia_test",
		UpstreamMediaID: "up-media",
		AccountID:       42,
		UserID:          11,
		APIKeyID:        22,
		UpstreamURL:     "https://upstream.example/media",
		MediaType:       "UPLOADED",
		ExpiresAt:       time.Now().Add(time.Hour),
	}
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO video_media (media_id, upstream_media_id, account_id, user_id, api_key_id, upstream_url, media_type, expires_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`)).
		WithArgs(media.MediaID, media.UpstreamMediaID, media.AccountID, media.UserID, media.APIKeyID, media.UpstreamURL, media.MediaType, media.ExpiresAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	repo := NewVideoRepository(db)
	require.NoError(t, repo.CreateMedia(context.Background(), media))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestVideoRepository_UpdateJobAndSettleDoesNotRegressTerminalState(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Now().UTC()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(videoJobSelect + ` WHERE job_id=$1 FOR UPDATE`)).
		WithArgs("vidjob_terminal").
		WillReturnRows(videoJobRows("vidjob_terminal", "up-job", "completed", 2, "captured", now))
	mock.ExpectRollback()

	repo := NewVideoRepository(db)
	job, err := repo.UpdateJobAndSettle(context.Background(), service.VideoJobUpdate{
		JobID: "vidjob_terminal", Status: "running", UpstreamResponse: []byte(`{"status":"running"}`),
	})
	require.NoError(t, err)
	require.Equal(t, "completed", job.Status)
	require.Equal(t, "captured", job.SettlementStatus)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestVideoRepository_UpdateJobAndSettleCapturesHeldAmount(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Now().UTC()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(videoJobSelect + ` WHERE job_id=$1 FOR UPDATE`)).
		WithArgs("vidjob_capture").
		WillReturnRows(videoJobRows("vidjob_capture", "up-job", "running", 2, "held", now))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE users SET frozen_balance=COALESCE(frozen_balance,0)-$1,updated_at=NOW() WHERE id=$2 AND COALESCE(frozen_balance,0)>=$1`)).
		WithArgs(2.0, int64(11)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE video_jobs SET status=$2,upstream_response=$3,updated_at=NOW(),finished_at=COALESCE($4,finished_at),settlement_status=$5,settled_at=$6 WHERE job_id=$1`)).
		WithArgs("vidjob_capture", "completed", []byte(`{"status":"completed"}`), sqlmock.AnyArg(), "captured", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(regexp.QuoteMeta(videoJobSelect + ` WHERE job_id=$1`)).
		WithArgs("vidjob_capture").
		WillReturnRows(videoJobRows("vidjob_capture", "up-job", "completed", 2, "captured", now))

	repo := NewVideoRepository(db)
	finished := now
	job, err := repo.UpdateJobAndSettle(context.Background(), service.VideoJobUpdate{
		JobID: "vidjob_capture", Status: "completed", UpstreamResponse: []byte(`{"status":"completed"}`), FinishedAt: &finished,
	})
	require.NoError(t, err)
	require.Equal(t, "captured", job.SettlementStatus)
	require.NoError(t, mock.ExpectationsWereMet())
}

func videoJobRows(jobID, upstreamID, status string, amount float64, settlement string, now time.Time) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"job_id", "upstream_job_id", "account_id", "user_id", "api_key_id", "group_id", "idempotency_key", "request_hash",
		"model", "resolution", "duration", "aspect_ratio", "status", "amount", "currency", "settlement_status", "upstream_response",
		"created_at", "updated_at", "finished_at", "settled_at",
	}).AddRow(jobID, upstreamID, int64(42), int64(11), int64(22), int64(7), nil, "hash", "video-public", "480p", 4, "16:9", status, amount, "USD", settlement, []byte(`{}`), now, now, nil, nil)
}
