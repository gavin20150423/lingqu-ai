//nolint:unused // Legacy ?async=true worker retained for backward compatibility with existing clients.
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

const (
	openAIImageAsyncWorkerDequeueWait    = 5 * time.Second
	openAIImageAsyncWorkerLeaseTTL       = 10 * time.Minute
	openAIImageAsyncWorkerHeartbeatEvery = 30 * time.Second
)

func wantsOpenAIImageAsyncTask(c *gin.Context) bool {
	value := strings.ToLower(strings.TrimSpace(c.Query("async")))
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

func (h *OpenAIGatewayHandler) enqueueOpenAIImageAsyncTask(c *gin.Context, apiKey *service.APIKey, subject middleware2.AuthSubject, body []byte) {
	if h.imageAsyncTaskStore == nil {
		h.errorResponse(c, http.StatusServiceUnavailable, "service_unavailable", "图片异步任务存储暂不可用，请稍后再试")
		return
	}
	taskRequest := snapshotOpenAIImageAsyncTaskRequest(c, apiKey, subject, body)
	task, err := h.imageAsyncTaskStore.Enqueue(c.Request.Context(), taskRequest)
	if errors.Is(err, service.ErrOpenAIImageAsyncTaskStoreFull) {
		h.errorResponse(c, http.StatusTooManyRequests, "rate_limit_error", "图片任务队列已满，请稍后再试")
		return
	}
	if err != nil {
		h.errorResponse(c, http.StatusServiceUnavailable, "service_unavailable", "图片任务创建失败，请稍后再试")
		return
	}

	c.Header("Location", openAIImageTaskLocation(c.Request.URL, task.ID))
	c.JSON(http.StatusAccepted, gin.H{
		"data": gin.H{
			"task_id": task.ID,
			"status":  task.Status,
		},
	})
}

func (h *OpenAIGatewayHandler) startOpenAIImageAsyncWorker() {
	if h == nil || h.imageAsyncTaskStore == nil {
		return
	}
	go h.openAIImageAsyncWorkerLoop(context.Background())
}

func (h *OpenAIGatewayHandler) openAIImageAsyncWorkerLoop(ctx context.Context) {
	workerID := "openai-image-worker-" + uuid.NewString()
	_, _ = h.imageAsyncTaskStore.RequeueStaleRunning(ctx, openAIImageAsyncWorkerLeaseTTL)
	nextRecoverAt := time.Now().Add(openAIImageAsyncWorkerHeartbeatEvery)
	for {
		if time.Now().After(nextRecoverAt) {
			_, _ = h.imageAsyncTaskStore.RequeueStaleRunning(ctx, openAIImageAsyncWorkerLeaseTTL)
			nextRecoverAt = time.Now().Add(openAIImageAsyncWorkerHeartbeatEvery)
		}
		task, err := h.imageAsyncTaskStore.Dequeue(ctx, openAIImageAsyncWorkerDequeueWait)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		if task == nil {
			continue
		}
		h.runOpenAIImageAsyncTask(ctx, workerID, task)
	}
}

func (h *OpenAIGatewayHandler) runOpenAIImageAsyncTask(ctx context.Context, workerID string, task *service.OpenAIImageAsyncTask) {
	if h.imageAsyncTaskStore == nil || task == nil {
		return
	}
	if task.Request == nil {
		_ = h.imageAsyncTaskStore.MarkFailed(ctx, task.ID, "图片任务请求已丢失，请重新提交")
		return
	}
	locked, err := h.imageAsyncTaskStore.MarkRunning(ctx, task.ID, workerID, openAIImageAsyncWorkerLeaseTTL)
	if err != nil || !locked {
		return
	}
	stopHeartbeat := h.startOpenAIImageAsyncHeartbeat(ctx, task.ID, workerID)
	defer stopHeartbeat()

	h.executeOpenAIImageAsyncTask(ctx, task.ID, task.Request)
}

func (h *OpenAIGatewayHandler) executeOpenAIImageAsyncTask(ctx context.Context, taskID string, taskRequest *service.OpenAIImageAsyncTaskRequest) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = openAIImageAsyncHTTPRequest(ctx, taskRequest)
	c.Set(string(middleware2.ContextKeyAPIKey), taskRequest.APIKey)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{
		UserID:      taskRequest.Subject.UserID,
		Concurrency: taskRequest.Subject.Concurrency,
	})
	if taskRequest.Role != "" {
		c.Set(string(middleware2.ContextKeyUserRole), taskRequest.Role)
	}
	if taskRequest.Subscription != nil {
		c.Set(string(middleware2.ContextKeySubscription), taskRequest.Subscription)
	}
	if taskRequest.Endpoint != "" {
		c.Set(ctxKeyInboundEndpoint, taskRequest.Endpoint)
	}

	h.Images(c)

	statusCode := rec.Code
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	bodyBytes := rec.Body.Bytes()
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		h.logOpenAIImageAsyncUpstreamFailure(c, taskID, statusCode)
		_ = h.imageAsyncTaskStore.MarkFailed(ctx, taskID, openAIImageAsyncErrorMessage(bodyBytes, statusCode))
		return
	}
	if !json.Valid(bodyBytes) {
		_ = h.imageAsyncTaskStore.MarkFailed(ctx, taskID, "图片任务完成但返回内容不是有效 JSON")
		return
	}
	_ = h.imageAsyncTaskStore.MarkCompleted(ctx, taskID, statusCode, bodyBytes, rec.Header().Get("Content-Type"))
}

