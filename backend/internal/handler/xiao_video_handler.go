package handler

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type XiaoVideoHandler struct {
	service *service.XiaoVideoService
	openAI  *OpenAIGatewayHandler
}

const xiaoVideoEnabledContextKey = "xiao_video_enabled"

func NewXiaoVideoHandler(svc *service.XiaoVideoService, openAI *OpenAIGatewayHandler) *XiaoVideoHandler {
	return &XiaoVideoHandler{service: svc, openAI: openAI}
}

func (h *XiaoVideoHandler) EnabledFor(c *gin.Context) bool {
	owner, ok := videoOwnerFromContext(c)
	enabled := ok && h != nil && h.service != nil && h.service.ActiveForGroup(c.Request.Context(), owner.GroupID)
	if enabled {
		c.Set(xiaoVideoEnabledContextKey, true)
	}
	return enabled
}

func (h *XiaoVideoHandler) Models(c *gin.Context) {
	owner, ok := h.requireEnabledOwner(c)
	if !ok {
		return
	}
	models, err := h.service.ListModels(c.Request.Context(), owner)
	if err != nil {
		videoError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"object": "list", "data": models})
}

func (h *XiaoVideoHandler) Upload(c *gin.Context) {
	owner, ok := h.requireEnabledOwner(c)
	if !ok {
		return
	}
	contentType := c.GetHeader("Content-Type")
	if !validVideoUploadContentType(contentType) {
		videoError(c, service.ErrVideoRequestInvalid)
		return
	}
	media, err := h.service.Upload(c.Request.Context(), owner, c.Request.Body, contentType)
	if err != nil {
		videoError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"media_id":   media.MediaID,
		"url":        h.service.PublicBaseURL() + "/v1/videos/uploads/" + media.MediaID + "/content",
		"type":       media.MediaType,
		"expires_at": media.ExpiresAt.UTC().Format(time.RFC3339),
	})
}

func (h *XiaoVideoHandler) Create(c *gin.Context) {
	owner, ok := h.requireEnabledOwner(c)
	if !ok {
		return
	}
	if !validVideoJSONContentType(c.GetHeader("Content-Type")) {
		videoError(c, service.ErrVideoRequestInvalid)
		return
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 2<<20))
	if err != nil || len(body) == 0 {
		videoError(c, service.ErrVideoRequestInvalid)
		return
	}
	if !h.checkSecurityAudit(c, body) {
		return
	}
	job, err := h.service.Create(c.Request.Context(), owner, body, c.GetHeader("Idempotency-Key"))
	if err != nil {
		videoError(c, err)
		return
	}
	location := "/v1/videos/jobs/" + job.JobID
	if strings.HasSuffix(strings.TrimRight(c.Request.URL.Path, "/"), "/videos") {
		location = "/v1/videos/" + job.JobID
	}
	if videoPreferRespondAsync(c.GetHeader("Prefer")) {
		c.Header("Preference-Applied", "respond-async")
	}
	c.Header("Location", location)
	c.JSON(http.StatusAccepted, gin.H{"job_id": job.JobID, "status": job.Status, "status_url": location})
}

func (h *XiaoVideoHandler) Get(c *gin.Context) {
	owner, ok := h.requireOwner(c)
	if !ok {
		return
	}
	jobID := c.Param("job_id")
	if jobID == "" {
		jobID = c.Param("request_id")
	}
	job, err := h.service.Get(c.Request.Context(), owner, jobID)
	if err != nil {
		videoError(c, err)
		return
	}
	c.JSON(http.StatusOK, publicVideoJob(job))
}

