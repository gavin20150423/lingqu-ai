package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	openAIImageAsyncTaskKeyPrefix     = "openai:image_async:task:"
	openAIImageAsyncTaskLockKeyPrefix = "openai:image_async:lock:"
	openAIImageAsyncTaskQueueKey      = "openai:image_async:queue"
	openAIImageAsyncTaskIndexKey      = "openai:image_async:tasks"
	openAIImageAsyncTaskTerminalKey   = "openai:image_async:terminal"
	openAIImageAsyncTaskRunningKey    = "openai:image_async:running"
	openAIImageAsyncTaskRetention     = 24 * time.Hour
	openAIImageAsyncTaskLimit         = 100
	openAIImageAsyncTaskEvictionBatch = 32
)

var openAIImageAsyncTaskStoreCreateScript = redis.NewScript(`
redis.call("ZREMRANGEBYSCORE", KEYS[1], "-inf", ARGV[1])
redis.call("ZREMRANGEBYSCORE", KEYS[2], "-inf", ARGV[1])
redis.call("ZREMRANGEBYSCORE", KEYS[3], "-inf", ARGV[1])

local count = redis.call("ZCARD", KEYS[1])
if count >= tonumber(ARGV[2]) then
  local evictable = redis.call("ZRANGE", KEYS[2], 0, tonumber(ARGV[6]) - 1)
  for _, task_id in ipairs(evictable) do
    redis.call("DEL", ARGV[7] .. task_id)
    redis.call("DEL", ARGV[8] .. task_id)
    redis.call("ZREM", KEYS[1], task_id)
    redis.call("ZREM", KEYS[2], task_id)
    redis.call("ZREM", KEYS[3], task_id)
  end
  count = redis.call("ZCARD", KEYS[1])
end

if count >= tonumber(ARGV[2]) then
  return 0
end

redis.call("SET", KEYS[4], ARGV[3], "EX", ARGV[4])
redis.call("ZADD", KEYS[1], ARGV[5], ARGV[9])
redis.call("RPUSH", KEYS[5], ARGV[9])
return 1
`)

var openAIImageAsyncTaskRenewScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  redis.call("EXPIRE", KEYS[1], ARGV[2])
  redis.call("ZADD", KEYS[2], ARGV[3], ARGV[4])
  return 1
