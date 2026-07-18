package service

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestCommunitySchedulerConfigMapsReferenceAnthropicControls(t *testing.T) {
	credentials, extra := communitySchedulerConfig("anthropic", map[string]any{
		"refresh_token": "oauth-refresh",
	}, map[string]any{
		"temp_unschedulable_enabled": true,
		"temp_unschedulable_rules": []any{map[string]any{
			"error_code": float64(529), "keywords": []any{"overloaded"}, "duration_minutes": float64(10),
		}},
		"intercept_warmup":             true,
		"session_limit":                float64(7),
		"session_idle_timeout_minutes": float64(9),
		"rpm_limit":                    float64(25),
		"rpm_strategy":                 "sticky_exempt",
		"rpm_sticky_buffer":            float64(6),
		"tls_fingerprint":              true,
		"session_affinity":             true,
		"cache_ttl_override":           true,
		"cache_ttl_target":             "1h",
		"user_msg_queue_mode":          "serialize",
		"window_cost_limit":            float64(18.5),
		"window_cost_sticky_reserve":   float64(4.5),
	}, 42)

	require.Equal(t, "oauth-refresh", credentials["refresh_token"])
	require.Equal(t, true, credentials["intercept_warmup_requests"])
	require.Equal(t, true, credentials["temp_unschedulable_enabled"])
	require.Len(t, credentials["temp_unschedulable_rules"], 1)
	require.Equal(t, int64(42), extra["community_account_id"])
	require.Equal(t, true, extra["community_owner_managed"])
	require.Equal(t, float64(7), extra["max_sessions"])
	require.Equal(t, float64(9), extra["session_idle_timeout_minutes"])
	require.Equal(t, float64(25), extra["base_rpm"])
	require.Equal(t, "sticky_exempt", extra["rpm_strategy"])
	require.Equal(t, float64(6), extra["rpm_sticky_buffer"])
	require.Equal(t, true, extra["enable_tls_fingerprint"])
	require.Equal(t, true, extra["session_id_masking_enabled"])
	require.Equal(t, true, extra["cache_ttl_override_enabled"])
	require.Equal(t, "1h", extra["cache_ttl_override_target"])
	require.Equal(t, "serialize", extra["user_msg_queue_mode"])
	require.Equal(t, float64(18.5), extra["window_cost_limit"])
	require.Equal(t, float64(4.5), extra["window_cost_sticky_reserve"])
}

func TestCommunitySchedulerConfigDoesNotLeakAnthropicControlsToOpenAI(t *testing.T) {
	credentials, extra := communitySchedulerConfig("openai", map[string]any{
		"refresh_token": "oauth-refresh",
	}, map[string]any{
		"tls_fingerprint": true,
		"session_limit":   float64(7),
	}, 43)

	require.Equal(t, map[string]any{"refresh_token": "oauth-refresh"}, credentials)
	require.Equal(t, map[string]any{
		"community_account_id":    int64(43),
		"community_owner_managed": true,
	}, extra)
}

func TestCommunitySchedulerConfigMapsOpenAIModelMapping(t *testing.T) {
	credentials, extra := communitySchedulerConfig("openai", map[string]any{
		"refresh_token": "oauth-refresh",
	}, map[string]any{
		"model_mapping": map[string]any{"claude-sonnet-*": "gpt-5.4"},
	}, 44)

	require.Equal(t, map[string]any{"claude-sonnet-*": "gpt-5.4"}, credentials["model_mapping"])
	require.Equal(t, int64(44), extra["community_account_id"])
}

func TestDeleteCommunityAccountPausesListingAndSchedulerAccount(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT scheduler_account_id FROM community_accounts`).
		WithArgs(int64(81), int64(17)).
		WillReturnRows(sqlmock.NewRows([]string{"scheduler_account_id"}).AddRow(int64(42)))
	mock.ExpectExec(`UPDATE community_accounts SET deleted_at=NOW\(\),status='paused',schedulable=FALSE`).
		WithArgs(int64(81), int64(17)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE community_listings SET status='paused'`).
		WithArgs(int64(81), int64(17)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE accounts SET status='disabled',schedulable=FALSE`).
		WithArgs(int64(42)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`INSERT INTO scheduler_outbox`).
		WithArgs(int64(42)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	service := NewCommunityService(db, communityTestEncryptor{})
	require.NoError(t, service.DeleteAccount(context.Background(), 17, 81))
	require.NoError(t, mock.ExpectationsWereMet())
}