func (h *XiaoVideoHandler) List(c *gin.Context) {
	owner, ok := h.requireOwner(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	jobs, err := h.service.List(c.Request.Context(), owner, limit)
	if err != nil {
		videoError(c, err)
		return
	}
	data := make([]map[string]any, 0, len(jobs))
	for _, job := range jobs {
		data = append(data, publicVideoJob(job))
	}
	c.JSON(http.StatusOK, gin.H{"object": "list", "data": data})
}

func (h *XiaoVideoHandler) Cancel(c *gin.Context) {
	owner, ok := h.requireOwner(c)
	if !ok {
		return
	}
	job, err := h.service.Cancel(c.Request.Context(), owner, c.Param("job_id"))
	if err != nil {
		videoError(c, err)
		return
	}
	c.JSON(http.StatusOK, publicVideoJob(job))
}

func (h *XiaoVideoHandler) Content(c *gin.Context) {
	owner, ok := h.requireOwner(c)
	if !ok {
		return
	}
	jobID := c.Param("job_id")
	if jobID == "" {
		jobID = c.Param("request_id")
	}
	resp, err := h.service.OpenContent(c.Request.Context(), owner, jobID, c.GetHeader("Range"), c.Request.URL.RawQuery)
	if err != nil {
		videoError(c, err)
		return
	}
	proxyVideoResponse(c, resp, true)
}

func (h *XiaoVideoHandler) MediaContent(c *gin.Context) {
	owner, ok := h.requireOwner(c)
	if !ok {
		return
	}
	resp, err := h.service.OpenMedia(c.Request.Context(), owner, c.Param("media_id"), c.GetHeader("Range"), c.Request.URL.RawQuery)
	if err != nil {
		videoError(c, err)
		return
	}
	proxyVideoResponse(c, resp, true)
}

func (h *XiaoVideoHandler) requireOwner(c *gin.Context) (service.VideoOwner, bool) {
	owner, ok := videoOwnerFromContext(c)
	if !ok {
		videoError(c, infraerrors.New(http.StatusUnauthorized, "API_KEY_REQUIRED", "API key is required"))
		return service.VideoOwner{}, false
	}
	if h == nil || h.service == nil {
		videoError(c, service.ErrVideoExecutionDisabled)
		return service.VideoOwner{}, false
	}
	return owner, true
}

func (h *XiaoVideoHandler) requireEnabledOwner(c *gin.Context) (service.VideoOwner, bool) {
	owner, ok := h.requireOwner(c)
	if !ok {
		return service.VideoOwner{}, false
	}
	if !h.service.Enabled() {
		videoError(c, service.ErrVideoExecutionDisabled)
		return service.VideoOwner{}, false
	}
	enabled, _ := c.Get(xiaoVideoEnabledContextKey)
	if enabled != true && !h.service.ActiveForGroup(c.Request.Context(), owner.GroupID) {
		videoError(c, service.ErrVideoGenerationDisabled)
		return service.VideoOwner{}, false
	}
	return owner, true
}

func videoOwnerFromContext(c *gin.Context) (service.VideoOwner, bool) {
	key, ok := middleware.GetAPIKeyFromContext(c)
	if !ok || key == nil || key.ID <= 0 || key.UserID <= 0 {
		return service.VideoOwner{}, false
	}
	return service.VideoOwner{UserID: key.UserID, APIKeyID: key.ID, GroupID: key.GroupID}, true
}

func (h *XiaoVideoHandler) checkSecurityAudit(c *gin.Context, body []byte) bool {
	if h == nil || h.openAI == nil {
		return true
	}
	apiKey, ok := middleware.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil {
		videoError(c, infraerrors.New(http.StatusUnauthorized, "API_KEY_REQUIRED", "API key is required"))
		return false
	}
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		videoError(c, infraerrors.New(http.StatusInternalServerError, "USER_CONTEXT_REQUIRED", "user context is required"))
		return false
	}
	var request struct {
		Model string `json:"model"`
	}
	_ = json.Unmarshal(body, &request)
	reqLog := requestLogger(c, "handler.xiao_video.security_audit",
		zap.Int64("user_id", subject.UserID), zap.Int64("api_key_id", apiKey.ID), zap.String("model", request.Model))
	decision := h.openAI.checkSecurityAudit(c, reqLog, apiKey, subject, service.ContentModerationProtocolOpenAIImages, request.Model, body)
	if decision != nil && !decision.AllowNextStage {
		h.openAI.openAISecurityAuditError(c, decision)
		return false
	}
	return true
}

func publicVideoJob(job *service.VideoJob) map[string]any {
	out := map[string]any{
		"job_id":       job.JobID,
		"status":       job.Status,
		"model":        job.Model,
		"resolution":   job.Resolution,
		"duration":     job.Duration,
		"aspect_ratio": job.AspectRatio,
		"amount":       strconv.FormatFloat(job.Amount, 'f', 8, 64),
		"currency":     job.Currency,
		"created_at":   job.CreatedAt.Format(time.RFC3339),
		"updated_at":   job.UpdatedAt.Format(time.RFC3339),
		"status_url":   "/v1/videos/jobs/" + job.JobID,
	}
	if job.FinishedAt != nil {
		out["finished_at"] = job.FinishedAt.Format(time.RFC3339)
	}
	if strings.TrimSpace(job.SettlementStatus) != "" {
		out["settlement_status"] = job.SettlementStatus
	}
	if job.Status == "completed" {
		out["content_url"] = "/v1/videos/jobs/" + job.JobID + "/content"
	}
	if job.Status == "failed" {
		out["error"] = publicVideoFailure(job)
	}
	return out
}