func (h *OpenAIGatewayHandler) startOpenAIImageAsyncHeartbeat(ctx context.Context, taskID string, workerID string) func() {
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(openAIImageAsyncWorkerHeartbeatEvery)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_ = h.imageAsyncTaskStore.RenewRunning(ctx, taskID, workerID, openAIImageAsyncWorkerLeaseTTL)
			case <-done:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
	var stopped bool
	return func() {
		if stopped {
			return
		}
		stopped = true
		close(done)
	}
}

func snapshotOpenAIImageAsyncTaskRequest(parent *gin.Context, apiKey *service.APIKey, subject middleware2.AuthSubject, body []byte) *service.OpenAIImageAsyncTaskRequest {
	req := &service.OpenAIImageAsyncTaskRequest{
		Method:   http.MethodPost,
		APIKey:   apiKey,
		Subject:  service.OpenAIImageAsyncTaskSubject{UserID: subject.UserID, Concurrency: subject.Concurrency},
		Endpoint: GetInboundEndpoint(parent),
		Body:     append([]byte(nil), body...),
	}
	if parent != nil && parent.Request != nil {
		req.Method = parent.Request.Method
		if parent.Request.URL != nil {
			copiedURL := *parent.Request.URL
			values := copiedURL.Query()
			values.Del("async")
			copiedURL.RawQuery = values.Encode()
			req.URL = copiedURL.String()
		}
		req.Header = parent.Request.Header.Clone()
		delete(req.Header, "Authorization")
		delete(req.Header, "X-Api-Key")
		delete(req.Header, "X-Goog-Api-Key")
	}
	if req.URL == "" {
		req.URL = "/v1/images/generations"
	}
	if apiKey != nil && apiKey.User != nil {
		req.Role = apiKey.User.Role
	}
	if subscription, ok := middleware2.GetSubscriptionFromContext(parent); ok {
		req.Subscription = subscription
	}
	return req
}

func openAIImageAsyncHTTPRequest(ctx context.Context, taskRequest *service.OpenAIImageAsyncTaskRequest) *http.Request {
	if ctx == nil {
		ctx = context.Background()
	}
	if taskRequest != nil && taskRequest.APIKey != nil && taskRequest.APIKey.Group != nil {
		ctx = context.WithValue(ctx, ctxkey.Group, taskRequest.APIKey.Group)
	}
	method := http.MethodPost
	target := "/v1/images/generations"
	var body []byte
	if taskRequest != nil {
		if strings.TrimSpace(taskRequest.Method) != "" {
			method = taskRequest.Method
		}
		if strings.TrimSpace(taskRequest.URL) != "" {
			target = taskRequest.URL
		}
		body = taskRequest.Body
	}
	req := httptest.NewRequest(method, target, bytes.NewReader(body)).WithContext(ctx)
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	req.ContentLength = int64(len(body))
	if taskRequest != nil && taskRequest.Header != nil {
		req.Header = cloneOpenAIImageAsyncHeader(taskRequest.Header)
	}
	return req
}

