//go:build integration

package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestVideoRepository_BindsAccountAndSettlesHoldExactlyOnce(t *testing.T) {
	ctx := context.Background()
	suffix := uniqueTestValue(t, "video")

	var groupID, userID, apiKeyID, accountOneID, accountTwoID int64
	require.NoError(t, integrationDB.QueryRowContext(ctx,
		`INSERT INTO groups (name) VALUES ($1) RETURNING id`, suffix+"-group").Scan(&groupID))
	require.NoError(t, integrationDB.QueryRowContext(ctx,
		`INSERT INTO users (email,password_hash,balance,frozen_balance) VALUES ($1,'test',20,0) RETURNING id`, suffix+"@example.test").Scan(&userID))
	require.NoError(t, integrationDB.QueryRowContext(ctx,
		`INSERT INTO api_keys (user_id,key,name,group_id) VALUES ($1,$2,$3,$4) RETURNING id`, userID, "sk-"+suffix, suffix+"-key", groupID).Scan(&apiKeyID))
	require.NoError(t, integrationDB.QueryRowContext(ctx,
		`INSERT INTO accounts (name,platform,type,credentials,extra) VALUES ($1,'openai','apikey','{}','{}') RETURNING id`, suffix+"-account-1").Scan(&accountOneID))
	require.NoError(t, integrationDB.QueryRowContext(ctx,
		`INSERT INTO accounts (name,platform,type,credentials,extra) VALUES ($1,'openai','apikey','{}','{}') RETURNING id`, suffix+"-account-2").Scan(&accountTwoID))
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(ctx, `DELETE FROM video_jobs WHERE api_key_id=$1`, apiKeyID)
		_, _ = integrationDB.ExecContext(ctx, `DELETE FROM video_media WHERE api_key_id=$1`, apiKeyID)
		_, _ = integrationDB.ExecContext(ctx, `DELETE FROM api_keys WHERE id=$1`, apiKeyID)
		_, _ = integrationDB.ExecContext(ctx, `DELETE FROM accounts WHERE id IN ($1,$2)`, accountOneID, accountTwoID)
		_, _ = integrationDB.ExecContext(ctx, `DELETE FROM groups WHERE id=$1`, groupID)
		_, _ = integrationDB.ExecContext(ctx, `DELETE FROM users WHERE id=$1`, userID)
	})

	repo := NewVideoRepository(integrationDB)
	media := &service.VideoMedia{
		MediaID:         "vidmedia_repo_bound",
		UpstreamMediaID: "upstream-media",
		AccountID:       accountOneID,
		UserID:          userID,
		APIKeyID:        apiKeyID,
		UpstreamURL:     "https://upstream.example/media",
		MediaType:       "UPLOADED",
		ExpiresAt:       time.Now().Add(time.Hour),
	}
	require.NoError(t, repo.CreateMedia(ctx, media))
	storedMedia, err := repo.GetMediaForOwner(ctx, media.MediaID, apiKeyID)
	require.NoError(t, err)
	require.Equal(t, accountOneID, storedMedia.AccountID)
	_, err = repo.GetMediaForOwner(ctx, media.MediaID, apiKeyID+1)
	require.ErrorIs(t, err, service.ErrVideoResourceNotFound)

	owner := service.VideoOwner{UserID: userID, APIKeyID: apiKeyID, GroupID: &groupID}
	reserve := func(jobID string, accountID int64, idempotencyKey string) *service.VideoJob {
		t.Helper()
		job, created, reserveErr := repo.ReserveJob(ctx, service.VideoJobReservation{
			JobID:                  jobID,
			AccountID:              accountID,
			Owner:                  owner,
			IdempotencyKey:         idempotencyKey,
			RequestHash:            strings.Repeat("a", 64),
			Model:                  "video-public",
			Resolution:             "480p",
			Duration:               4,
			AspectRatio:            "16:9",
			PreauthorizationAmount: 10,
		})
		require.NoError(t, reserveErr)
		require.True(t, created)
		return job
	}

	first := reserve("vidjob_repo_one", accountOneID, "repo-one")
	second := reserve("vidjob_repo_two", accountTwoID, "repo-two")
	require.Equal(t, accountOneID, first.AccountID)
	require.Equal(t, accountTwoID, second.AccountID)
	assertVideoBalance(t, ctx, userID, 0, 20)

	first, err = repo.FinalizeJobAndReconcileHold(ctx, service.VideoJobFinalization{
		JobID: first.JobID, UpstreamJobID: "same-upstream-id", Status: "running", Amount: 2, Currency: "USD",
		Resolution: "720p", Duration: 8, AspectRatio: "9:16", UpstreamResponse: []byte(`{"status":"running"}`),
	})
	require.NoError(t, err)
	require.Equal(t, "720p", first.Resolution)
	require.Equal(t, 8, first.Duration)
	require.Equal(t, "9:16", first.AspectRatio)
	second, err = repo.FinalizeJobAndReconcileHold(ctx, service.VideoJobFinalization{
		JobID: second.JobID, UpstreamJobID: "same-upstream-id", Status: "running", Amount: 3, Currency: "USD", UpstreamResponse: []byte(`{"status":"running"}`),
	})
	require.NoError(t, err, "upstream IDs are unique within an account, not globally")

	assertVideoBalance(t, ctx, userID, 15, 5)
	now := time.Now()
	first, err = repo.UpdateJobAndSettle(ctx, service.VideoJobUpdate{
		JobID: first.JobID, Status: "completed", Resolution: "1080p", Duration: 12, AspectRatio: "21:9",
		UpstreamResponse: []byte(`{"status":"completed"}`), FinishedAt: &now,
	})
	require.NoError(t, err)
	require.Equal(t, "captured", first.SettlementStatus)
	require.Equal(t, "1080p", first.Resolution)
	require.Equal(t, 12, first.Duration)
	require.Equal(t, "21:9", first.AspectRatio)
	assertVideoBalance(t, ctx, userID, 15, 3)

	first, err = repo.UpdateJobAndSettle(ctx, service.VideoJobUpdate{JobID: first.JobID, Status: "running", UpstreamResponse: []byte(`{"status":"running"}`)})
	require.NoError(t, err)
	require.Equal(t, "completed", first.Status, "a stale poll must not regress a terminal job")
	assertVideoBalance(t, ctx, userID, 15, 3)

	second, err = repo.UpdateJobAndSettle(ctx, service.VideoJobUpdate{JobID: second.JobID, Status: "failed", UpstreamResponse: []byte(`{"status":"failed"}`), FinishedAt: &now})
	require.NoError(t, err)
	require.Equal(t, "released", second.SettlementStatus)
	assertVideoBalance(t, ctx, userID, 18, 0)

	second, err = repo.UpdateJobAndSettle(ctx, service.VideoJobUpdate{JobID: second.JobID, Status: "failed", UpstreamResponse: []byte(`{"status":"failed"}`), FinishedAt: &now})
	require.NoError(t, err)
	require.Equal(t, "released", second.SettlementStatus)
	assertVideoBalance(t, ctx, userID, 18, 0)

	_, err = repo.UpdateJobAndSettle(ctx, service.VideoJobUpdate{JobID: second.JobID, Status: "unknown"})
	require.EqualError(t, err, "invalid video job status")
}

func assertVideoBalance(t *testing.T, ctx context.Context, userID int64, wantBalance, wantFrozen float64) {
	t.Helper()
	var balance, frozen float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT balance,frozen_balance FROM users WHERE id=$1`, userID).Scan(&balance, &frozen))
	require.InDelta(t, wantBalance, balance, 0.00000001)
	require.InDelta(t, wantFrozen, frozen, 0.00000001)
}