func publicVideoFailure(job *service.VideoJob) map[string]any {
	failedAt := job.UpdatedAt
	if job.FinishedAt != nil {
		failedAt = *job.FinishedAt
	}
	failure := map[string]any{
		"code":      "VIDEO_GENERATION_FAILED",
		"message":   "video generation failed; use the task ID and error code for troubleshooting",
		"stage":     "upstream_generation",
		"task_id":   job.JobID,
		"failed_at": failedAt.Format(time.RFC3339),
	}
	var payload map[string]any
	if json.Unmarshal(job.UpstreamResponse, &payload) != nil {
		return failure
	}
	errorPayload, _ := payload["error"].(map[string]any)
	if len(errorPayload) == 0 {
		errorPayload, _ = payload["last_error"].(map[string]any)
	}
	code := firstVideoString(errorPayload, "code", "error_code")
	if code == "" {
		code = firstVideoString(payload, "error_code", "code")
	}
	if safeCode := safePublicVideoCode(code); safeCode != "" {
		failure["upstream_code"] = safeCode
	}
	if stage := safePublicVideoStage(firstVideoString(errorPayload, "stage"), firstVideoString(payload, "stage")); stage != "" {
		failure["stage"] = stage
	}
	if requestID := safePublicVideoIdentifier(
		firstVideoString(errorPayload, "request_id", "requestId", "trace_id", "traceId"),
		firstVideoString(payload, "request_id", "requestId", "trace_id", "traceId"),
	); requestID != "" {
		failure["request_id"] = requestID
	}
	if message := safePublicVideoMessage(code); message != "" {
		failure["message"] = message
	}
	return failure
}

func firstVideoString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func safePublicVideoCode(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if len(value) < 2 || len(value) > 64 {
		return ""
	}
	for _, character := range value {
		if (character < 'A' || character > 'Z') && (character < '0' || character > '9') && character != '_' && character != '-' {
			return ""
		}
	}
	switch value {
	case "VIDEO_REQUEST_INVALID", "VIDEO_RESOURCE_NOT_FOUND", "VIDEO_JOB_NOT_CANCELABLE",
		"VIDEO_MODEL_INVALID", "VIDEO_PROMPT_INVALID", "VIDEO_RESOLUTION_INVALID",
		"VIDEO_DURATION_INVALID", "VIDEO_ASPECT_RATIO_INVALID", "VIDEO_MEDIA_INVALID",
		"VIDEO_REFERENCE_IMAGE_STRENGTH_INVALID", "VIDEO_PROMPT_ASPECT_RATIO_CONFLICT", "VIDEO_PROMPT_DURATION_CONFLICT",
		"VIDEO_OPTION_UNSUPPORTED", "VIDEO_CAPACITY_EXHAUSTED", "VIDEO_GENERATION_FAILED",
		"CONTENT_POLICY_VIOLATION", "SAFETY_FILTER_TRIGGERED", "MODERATION_FAILED",
		"RATE_LIMIT_EXCEEDED", "INTERNAL_ERROR":
		return value
	default:
		return ""
	}
}

func safePublicVideoIdentifier(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if len(value) < 3 || len(value) > 128 {
			continue
		}
		valid := true
		for _, character := range value {
			if (character < 'A' || character > 'Z') && (character < 'a' || character > 'z') && (character < '0' || character > '9') && !strings.ContainsRune("._:-", character) {
				valid = false
				break
			}
		}
		if valid {
			return value
		}
	}
	return ""
}

func safePublicVideoStage(values ...string) string {
	for _, value := range values {
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "validation", "queued", "processing", "content", "settlement", "upstream_generation":
			return strings.ToLower(strings.TrimSpace(value))
		}
	}
	return ""
}