end
return 0
`)

type redisOpenAIImageAsyncTask struct {
	ID             string                               `json:"id"`
	UserID         int64                                `json:"user_id"`
	APIKeyID       int64                                `json:"api_key_id"`
	Status         string                               `json:"status"`
	Request        *service.OpenAIImageAsyncTaskRequest `json:"request,omitempty"`
	ResultStatus   int                                  `json:"result_status,omitempty"`
	ResultBody     []byte                               `json:"result_body,omitempty"`
	ResultMimeType string                               `json:"result_mime_type,omitempty"`
	ErrorMessage   string                               `json:"error_message,omitempty"`
	CreatedAt      int64                                `json:"created_at"`
	UpdatedAt      int64                                `json:"updated_at"`
	FinishedAt     int64                                `json:"finished_at,omitempty"`
}

type openAIImageAsyncTaskStore struct {
	rdb       *redis.Client
	retention time.Duration
	limit     int
}

func NewOpenAIImageAsyncTaskStore(rdb *redis.Client) service.OpenAIImageAsyncTaskStore {
	return &openAIImageAsyncTaskStore{
		rdb:       rdb,
		retention: openAIImageAsyncTaskRetention,
		limit:     openAIImageAsyncTaskLimit,
	}
}

func (s *openAIImageAsyncTaskStore) Enqueue(ctx context.Context, request *service.OpenAIImageAsyncTaskRequest) (*service.OpenAIImageAsyncTask, error) {
	if s == nil || s.rdb == nil {
		return nil, service.ErrOpenAIImageAsyncTaskNotFound
	}
	if request == nil || request.APIKey == nil {
		return nil, fmt.Errorf("openai image async task request is incomplete")
	}
	now := time.Now()
	task := &service.OpenAIImageAsyncTask{
		ID:        "imgtask_" + uuid.NewString(),
		UserID:    request.Subject.UserID,
		APIKeyID:  request.APIKey.ID,
		Status:    service.OpenAIImageAsyncTaskQueued,
		Request:   request,
		CreatedAt: now,
		UpdatedAt: now,
	}
	payload, err := json.Marshal(redisOpenAIImageAsyncTaskFromService(task))
	if err != nil {
		return nil, fmt.Errorf("marshal openai image async task: %w", err)
	}

	expireBefore := now.Add(-s.retention).Unix()
	ok, err := openAIImageAsyncTaskStoreCreateScript.Run(
		ctx,
		s.rdb,
		[]string{
			openAIImageAsyncTaskIndexKey,
			openAIImageAsyncTaskTerminalKey,
			openAIImageAsyncTaskRunningKey,
			openAIImageAsyncTaskKey(task.ID),
			openAIImageAsyncTaskQueueKey,
		},
		expireBefore,
		s.limit,
		string(payload),
		int(s.retention.Seconds()),
		now.Unix(),
		openAIImageAsyncTaskEvictionBatch,
		openAIImageAsyncTaskKeyPrefix,
		openAIImageAsyncTaskLockKeyPrefix,
		task.ID,
	).Int()
	if err != nil {
		return nil, err
	}
	if ok != 1 {
		return nil, service.ErrOpenAIImageAsyncTaskStoreFull
	}
	return cloneOpenAIImageAsyncTask(task), nil
}

func (s *openAIImageAsyncTaskStore) Dequeue(ctx context.Context, wait time.Duration) (*service.OpenAIImageAsyncTask, error) {
	if s == nil || s.rdb == nil {
		return nil, service.ErrOpenAIImageAsyncTaskNotFound
	}
	items, err := s.rdb.BLPop(ctx, wait, openAIImageAsyncTaskQueueKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	if len(items) < 2 {
		return nil, nil
	}
	task, err := s.Get(ctx, items[1])
	if err != nil {
		if err == service.ErrOpenAIImageAsyncTaskNotFound {
			return nil, nil
		}
		return nil, err
	}
	return task, nil
}

func (s *openAIImageAsyncTaskStore) RequeueStaleRunning(ctx context.Context, staleAfter time.Duration) (int64, error) {
	if s == nil || s.rdb == nil {
		return 0, nil
	}
	if staleAfter <= 0 {
		return 0, nil
	}
	cutoff := time.Now().Add(-staleAfter).Unix()
	taskIDs, err := s.rdb.ZRangeByScore(ctx, openAIImageAsyncTaskRunningKey, &redis.ZRangeBy{
		Min: "-inf",
		Max: fmt.Sprintf("%d", cutoff),
	}).Result()
	if err != nil {
		return 0, err
	}
	var count int64
	for _, taskID := range taskIDs {
		task, err := s.Get(ctx, taskID)
		if err != nil {
			if err == service.ErrOpenAIImageAsyncTaskNotFound {
				_ = s.rdb.ZRem(ctx, openAIImageAsyncTaskRunningKey, taskID).Err()
				continue
			}
			return count, err
		}
		if task.Status != service.OpenAIImageAsyncTaskRunning {
			_ = s.rdb.ZRem(ctx, openAIImageAsyncTaskRunningKey, taskID).Err()
			continue
		}
		if err := s.update(ctx, taskID, func(task *service.OpenAIImageAsyncTask, now time.Time) {
			task.Status = service.OpenAIImageAsyncTaskQueued
			task.UpdatedAt = now
		}); err != nil {
			return count, err
		}
		pipe := s.rdb.TxPipeline()
		pipe.Del(ctx, openAIImageAsyncTaskLockKey(taskID))
		pipe.ZRem(ctx, openAIImageAsyncTaskRunningKey, taskID)
		pipe.RPush(ctx, openAIImageAsyncTaskQueueKey, taskID)
		if _, err := pipe.Exec(ctx); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (s *openAIImageAsyncTaskStore) MarkRunning(ctx context.Context, taskID string, owner string, leaseTTL time.Duration) (bool, error) {
	if leaseTTL <= 0 {
		leaseTTL = 30 * time.Minute
	}
	owner = strings.TrimSpace(owner)
	if owner == "" {
		owner = "unknown"
	}
	task, err := s.Get(ctx, taskID)
	if err != nil {
		return false, err
	}
	if task.Status != service.OpenAIImageAsyncTaskQueued {
		return false, nil
	}
	locked, err := s.rdb.SetNX(ctx, openAIImageAsyncTaskLockKey(taskID), owner, leaseTTL).Result()
	if err != nil || !locked {
		return locked, err
	}
	if err := s.update(ctx, taskID, func(task *service.OpenAIImageAsyncTask, now time.Time) {
		task.Status = service.OpenAIImageAsyncTaskRunning
		task.UpdatedAt = now
	}); err != nil {
		_ = s.rdb.Del(ctx, openAIImageAsyncTaskLockKey(taskID)).Err()
		return false, err
	}
	return true, nil
}

func (s *openAIImageAsyncTaskStore) RenewRunning(ctx context.Context, taskID string, owner string, leaseTTL time.Duration) error {
	if s == nil || s.rdb == nil {
		return service.ErrOpenAIImageAsyncTaskNotFound
	}
	if leaseTTL <= 0 {
		leaseTTL = 30 * time.Minute
	}
	owner = strings.TrimSpace(owner)
	if owner == "" {
		owner = "unknown"
	}
	task, err := s.Get(ctx, taskID)
	if err != nil {
		return err
	}
	if task.Status != service.OpenAIImageAsyncTaskRunning {
		return nil
	}
	renewed, err := openAIImageAsyncTaskRenewScript.Run(
		ctx,
		s.rdb,
		[]string{openAIImageAsyncTaskLockKey(taskID), openAIImageAsyncTaskRunningKey},
		owner,
		int(leaseTTL.Seconds()),
		time.Now().Unix(),
		taskID,
	).Int()
	if err != nil {
		return err
	}
	if renewed != 1 {
		return nil
	}
	return s.update(ctx, taskID, func(task *service.OpenAIImageAsyncTask, now time.Time) {
		task.Status = service.OpenAIImageAsyncTaskRunning
		task.UpdatedAt = now
	})
}

func (s *openAIImageAsyncTaskStore) MarkCompleted(ctx context.Context, taskID string, statusCode int, body []byte, contentType string) error {
	if err := s.update(ctx, taskID, func(task *service.OpenAIImageAsyncTask, now time.Time) {
		task.Status = service.OpenAIImageAsyncTaskCompleted
		task.ResultStatus = statusCode
		task.ResultBody = append(task.ResultBody[:0], body...)
		task.ResultMimeType = strings.TrimSpace(contentType)
		task.UpdatedAt = now
		task.FinishedAt = now
	}); err != nil {
		return err
	}
	return s.finish(ctx, taskID)
}

func (s *openAIImageAsyncTaskStore) MarkFailed(ctx context.Context, taskID string, message string) error {
	if err := s.update(ctx, taskID, func(task *service.OpenAIImageAsyncTask, now time.Time) {
		task.Status = service.OpenAIImageAsyncTaskFailed
		task.ErrorMessage = strings.TrimSpace(message)
		if task.ErrorMessage == "" {
			task.ErrorMessage = "图片任务执行失败"
		}
		task.UpdatedAt = now
		task.FinishedAt = now
	}); err != nil {
		return err
	}
	return s.finish(ctx, taskID)
}

func (s *openAIImageAsyncTaskStore) Get(ctx context.Context, taskID string) (*service.OpenAIImageAsyncTask, error) {
	if s == nil || s.rdb == nil {
		return nil, service.ErrOpenAIImageAsyncTaskNotFound
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, service.ErrOpenAIImageAsyncTaskNotFound
	}
	payload, err := s.rdb.Get(ctx, openAIImageAsyncTaskKey(taskID)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, service.ErrOpenAIImageAsyncTaskNotFound
		}
		return nil, err
	}
	var stored redisOpenAIImageAsyncTask
	if err := json.Unmarshal(payload, &stored); err != nil {
		return nil, fmt.Errorf("unmarshal openai image async task: %w", err)
	}
	return stored.toServiceTask(), nil
}

func (s *openAIImageAsyncTaskStore) update(ctx context.Context, taskID string, mutate func(*service.OpenAIImageAsyncTask, time.Time)) error {
	task, err := s.Get(ctx, taskID)
	if err != nil {
		return err
	}
	now := time.Now()
	mutate(task, now)
	payload, err := json.Marshal(redisOpenAIImageAsyncTaskFromService(task))
	if err != nil {
		return fmt.Errorf("marshal openai image async task: %w", err)
	}
	pipe := s.rdb.TxPipeline()
	pipe.Set(ctx, openAIImageAsyncTaskKey(task.ID), payload, s.retention)
	pipe.ZAdd(ctx, openAIImageAsyncTaskIndexKey, redis.Z{
		Score:  float64(task.UpdatedAt.Unix()),
		Member: task.ID,
	})
	if task.Status == service.OpenAIImageAsyncTaskCompleted || task.Status == service.OpenAIImageAsyncTaskFailed {
		pipe.ZAdd(ctx, openAIImageAsyncTaskTerminalKey, redis.Z{
			Score:  float64(task.UpdatedAt.Unix()),
			Member: task.ID,
		})
		pipe.ZRem(ctx, openAIImageAsyncTaskRunningKey, task.ID)
	} else if task.Status == service.OpenAIImageAsyncTaskRunning {
		pipe.ZAdd(ctx, openAIImageAsyncTaskRunningKey, redis.Z{
			Score:  float64(task.UpdatedAt.Unix()),
			Member: task.ID,
		})
	} else {
		pipe.ZRem(ctx, openAIImageAsyncTaskTerminalKey, task.ID)
		pipe.ZRem(ctx, openAIImageAsyncTaskRunningKey, task.ID)
	}
	_, err = pipe.Exec(ctx)
	return err
}

func (s *openAIImageAsyncTaskStore) finish(ctx context.Context, taskID string) error {
	pipe := s.rdb.TxPipeline()
	pipe.Del(ctx, openAIImageAsyncTaskLockKey(taskID))
	pipe.ZRem(ctx, openAIImageAsyncTaskRunningKey, taskID)
	_, err := pipe.Exec(ctx)
	return err
}

func redisOpenAIImageAsyncTaskFromService(task *service.OpenAIImageAsyncTask) redisOpenAIImageAsyncTask {
	if task == nil {
		return redisOpenAIImageAsyncTask{}
	}
	return redisOpenAIImageAsyncTask{
		ID:             task.ID,
		UserID:         task.UserID,
		APIKeyID:       task.APIKeyID,
		Status:         string(task.Status),
		Request:        cloneOpenAIImageAsyncTaskRequest(task.Request),
		ResultStatus:   task.ResultStatus,
		ResultBody:     append([]byte(nil), task.ResultBody...),
		ResultMimeType: task.ResultMimeType,
		ErrorMessage:   task.ErrorMessage,
		CreatedAt:      unixOrZero(task.CreatedAt),
		UpdatedAt:      unixOrZero(task.UpdatedAt),
		FinishedAt:     unixOrZero(task.FinishedAt),
	}
}

func (t redisOpenAIImageAsyncTask) toServiceTask() *service.OpenAIImageAsyncTask {
	return &service.OpenAIImageAsyncTask{
		ID:             t.ID,
		UserID:         t.UserID,
		APIKeyID:       t.APIKeyID,
		Status:         service.OpenAIImageAsyncTaskStatus(t.Status),
		Request:        cloneOpenAIImageAsyncTaskRequest(t.Request),
		ResultStatus:   t.ResultStatus,
		ResultBody:     append([]byte(nil), t.ResultBody...),
		ResultMimeType: t.ResultMimeType,
		ErrorMessage:   t.ErrorMessage,
		CreatedAt:      timeFromUnix(t.CreatedAt),
		UpdatedAt:      timeFromUnix(t.UpdatedAt),
		FinishedAt:     timeFromUnix(t.FinishedAt),
	}
}

func cloneOpenAIImageAsyncTask(task *service.OpenAIImageAsyncTask) *service.OpenAIImageAsyncTask {
	if task == nil {
		return nil
	}
	cloned := *task
	cloned.Request = cloneOpenAIImageAsyncTaskRequest(task.Request)
	if task.ResultBody != nil {
		cloned.ResultBody = append([]byte(nil), task.ResultBody...)
	}
	return &cloned
}

func cloneOpenAIImageAsyncTaskRequest(req *service.OpenAIImageAsyncTaskRequest) *service.OpenAIImageAsyncTaskRequest {
	if req == nil {
		return nil
	}
	cloned := *req
	if req.Header != nil {
		cloned.Header = make(map[string][]string, len(req.Header))
		for key, values := range req.Header {
			cloned.Header[key] = append([]string(nil), values...)
		}
	}
	cloned.Body = append([]byte(nil), req.Body...)
	cloned.APIKey = cloneAPIKeyForOpenAIImageTask(req.APIKey)
	cloned.Subscription = cloneUserSubscriptionForOpenAIImageTask(req.Subscription)
	return &cloned
}

func cloneAPIKeyForOpenAIImageTask(apiKey *service.APIKey) *service.APIKey {
	if apiKey == nil {
		return nil
	}
	cloned := *apiKey
	cloned.IPWhitelist = append([]string(nil), apiKey.IPWhitelist...)
	cloned.IPBlacklist = append([]string(nil), apiKey.IPBlacklist...)
	cloned.User = cloneUserForOpenAIImageTask(apiKey.User)
	cloned.Group = cloneGroupForOpenAIImageTask(apiKey.Group)
	cloned.Key = ""
	return &cloned
}

func cloneUserForOpenAIImageTask(user *service.User) *service.User {
	if user == nil {
		return nil
	}
	cloned := *user
	cloned.AllowedGroups = append([]int64(nil), user.AllowedGroups...)
	cloned.BalanceNotifyExtraEmails = append([]service.NotifyEmailEntry(nil), user.BalanceNotifyExtraEmails...)
	cloned.GroupRates = cloneInt64Float64Map(user.GroupRates)
	cloned.APIKeys = nil
	cloned.Subscriptions = nil
	cloned.PasswordHash = ""
	cloned.TotpSecretEncrypted = nil
	return &cloned
}

func cloneGroupForOpenAIImageTask(group *service.Group) *service.Group {
	if group == nil {
		return nil
	}
	cloned := *group
	cloned.ModelRouting = cloneStringInt64SliceMap(group.ModelRouting)
	cloned.SupportedModelScopes = append([]string(nil), group.SupportedModelScopes...)
	cloned.MessagesDispatchModelConfig = group.MessagesDispatchModelConfig
	cloned.ModelsListConfig = group.ModelsListConfig
	cloned.AccountGroups = nil
	return &cloned
}

func cloneUserSubscriptionForOpenAIImageTask(sub *service.UserSubscription) *service.UserSubscription {
	if sub == nil {
		return nil
	}
	cloned := *sub
	cloned.User = nil
	cloned.Group = cloneGroupForOpenAIImageTask(sub.Group)
	cloned.AssignedByUser = nil
	return &cloned
}

func cloneInt64Float64Map(in map[int64]float64) map[int64]float64 {
	if in == nil {
		return nil
	}
	out := make(map[int64]float64, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneStringInt64SliceMap(in map[string][]int64) map[string][]int64 {
	if in == nil {
		return nil
	}
	out := make(map[string][]int64, len(in))
	for key, values := range in {
		out[key] = append([]int64(nil), values...)
	}
	return out
}

func openAIImageAsyncTaskKey(taskID string) string {
	return openAIImageAsyncTaskKeyPrefix + strings.TrimSpace(taskID)
}

func openAIImageAsyncTaskLockKey(taskID string) string {
	return openAIImageAsyncTaskLockKeyPrefix + strings.TrimSpace(taskID)
}

func unixOrZero(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

func timeFromUnix(ts int64) time.Time {
	if ts <= 0 {
		return time.Time{}
	}
	return time.Unix(ts, 0)
}

var _ service.OpenAIImageAsyncTaskStore = (*openAIImageAsyncTaskStore)(nil)
