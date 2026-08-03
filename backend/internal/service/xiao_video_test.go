package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/stretchr/testify/require"
)

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
}

func newVideoRepositoryStub() *videoRepositoryStub {
	return &videoRepositoryStub{
		media:       make(map[string]*VideoMedia),
		jobs:        make(map[string]*VideoJob),
		idempotency: make(map[string]*VideoJob),
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
	key := strings.TrimSpace(reservation.IdempotencyKey)
	if key != "" {
		if existing := r.idempotency[key]; existing != nil {
			copy := *existing
			return &copy, false, nil
		}
	}
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
		Currency:         "USD",
		SettlementStatus: "released",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	if key != "" {
		job.IdempotencyKey = &key
		r.idempotency[key] = job
	}
	r.jobs[job.JobID] = job
	copy := *job
	return &copy, true, nil
}

func (r *videoRepositoryStub) DeleteJobReservation(_ context.Context, jobID string) error {
	job := r.jobs[jobID]
	if job != nil && strings.HasPrefix(job.UpstreamJobID, videoCreatingUpstreamPrefix) {
		delete(r.jobs, jobID)
		if job.IdempotencyKey != nil {
			delete(r.idempotency, *job.IdempotencyKey)
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

func (r *videoRepositoryStub) FinalizeJobAndHold(_ context.Context, finalization VideoJobFinalization) (*VideoJob, error) {
	job := r.jobs[finalization.JobID]
	if job == nil {
		return nil, ErrVideoResourceNotFound
	}
	job.UpstreamJobID = finalization.UpstreamJobID
	job.Status = finalization.Status
	job.Amount = finalization.Amount
	job.Currency = finalization.Currency
	job.UpstreamResponse = append([]byte(nil), finalization.UpstreamResponse...)
	job.SettlementStatus = "held"
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
	rate := 1.5
	return Account{
		ID:             id,
		Name:           "video-upstream",
		Platform:       PlatformOpenAI,
		Type:           AccountTypeAPIKey,
		Status:         StatusActive,
		Schedulable:    true,
		Concurrency:    2,
		RateMultiplier: &rate,
		GroupIDs:       []int64{groupID},
		Credentials: map[string]any{
			"base_url":            baseURL,
			"api_key":             "upstream-secret",
			"openai_capabilities": []any{"video_api"},
			"model_mapping":       map[string]any{"video-public": "video-upstream-v2"},
		},
	}
}

func newVideoServiceForTest(repo VideoRepository, accountRepo *videoAccountRepoStub, upstream HTTPUpstream) *XiaoVideoService {
	cfg := &config.Config{VideoAPI: config.VideoAPIConfig{Enabled: true, PublicBaseURL: "https://video.example.test"}}
	gateway := &OpenAIGatewayService{accountRepo: accountRepo}
	return NewXiaoVideoService(repo, accountRepo, gateway, upstream, nil, nil, cfg)
}

func TestVideoCapabilityRequiresExplicitAPIKeyOptIn(t *testing.T) {
	base := Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{}}
	require.False(t, base.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityVideoAPI))

	base.Credentials["openai_capabilities"] = []any{"video_api"}
	require.True(t, base.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityVideoAPI))

	base.Type = AccountTypeOAuth
	require.False(t, base.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityVideoAPI))
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
	job, err := svc.Create(context.Background(), VideoOwner{UserID: 11, APIKeyID: 22, GroupID: videoInt64Ptr(groupID)}, []byte(`{"model":"video-public","prompt":"make a video","duration":4,"start_frame_url":"https://video.example.test/v1/videos/uploads/vidmedia_local/content"}`), "order-1")
	require.NoError(t, err)
	require.Equal(t, int64(42), capturedAccountID)
	require.Equal(t, "video-upstream-v2", captured["model"])
	require.Equal(t, "https://upstream.example.test/media/up-media/content", captured["start_frame_url"])
	require.Equal(t, int64(42), job.AccountID)
	require.Equal(t, 3.0, job.Amount)
	require.NotContains(t, string(job.UpstreamResponse), "video.example.test")
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
	require.Equal(t, []any{"720p"}, models[0]["resolutions"])
}

func TestXiaoVideoOpenMediaUsesBoundAccountAndForwardsRange(t *testing.T) {
	const groupID int64 = 7
	repo := newVideoRepositoryStub()
	repo.media["vidmedia_local"] = &VideoMedia{MediaID: "vidmedia_local", UpstreamMediaID: "up media", AccountID: 42, UserID: 11, APIKeyID: 22, UpstreamURL: "https://upstream.example/media", ExpiresAt: time.Now().Add(time.Hour)}
	boundAccount := videoTestAccount(42, groupID, "https://upstream.example/v1")
	boundAccount.Status = StatusDisabled
	boundAccount.Schedulable = false
	delete(boundAccount.Credentials, "openai_capabilities")
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