func safePublicVideoMessage(code string) string {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case "VIDEO_REQUEST_INVALID":
		return "video request is invalid"
	case "VIDEO_MODEL_INVALID":
		return "model is not supported"
	case "VIDEO_PROMPT_INVALID":
		return "prompt is invalid"
	case "VIDEO_RESOLUTION_INVALID":
		return "resolution is not supported by this model"
	case "VIDEO_DURATION_INVALID":
		return "duration is not supported by this model"
	case "VIDEO_ASPECT_RATIO_INVALID":
		return "aspect ratio is not supported by this model"
	case "VIDEO_MEDIA_INVALID":
		return "video media is invalid"
	case "VIDEO_REFERENCE_IMAGE_STRENGTH_INVALID":
		return "reference image strength must be low, medium, or high"
	case "VIDEO_PROMPT_ASPECT_RATIO_CONFLICT":
		return "prompt aspect ratio conflicts with the selected option"
	case "VIDEO_PROMPT_DURATION_CONFLICT":
		return "prompt duration conflicts with the selected option"
	case "VIDEO_OPTION_UNSUPPORTED":
		return "video option is not supported by this model"
	case "VIDEO_CAPACITY_EXHAUSTED":
		return "video capacity is temporarily exhausted"
	case "CONTENT_POLICY_VIOLATION", "SAFETY_FILTER_TRIGGERED", "MODERATION_FAILED":
		return "video generation was rejected by content safety checks"
	case "RATE_LIMIT_EXCEEDED":
		return "video generation rate limit was exceeded"
	case "INTERNAL_ERROR":
		return "video generation failed because of an upstream internal error"
	default:
		return ""
	}
}

func proxyVideoResponse(c *gin.Context, resp *http.Response, stream bool) {
	if resp == nil {
		videoError(c, service.ErrVideoUpstreamUnavailable)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	for _, name := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges", "Content-Disposition", "Cache-Control", "ETag", "Last-Modified"} {
		if value := resp.Header.Get(name); value != "" {
			c.Header(name, value)
		}
	}
	c.Status(resp.StatusCode)
	if stream {
		_, _ = io.CopyBuffer(c.Writer, resp.Body, make([]byte, 64*1024))
		return
	}
	_, _ = io.Copy(c.Writer, io.LimitReader(resp.Body, 2<<20))
}

func videoError(c *gin.Context, err error) {
	var upstream *service.VideoUpstreamError
	if errors.As(err, &upstream) {
		clientRequestID, path := "", ""
		if c != nil && c.Request != nil {
			clientRequestID = c.GetHeader("X-Client-Request-Id")
			if c.Request.URL != nil {
				path = c.Request.URL.Path
			}
		}
		service.LogVideoUpstreamErrorForRequest(upstream, clientRequestID, path)
		if requestID := safeVideoUpstreamRequestID(upstream.Header.Get("X-Request-Id")); requestID != "" {
			c.Header("X-Request-Id", requestID)
		}
		if retryAfter := upstream.Header.Get("Retry-After"); retryAfter != "" {
			c.Header("Retry-After", retryAfter)
		}
		status := upstream.Status
		if status == http.StatusPaymentRequired || status == http.StatusUnauthorized || status == http.StatusForbidden || status >= 500 {
			status = http.StatusServiceUnavailable
		}
		if status <= 0 {
			status = http.StatusBadGateway
		}
		code, message := "VIDEO_UPSTREAM_ERROR", "video upstream request failed"
		if safeCode, safeMessage, ok := safeVideoUpstreamError(upstream.Status, upstream.Body); ok {
			code, message = safeCode, safeMessage
		}
		c.JSON(status, gin.H{"error": gin.H{"type": "invalid_request_error", "code": code, "message": message}})
		return
	}
	status := infraerrors.Code(err)
	code := infraerrors.Reason(err)
	message := infraerrors.Message(err)
	if status == 0 {
		status = http.StatusInternalServerError
	}
	if strings.TrimSpace(code) == "" {
		code = "INTERNAL_ERROR"
	}
	if strings.TrimSpace(message) == "" {
		message = "internal error"
	}
	c.JSON(status, gin.H{"error": gin.H{"type": "invalid_request_error", "code": code, "message": message}})
}

func validVideoJSONContentType(value string) bool {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(value))
	return err == nil && strings.EqualFold(mediaType, "application/json")
}

func validVideoUploadContentType(value string) bool {
	mediaType, params, err := mime.ParseMediaType(strings.TrimSpace(value))
	return err == nil && strings.EqualFold(mediaType, "multipart/form-data") && strings.TrimSpace(params["boundary"]) != ""
}

func videoPreferRespondAsync(value string) bool {
	for _, preference := range strings.Split(value, ",") {
		name := strings.TrimSpace(strings.SplitN(preference, ";", 2)[0])
		if strings.EqualFold(name, "respond-async") {
			return true
		}
	}
	return false
}

