package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/stretchr/testify/require"
)

func TestVideoUpstreamDiagnosticSanitizers(t *testing.T) {
	require.Equal(t, "req_123:attempt-2", sanitizeVideoUpstreamRequestID(" req_123:attempt-2 "))
	require.Empty(t, sanitizeVideoUpstreamRequestID("https://secret.example/request?id=9"))
	require.Equal(t, "unsupported image codec", sanitizeVideoUpstreamDiagnostic(" unsupported  image codec "))
	require.Empty(t, sanitizeVideoUpstreamDiagnostic("unsupported codec\naccount 99"))
	require.Empty(t, sanitizeVideoUpstreamDiagnostic("provider https://secret.example failed"))
	require.Empty(t, sanitizeVideoUpstreamDiagnostic("token abc123 was rejected"))
}

func TestLogVideoUpstreamErrorForRequestDoesNotLeakSensitiveValues(t *testing.T) {
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	LogVideoUpstreamErrorForRequest(&VideoUpstreamError{
		Status: http.StatusUnprocessableEntity,
		Header: http.Header{"X-Request-Id": []string{"https://secret.example/request?id=9"}},
		Body:   []byte(`{"error":{"code":"SECRET_ACCOUNT_999","message":"account 999 at https://secret.example failed","request_id":"internal secret"}}`),
	}, "Bearer secret-token", "/v1/videos/generations")

	var entry map[string]any
	require.NoError(t, json.Unmarshal(output.Bytes(), &entry))
	require.Equal(t, float64(http.StatusUnprocessableEntity), entry["status"])
	require.Equal(t, "/v1/videos/generations", entry["path"])
	require.Len(t, entry["body_sha256"], 64)
	require.Empty(t, entry["request_id"])
	require.Empty(t, entry["upstream_request_id"])
	require.NotContains(t, output.String(), "secret.example")
	require.NotContains(t, output.String(), "SECRET_ACCOUNT_999")
	require.NotContains(t, output.String(), "account 999")
}

type videoAccountRepoStub struct {
	AccountRepository
	accounts []Account
}

func (r *videoAccountRepoStub) GetByID(_ context.Context, id int64) (*Account, error) {
	for i := range r.accounts {
		if r.accounts[i].ID == id {
			account := r.accounts[i]
			return &account, nil
		}
	}
	return nil, errors.New("account not found")
}

func (r *videoAccountRepoStub) ListModelAvailabilityCandidates(_ context.Context, groupID *int64, platforms []string, _ bool) ([]Account, error) {
	result := make([]Account, 0, len(r.accounts))
	for _, account := range r.accounts {
		if len(platforms) > 0 && account.Platform != platforms[0] {
			continue
		}
		if !openAIStickyAccountMatchesGroup(&account, groupID) {
			continue
		}
		result = append(result, account)
	}
	return result, nil
}

func (r *videoAccountRepoStub) ListSchedulableByGroupIDAndPlatform(ctx context.Context, groupID int64, platform string) ([]Account, error) {
	return r.ListSchedulableByGroupIDAndPlatforms(ctx, groupID, []string{platform})
}

func (r *videoAccountRepoStub) ListSchedulableByGroupIDAndPlatforms(ctx context.Context, groupID int64, platforms []string) ([]Account, error) {
	candidates, err := r.ListModelAvailabilityCandidates(ctx, &groupID, platforms, true)
	if err != nil {
		return nil, err
	}
	result := make([]Account, 0, len(candidates))
	for _, account := range candidates {
		if account.IsSchedulable() {
			result = append(result, account)
		}
	}
	return result, nil
}

type videoHTTPUpstreamStub struct {
	do func(*http.Request, string, int64, int) (*http.Response, error)
}

func (s *videoHTTPUpstreamStub) Do(req *http.Request, proxyURL string, accountID int64, concurrency int) (*http.Response, error) {
	return s.do(req, proxyURL, accountID, concurrency)
}

func (s *videoHTTPUpstreamStub) DoWithTLS(req *http.Request, proxyURL string, accountID int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return s.Do(req, proxyURL, accountID, concurrency)
}

type videoRepositoryStub struct {
	media       map[string]*VideoMedia
	jobs        map[string]*VideoJob
	idempotency map[string]*VideoJob
	balance     float64
	frozen      float64
}

func newVideoRepositoryStub() *videoRepositoryStub {
	return &videoRepositoryStub{
		media:       make(map[string]*VideoMedia),
		jobs:        make(map[string]*VideoJob),
		idempotency: make(map[string]*VideoJob),
		balance:     100,
	}
}

func (r *videoRepositoryStub) CreateMedia(_ context.Context, media *VideoMedia) error {
	copy := *media
	r.media[media.MediaID] = &copy
	return nil
}

func (r *videoRepositoryStub) GetMediaForOwner(_ context.Context, mediaID string, apiKeyID int64) (*VideoMedia, error) {
	media := r.media[mediaID]
	if media == nil || media.APIKeyID != apiKeyID || !media.ExpiresAt.After(time.Now()) {
		return nil, ErrVideoResourceNotFound
	}
	copy := *media
	return &copy, nil
}

