package repository

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newOpenAIImageAsyncTaskStoreForTest(t *testing.T) (*openAIImageAsyncTaskStore, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rdb.Close()
	})
	store, ok := NewOpenAIImageAsyncTaskStore(rdb).(*openAIImageAsyncTaskStore)
	require.True(t, ok)
	return store, mr
}

func newOpenAIImageAsyncTaskRequestForTest(userID int64, apiKeyID int64) *service.OpenAIImageAsyncTaskRequest {
	groupID := int64(3)
	return &service.OpenAIImageAsyncTaskRequest{
		Method: "POST",
		URL:    "/v1/images/generations",
		Header: map[string][]string{
			"Content-Type": {"application/json"},
		},
		Body: []byte(`{"model":"gpt-image-2","prompt":"draw"}`),
		APIKey: &service.APIKey{
			ID:      apiKeyID,
			UserID:  userID,
			Key:     "sk-secret-should-not-be-stored",
			GroupID: &groupID,
			User: &service.User{
				ID:          userID,
				Status:      service.StatusActive,
				Concurrency: 2,
			},
			Group: &service.Group{
				ID:                   groupID,
				Status:               service.StatusActive,
				Platform:             service.PlatformOpenAI,
				Hydrated:             true,
				AllowImageGeneration: true,
			},
			Status: service.StatusActive,
		},
		Subject: service.OpenAIImageAsyncTaskSubject{UserID: userID, Concurrency: 2},
		Role:    service.RoleUser,
	}
}

func TestOpenAIImageAsyncTaskStoreLifecycle(t *testing.T) {
	store, _ := newOpenAIImageAsyncTaskStoreForTest(t)
	ctx := context.Background()

	task, err := store.Enqueue(ctx, newOpenAIImageAsyncTaskRequestForTest(42, 99))
	require.NoError(t, err)
	require.NotEmpty(t, task.ID)
	require.Equal(t, int64(42), task.UserID)
	require.Equal(t, int64(99), task.APIKeyID)
	require.Equal(t, service.OpenAIImageAsyncTaskQueued, task.Status)
	require.NotNil(t, task.Request)
	require.NotNil(t, task.Request.APIKey)
	require.Empty(t, task.Request.APIKey.Key)

	dequeued, err := store.Dequeue(ctx, time.Millisecond)
	require.NoError(t, err)
	require.NotNil(t, dequeued)
	require.Equal(t, task.ID, dequeued.ID)

	locked, err := store.MarkRunning(ctx, task.ID, "worker-1", time.Minute)
	require.NoError(t, err)
	require.True(t, locked)
	require.NoError(t, store.RenewRunning(ctx, task.ID, "worker-1", time.Minute))
	running, err := store.Get(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, service.OpenAIImageAsyncTaskRunning, running.Status)
	require.False(t, running.CreatedAt.IsZero())
	require.False(t, running.UpdatedAt.IsZero())

	body := []byte(`{"data":[{"b64_json":"abc"}]}`)
	require.NoError(t, store.MarkCompleted(ctx, task.ID, 200, body, "application/json"))
	completed, err := store.Get(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, service.OpenAIImageAsyncTaskCompleted, completed.Status)
	require.Equal(t, 200, completed.ResultStatus)
	require.JSONEq(t, string(body), string(completed.ResultBody))
	require.Equal(t, "application/json", completed.ResultMimeType)
	require.False(t, completed.FinishedAt.IsZero())

	var payload map[string]any
	require.NoError(t, json.Unmarshal(completed.ResultBody, &payload))
	require.Contains(t, payload, "data")
}

func TestOpenAIImageAsyncTaskStoreFailureAndNotFound(t *testing.T) {
	store, _ := newOpenAIImageAsyncTaskStoreForTest(t)
	ctx := context.Background()

	_, err := store.Get(ctx, "missing")
	require.ErrorIs(t, err, service.ErrOpenAIImageAsyncTaskNotFound)

	task, err := store.Enqueue(ctx, newOpenAIImageAsyncTaskRequestForTest(7, 8))
	require.NoError(t, err)

	require.NoError(t, store.MarkFailed(ctx, task.ID, "upstream timeout"))
	failed, err := store.Get(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, service.OpenAIImageAsyncTaskFailed, failed.Status)
	require.Equal(t, "upstream timeout", failed.ErrorMessage)
	require.False(t, failed.FinishedAt.IsZero())
}

func TestOpenAIImageAsyncTaskStoreLimitEvictsTerminalTasks(t *testing.T) {
	store, _ := newOpenAIImageAsyncTaskStoreForTest(t)
	store.limit = 2
	ctx := context.Background()

	first, err := store.Enqueue(ctx, newOpenAIImageAsyncTaskRequestForTest(1, 1))
	require.NoError(t, err)
	require.NoError(t, store.MarkCompleted(ctx, first.ID, 200, []byte(`{"data":[]}`), "application/json"))

	second, err := store.Enqueue(ctx, newOpenAIImageAsyncTaskRequestForTest(1, 2))
	require.NoError(t, err)
	locked, err := store.MarkRunning(ctx, second.ID, "worker-2", time.Minute)
	require.NoError(t, err)
	require.True(t, locked)

	third, err := store.Enqueue(ctx, newOpenAIImageAsyncTaskRequestForTest(1, 3))
	require.NoError(t, err)
	require.NotEmpty(t, third.ID)

	_, err = store.Get(ctx, first.ID)
	require.ErrorIs(t, err, service.ErrOpenAIImageAsyncTaskNotFound)
	_, err = store.Get(ctx, second.ID)
	require.NoError(t, err)
	_, err = store.Get(ctx, third.ID)
	require.NoError(t, err)
}

func TestOpenAIImageAsyncTaskStoreRequeuesStaleRunningTasks(t *testing.T) {
	store, _ := newOpenAIImageAsyncTaskStoreForTest(t)
	ctx := context.Background()

	task, err := store.Enqueue(ctx, newOpenAIImageAsyncTaskRequestForTest(11, 22))
	require.NoError(t, err)
	dequeued, err := store.Dequeue(ctx, time.Millisecond)
	require.NoError(t, err)
	require.Equal(t, task.ID, dequeued.ID)

	locked, err := store.MarkRunning(ctx, task.ID, "worker-stale", time.Minute)
	require.NoError(t, err)
	require.True(t, locked)

	require.NoError(t, store.rdb.ZAdd(ctx, openAIImageAsyncTaskRunningKey, redis.Z{
		Score:  float64(time.Now().Add(-time.Hour).Unix()),
		Member: task.ID,
	}).Err())
	count, err := store.RequeueStaleRunning(ctx, time.Minute)
	require.NoError(t, err)
	require.Equal(t, int64(1), count)

	requeued, err := store.Get(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, service.OpenAIImageAsyncTaskQueued, requeued.Status)

	dequeuedAgain, err := store.Dequeue(ctx, time.Millisecond)
	require.NoError(t, err)
	require.NotNil(t, dequeuedAgain)
	require.Equal(t, task.ID, dequeuedAgain.ID)
}