func cloneOpenAIImageAsyncHeader(header map[string][]string) http.Header {
	if header == nil {
		return nil
	}
	out := make(http.Header, len(header))
	for key, values := range header {
		out[key] = append([]string(nil), values...)
	}
	return out
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

func (h *OpenAIGatewayHandler) logOpenAIImageAsyncUpstreamFailure(c *gin.Context, taskID string, statusCode int) {
	upstreamMessage := openAIImageAsyncUpstreamErrorMessage(c)
	upstreamStatus := openAIImageAsyncUpstreamStatusCode(c)
	if upstreamMessage == "" && upstreamStatus <= 0 {
		return
	}
	logger.L().Warn("openai.images.async_task_failed_with_upstream_context",
		zap.String("task_id", taskID),
		zap.Int("client_status_code", statusCode),
		zap.Int("upstream_status_code", upstreamStatus),
		zap.String("upstream_message", upstreamMessage),
	)
}

func openAIImageAsyncErrorMessage(body []byte, statusCode int) string {
	if statusCode == http.StatusTooManyRequests {
		return "图片生成请求过于频繁，请稍后重试"
	}
	if statusCode == http.StatusRequestEntityTooLarge {
		return "图片或请求内容过大，请压缩后重试"
	}
	if statusCode >= http.StatusInternalServerError {
		return "图片生成失败，请稍后重试"
	}
	if len(body) > 0 && gjson.ValidBytes(body) {
		for _, path := range []string{"error.message", "message"} {
			if extracted := strings.TrimSpace(gjson.GetBytes(body, path).String()); extracted != "" {
				if isOpenAIImageAsyncInternalErrorMessage(extracted) {
					return "图片生成失败，请稍后重试"
				}
				return extracted
			}
		}
	}
	if statusCode > 0 {
		return http.StatusText(statusCode)
	}
	return "图片任务执行失败"
}

func isOpenAIImageAsyncInternalErrorMessage(message string) bool {
	message = strings.TrimSpace(message)
	if message == "" {
		return false
	}
	lower := strings.ToLower(message)
	internalMarkers := []string{
		"upstream",
		"temporarily unavailable",
		"bad gateway",
		"gateway timeout",
		"service unavailable",
		"failed to fetch",
		"fetch failed",
		"networkerror",
		"cors",
		"proxy",
		"nginx",
		"cloudflare",
		"multipart/form-data",
		"/v1/images/",
		"/images/",
	}
	for _, marker := range internalMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return strings.Contains(message, "跨域") ||
		strings.Contains(message, "反向代理") ||
		strings.Contains(message, "网关") ||
		strings.Contains(message, "上游")
}

func openAIImageAsyncUpstreamErrorMessage(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if v, ok := c.Get(service.OpsUpstreamErrorMessageKey); ok {
		if message, ok := v.(string); ok {
			return strings.TrimSpace(message)
		}
	}
	return ""
}

func openAIImageAsyncUpstreamStatusCode(c *gin.Context) int {
	if c == nil {
		return 0
	}
	if v, ok := c.Get(service.OpsUpstreamStatusCodeKey); ok {
		switch value := v.(type) {
		case int:
			return value
		case int64:
			return int(value)
		}
	}
	return 0
}

func (h *OpenAIGatewayHandler) ImageTask(c *gin.Context) {
	if h.imageAsyncTaskStore == nil {
		h.errorResponse(c, http.StatusServiceUnavailable, "service_unavailable", "图片异步任务存储暂不可用，请稍后再试")
		return
	}
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
	task, err := h.imageAsyncTaskStore.Get(c.Request.Context(), taskID)
	if errors.Is(err, service.ErrOpenAIImageAsyncTaskNotFound) || (err == nil && (task.UserID != subject.UserID || task.APIKeyID != apiKey.ID)) {
		h.errorResponse(c, http.StatusNotFound, "not_found_error", "图片任务不存在或已过期")
		return
	}
	if err != nil {
		h.errorResponse(c, http.StatusServiceUnavailable, "service_unavailable", "图片任务查询失败，请稍后再试")
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
	case service.OpenAIImageAsyncTaskCompleted:
		data["result"] = json.RawMessage(task.ResultBody)
	case service.OpenAIImageAsyncTaskFailed:
		message := task.ErrorMessage
		if message == "" {
			message = "图片任务执行失败"
		}
		data["error"] = gin.H{"message": message}
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}