func (r *videoRepositoryStub) ReserveJob(_ context.Context, reservation VideoJobReservation) (*VideoJob, bool, error) {
	key := reservation.IdempotencyKey
	scopedKey := strconv.FormatInt(reservation.Owner.APIKeyID, 10) + ":" + key
	if key != "" {
		if existing := r.idempotency[scopedKey]; existing != nil {
			copy := *existing
			return &copy, false, nil
		}
	}
	if r.balance < reservation.PreauthorizationAmount {
		return nil, false, ErrVideoInsufficientBalance
	}
	r.balance -= reservation.PreauthorizationAmount
	r.frozen += reservation.PreauthorizationAmount
	job := &VideoJob{
		JobID:            reservation.JobID,
		UpstreamJobID:    videoCreatingUpstreamPrefix + reservation.JobID,
		AccountID:        reservation.AccountID,
		UserID:           reservation.Owner.UserID,
		APIKeyID:         reservation.Owner.APIKeyID,
		GroupID:          reservation.Owner.GroupID,
		RequestHash:      reservation.RequestHash,
		Model:            reservation.Model,
		Resolution:       reservation.Resolution,
		Duration:         reservation.Duration,
		AspectRatio:      reservation.AspectRatio,
		Status:           "pending",
		Amount:           reservation.PreauthorizationAmount,
		Currency:         "USD",
		SettlementStatus: "held",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	if key != "" {
		job.IdempotencyKey = &key
		r.idempotency[scopedKey] = job
	}
	r.jobs[job.JobID] = job
	copy := *job
	return &copy, true, nil
}

func (r *videoRepositoryStub) ReleaseJobReservation(_ context.Context, jobID string) error {
	job := r.jobs[jobID]
	if job != nil && job.SettlementStatus == "held" &&
		(strings.HasPrefix(job.UpstreamJobID, videoCreatingUpstreamPrefix) || strings.HasPrefix(job.UpstreamJobID, videoRetryableUpstreamPrefix)) {
		r.balance += job.Amount
		r.frozen -= job.Amount
		delete(r.jobs, jobID)
		if job.IdempotencyKey != nil {
			delete(r.idempotency, strconv.FormatInt(job.APIKeyID, 10)+":"+*job.IdempotencyKey)
		}
	}
	return nil
}

func (r *videoRepositoryStub) MarkJobReservationRetryable(_ context.Context, jobID string) error {
	job := r.jobs[jobID]
	if job != nil && strings.HasPrefix(job.UpstreamJobID, videoCreatingUpstreamPrefix) {
		job.UpstreamJobID = videoRetryableUpstreamPrefix + jobID
		job.UpdatedAt = time.Now()
	}
	return nil
}

func (r *videoRepositoryStub) ClaimJobReservationRetry(_ context.Context, jobID string, staleBefore time.Time) (bool, error) {
	job := r.jobs[jobID]
	if job == nil {
		return false, nil
	}
	if strings.HasPrefix(job.UpstreamJobID, videoRetryableUpstreamPrefix) ||
		(strings.HasPrefix(job.UpstreamJobID, videoCreatingUpstreamPrefix) && job.UpdatedAt.Before(staleBefore)) {
		job.UpstreamJobID = videoCreatingUpstreamPrefix + jobID
		job.UpdatedAt = time.Now()
		return true, nil
	}
	return false, nil
}

func (r *videoRepositoryStub) FinalizeJobAndReconcileHold(_ context.Context, finalization VideoJobFinalization) (*VideoJob, error) {
	job := r.jobs[finalization.JobID]
	if job == nil {
		return nil, ErrVideoResourceNotFound
	}
	job.UpstreamJobID = finalization.UpstreamJobID
	job.Status = finalization.Status
	upstreamAmount := finalization.UpstreamAmount
	job.UpstreamAmount = &upstreamAmount
	job.UpstreamCurrency = finalization.UpstreamCurrency
	if finalization.Resolution != "" {
		job.Resolution = finalization.Resolution
	}
	if finalization.Duration > 0 {
		job.Duration = finalization.Duration
	}
	if finalization.AspectRatio != "" {
		job.AspectRatio = finalization.AspectRatio
	}
	job.UpstreamResponse = append([]byte(nil), finalization.UpstreamResponse...)
	job.SettlementStatus = "held"
	switch finalization.Status {
	case "completed":
		r.frozen -= job.Amount
		job.SettlementStatus = "captured"
	case "failed", "canceled":
		r.balance += job.Amount
		r.frozen -= job.Amount
		job.SettlementStatus = "released"
	}
	job.UpdatedAt = time.Now()
	copy := *job
	return &copy, nil
}

func (r *videoRepositoryStub) GetJobForOwner(_ context.Context, jobID string, apiKeyID int64) (*VideoJob, error) {
	job := r.jobs[jobID]
	if job == nil || job.APIKeyID != apiKeyID {
		return nil, ErrVideoResourceNotFound
	}
	copy := *job
	return &copy, nil
}

func (r *videoRepositoryStub) ListJobsForOwner(_ context.Context, apiKeyID int64, limit int) ([]*VideoJob, error) {
	result := make([]*VideoJob, 0)
	for _, job := range r.jobs {
		if job.APIKeyID == apiKeyID && len(result) < limit {
			copy := *job
			result = append(result, &copy)
		}
	}
	return result, nil
}

func (r *videoRepositoryStub) ListActiveJobs(_ context.Context, _ int) ([]*VideoJob, error) {
	return nil, nil
}

func (r *videoRepositoryStub) UpdateJobAndSettle(_ context.Context, update VideoJobUpdate) (*VideoJob, error) {
	job := r.jobs[update.JobID]
	if job == nil {
		return nil, ErrVideoResourceNotFound
	}
	job.Status = update.Status
	if update.Resolution != "" {
		job.Resolution = update.Resolution
	}
	if update.Duration > 0 {
		job.Duration = update.Duration
	}
	if update.AspectRatio != "" {
		job.AspectRatio = update.AspectRatio
	}
	job.UpstreamResponse = append([]byte(nil), update.UpstreamResponse...)
	job.FinishedAt = update.FinishedAt
	job.UpdatedAt = time.Now()
	if job.SettlementStatus == "held" && update.Status == "completed" {
		job.SettlementStatus = "captured"
	} else if job.SettlementStatus == "held" && (update.Status == "failed" || update.Status == "canceled") {
		job.SettlementStatus = "released"
	}
	copy := *job
	return &copy, nil
}

func videoTestAccount(id, groupID int64, baseURL string) Account {
	return Account{
		ID:          id,
		Name:        "video-upstream",
		Platform:    PlatformXiaoAPI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 2,
		GroupIDs:    []int64{groupID},
		Credentials: map[string]any{
			"base_url":      baseURL,
			"api_key":       "upstream-secret",
			"model_mapping": map[string]any{"video-public": "video-upstream-v2"},
			XiaoVideoPricingCredentialKey: []any{
				map[string]any{
					"model":                  "video-public",
					"resolution":             "720p",
					"price_per_second":       0.75,
					"audio_price_per_second": 0.25,
					"default_resolution":     true,
					"default_duration":       20,
				},
			},
		},
	}
}

func seedanceVideoTestAccount(id, groupID int64, model string) Account {
	account := videoTestAccount(id, groupID, "https://upstream.example.test/v1")
	account.Credentials["model_mapping"] = map[string]any{model: model}
	account.Credentials[XiaoVideoPricingCredentialKey] = []any{
		map[string]any{
			"model":                  model,
			"resolution":             "480p",
			"price_per_second":       0.75,
			"audio_price_per_second": 0.25,
			"default_resolution":     true,
			"default_duration":       4,
		},
	}
	return account
}

func newVideoServiceForTest(repo VideoRepository, accountRepo *videoAccountRepoStub, upstream HTTPUpstream) *XiaoVideoService {
	cfg := &config.Config{VideoAPI: config.VideoAPIConfig{Enabled: true, PublicBaseURL: "https://video.example.test"}}
	gateway := &OpenAIGatewayService{accountRepo: accountRepo}
	return NewXiaoVideoService(repo, accountRepo, gateway, upstream, nil, nil, cfg)
}

func TestXiaoVideoAccountUsesIndependentPlatform(t *testing.T) {
	xiao := videoTestAccount(42, 7, "https://upstream.example/v1")
	require.True(t, xiao.IsXiaoAPI())

	xiao.Platform = PlatformOpenAI
	require.False(t, xiao.IsXiaoAPI())
}

func TestXiaoVideoOpenAISoraProtocolAdapter(t *testing.T) {
	account := videoTestAccount(42, 7, "https://api.video.aistarslab.com/openai")
	account.Credentials[XiaoVideoProtocolCredentialKey] = XiaoVideoProtocolOpenAISora

	body, err := rewriteVideoRequestForAccount(
		&account,
		[]byte(`{"model":"public-video","prompt":"Ocean waves","aspect_ratio":"16:9","start_frame_url":"https://example.test/start.jpg","end_frame_url":"https://example.test/end.jpg"}`),
		"12:provider-video",
		"720p",
		5,
	)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Equal(t, "12:provider-video", payload["model"])
	require.Equal(t, "5", payload["seconds"])
	require.Equal(t, "16:9", payload["size"])
	require.Equal(t, float64(1), payload["n"])
	metadata, ok := payload["metadata"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "720p", metadata["resolution"])
	require.Equal(t, "frames2video", metadata["mode_type"])
	require.Equal(t, []any{"https://example.test/start.jpg", "https://example.test/end.jpg"}, metadata["images"])

	normalized, err := decodeVideoUpstreamResponse(&account, []byte(`{"id":"task-1","status":"in_progress","seconds":"5","size":"16:9","metadata":{"resolution":"720p"}}`))
	require.NoError(t, err)
	require.Equal(t, "task-1", normalized["job_id"])
	require.Equal(t, "running", normalized["status"])
	require.Equal(t, "5", normalized["duration"])
	require.Equal(t, "16:9", normalized["aspect_ratio"])
	require.Equal(t, "720p", normalized["resolution"])
	require.Equal(t, "0", normalized["amount"])
	require.Equal(t, "CREDITS", normalized["currency"])
}

func TestXiaoVideoCustomJSONAdapterMapsProviderContract(t *testing.T) {
	account := videoTestAccount(42, 7, "https://provider.example.test/api")
	account.Credentials[XiaoVideoProtocolCredentialKey] = XiaoVideoProtocolCustomJSON
	account.Credentials[XiaoVideoAdapterCredentialKey] = map[string]any{
		"version": 1,
		"auth":    map[string]any{"type": "header", "header": "X-API-Key", "prefix": "Token "},
		"endpoints": map[string]any{
			"models":  map[string]any{"method": "GET", "path": "/catalog"},
			"create":  map[string]any{"method": "POST", "path": "/tasks"},
			"status":  map[string]any{"method": "GET", "path": "/tasks/{job_id}"},
			"content": map[string]any{"method": "GET", "path": "/tasks/{job_id}/file"},
		},
		"request": map[string]any{
			"fields": map[string]any{"model": "engine", "prompt": "input.text", "duration": "input.seconds"},
		},
		"response": map[string]any{
			"data_path":  "task",
			"fields":     map[string]any{"job_id": "id", "status": "state", "result_url": "video.url"},
			"status_map": map[string]any{"processing": "running", "done": "completed"},
		},
		"models": map[string]any{"items_path": "items", "id_path": "name"},
	}
	adapter, err := videoProviderAdapterForAccount(&account)
	require.NoError(t, err)
	endpoint, supported := adapter.Endpoint(videoOperationStatus, "task/one")
	require.True(t, supported)
	require.Equal(t, "/tasks/task%2Fone", endpoint.Path)
	require.Equal(t, http.MethodGet, endpoint.Method)
	require.Equal(t, "running", mustVideoStatus(t, adapter, []byte(`{"task":{"id":"job-1","state":"processing"}}`)))
	body, err := adapter.RewriteCreate(&account, []byte(`{"model":"public","prompt":"waves","duration":4}`), "provider-model", "720p", 8)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Equal(t, "provider-model", payload["engine"])
	require.Equal(t, map[string]any{"text": "waves", "seconds": float64(8)}, payload["input"])
	require.Equal(t, "Token provider-secret", customAdapterAuthHeader(t, adapter, "provider-secret"))
	models, err := adapter.DecodeModels([]byte(`{"items":[{"name":"model-a"}]}`))
	require.NoError(t, err)
	require.Equal(t, "model-a", models[0]["id"])
	resultURL, ok := adapter.ResultURL([]byte(`{"task":{"id":"job-1","state":"done","video":{"url":"https://cdn.example.test/video.mp4"}}}`))
	require.True(t, ok)
	require.Equal(t, "https://cdn.example.test/video.mp4", resultURL)
}

func TestXiaoVideoCTMOAIAdapterMapsSeedanceAndH3Contracts(t *testing.T) {
	account := videoTestAccount(42, 7, "https://video.ctmoai.com")
	account.Credentials[XiaoVideoProtocolCredentialKey] = XiaoVideoProtocolCTMOAI
	adapter, err := videoProviderAdapterForAccount(&account)
	require.NoError(t, err)
	create, ok := adapter.Endpoint(videoOperationCreate, "")
	require.True(t, ok)
	require.Equal(t, http.MethodPost, create.Method)
	require.Equal(t, "/v1/videos", create.Path)
	status, ok := adapter.Endpoint(videoOperationStatus, "task/1")
	require.True(t, ok)
	require.Equal(t, "/v1/videos/task%2F1", status.Path)
	models, err := adapter.DecodeModels([]byte(`{"data":[{"id":"minimax-h3-quantized-768p","resolution":"768p","durations_seconds":[4,10],"ratios":["16:9"],"max_images":4,"max_videos":0,"max_audios":0}]}`))
	require.NoError(t, err)
	require.Equal(t, map[string]any{"image": json.Number("4"), "video": json.Number("0"), "audio": json.Number("0")}, models[0]["max_references"])

	body, err := adapter.RewriteCreate(&account, []byte(`{"prompt":"A city at night","aspect_ratio":"16:9","guidances":{"image_reference":[{"image":{"url":"https://cdn.test/ref.jpg"}}]}}`), "minimax-h3-original-768p", "768p", 8)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Equal(t, "minimax-h3-original-768p", payload["model"])
	require.Equal(t, float64(8), payload["seconds"])
	require.Equal(t, "16:9", payload["aspect_ratio"])
	require.Equal(t, "https://cdn.test/ref.jpg", payload["images"].([]any)[0])

	cfBody, err := adapter.RewriteCreate(&account, []byte(`{"prompt":"Animate this","guidances":{"image_reference":[{"image":{"url":"https://cdn.test/ref.jpg"}}]}}`), "minimax-h3-original-cf-4k", "4K", 6)
	require.NoError(t, err)
	var cfPayload map[string]any
	require.NoError(t, json.Unmarshal(cfBody, &cfPayload))
	require.Equal(t, "https://cdn.test/ref.jpg", cfPayload["input_reference"])
	_, err = adapter.RewriteCreate(&account, []byte(`{"prompt":"Missing reference"}`), "minimax-h3-original-cf-4k", "4K", 6)
	require.ErrorIs(t, err, ErrVideoOptionUnsupported)

	fl2vBody, err := adapter.RewriteCreate(&account, []byte(`{"prompt":"Transition","start_frame_url":"https://cdn.test/start.jpg","end_frame_url":"https://cdn.test/end.jpg"}`), "minimax-h3-original-768p", "768p", 5)
	require.NoError(t, err)
	var fl2vPayload map[string]any
	require.NoError(t, json.Unmarshal(fl2vBody, &fl2vPayload))
	require.Equal(t, "fl2v", fl2vPayload["workflow_id"])
	require.Equal(t, []any{"https://cdn.test/start.jpg", "https://cdn.test/end.jpg"}, fl2vPayload["images"])

	normalized, err := adapter.DecodeJob([]byte(`{"task_id":"task-1","status":"in_progress","seconds":8,"aspect_ratio":"16:9"}`))
	require.NoError(t, err)
	require.Equal(t, "task-1", normalized["job_id"])
	require.Equal(t, "running", normalized["status"])

	_, err = adapter.RewriteCreate(&account, []byte(`{"prompt":"Too long"}`), "minimax-h3-quantized-768p", "768p", 11)
	require.ErrorIs(t, err, ErrVideoOptionUnsupported)
	_, err = adapter.RewriteCreate(&account, []byte(`{"prompt":"Video reference","guidances":{"video_reference_base":[{"video":{"url":"https://cdn.test/ref.mp4"}}]}}`), "minimax-h3-quantized-768p", "768p", 8)
	require.ErrorIs(t, err, ErrVideoOptionUnsupported)
}

func TestCTMOAIVideoAdapterDecodesUploadCollections(t *testing.T) {
	adapter := ctmoaiVideoAdapter{}

	for _, tc := range []struct {
		name string
		raw  string
		want string
	}{
		{name: "canonical URL", raw: `{"url":"https://video.ctmoai.com/sd-media/images/direct.png"}`, want: "https://video.ctmoai.com/sd-media/images/direct.png"},
		{name: "image collection", raw: `{"images":["https://video.ctmoai.com/sd-media/images/reference.png?date=2026-08-28"]}`, want: "https://video.ctmoai.com/sd-media/images/reference.png?date=2026-08-28"},
		{name: "video collection", raw: `{"videos":["https://video.ctmoai.com/sd-media/videos/reference.mp4"]}`, want: "https://video.ctmoai.com/sd-media/videos/reference.mp4"},
		{name: "audio collection", raw: `{"audios":["https://video.ctmoai.com/sd-media/audios/reference.mp3"]}`, want: "https://video.ctmoai.com/sd-media/audios/reference.mp3"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			decoded, err := adapter.DecodeUpload([]byte(tc.raw))
			require.NoError(t, err)
			require.Equal(t, tc.want, decoded["url"])
			require.NotEmpty(t, decoded["media_id"])
			require.Equal(t, "UPLOADED", decoded["type"])
		})
	}
}

