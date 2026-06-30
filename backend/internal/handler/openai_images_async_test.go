package handler

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSnapshotOpenAIImageAsyncTaskRequestStripsSensitiveHeadersAndAsyncQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations?async=true&foo=bar", bytes.NewReader([]byte(`{"prompt":"draw"}`)))
	req.Header.Set("Authorization", "Bearer sk-secret")
	req.Header.Set("X-Api-Key", "sk-secret")
	req.Header.Set("X-Goog-Api-Key", "sk-secret")
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	apiKey := &service.APIKey{
		ID:     9,
		UserID: 7,
		Key:    "sk-secret",
		User: &service.User{
			ID:          7,
			Role:        service.RoleUser,
			Concurrency: 3,
		},
		Group: &service.Group{ID: 5, Platform: service.PlatformOpenAI, AllowImageGeneration: true},
	}
	got := snapshotOpenAIImageAsyncTaskRequest(c, apiKey, middleware2.AuthSubject{UserID: 7, Concurrency: 3}, []byte(`{"prompt":"draw"}`))
	header := http.Header(got.Header)

	require.Equal(t, "/v1/images/generations?foo=bar", got.URL)
	require.Equal(t, "application/json", header.Get("Content-Type"))
	require.Empty(t, header.Get("Authorization"))
	require.Empty(t, header.Get("X-Api-Key"))
	require.Empty(t, header.Get("X-Goog-Api-Key"))
	require.Equal(t, int64(9), got.APIKey.ID)
	require.Equal(t, service.RoleUser, got.Role)
}

func TestOpenAIImageAsyncEditRequestPreservesEditsURLAndMultipartHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "gpt-image-2"))
	require.NoError(t, writer.WriteField("prompt", "edit with reference"))
	part, err := writer.CreateFormFile("image[]", "reference.png")
	require.NoError(t, err)
	_, err = part.Write([]byte("fake-png"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/v1/images/edits?async=true&foo=bar", bytes.NewReader(body.Bytes()))
	req.Header.Set("Authorization", "Bearer sk-secret")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	c.Request = req

	apiKey := &service.APIKey{
		ID:     9,
		UserID: 7,
		Key:    "sk-secret",
		User: &service.User{
			ID:          7,
			Role:        service.RoleUser,
			Concurrency: 3,
		},
		Group: &service.Group{ID: 5, Platform: service.PlatformOpenAI, AllowImageGeneration: true},
	}
	got := snapshotOpenAIImageAsyncTaskRequest(c, apiKey, middleware2.AuthSubject{UserID: 7, Concurrency: 3}, body.Bytes())
	replayed := openAIImageAsyncHTTPRequest(c.Request.Context(), got)

	require.Equal(t, "/v1/images/edits?foo=bar", got.URL)
	require.Equal(t, "/v1/images/edits", replayed.URL.Path)
	require.Equal(t, "foo=bar", replayed.URL.RawQuery)
	require.Contains(t, replayed.Header.Get("Content-Type"), "multipart/form-data")
	require.Empty(t, replayed.Header.Get("Authorization"))
	require.Equal(t, int64(body.Len()), replayed.ContentLength)
}

func TestOpenAIImageAsyncErrorMessageDoesNotExposeUpstreamContext(t *testing.T) {
	message := openAIImageAsyncErrorMessage(
		[]byte(`{"error":{"message":"Upstream service temporarily unavailable"}}`),
		http.StatusBadGateway,
	)

	require.Equal(t, "图片生成失败，请稍后重试", message)
}

func TestOpenAIImageAsyncErrorMessageKeepsValidationMessage(t *testing.T) {
	message := openAIImageAsyncErrorMessage(
		[]byte(`{"error":{"message":"prompt is required"}}`),
		http.StatusBadRequest,
	)

	require.Equal(t, "prompt is required", message)
}
