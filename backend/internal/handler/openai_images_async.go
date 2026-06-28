package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

const (
	openAIImageAsyncTaskRetention = 24 * time.Hour
	openAIImageAsyncTaskLimit     = 100
)

type openAIImageAsyncTaskStatus string

const (
	openAIImageAsyncTaskQueued    openAIImageAsyncTaskStatus = "queued"
	openAIImageAsyncTaskRunning   openAIImageAsyncTaskStatus = "running"
	openAIImageAsyncTaskCompleted openAIImageAsyncTaskStatus = "completed"
	openAIImageAsyncTaskFailed    openAIImageAsyncTaskStatus = "failed"
)

type openAIImageAsyncTask struct {
	ID             string
	UserID         int64
	APIKeyID       int64
	Status         openAIImageAsyncTaskStatus
	ResultStatus   int
	ResultBody     []byte
	ResultMimeType string
	ErrorMessage   string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	FinishedAt     time.Time
}

type openAIImageAsyncJob struct {
	Request      *http.Request
	APIKey       *service.APIKey
	Subject      middleware2.AuthSubject
	Role         string
	Subscription *service.UserSubscription
	Endpoint     string
}

type openAIImageAsyncTaskStore struct {
	mu    sync.Mutex
	tasks map[string]*openAIImageAsyncTask
}

var openAIImageAsyncTasks = &openAIImageAsyncTaskStore{
	tasks: make(map[string]*openAIImageAsyncTask),
}

func wantsOpenAIImageAsyncTask(c *gin.Context) bool {
	value := strings.ToLower(strings.TrimSpace(c.Query("async")))
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

func (s *openAIImageAsyncTaskStore) enqueue(userID int64, apiKeyID int64) (*openAIImageAsyncTask, bool) {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanupLocked(now)
	if len(s.tasks) >= openAIImageAsyncTaskLimit {
		s.evictOldestTerminalLocked()
	}
	if len(s.tasks) >= openAIImageAsyncTaskLimit {
		return nil, false
	}

	task := &openAIImageAsyncTask{
		ID:        "imgtask_" + uuid.NewString(),
		UserID:    userID,
		APIKeyID:  apiKeyID,
		Status:    openAIImageAsyncTaskQueued,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.tasks[task.ID] = task
	return cloneOpenAIImageAsyncTask(task), true
}

func (s *openAIImageAsyncTaskStore) markRunning(taskID string) {
	s.update(taskID, func(task *openAIImageAsyncTask, now time.Time) {
		task.Status = openAIImageAsyncTaskRunning
		task.UpdatedAt = now
	})
}

func (s *openAIImageAsyncTaskStore) markCompleted(taskID string, statusCode int, body []byte, contentType string) {
	s.update(taskID, func(task *openAIImageAsyncTask, now time.Time) {
		task.Status = openAIImageAsyncTaskCompleted
		task.ResultStatus = statusCode
		task.ResultBody = append(task.ResultBody[:0], body...)
		task.ResultMimeType = strings.TrimSpace(contentType)
		task.UpdatedAt = now
		task.FinishedAt = now
	})
}

func (s *openAIImageAsyncTaskStore) markFailed(taskID string, message string) {
	s.update(taskID, func(task *openAIImageAsyncTask, now time.Time) {
		task.Status = openAIImageAsyncTaskFailed
		task.ErrorMessage = strings.TrimSpace(message)
		if task.ErrorMessage == "" {
			task.ErrorMessage = "图片任务执行失败"
		}
		task.UpdatedAt = now
		task.FinishedAt = now
	})
}

func (s *openAIImageAsyncTaskStore) get(taskID string) (*openAIImageAsyncTask, bool) {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanupLocked(now)
	task, ok := s.tasks[taskID]
	if !ok {
		return nil, false
	}
	return cloneOpenAIImageAsyncTask(task), true
}

func (s *openAIImageAsyncTaskStore) update(taskID string, mutate func(*openAIImageAsyncTask, time.Time)) {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return
	}
	mutate(task, now)
}

func (s *openAIImageAsyncTaskStore) cleanupLocked(now time.Time) {
	for id, task := range s.tasks {
		anchor := task.UpdatedAt
		if !task.FinishedAt.IsZero() {
			anchor = task.FinishedAt
		}
		if now.Sub(anchor) > openAIImageAsyncTaskRetention {
			delete(s.tasks, id)
		}
	}
}

func (s *openAIImageAsyncTaskStore) evictOldestTerminalLocked() {
	var oldest *openAIImageAsyncTask
	for _, task := range s.tasks {
		if task.Status != openAIImageAsyncTaskCompleted && task.Status != openAIImageAsyncTaskFailed {
			continue
		}
		if oldest == nil || task.UpdatedAt.Before(oldest.UpdatedAt) {
			oldest = task
		}
	}
	if oldest != nil {
		delete(s.tasks, oldest.ID)
	}
}

func cloneOpenAIImageAsyncTask(task *openAIImageAsyncTask) *openAIImageAsyncTask {
	if task == nil {
		return nil
	}
	cloned := *task
	if task.ResultBody != nil {
		cloned.ResultBody = append([]byte(nil), task.ResultBody...)
	}
	return &cloned
}

func (h *OpenAIGatewayHandler) enqueueOpenAIImageAsyncTask(c *gin.Context, apiKey *service.APIKey, subject middleware2.AuthSubject, body []byte) {
	task, ok := openAIImageAsyncTasks.enqueue(subject.UserID, apiKey.ID)
	if !ok {
		h.errorResponse(c, http.StatusTooManyRequests, "rate_limit_error", "图片任务队列已满，请稍后再试")
		return
	}
	job := snapshotOpenAIImageAsyncJob(c, apiKey, subject, body)

	c.Header("Location", openAIImageTaskLocation(c.Request.URL, task.ID))
	c.JSON(http.StatusAccepted, gin.H{
		"data": gin.H{
			"task_id": task.ID,
			"status":  task.Status,
		},
	})

	go h.runOpenAIImageAsyncTask(job, task.ID)
}

func (h *OpenAIGatewayHandler) runOpenAIImageAsyncTask(job openAIImageAsyncJob, taskID string) {
	openAIImageAsyncTasks.markRunning(taskID)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = job.Request
	c.Set(string(middleware2.ContextKeyAPIKey), job.APIKey)
	c.Set(string(middleware2.ContextKeyUser), job.Subject)
	if job.Role != "" {
		c.Set(string(middleware2.ContextKeyUserRole), job.Role)
	}
	if job.Subscription != nil {
		c.Set(string(middleware2.ContextKeySubscription), job.Subscription)
	}
	if job.Endpoint != "" {
		c.Set(ctxKeyInboundEndpoint, job.Endpoint)
	}

	h.Images(c)

	statusCode := rec.Code
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	bodyBytes := rec.Body.Bytes()
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		openAIImageAsyncTasks.markFailed(taskID, openAIImageAsyncErrorMessage(bodyBytes, statusCode))
		return
	}
	if !json.Valid(bodyBytes) {
		openAIImageAsyncTasks.markFailed(taskID, "图片任务完成但返回内容不是有效 JSON")
		return
	}
	openAIImageAsyncTasks.markCompleted(taskID, statusCode, bodyBytes, rec.Header().Get("Content-Type"))
}