func mustVideoStatus(t *testing.T, adapter videoProviderAdapter, raw []byte) string {
	t.Helper()
	value, err := adapter.DecodeJob(raw)
	require.NoError(t, err)
	return videoStringValue(value["status"])
}

func TestCTMOAIVideoAdapterMapsTransientAndTerminalStatuses(t *testing.T) {
	adapter := ctmoaiVideoAdapter{}
	for _, tc := range []struct {
		status string
		want   string
	}{
		{status: "unknown", want: "pending"},
		{status: "timeout", want: "failed"},
		{status: "timed_out", want: "failed"},
		{status: "expired", want: "failed"},
	} {
		normalized, err := adapter.DecodeJob([]byte(`{"task_id":"task-status","status":"` + tc.status + `"}`))
		require.NoError(t, err)
		require.Equal(t, tc.want, normalized["status"], tc.status)
	}
}

func customAdapterAuthHeader(t *testing.T, adapter videoProviderAdapter, apiKey string) string {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, "https://provider.example.test", nil)
	require.NoError(t, err)
	adapter.Authorize(req, apiKey)
	return req.Header.Get("X-Api-Key")
}

func TestXiaoVideoOpenAISoraEndToEndCompatibility(t *testing.T) {
	const groupID int64 = 7
	repo := newVideoRepositoryStub()
	account := videoTestAccount(42, groupID, "https://api.video.aistarslab.com/openai")
	account.Credentials[XiaoVideoProtocolCredentialKey] = XiaoVideoProtocolOpenAISora
	account.Credentials["model_mapping"] = map[string]any{"video-public": "12:provider-video"}
	accounts := &videoAccountRepoStub{accounts: []Account{account}}
	requests := 0
	upstream := &videoHTTPUpstreamStub{do: func(req *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
		require.Equal(t, int64(42), accountID)
		requests++
		switch requests {
		case 1:
			require.Equal(t, "Bearer upstream-secret", req.Header.Get("Authorization"))
			require.Equal(t, http.MethodPost, req.Method)
			require.Equal(t, "/openai/v1/videos", req.URL.Path)
			var payload map[string]any
			require.NoError(t, json.NewDecoder(req.Body).Decode(&payload))
			require.Equal(t, "12:provider-video", payload["model"])
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"id":"task-1","status":"queued","seconds":"5","size":"16:9"}`)),
			}, nil
		case 2:
			require.Equal(t, "Bearer upstream-secret", req.Header.Get("Authorization"))
			require.Equal(t, http.MethodGet, req.Method)
			require.Equal(t, "/openai/v1/videos/task-1", req.URL.Path)
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"id":"task-1","status":"completed","seconds":"5","size":"16:9","metadata":{"resolution":"720p","result_url":"https://cdn.example.test/result.mp4"}}`)),
			}, nil
		case 3:
			require.Equal(t, http.MethodGet, req.Method)
			require.Equal(t, "https://cdn.example.test/result.mp4", req.URL.String())
			require.Empty(t, req.Header.Get("Authorization"))
			require.Equal(t, "bytes=0-99", req.Header.Get("Range"))
			return &http.Response{
				StatusCode: http.StatusPartialContent,
				Header:     http.Header{"Content-Range": []string{"bytes 0-99/1000"}},
				Body:       io.NopCloser(strings.NewReader("video")),
			}, nil
		default:
			t.Fatalf("unexpected upstream request %d", requests)
			return nil, nil
		}
	}}
	svc := newVideoServiceForTest(repo, accounts, upstream)
	owner := VideoOwner{UserID: 11, APIKeyID: 22, GroupID: videoInt64Ptr(groupID)}

	job, err := svc.Create(context.Background(), owner, []byte(`{"model":"video-public","prompt":"Ocean waves","resolution":"720p","duration":5,"aspect_ratio":"16:9"}`), "aistartlab-1")
	require.NoError(t, err)
	require.Equal(t, "task-1", job.UpstreamJobID)
	require.Equal(t, "pending", job.Status)
	require.InDelta(t, 3.75, job.Amount, 0.00000001)
	require.NotNil(t, job.UpstreamAmount)
	require.Zero(t, *job.UpstreamAmount)

	job, err = svc.Get(context.Background(), owner, job.JobID)
	require.NoError(t, err)
	require.Equal(t, "completed", job.Status)
	require.Equal(t, "720p", job.Resolution)
	require.Equal(t, 5, job.Duration)

	content, err := svc.OpenContent(context.Background(), owner, job.JobID, "bytes=0-99", "")
	require.NoError(t, err)
	require.Equal(t, http.StatusPartialContent, content.StatusCode)
	require.Equal(t, "bytes 0-99/1000", content.Header.Get("Content-Range"))
	require.NoError(t, content.Body.Close())
	require.Equal(t, 3, requests)
}