func safeVideoUpstreamError(status int, body []byte) (string, string, bool) {
	// A few providers return an empty body for permission failures. Preserve
	// the actionable cause instead of reducing a definitive 403 to a generic
	// upstream failure message.
	if status == http.StatusForbidden {
		return "VIDEO_UPSTREAM_FORBIDDEN", "video upstream denied this model or API key permission", true
	}
	var envelope struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &envelope) != nil {
		return "", "", false
	}
	type publicError struct {
		status  int
		message string
	}
	allowed := map[string]publicError{
		"VIDEO_REQUEST_INVALID":                  {http.StatusBadRequest, "video request is invalid"},
		"VIDEO_RESOURCE_NOT_FOUND":               {http.StatusNotFound, "video resource not found"},
		"VIDEO_JOB_NOT_CANCELABLE":               {http.StatusConflict, "video job is not cancelable"},
		"VIDEO_MODEL_INVALID":                    {http.StatusUnprocessableEntity, "model is not supported"},
		"VIDEO_PROMPT_INVALID":                   {http.StatusUnprocessableEntity, "prompt is invalid"},
		"VIDEO_RESOLUTION_INVALID":               {http.StatusUnprocessableEntity, "resolution is not supported by this model"},
		"VIDEO_DURATION_INVALID":                 {http.StatusUnprocessableEntity, "duration is not supported by this model"},
		"VIDEO_ASPECT_RATIO_INVALID":             {http.StatusUnprocessableEntity, "aspect ratio is not supported by this model"},
		"VIDEO_MEDIA_INVALID":                    {http.StatusUnprocessableEntity, "video media is invalid"},
		"VIDEO_REFERENCE_IMAGE_STRENGTH_INVALID": {http.StatusUnprocessableEntity, "reference image strength must be low, medium, or high"},
		"VIDEO_PROMPT_ASPECT_RATIO_CONFLICT":     {http.StatusUnprocessableEntity, "prompt aspect ratio conflicts with the selected option"},
		"VIDEO_PROMPT_DURATION_CONFLICT":         {http.StatusUnprocessableEntity, "prompt duration conflicts with the selected option"},
		"VIDEO_OPTION_UNSUPPORTED":               {http.StatusUnprocessableEntity, "video option is not supported by this model"},
		"VIDEO_UPSTREAM_FORBIDDEN":               {http.StatusForbidden, "video upstream denied this model or API key permission"},
		"VIDEO_CAPACITY_EXHAUSTED":               {http.StatusTooManyRequests, "video capacity is temporarily exhausted"},
	}
	code := strings.TrimSpace(envelope.Error.Code)
	if code == "" {
		code = strings.TrimSpace(envelope.Code)
	}
	definition, ok := allowed[code]
	if !ok || definition.status != status {
		return safeUpstreamFallback(status, code, envelope.Error.Message, envelope.Message)
	}
	return code, definition.message, true
}

func safeUpstreamFallback(status int, code, nestedMessage, topMessage string) (string, string, bool) {
	// Provider-specific codes are useful to support staff, while provider text
	// is normalized to avoid leaking URLs, account IDs, or internal details.
	code = strings.TrimSpace(code)
	if !safeVideoUpstreamCode(code) {
		code = ""
	}
	message := strings.TrimSpace(nestedMessage)
	if message == "" {
		message = strings.TrimSpace(topMessage)
	}
	message = sanitizeVideoUpstreamMessage(message)
	if code == "" && message == "" {
		return "", "", false
	}
	if message == "" {
		message = "video upstream rejected the request"
	}
	if code == "" {
		code = "VIDEO_UPSTREAM_ERROR"
	}
	return code, message, true
}

func sanitizeVideoUpstreamMessage(value string) string {
	if strings.ContainsAny(value, "\r\n") {
		return ""
	}
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if value == "" || len(value) > 240 {
		return ""
	}
	for _, marker := range []string{"http://", "https://", "bearer ", "api_key", "account_id", "account id", "internal", "secret", "token"} {
		if strings.Contains(strings.ToLower(value), strings.ToLower(marker)) {
			return ""
		}
	}
	return value
}

func safeVideoUpstreamRequestID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 128 {
		return ""
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || strings.ContainsRune("_-.:", r) {
			continue
		}
		return ""
	}
	return value
}

func safeVideoUpstreamCode(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 80 {
		return false
	}
	lower := strings.ToLower(value)
	for _, marker := range []string{"http", "secret", "account", "internal", "api_key", "token"} {
		if strings.Contains(lower, marker) {
			return false
		}
	}
	for _, r := range value {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || strings.ContainsRune("_-.", r) {
			continue
		}
		return false
	}
	return true
}
