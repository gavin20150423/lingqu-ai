package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestWriteRawUpstreamErrorPreservesStatusBodyAndRetryAfter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/", func(c *gin.Context) {
		if !writeRawUpstreamError(c, http.StatusServiceUnavailable, http.Header{
			"Content-Type": []string{"application/json"},
			"Retry-After":  []string{"7"},
		}, []byte(`{"error":{"type":"overloaded_error","message":"upstream busy"}}`)) {
			t.Fatal("expected raw upstream response to be written")
		}
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d, want 503", rec.Code)
	}
	if rec.Header().Get("Retry-After") != "7" {
		t.Fatalf("retry-after=%q, want 7", rec.Header().Get("Retry-After"))
	}
	if rec.Body.String() != `{"error":{"type":"overloaded_error","message":"upstream busy"}}` {
		t.Fatalf("body was rewritten: %s", rec.Body.String())
	}
}