func TestOpenAISoraResultURLRejectsUnsafeValues(t *testing.T) {
	value, ok := openAISoraResultURL([]byte(`{"metadata":{"result_url":"https://cdn.example.test/result.mp4"}}`))
	require.True(t, ok)
	require.Equal(t, "https://cdn.example.test/result.mp4", value)

	for _, raw := range []string{
		`{}`,
		`{"metadata":{"result_url":"javascript:alert(1)"}}`,
		`{"metadata":{"result_url":"/relative.mp4"}}`,
	} {
		_, ok := openAISoraResultURL([]byte(raw))
		require.False(t, ok)
	}
}

func TestXiaoVideoOpenAISoraRejectsUnsupportedOperations(t *testing.T) {
	const groupID int64 = 7
	repo := newVideoRepositoryStub()
	account := videoTestAccount(42, groupID, "https://api.video.aistarslab.com/openai")
	account.Credentials[XiaoVideoProtocolCredentialKey] = XiaoVideoProtocolOpenAISora
	accounts := &videoAccountRepoStub{accounts: []Account{account}}
	svc := newVideoServiceForTest(repo, accounts, &videoHTTPUpstreamStub{do: func(*http.Request, string, int64, int) (*http.Response, error) {
		t.Fatal("unsupported operations must not reach upstream")
		return nil, nil
	}})
	owner := VideoOwner{UserID: 11, APIKeyID: 22, GroupID: videoInt64Ptr(groupID)}

	_, err := svc.Upload(context.Background(), owner, strings.NewReader("media"), "multipart/form-data; boundary=x")
	require.ErrorIs(t, err, ErrVideoUploadUnsupported)

	repo.jobs["vidjob-ai"] = &VideoJob{JobID: "vidjob-ai", UpstreamJobID: "task-ai", AccountID: 42, UserID: 11, APIKeyID: 22, Status: "pending", Amount: 1, Currency: "USD", SettlementStatus: "held"}
	_, err = svc.Cancel(context.Background(), owner, "vidjob-ai")
	require.ErrorIs(t, err, ErrVideoJobNotCancelable)
}

func TestXiaoVideoSeedanceAudioCapabilityMatrix(t *testing.T) {
	for _, model := range []string{"seedance-2.0", "seedance-2.0-fast", "seedance-2.0-mini"} {
		require.True(t, seedanceSupportsGeneratedAudio(model), model)
	}
	require.False(t, seedanceSupportsGeneratedAudio("other-video-model"))
}

func TestXiaoVideoNormalizesAndValidatesAudio(t *testing.T) {
	svc := newVideoServiceForTest(newVideoRepositoryStub(), &videoAccountRepoStub{}, nil)
	owner := VideoOwner{UserID: 11, APIKeyID: 22, GroupID: videoInt64Ptr(7)}

	decode := func(body string) (map[string]any, string, error) {
		rewritten, _, hash, _, err := svc.rewriteGenerationRequest(context.Background(), owner, []byte(body))
		if err != nil {
			return nil, "", err
		}
		var request map[string]any
		require.NoError(t, json.Unmarshal(rewritten, &request))
		return request, hash, nil
	}

	omitted, omittedHash, err := decode(`{"model":"seedance-2.0","prompt":"waves"}`)
	require.NoError(t, err)
	require.Equal(t, false, omitted["audio"])

	explicitFalse, falseHash, err := decode(`{"model":"seedance-2.0","prompt":"waves","audio":false}`)
	require.NoError(t, err)
	require.Equal(t, false, explicitFalse["audio"])
	require.Equal(t, omittedHash, falseHash)

	explicitTrue, trueHash, err := decode(`{"model":"seedance-2.0","prompt":"waves","audio":true}`)
	require.NoError(t, err)
	require.Equal(t, true, explicitTrue["audio"])
	require.NotEqual(t, falseHash, trueHash)

	for _, invalid := range []string{
		`{"model":"seedance-2.0","prompt":"waves","audio":"true"}`,
		`{"model":"seedance-2.0","prompt":"waves","audio":1}`,
		`{"model":"seedance-2.0","prompt":"waves","audio":null}`,
	} {
		_, _, err := decode(invalid)
		require.ErrorIs(t, err, ErrVideoRequestInvalid)
	}
}

func TestXiaoVideoPreservesCapabilityDrivenGenerationOptions(t *testing.T) {
	svc := newVideoServiceForTest(newVideoRepositoryStub(), &videoAccountRepoStub{}, nil)
	owner := VideoOwner{UserID: 11, APIKeyID: 22, GroupID: videoInt64Ptr(7)}

	rewritten, _, _, _, err := svc.rewriteGenerationRequest(context.Background(), owner, []byte(`{"model":"seedance-2.0","prompt":"waves","generation_mode":"text_to_video","quality":"standard","watermark":false}`))
	require.NoError(t, err)
	var request map[string]any
	require.NoError(t, json.Unmarshal(rewritten, &request))
	require.Equal(t, "text_to_video", request["generation_mode"])
	require.Equal(t, "standard", request["quality"])
	require.Equal(t, false, request["watermark"])

	_, _, _, _, err = svc.rewriteGenerationRequest(context.Background(), owner, []byte(`{"model":"seedance-2.0","prompt":"waves","watermark":"false"}`))
	require.ErrorIs(t, err, ErrVideoRequestInvalid)
}

func TestXiaoVideoDetectsReferenceVideoGuidance(t *testing.T) {
	svc := newVideoServiceForTest(newVideoRepositoryStub(), &videoAccountRepoStub{}, nil)
	owner := VideoOwner{UserID: 11, APIKeyID: 22, GroupID: videoInt64Ptr(7)}

	_, meta, _, _, err := svc.rewriteGenerationRequest(
		context.Background(),
		owner,
		[]byte(`{"model":"seedance-2.0","prompt":"waves","guidances":{"video_reference_base":[{"video":{"url":"https://example.test/reference.mp4","type":"UPLOADED"}}]}}`),
	)
	require.NoError(t, err)
	require.True(t, meta.ReferenceVideo)

	_, meta, _, _, err = svc.rewriteGenerationRequest(
		context.Background(),
		owner,
		[]byte(`{"model":"seedance-2.0","prompt":"waves","guidances":{"image_reference":[{"image":{"url":"https://example.test/reference.png","type":"UPLOADED"}}]}}`),
	)
	require.NoError(t, err)
	require.False(t, meta.ReferenceVideo)
}

func TestXiaoVideoAcceptsProviderSecondsAlias(t *testing.T) {
	svc := newVideoServiceForTest(newVideoRepositoryStub(), &videoAccountRepoStub{}, nil)
	owner := VideoOwner{UserID: 11, APIKeyID: 22, GroupID: videoInt64Ptr(7)}
	rewritten, meta, _, _, err := svc.rewriteGenerationRequest(
		context.Background(), owner,
		[]byte(`{"model":"seedance-2.0","prompt":"waves","seconds":6}`),
	)
	require.NoError(t, err)
	require.Equal(t, 6, meta.Duration)
	var request map[string]any
	require.NoError(t, json.Unmarshal(rewritten, &request))
	require.Equal(t, float64(6), request["duration"])
	_, _, _, _, err = svc.rewriteGenerationRequest(
		context.Background(), owner,
		[]byte(`{"model":"seedance-2.0","prompt":"waves","duration":5,"seconds":6}`),
	)
	require.ErrorIs(t, err, ErrVideoRequestInvalid)
}

func TestXiaoVideoNormalizesReferenceImageStrengthAndOrder(t *testing.T) {
	svc := newVideoServiceForTest(newVideoRepositoryStub(), &videoAccountRepoStub{}, nil)
	owner := VideoOwner{UserID: 11, APIKeyID: 22, GroupID: videoInt64Ptr(7)}

	rewritten, _, _, _, err := svc.rewriteGenerationRequest(
		context.Background(),
		owner,
		[]byte(`{"model":"seedance-2.0","prompt":"waves","guidances":{"image_reference":[{"image":{"url":"https://example.test/a.png","type":"UPLOADED"},"strength":"HIGH","order":9},{"image":{"url":"https://example.test/b.png","type":"UPLOADED"},"strength":"AUTO","order":8}]}}`),
	)
	require.NoError(t, err)
	var request map[string]any
	require.NoError(t, json.Unmarshal(rewritten, &request))
	guidances, ok := request["guidances"].(map[string]any)
	require.True(t, ok)
	items, ok := guidances["image_reference"].([]any)
	require.True(t, ok)
	first, ok := items[0].(map[string]any)
	require.True(t, ok)
	second, ok := items[1].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "HIGH", first["strength"])
	require.EqualValues(t, 0, first["order"])
	require.Equal(t, "MID", second["strength"])
	require.EqualValues(t, 1, second["order"])

	_, _, _, _, err = svc.rewriteGenerationRequest(
		context.Background(),
		owner,
		[]byte(`{"model":"seedance-2.0","prompt":"waves","guidances":{"image_reference":[{"image":{"url":"https://example.test/a.png","type":"UPLOADED"},"strength":"strong"}]}}`),
	)
	require.ErrorIs(t, err, ErrVideoReferenceImageStrengthInvalid)
}

