package handler

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// VideoWorkbenchHandler exposes the browser workbench through the user's
// authenticated session. The browser sends only an API key ID; the actual
// credential remains in the server-side account/key store.
type VideoWorkbenchHandler struct {
	video *service.XiaoVideoService
	keys  *service.APIKeyService
}

func NewVideoWorkbenchHandler(video *service.XiaoVideoService, keys *service.APIKeyService) *VideoWorkbenchHandler {
	return &VideoWorkbenchHandler{video: video, keys: keys}
}

func (h *VideoWorkbenchHandler) owner(c *gin.Context) (service.VideoOwner, bool) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		videoError(c, infraerrors.New(http.StatusUnauthorized, "AUTH_REQUIRED", "authentication is required"))
		return service.VideoOwner{}, false
	}
	if h == nil || h.video == nil || h.keys == nil {
		videoError(c, service.ErrVideoExecutionDisabled)
		return service.VideoOwner{}, false
	}
	value := strings.TrimSpace(c.GetHeader("X-Video-Key-Id"))
	if value == "" {
		value = strings.TrimSpace(c.Query("key_id"))
	}
	keyID, err := strconv.ParseInt(value, 10, 64)
	if err != nil || keyID <= 0 {
		videoError(c, infraerrors.New(http.StatusBadRequest, "VIDEO_KEY_REQUIRED", "a video API key must be selected"))
		return service.VideoOwner{}, false
	}
	key, err := h.keys.GetByID(c.Request.Context(), keyID)
	if err != nil || key == nil || key.UserID != subject.UserID {
		videoError(c, service.ErrVideoResourceNotFound)
		return service.VideoOwner{}, false
	}
	if !key.IsActive() || key.GroupID == nil {
		videoError(c, service.ErrVideoGenerationDisabled)
		return service.VideoOwner{}, false
	}
	return service.VideoOwner{UserID: subject.UserID, APIKeyID: key.ID, GroupID: key.GroupID}, true
}

func (h *VideoWorkbenchHandler) Models(c *gin.Context) {
	owner, ok := h.owner(c)
	if !ok {
		return
	}
	models, err := h.video.ListModels(c.Request.Context(), owner)
	if err != nil {
		videoError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"object": "list", "data": models})
}

func (h *VideoWorkbenchHandler) Capabilities(c *gin.Context) {
	owner, ok := h.owner(c)
	if !ok {
		return
	}
	schemaVersion, data, err := h.video.ListCapabilities(c.Request.Context(), owner)
	if err != nil {
		videoError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"schema_version": schemaVersion, "data": data})
}

func (h *VideoWorkbenchHandler) Bootstrap(c *gin.Context) {
	owner, ok := h.owner(c)
	if !ok {
		return
	}
	models, modelErr := h.video.ListModels(c.Request.Context(), owner)
	if modelErr != nil {
		videoError(c, modelErr)
		return
	}
	schemaVersion, capabilities, capabilityErr := h.video.ListCapabilities(c.Request.Context(), owner)
	if capabilityErr != nil {
		videoError(c, capabilityErr)
		return
	}
	c.Header("Cache-Control", "private, max-age=30")
	c.JSON(http.StatusOK, gin.H{
		"models":       gin.H{"object": "list", "data": models},
		"capabilities": gin.H{"schema_version": schemaVersion, "data": capabilities},
	})
}

func (h *VideoWorkbenchHandler) Upload(c *gin.Context) {
	owner, ok := h.owner(c)
	if !ok {
		return
	}
	contentType := c.GetHeader("Content-Type")
	if !validVideoUploadContentType(contentType) {
		videoError(c, service.ErrVideoRequestInvalid)
		return
	}
	media, err := h.video.Upload(c.Request.Context(), owner, c.Request.Body, contentType, c.GetHeader("Idempotency-Key"))
	if err != nil {
		videoError(c, err)
		return
	}
	payload := gin.H{
		"media_id":   media.MediaID,
		"url":        "/api/v1/video/uploads/" + media.MediaID + "/content",
		"type":       media.MediaType,
		"expires_at": media.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
	if media.MediaContentType != "" {
		payload["media_type"] = media.MediaContentType
	}
	if media.MIMEType != "" {
		payload["mime_type"] = media.MIMEType
	}
	if media.Container != "" {
		payload["container"] = media.Container
	}
	if media.DurationUS > 0 {
		payload["duration_us"] = media.DurationUS
	}
	c.JSON(http.StatusCreated, payload)
}

func (h *VideoWorkbenchHandler) Create(c *gin.Context) {
	owner, ok := h.owner(c)
	if !ok {
		return
	}
	if !validVideoJSONContentType(c.GetHeader("Content-Type")) {
		videoError(c, service.ErrVideoRequestInvalid)
		return
	}
	if !videoPreferRespondAsync(c.GetHeader("Prefer")) {
		videoError(c, infraerrors.New(http.StatusBadRequest, "ASYNC_REQUIRED", "Prefer: respond-async is required"))
		return
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 2<<20))
	if err != nil || len(body) == 0 {
		videoError(c, service.ErrVideoRequestInvalid)
		return
	}
	job, err := h.video.Create(c.Request.Context(), owner, body, c.GetHeader("Idempotency-Key"))
	if err != nil {
		videoError(c, err)
		return
	}
	location := "/api/v1/video/jobs/" + job.JobID
	c.Header("Preference-Applied", "respond-async")
	c.Header("Location", location)
	c.JSON(http.StatusAccepted, gin.H{"job_id": job.JobID, "status": job.Status, "status_url": location})
}

func (h *VideoWorkbenchHandler) List(c *gin.Context) {
	owner, ok := h.owner(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	jobs, err := h.video.List(c.Request.Context(), owner, limit)
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

func (h *VideoWorkbenchHandler) Get(c *gin.Context) {
	owner, ok := h.owner(c)
	if !ok {
		return
	}
	job, err := h.video.Get(c.Request.Context(), owner, c.Param("job_id"))
	if err != nil {
		videoError(c, err)
		return
	}
	c.JSON(http.StatusOK, publicVideoJob(job))
}

func (h *VideoWorkbenchHandler) Cancel(c *gin.Context) {
	owner, ok := h.owner(c)
	if !ok {
		return
	}
	job, err := h.video.Cancel(c.Request.Context(), owner, c.Param("job_id"))
	if err != nil {
		videoError(c, err)
		return
	}
	c.JSON(http.StatusOK, publicVideoJob(job))
}

func (h *VideoWorkbenchHandler) Content(c *gin.Context) {
	owner, ok := h.owner(c)
	if !ok {
		return
	}
	resp, err := h.video.OpenContent(c.Request.Context(), owner, c.Param("job_id"), c.GetHeader("Range"), c.Request.URL.RawQuery)
	if err != nil {
		videoError(c, err)
		return
	}
	proxyVideoResponse(c, resp, true)
}

func (h *VideoWorkbenchHandler) MediaContent(c *gin.Context) {
	owner, ok := h.owner(c)
	if !ok {
		return
	}
	resp, err := h.video.OpenMedia(c.Request.Context(), owner, c.Param("media_id"), c.GetHeader("Range"), c.Request.URL.RawQuery)
	if err != nil {
		videoError(c, err)
		return
	}
	proxyVideoResponse(c, resp, true)
}
