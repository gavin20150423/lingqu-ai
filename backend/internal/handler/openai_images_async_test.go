package handler

import (
	"bytes"
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