func TestXiaoVideoRejectsPromptParameterConflictsAndFrameImageMix(t *testing.T) {
	svc := newVideoServiceForTest(newVideoRepositoryStub(), &videoAccountRepoStub{}, nil)
	owner := VideoOwner{UserID: 11, APIKeyID: 22, GroupID: videoInt64Ptr(7)}

	_, _, _, _, err := svc.rewriteGenerationRequest(context.Background(), owner, []byte(`{"model":"seedance-2.0","prompt":"9:16 portrait","aspect_ratio":"16:9","duration":8}`))
	require.ErrorIs(t, err, ErrVideoPromptAspectRatioConflict)
	_, _, _, _, err = svc.rewriteGenerationRequest(context.Background(), owner, []byte(`{"model":"seedance-2.0","prompt":"总时长 15 秒","aspect_ratio":"16:9","duration":8}`))
	require.ErrorIs(t, err, ErrVideoPromptDurationConflict)
	_, _, _, _, err = svc.rewriteGenerationRequest(context.Background(), owner, []byte(`{"model":"seedance-2.0","prompt":"keep subject","start_frame_url":"https://example.test/start.png","guidances":{"image_reference":[{"image":{"url":"https://example.test/ref.png","type":"UPLOADED"},"strength":"MID"}]}}`))
	require.ErrorIs(t, err, ErrVideoOptionUnsupported)
}

func TestXiaoVideoSeedanceAudioReachesUpstream(t *testing.T) {
	const groupID int64 = 7
	models := []string{"seedance-2.0", "seedance-2.0-fast", "seedance-2.0-mini"}
	for _, model := range models {
		t.Run(model, func(t *testing.T) {
			repo := newVideoRepositoryStub()
			accounts := &videoAccountRepoStub{accounts: []Account{seedanceVideoTestAccount(42, groupID, model)}}
			var captured map[string]any
			upstream := &videoHTTPUpstreamStub{do: func(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
				require.NoError(t, json.NewDecoder(req.Body).Decode(&captured))
				return &http.Response{
					StatusCode: http.StatusAccepted,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{"job_id":"up-audio-job","status":"pending","amount":"1","currency":"USD"}`)),
				}, nil
			}}
			svc := newVideoServiceForTest(repo, accounts, upstream)

			job, err := svc.Create(
				context.Background(),
				VideoOwner{UserID: 11, APIKeyID: 22, GroupID: videoInt64Ptr(groupID)},
				[]byte(`{"model":"`+model+`","prompt":"Ocean waves","resolution":"480p","duration":4,"aspect_ratio":"16:9","audio":true}`),
				"seedance-audio-"+model,
			)
			require.NoError(t, err)
			require.Equal(t, model, captured["model"])
			require.Equal(t, true, captured["audio"])
			require.InDelta(t, 4.0, job.Amount, 0.00000001)
			require.NotEmpty(t, job.RequestHash)
		})
	}
}

func TestXiaoVideoCreateBindsMediaAccountAndMapsModel(t *testing.T) {
	const groupID int64 = 7
	repo := newVideoRepositoryStub()
	repo.media["vidmedia_local"] = &VideoMedia{
		MediaID:         "vidmedia_local",
		UpstreamMediaID: "up-media",
		AccountID:       42,
		UserID:          11,
		APIKeyID:        22,
		UpstreamURL:     "https://upstream.example.test/media/up-media/content",
		ExpiresAt:       time.Now().Add(time.Hour),
	}
	accounts := &videoAccountRepoStub{accounts: []Account{videoTestAccount(42, groupID, "https://upstream.example.test/v1")}}
	var captured map[string]any
	var capturedAccountID int64
	upstream := &videoHTTPUpstreamStub{do: func(req *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
		capturedAccountID = accountID
		require.Equal(t, "/v1/videos/generations", req.URL.Path)
		require.Equal(t, "Bearer upstream-secret", req.Header.Get("Authorization"))
		require.Equal(t, "respond-async", req.Header.Get("Prefer"))
		require.NotEmpty(t, req.Header.Get("Idempotency-Key"))
		require.NoError(t, json.NewDecoder(req.Body).Decode(&captured))
		return &http.Response{
			StatusCode: http.StatusAccepted,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"job_id":"up-job-secret","status":"pending","amount":"2.00000000","currency":"USD"}`)),
		}, nil
	}}
	svc := newVideoServiceForTest(repo, accounts, upstream)
	job, err := svc.Create(context.Background(), VideoOwner{UserID: 11, APIKeyID: 22, GroupID: videoInt64Ptr(groupID)}, []byte(`{"model":"video-public","prompt":"make a video","duration":4,"start_frame_url":"https://app.example.test/api/v1/video/uploads/vidmedia_local/content"}`), "order-1")
	require.NoError(t, err)
	require.Equal(t, int64(42), capturedAccountID)
	require.Equal(t, "video-upstream-v2", captured["model"])
	require.Equal(t, "https://upstream.example.test/media/up-media/content", captured["start_frame_url"])
	require.Equal(t, int64(42), job.AccountID)
	require.Equal(t, 3.0, job.Amount)
	require.NotNil(t, job.UpstreamAmount)
	require.Equal(t, 2.0, *job.UpstreamAmount)
	require.InDelta(t, 97.0, repo.balance, 0.00000001)
	require.InDelta(t, 3.0, repo.frozen, 0.00000001)
	require.NotContains(t, string(job.UpstreamResponse), "video.example.test")
}

func TestXiaoVideoCreateSwitchesMediaAccountWithinSameUpstream(t *testing.T) {
	const groupID int64 = 7
	repo := newVideoRepositoryStub()
	repo.media["vidmedia_shared_upstream"] = &VideoMedia{
		MediaID:     "vidmedia_shared_upstream",
		AccountID:   41,
		UserID:      11,
		APIKeyID:    22,
		UpstreamURL: "https://upstream.example.test/media/reference",
		ExpiresAt:   time.Now().Add(time.Hour),
	}
	mediaAccount := seedanceVideoTestAccount(41, groupID, "other-model")
	selectedAccount := videoTestAccount(42, groupID, "https://upstream.example.test/v1")
	accounts := &videoAccountRepoStub{accounts: []Account{mediaAccount, selectedAccount}}
	var selectedAccountID int64
	upstream := &videoHTTPUpstreamStub{do: func(req *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
		selectedAccountID = accountID
		require.Equal(t, "/v1/videos/generations", req.URL.Path)
		return &http.Response{
			StatusCode: http.StatusAccepted,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"job_id":"up-job-shared","status":"pending","amount":"1"}`)),
		}, nil
	}}
	svc := newVideoServiceForTest(repo, accounts, upstream)

	job, err := svc.Create(
		context.Background(),
		VideoOwner{UserID: 11, APIKeyID: 22, GroupID: videoInt64Ptr(groupID)},
		[]byte(`{"model":"video-public","prompt":"use shared upstream media","start_frame_url":"https://video.example.test/v1/videos/uploads/vidmedia_shared_upstream/content"}`),
		"shared-upstream-account",
	)
	require.NoError(t, err)
	require.Equal(t, int64(42), selectedAccountID)
	require.Equal(t, int64(42), job.AccountID)
}

func TestXiaoVideoCreateDoesNotSwitchMediaAccountAcrossUpstreams(t *testing.T) {
	const groupID int64 = 7
	repo := newVideoRepositoryStub()
	repo.media["vidmedia_different_upstream"] = &VideoMedia{
		MediaID:     "vidmedia_different_upstream",
		AccountID:   41,
		UserID:      11,
		APIKeyID:    22,
		UpstreamURL: "https://first.example.test/media/reference",
		ExpiresAt:   time.Now().Add(time.Hour),
	}
	mediaAccount := seedanceVideoTestAccount(41, groupID, "other-model")
	selectedAccount := videoTestAccount(42, groupID, "https://second.example.test/v1")
	accounts := &videoAccountRepoStub{accounts: []Account{mediaAccount, selectedAccount}}
	requests := 0
	upstream := &videoHTTPUpstreamStub{do: func(*http.Request, string, int64, int) (*http.Response, error) {
		requests++
		return nil, errors.New("upstream must not be called")
	}}
	svc := newVideoServiceForTest(repo, accounts, upstream)

	_, err := svc.Create(
		context.Background(),
		VideoOwner{UserID: 11, APIKeyID: 22, GroupID: videoInt64Ptr(groupID)},
		[]byte(`{"model":"video-public","prompt":"do not cross upstream","start_frame_url":"https://video.example.test/v1/videos/uploads/vidmedia_different_upstream/content"}`),
		"different-upstream-account",
	)
	require.ErrorIs(t, err, ErrVideoUpstreamUnavailable)
	require.Zero(t, requests)
}

func TestXiaoVideoCreatePersistsResolvedUpstreamDefaults(t *testing.T) {
	const groupID int64 = 7
	repo := newVideoRepositoryStub()
	accounts := &videoAccountRepoStub{accounts: []Account{videoTestAccount(42, groupID, "https://upstream.example.test/v1")}}
	upstream := &videoHTTPUpstreamStub{do: func(req *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
		require.Equal(t, int64(42), accountID)
		require.Equal(t, http.MethodPost, req.Method)
		return &http.Response{
			StatusCode: http.StatusAccepted,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{
				"job_id":"up-job-defaults",
				"status":"pending",
				"amount":"2.00000000",
				"currency":"USD",
				"resolution":"720p",
				"duration":8,
				"aspect_ratio":"16:9"
			}`)),
		}, nil
	}}
	svc := newVideoServiceForTest(repo, accounts, upstream)

	job, err := svc.Create(
		context.Background(),
		VideoOwner{UserID: 11, APIKeyID: 22, GroupID: videoInt64Ptr(groupID)},
		[]byte(`{"model":"video-public","prompt":"use provider defaults"}`),
		"defaults-1",
	)
	require.NoError(t, err)
	require.Equal(t, "video-public", job.Model)
	require.Equal(t, "720p", job.Resolution)
	require.Equal(t, 8, job.Duration)
	require.Equal(t, "16:9", job.AspectRatio)
}