func snapshotOpenAIImageAsyncJob(parent *gin.Context, apiKey *service.APIKey, subject middleware2.AuthSubject, body []byte) openAIImageAsyncJob {
	job := openAIImageAsyncJob{
		Request:  cloneOpenAIImageAsyncRequest(parent, apiKey, body),
		APIKey:   apiKey,
		Subject:  subject,
		Endpoint: GetInboundEndpoint(parent),
	}
	if apiKey != nil && apiKey.User != nil {
		job.Role = apiKey.User.Role
	}
	if subscription, ok := middleware2.GetSubscriptionFromContext(parent); ok {
		job.Subscription = subscription
	}
	return job
}

func cloneOpenAIImageAsyncRequest(parent *gin.Context, apiKey *service.APIKey, body []byte) *http.Request {
	ctx := context.Background()
	if apiKey != nil && apiKey.Group != nil {
		ctx = context.WithValue(ctx, ctxkey.Group, apiKey.Group)
	}
	if parent != nil && parent.Request != nil {
		ctx = usageRecordContext(parent.Request.Context(), ctx)
	}

	req := parent.Request.Clone(ctx)
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	req.ContentLength = int64(len(body))

	if req.URL != nil {
		copiedURL := *req.URL
		values := copiedURL.Query()
		values.Del("async")
		copiedURL.RawQuery = values.Encode()
		req.URL = &copiedURL
	}
	return req
}

func openAIImageTaskLocation(current *url.URL, taskID string) string {
	if current == nil {
		return "/v1/images/tasks/" + taskID
	}
	prefix := "/v1"
	if !strings.HasPrefix(current.Path, "/v1/") {
		prefix = ""
	}
	return strings.TrimRight(prefix, "/") + "/images/tasks/" + taskID
}

func openAIImageAsyncErrorMessage(body []byte, statusCode int) string {
	if len(body) > 0 && gjson.ValidBytes(body) {
		for _, path := range []string{"error.message", "message"} {
			if message := strings.TrimSpace(gjson.GetBytes(body, path).String()); message != "" {
				return message
			}
		}
	}
	if statusCode > 0 {
		return http.StatusText(statusCode)
	}
	return "图片任务执行失败"
}

func (h *OpenAIGatewayHandler) ImageTask(c *gin.Context) {
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}

	taskID := strings.TrimSpace(c.Param("task_id"))
	task, ok := openAIImageAsyncTasks.get(taskID)
	if !ok || task.UserID != subject.UserID || task.APIKeyID != apiKey.ID {
		h.errorResponse(c, http.StatusNotFound, "not_found_error", "图片任务不存在或已过期")
		return
	}

	data := gin.H{
		"task_id": task.ID,
		"status":  task.Status,
	}
	if !task.CreatedAt.IsZero() {
		data["created_at"] = task.CreatedAt.Unix()
	}
	if !task.UpdatedAt.IsZero() {
		data["updated_at"] = task.UpdatedAt.Unix()
	}

	switch task.Status {
	case openAIImageAsyncTaskCompleted:
		data["result"] = json.RawMessage(task.ResultBody)
	case openAIImageAsyncTaskFailed:
		message := task.ErrorMessage
		if message == "" {
			message = "图片任务执行失败"
		}
		data["error"] = gin.H{"message": message}
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}