func TestXiaoVideoGetRefreshesResolvedUpstreamMetadata(t *testing.T) {
	const groupID int64 = 7
	repo := newVideoRepositoryStub()
	repo.jobs["vidjob_defaults"] = &VideoJob{
		JobID:            "vidjob_defaults",
		UpstreamJobID:    "up-job-defaults",
		AccountID:        42,
		UserID:           11,
		APIKeyID:         22,
		Model:            "video-public",
		Status:           "pending",
		Amount:           2,
		Currency:         "USD",
		SettlementStatus: "held",
	}
	accounts := &videoAccountRepoStub{accounts: []Account{videoTestAccount(42, groupID, "https://upstream.example.test/v1")}}
	upstream := &videoHTTPUpstreamStub{do: func(req *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
		require.Equal(t, int64(42), accountID)
		require.Equal(t, http.MethodGet, req.Method)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{
				"job_id":"up-job-defaults",
				"status":"running",
				"resolution":"480p",
				"duration":4,
				"aspect_ratio":"9:16"
			}`)),
		}, nil
	}}
	svc := newVideoServiceForTest(repo, accounts, upstream)

	job, err := svc.Get(
		context.Background(),
		VideoOwner{UserID: 11, APIKeyID: 22, GroupID: videoInt64Ptr(groupID)},
		"vidjob_defaults",
	)
	require.NoError(t, err)
	require.Equal(t, "video-public", job.Model)
	require.Equal(t, "480p", job.Resolution)
	require.Equal(t, 4, job.Duration)
	require.Equal(t, "9:16", job.AspectRatio)
}

func TestXiaoVideoCreateRejectsInsufficientBalanceBeforeCallingUpstream(t *testing.T) {
	const groupID int64 = 7
	repo := newVideoRepositoryStub()
	repo.balance = 14.99
	repo.media["vidmedia_balance"] = &VideoMedia{MediaID: "vidmedia_balance", AccountID: 42, UserID: 11, APIKeyID: 22, UpstreamURL: "https://upstream.example/media/balance", ExpiresAt: time.Now().Add(time.Hour)}
	accounts := &videoAccountRepoStub{accounts: []Account{videoTestAccount(42, groupID, "https://upstream.example/v1")}}
	requests := 0
	upstream := &videoHTTPUpstreamStub{do: func(_ *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
		requests++
		return nil, errors.New("upstream must not be called")
	}}
	svc := newVideoServiceForTest(repo, accounts, upstream)

	_, err := svc.Create(context.Background(), VideoOwner{UserID: 11, APIKeyID: 22, GroupID: videoInt64Ptr(groupID)}, []byte(`{"model":"video-public","prompt":"no balance","start_frame_url":"https://video.example.test/v1/videos/uploads/vidmedia_balance/content"}`), "insufficient-balance")
	require.ErrorIs(t, err, ErrVideoInsufficientBalance)
	require.Zero(t, requests)
	require.Empty(t, repo.jobs)
	require.InDelta(t, 14.99, repo.balance, 0.00000001)
	require.Zero(t, repo.frozen)
}

func TestXiaoVideoCreateRejectsMissingPreauthorizationBeforeCallingUpstream(t *testing.T) {
	const groupID int64 = 7
	repo := newVideoRepositoryStub()
	repo.media["vidmedia_pricing"] = &VideoMedia{MediaID: "vidmedia_pricing", AccountID: 42, UserID: 11, APIKeyID: 22, UpstreamURL: "https://upstream.example/media/pricing", ExpiresAt: time.Now().Add(time.Hour)}
	account := videoTestAccount(42, groupID, "https://upstream.example/v1")
	delete(account.Credentials, XiaoVideoPricingCredentialKey)
	accounts := &videoAccountRepoStub{accounts: []Account{account}}
	requests := 0
	upstream := &videoHTTPUpstreamStub{do: func(_ *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
		requests++
		return nil, errors.New("upstream must not be called")
	}}
	svc := newVideoServiceForTest(repo, accounts, upstream)

	_, err := svc.Create(context.Background(), VideoOwner{UserID: 11, APIKeyID: 22, GroupID: videoInt64Ptr(groupID)}, []byte(`{"model":"video-public","prompt":"missing pricing","start_frame_url":"https://video.example.test/v1/videos/uploads/vidmedia_pricing/content"}`), "missing-pricing")
	require.ErrorIs(t, err, ErrVideoPricingUnavailable)
	require.Zero(t, requests)
	require.Empty(t, repo.jobs)
}

func TestXiaoVideoCreateKeepsSellingPriceWhenUpstreamCostIsHigher(t *testing.T) {
	const groupID int64 = 7
	repo := newVideoRepositoryStub()
	repo.media["vidmedia_ceiling"] = &VideoMedia{MediaID: "vidmedia_ceiling", AccountID: 42, UserID: 11, APIKeyID: 22, UpstreamURL: "https://upstream.example/media/ceiling", ExpiresAt: time.Now().Add(time.Hour)}
	accounts := &videoAccountRepoStub{accounts: []Account{videoTestAccount(42, groupID, "https://upstream.example/v1")}}
	posts := 0
	deletes := 0
	upstream := &videoHTTPUpstreamStub{do: func(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
		switch req.Method {
		case http.MethodPost:
			posts++
			return &http.Response{StatusCode: http.StatusAccepted, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"job_id":"up-too-expensive","status":"pending","amount":"11"}`))}, nil
		case http.MethodDelete:
			deletes++
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`))}, nil
		default:
			return nil, errors.New("unexpected request")
		}
	}}
	svc := newVideoServiceForTest(repo, accounts, upstream)

	job, err := svc.Create(context.Background(), VideoOwner{UserID: 11, APIKeyID: 22, GroupID: videoInt64Ptr(groupID)}, []byte(`{"model":"video-public","prompt":"provider cost is independent","duration":4,"start_frame_url":"https://video.example.test/v1/videos/uploads/vidmedia_ceiling/content"}`), "cost-isolation")
	require.NoError(t, err)
	require.Equal(t, 1, posts)
	require.Zero(t, deletes)
	require.Equal(t, 3.0, job.Amount)
	require.NotNil(t, job.UpstreamAmount)
	require.Equal(t, 11.0, *job.UpstreamAmount)
	require.InDelta(t, 97.0, repo.balance, 0.00000001)
	require.InDelta(t, 3.0, repo.frozen, 0.00000001)
}

func TestXiaoVideoCreateRejectsCrossKeyMediaAndIdempotencyConflict(t *testing.T) {
	const groupID int64 = 7
	repo := newVideoRepositoryStub()
	repo.media["vidmedia_local"] = &VideoMedia{MediaID: "vidmedia_local", AccountID: 42, UserID: 11, APIKeyID: 22, UpstreamURL: "https://upstream.example/media", ExpiresAt: time.Now().Add(time.Hour)}
	accounts := &videoAccountRepoStub{accounts: []Account{videoTestAccount(42, groupID, "https://upstream.example/v1")}}
	requests := 0
	upstream := &videoHTTPUpstreamStub{do: func(_ *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
		requests++
		return &http.Response{StatusCode: http.StatusAccepted, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"job_id":"up-job","status":"pending","amount":"1"}`))}, nil
	}}
	svc := newVideoServiceForTest(repo, accounts, upstream)
	_, _, _, _, err := svc.rewriteGenerationRequest(context.Background(), VideoOwner{UserID: 11, APIKeyID: 99, GroupID: videoInt64Ptr(groupID)}, []byte(`{"model":"video-public","prompt":"x","start_frame_url":"https://video.example/v1/videos/uploads/vidmedia_local/content"}`))
	require.ErrorIs(t, err, ErrVideoResourceNotFound)

	owner := VideoOwner{UserID: 11, APIKeyID: 22, GroupID: videoInt64Ptr(groupID)}
	_, err = svc.Create(context.Background(), owner, []byte(`{"model":"video-public","prompt":"first","start_frame_url":"https://video.example/v1/videos/uploads/vidmedia_local/content"}`), "same-key")
	require.NoError(t, err)
	_, err = svc.Create(context.Background(), owner, []byte(`{"model":"video-public","prompt":"different","start_frame_url":"https://video.example/v1/videos/uploads/vidmedia_local/content"}`), "same-key")
	require.ErrorIs(t, err, ErrVideoIdempotencyConflict)
	require.Equal(t, 1, requests)
}

func TestXiaoVideoCreateScopesUpstreamIdempotencyByDownstreamKey(t *testing.T) {
	const groupID int64 = 7
	repo := newVideoRepositoryStub()
	accounts := &videoAccountRepoStub{accounts: []Account{videoTestAccount(42, groupID, "https://upstream.example/v1")}}
	upstreamKeys := make([]string, 0, 2)
	upstream := &videoHTTPUpstreamStub{do: func(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
		upstreamKeys = append(upstreamKeys, req.Header.Get("Idempotency-Key"))
		jobID := "up-job-" + strconv.Itoa(len(upstreamKeys))
		return &http.Response{
			StatusCode: http.StatusAccepted,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"job_id":"` + jobID + `","status":"pending","amount":"1"}`)),
		}, nil
	}}
	svc := newVideoServiceForTest(repo, accounts, upstream)
	body := []byte(`{"model":"video-public","prompt":"same client order id"}`)

	_, err := svc.Create(context.Background(), VideoOwner{UserID: 11, APIKeyID: 22, GroupID: videoInt64Ptr(groupID)}, body, "shared-order-id")
	require.NoError(t, err)
	_, err = svc.Create(context.Background(), VideoOwner{UserID: 11, APIKeyID: 23, GroupID: videoInt64Ptr(groupID)}, body, "shared-order-id")
	require.NoError(t, err)
	require.Len(t, upstreamKeys, 2)
	require.NotEmpty(t, upstreamKeys[0])
	require.NotEqual(t, upstreamKeys[0], upstreamKeys[1])
}

func TestXiaoVideoCreateMapsAccountExhaustionToVideoCapacityError(t *testing.T) {
	const groupID int64 = 7
	repo := newVideoRepositoryStub()
	accounts := &videoAccountRepoStub{}
	svc := newVideoServiceForTest(repo, accounts, &videoHTTPUpstreamStub{do: func(*http.Request, string, int64, int) (*http.Response, error) {
		t.Fatal("upstream must not be called")
		return nil, nil
	}})

	_, err := svc.Create(
		context.Background(),
		VideoOwner{UserID: 11, APIKeyID: 22, GroupID: videoInt64Ptr(groupID)},
		[]byte(`{"model":"video-public","prompt":"no capacity"}`),
		"no-capacity",
	)
	require.ErrorIs(t, err, ErrVideoCapacityExhausted)
}

func TestXiaoVideoCreateSelectsNextConfiguredAccountForModel(t *testing.T) {
	const groupID int64 = 7
	repo := newVideoRepositoryStub()
	unsupported := videoTestAccount(41, groupID, "https://first.example.test/v1")
	unsupported.Priority = 1
	unsupported.Credentials["model_mapping"] = map[string]any{"another-video-model": "upstream-a"}
	selected := videoTestAccount(42, groupID, "https://second.example.test/v1")
	selected.Priority = 2
	accounts := &videoAccountRepoStub{accounts: []Account{unsupported, selected}}

	var selectedAccountID int64
	upstream := &videoHTTPUpstreamStub{do: func(_ *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
		selectedAccountID = accountID
		return &http.Response{
			StatusCode: http.StatusAccepted,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"job_id":"up-job-selected","status":"pending","amount":"1"}`)),
		}, nil
	}}
	svc := newVideoServiceForTest(repo, accounts, upstream)

	job, err := svc.Create(
		context.Background(),
		VideoOwner{UserID: 11, APIKeyID: 22, GroupID: videoInt64Ptr(groupID)},
		[]byte(`{"model":"video-public","prompt":"select configured upstream"}`),
		"select-configured-account",
	)
	require.NoError(t, err)
	require.Equal(t, int64(42), selectedAccountID)
	require.Equal(t, int64(42), job.AccountID)
}

func TestXiaoVideoListModelsMapsAliases(t *testing.T) {
	const groupID int64 = 7
	accounts := &videoAccountRepoStub{accounts: []Account{videoTestAccount(42, groupID, "https://upstream.example/v1")}}
	upstream := &videoHTTPUpstreamStub{do: func(req *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
		require.Equal(t, int64(42), accountID)
		require.Equal(t, "/v1/models", req.URL.Path)
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"object":"list","data":[{"id":"video-upstream-v2","object":"model","resolutions":["720p"]}]}`))}, nil
	}}
	svc := newVideoServiceForTest(newVideoRepositoryStub(), accounts, upstream)
	models, err := svc.ListModels(context.Background(), VideoOwner{UserID: 11, APIKeyID: 22, GroupID: videoInt64Ptr(groupID)})
	require.NoError(t, err)
	require.Len(t, models, 1)
	require.Equal(t, "video-public", models[0]["id"])
	require.Equal(t, []string{"720p"}, models[0]["resolutions"])
	require.Equal(t, XiaoVideoProtocolNative, models[0]["capability_source"])
}

func TestPricedVideoModelsUsesStoredAIStartLabCapabilities(t *testing.T) {
	account := videoTestAccount(42, 7, "https://api.video.aistarslab.com/openai")
	account.Credentials[XiaoVideoProtocolCredentialKey] = XiaoVideoProtocolOpenAISora
	account.Credentials["model_mapping"] = map[string]any{"public-seedance": "47:seedance-2.0"}
	account.Credentials[xiaoVideoCapabilitiesCredentialKey] = map[string]any{
		"47:seedance-2.0": map[string]any{
			"durations":            []any{4, 5, 6},
			"aspect_ratios":        []any{"16:9", "9:16"},
			"default_aspect_ratio": "16:9",
			"supports_guidances":   true,
			"supports_start_frame": true,
			"max_references":       map[string]any{"image": 9, "video": 3, "audio": 3},
		},
	}
	rules := []XiaoVideoPricingRule{{Model: "public-seedance", Resolution: "720p", PricePerSecond: 1, DefaultResolution: true, DefaultDuration: 4}}
	models := pricedVideoModelsForAccount(&account, rules, []map[string]any{{"id": "47:seedance-2.0"}})
	require.Contains(t, models, "public-seedance")
	model := models["public-seedance"]
	require.Equal(t, []string{"720p"}, model["resolutions"])
	require.Equal(t, []any{float64(4), float64(5), float64(6)}, model["durations"])
	require.Equal(t, true, model["supports_guidances"])
	require.Equal(t, true, model["supports_start_frame"])
	require.Equal(t, false, model["supports_audio"])
	require.Equal(t, map[string]any{"image": float64(9), "video": float64(3), "audio": float64(3)}, model["max_references"])
}

func TestPricedVideoModelsPreferLiveCapabilityLimits(t *testing.T) {
	account := videoTestAccount(42, 7, "https://video.ctmoai.com")
	account.Credentials[XiaoVideoProtocolCredentialKey] = XiaoVideoProtocolCTMOAI
	account.Credentials["model_mapping"] = map[string]any{"h3": "minimax-h3-quantized-768p"}
	account.Credentials[xiaoVideoCapabilitiesCredentialKey] = map[string]any{
		"minimax-h3-quantized-768p": map[string]any{
			"durations":          []any{4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
			"max_references":     map[string]any{"image": 9, "video": 3, "audio": 3},
			"supports_guidances": true,
		},
	}
	rules := []XiaoVideoPricingRule{{Model: "h3", Resolution: "768p", PricePerSecond: 1, DefaultResolution: true, DefaultDuration: 4}}
	upstream := []map[string]any{{
		"id":             "minimax-h3-quantized-768p",
		"resolutions":    []any{"768p"},
		"durations":      []any{4, 5, 6, 7, 8, 9, 10},
		"max_references": map[string]any{"image": 4, "video": 0, "audio": 0},
	}}
	models := pricedVideoModelsForAccount(&account, rules, upstream)
	model := models["h3"]
	require.Equal(t, []any{4, 5, 6, 7, 8, 9, 10}, model["durations"])
	require.Equal(t, map[string]any{"image": 4, "video": 0, "audio": 0}, model["max_references"])
}

func TestPricedVideoModelsRestoresLegacyPerRequestPrice(t *testing.T) {
	account := videoTestAccount(42, 7, "https://video.ctmoai.com")
	account.Credentials[XiaoVideoProtocolCredentialKey] = XiaoVideoProtocolCTMOAI
	rules := []XiaoVideoPricingRule{{
		Model:             "seedance2.0-stable-full-480p",
		Resolution:        "480p",
		PricePerSecond:    1.125,
		DefaultResolution: true,
		DefaultDuration:   4,
	}}
	models := pricedVideoModelsForAccount(&account, rules, []map[string]any{{
		"id":          "seedance2.0-stable-full-480p",
		"resolutions": []any{"480p"},
		"pricing": map[string]any{
			"mode":   "per_task",
			"amount": json.Number("4.5"),
		},
	}})

	variants, ok := models["seedance2.0-stable-full-480p"]["pricing_variants"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, variants, 1)
	require.Equal(t, "per_request", variants[0]["billing_unit"])
	require.Equal(t, "4.5", variants[0]["unit_price"])
}

func TestPreferredVideoUpstreamModelKeepsLegacyBareMappingsDeterministic(t *testing.T) {
	t.Parallel()

	require.Equal(t, "47:seedance-2.0", preferredVideoUpstreamModel([]string{
		"51:seedance-2.0",
		"47:seedance-2.0",
		"48:seedance-2.0",
	}))
	require.Equal(t, "48:seedance-2.0", preferredVideoUpstreamModel([]string{
		"51:seedance-2.0",
		"48:seedance-2.0",
	}))
}

func TestMergePricedVideoModelMarksMixedCapabilitySource(t *testing.T) {
	target := map[string]any{"resolutions": []string{"720p"}, "capability_source": XiaoVideoProtocolNative}
	source := map[string]any{"resolutions": []string{"1080p"}, "capability_source": XiaoVideoProtocolOpenAISora}

	mergePricedVideoModel(target, source)

	require.Equal(t, "mixed", target["capability_source"])
	require.Equal(t, []string{"1080p", "720p"}, target["resolutions"])
}

func TestXiaoVideoListModelsEnablesSeedanceGeneratedAudio(t *testing.T) {
	const groupID int64 = 7
	tests := []struct {
		name          string
		publicModel   string
		upstreamModel string
	}{
		{name: "seedance-2.0", publicModel: "seedance-2.0", upstreamModel: "seedance-2.0"},
		{name: "seedance-2.0-fast", publicModel: "seedance-2.0-fast", upstreamModel: "seedance-2.0-fast"},
		{name: "seedance-2.0-mini", publicModel: "seedance-2.0-mini", upstreamModel: "seedance-2.0-mini"},
		{name: "dynamic public alias", publicModel: "customer-seedance", upstreamModel: "seedance-2.0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := seedanceVideoTestAccount(42, groupID, tt.publicModel)
			account.Credentials["model_mapping"] = map[string]any{tt.publicModel: tt.upstreamModel}
			accounts := &videoAccountRepoStub{accounts: []Account{account}}
			upstream := &videoHTTPUpstreamStub{do: func(*http.Request, string, int64, int) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body: io.NopCloser(strings.NewReader(`{"object":"list","data":[{"id":"` + tt.upstreamModel +
						`","object":"model","resolutions":["480p"],"supports_audio":false}]}`)),
				}, nil
			}}
			svc := newVideoServiceForTest(newVideoRepositoryStub(), accounts, upstream)

			models, err := svc.ListModels(context.Background(), VideoOwner{UserID: 11, APIKeyID: 22, GroupID: videoInt64Ptr(groupID)})
			require.NoError(t, err)
			require.Len(t, models, 1)
			require.Equal(t, tt.publicModel, models[0]["id"])
			require.Equal(t, true, models[0]["supports_audio"])
		})
	}
}

func TestXiaoVideoOpenMediaUsesBoundAccountAndForwardsRange(t *testing.T) {
	const groupID int64 = 7
	repo := newVideoRepositoryStub()
	repo.media["vidmedia_local"] = &VideoMedia{MediaID: "vidmedia_local", UpstreamMediaID: "up media", AccountID: 42, UserID: 11, APIKeyID: 22, UpstreamURL: "https://upstream.example/media", ExpiresAt: time.Now().Add(time.Hour)}
	boundAccount := videoTestAccount(42, groupID, "https://upstream.example/v1")
	boundAccount.Status = StatusDisabled
	boundAccount.Schedulable = false
	accounts := &videoAccountRepoStub{accounts: []Account{boundAccount}}
	upstream := &videoHTTPUpstreamStub{do: func(req *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
		require.Equal(t, int64(42), accountID)
		require.Equal(t, "/v1/videos/uploads/up media/content", req.URL.Path)
		require.Equal(t, "download=1", req.URL.RawQuery)
		require.Equal(t, "bytes=0-99", req.Header.Get("Range"))
		return &http.Response{StatusCode: http.StatusPartialContent, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("video"))}, nil
	}}
	svc := newVideoServiceForTest(repo, accounts, upstream)
	resp, err := svc.OpenMedia(context.Background(), VideoOwner{UserID: 11, APIKeyID: 22, GroupID: videoInt64Ptr(groupID)}, "vidmedia_local", "bytes=0-99", "download=1")
	require.NoError(t, err)
	require.Equal(t, http.StatusPartialContent, resp.StatusCode)
	require.NoError(t, resp.Body.Close())
}

func TestXiaoVideoCancelDoesNotReleaseUntilUpstreamConfirmsTerminalStatus(t *testing.T) {
	const groupID int64 = 7
	repo := newVideoRepositoryStub()
	repo.jobs["vidjob_local"] = &VideoJob{
		JobID:            "vidjob_local",
		UpstreamJobID:    "up-job",
		AccountID:        42,
		UserID:           11,
		APIKeyID:         22,
		Status:           "running",
		Amount:           2,
		Currency:         "USD",
		SettlementStatus: "held",
	}
	accounts := &videoAccountRepoStub{accounts: []Account{videoTestAccount(42, groupID, "https://upstream.example/v1")}}
	requests := 0
	upstream := &videoHTTPUpstreamStub{do: func(req *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
		require.Equal(t, int64(42), accountID)
		requests++
		switch req.Method {
		case http.MethodDelete:
			require.Equal(t, "/v1/videos/jobs/up-job", req.URL.Path)
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"job_id":"up-job","status":"running"}`))}, nil
		case http.MethodGet:
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"job_id":"up-job","status":"canceled"}`))}, nil
		default:
			t.Fatalf("unexpected method %s", req.Method)
			return nil, nil
		}
	}}
	svc := newVideoServiceForTest(repo, accounts, upstream)
	owner := VideoOwner{UserID: 11, APIKeyID: 22, GroupID: videoInt64Ptr(groupID)}

	job, err := svc.Cancel(context.Background(), owner, "vidjob_local")
	require.NoError(t, err)
	require.Equal(t, "running", job.Status)
	require.Equal(t, "held", job.SettlementStatus)

	job, err = svc.Get(context.Background(), owner, "vidjob_local")
	require.NoError(t, err)
	require.Equal(t, "canceled", job.Status)
	require.Equal(t, "released", job.SettlementStatus)
	require.Equal(t, 2, requests)
}

func TestXiaoVideoCreateRetriesUncertainResultOnOriginalReservation(t *testing.T) {
	const groupID int64 = 7
	repo := newVideoRepositoryStub()
	repo.media["vidmedia_retry"] = &VideoMedia{
		MediaID: "vidmedia_retry", AccountID: 42, UserID: 11, APIKeyID: 22,
		UpstreamURL: "https://upstream.example/media/retry", ExpiresAt: time.Now().Add(time.Hour),
	}
	accounts := &videoAccountRepoStub{accounts: []Account{videoTestAccount(42, groupID, "https://upstream.example/v1")}}
	requests := 0
	var upstreamIdempotencyKey string
	upstream := &videoHTTPUpstreamStub{do: func(req *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
		require.Equal(t, int64(42), accountID)
		requests++
		if requests == 1 {
			upstreamIdempotencyKey = req.Header.Get("Idempotency-Key")
			return nil, errors.New("connection reset after request write")
		}
		require.Equal(t, upstreamIdempotencyKey, req.Header.Get("Idempotency-Key"))
		return &http.Response{StatusCode: http.StatusAccepted, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"job_id":"up-job-stable","status":"pending","amount":"1"}`))}, nil
	}}
	svc := newVideoServiceForTest(repo, accounts, upstream)
	owner := VideoOwner{UserID: 11, APIKeyID: 22, GroupID: videoInt64Ptr(groupID)}
	body := []byte(`{"model":"video-public","prompt":"stable retry","start_frame_url":"https://video.example.test/v1/videos/uploads/vidmedia_retry/content"}`)

	_, err := svc.Create(context.Background(), owner, body, "stable-key")
	require.Error(t, err)
	require.Len(t, repo.jobs, 1)
	var reservedJobID string
	for _, job := range repo.jobs {
		reservedJobID = job.JobID
		require.Equal(t, int64(42), job.AccountID)
		require.True(t, strings.HasPrefix(job.UpstreamJobID, videoRetryableUpstreamPrefix))
	}

	job, err := svc.Create(context.Background(), owner, body, "stable-key")
	require.NoError(t, err)
	require.Equal(t, reservedJobID, job.JobID)
	require.Equal(t, "up-job-stable", job.UpstreamJobID)
	require.Equal(t, 2, requests)
}

func videoInt64Ptr(value int64) *int64 { return &value }
