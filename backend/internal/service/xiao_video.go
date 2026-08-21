package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	videoCreatingUpstreamPrefix  = "creating:"
	videoRetryableUpstreamPrefix = "retry:"
)

var (
	ErrVideoGenerationDisabled            = infraerrors.New(http.StatusForbidden, "VIDEO_GENERATION_DISABLED", "video generation is not enabled for this API key")
	ErrVideoExecutionDisabled             = infraerrors.New(http.StatusServiceUnavailable, "VIDEO_EXECUTION_DISABLED", "video execution is disabled")
	ErrVideoResourceNotFound              = infraerrors.New(http.StatusNotFound, "VIDEO_RESOURCE_NOT_FOUND", "video resource not found")
	ErrVideoRequestInvalid                = infraerrors.New(http.StatusBadRequest, "VIDEO_REQUEST_INVALID", "video request is invalid")
	ErrVideoIdempotencyInvalid            = infraerrors.New(http.StatusBadRequest, "IDEMPOTENCY_KEY_INVALID", "idempotency key must be 1 to 128 printable ASCII characters")
	ErrVideoIdempotencyConflict           = infraerrors.New(http.StatusConflict, "IDEMPOTENCY_KEY_CONFLICT", "idempotency key was reused with a different request")
	ErrVideoRequestInProgress             = infraerrors.New(http.StatusServiceUnavailable, "VIDEO_REQUEST_IN_PROGRESS", "the idempotent request is still being created")
	ErrVideoInsufficientBalance           = infraerrors.New(http.StatusPaymentRequired, "INSUFFICIENT_BALANCE", "insufficient balance")
	ErrVideoJobNotCancelable              = infraerrors.New(http.StatusConflict, "VIDEO_JOB_NOT_CANCELABLE", "video job is not cancelable")
	ErrVideoCapacityExhausted             = infraerrors.New(http.StatusTooManyRequests, "VIDEO_CAPACITY_EXHAUSTED", "video capacity is temporarily exhausted")
	ErrVideoPricingUnavailable            = infraerrors.New(http.StatusServiceUnavailable, "VIDEO_PRICING_UNAVAILABLE", "video pricing is unavailable")
	ErrVideoUpstreamUnavailable           = infraerrors.New(http.StatusServiceUnavailable, "VIDEO_UPSTREAM_UNAVAILABLE", "video upstream is unavailable")
	ErrVideoMediaAccountMismatch          = infraerrors.New(http.StatusUnprocessableEntity, "VIDEO_MEDIA_INVALID", "all uploaded media must belong to the same video upstream")
	ErrVideoUploadUnsupported             = infraerrors.New(http.StatusUnprocessableEntity, "VIDEO_UPLOAD_UNSUPPORTED", "the selected video upstream only accepts public media URLs")
	ErrVideoOptionUnsupported             = infraerrors.New(http.StatusUnprocessableEntity, "VIDEO_OPTION_UNSUPPORTED", "the selected video upstream does not support this video option")
	ErrVideoReferenceImageStrengthInvalid = infraerrors.New(http.StatusUnprocessableEntity, "VIDEO_REFERENCE_IMAGE_STRENGTH_INVALID", "reference image strength must be LOW, MID, or HIGH")
	ErrVideoPromptAspectRatioConflict     = infraerrors.New(http.StatusUnprocessableEntity, "VIDEO_PROMPT_ASPECT_RATIO_CONFLICT", "prompt aspect ratio conflicts with the selected option")
	ErrVideoPromptDurationConflict        = infraerrors.New(http.StatusUnprocessableEntity, "VIDEO_PROMPT_DURATION_CONFLICT", "prompt duration conflicts with the selected option")
)

type VideoOwner struct {
	UserID   int64
	APIKeyID int64
	GroupID  *int64
}

type VideoMedia struct {
	MediaID         string
	UpstreamMediaID string
	AccountID       int64
	UserID          int64
	APIKeyID        int64
	UpstreamURL     string
	MediaType       string
	// The following fields are optional metadata returned by newer video
	// upstreams. Keep them on the internal media value so the browser
	// workbench can use the richer upload response without breaking older
	// providers that only return media_id, url, and type.
	MediaContentType string
	MIMEType         string
	Container        string
	DurationUS       int64
	ExpiresAt        time.Time
	CreatedAt        time.Time
}

type VideoJob struct {
	JobID            string
	UpstreamJobID    string
	AccountID        int64
	UserID           int64
	APIKeyID         int64
	GroupID          *int64
	IdempotencyKey   *string
	RequestHash      string
	Model            string
	Resolution       string
	Duration         int
	AspectRatio      string
	Status           string
	Amount           float64
	Currency         string
	UpstreamAmount   *float64
	UpstreamCurrency string
	SettlementStatus string
	UpstreamResponse []byte
	CreatedAt        time.Time
	UpdatedAt        time.Time
	FinishedAt       *time.Time
	SettledAt        *time.Time
}

type VideoJobReservation struct {
	JobID                  string
	AccountID              int64
	Owner                  VideoOwner
	IdempotencyKey         string
	RequestHash            string
	Model                  string
	Resolution             string
	Duration               int
	AspectRatio            string
	PreauthorizationAmount float64
}

type VideoJobFinalization struct {
	JobID            string
	UpstreamJobID    string
	Status           string
	UpstreamAmount   float64
	UpstreamCurrency string
	Resolution       string
	Duration         int
	AspectRatio      string
	UpstreamResponse []byte
}

type VideoJobUpdate struct {
	JobID            string
	Status           string
	Resolution       string
	Duration         int
	AspectRatio      string
	UpstreamResponse []byte
	FinishedAt       *time.Time
}

type VideoRepository interface {
	CreateMedia(context.Context, *VideoMedia) error
	GetMediaForOwner(context.Context, string, int64) (*VideoMedia, error)
	ReserveJob(context.Context, VideoJobReservation) (*VideoJob, bool, error)
	MarkJobReservationRetryable(context.Context, string) error
	ClaimJobReservationRetry(context.Context, string, time.Time) (bool, error)
	ReleaseJobReservation(context.Context, string) error
	FinalizeJobAndReconcileHold(context.Context, VideoJobFinalization) (*VideoJob, error)
	GetJobForOwner(context.Context, string, int64) (*VideoJob, error)
	ListJobsForOwner(context.Context, int64, int) ([]*VideoJob, error)
	ListActiveJobs(context.Context, int) ([]*VideoJob, error)
	UpdateJobAndSettle(context.Context, VideoJobUpdate) (*VideoJob, error)
}

type VideoUpstreamError struct {
	Status int
	Header http.Header
	Body   []byte
}

// VideoUpstreamDiagnostic contains only provider fields that are safe to log
// and, when recognized, show to the caller. The raw provider body is never
// exposed by the API.
type VideoUpstreamDiagnostic struct {
	Code      string
	Message   string
	RequestID string
}

func (e *VideoUpstreamError) Error() string {
	return "video upstream returned HTTP " + strconv.Itoa(e.Status)
}

type XiaoVideoService struct {
	repo          VideoRepository
	accountRepo   AccountRepository
	openAIGateway *OpenAIGatewayService
	httpUpstream  HTTPUpstream
	cfg           *config.Config
	client        *http.Client
	authCache     APIKeyAuthCacheInvalidator
	billingCache  *BillingCacheService
}

func NewXiaoVideoService(
	repo VideoRepository,
	accountRepo AccountRepository,
	openAIGateway *OpenAIGatewayService,
	httpUpstream HTTPUpstream,
	authCache APIKeyAuthCacheInvalidator,
	billingCache *BillingCacheService,
	cfg *config.Config,
) *XiaoVideoService {
	timeout := 30 * time.Second
	if cfg != nil && cfg.VideoAPI.RequestTimeoutSeconds > 0 {
		timeout = time.Duration(cfg.VideoAPI.RequestTimeoutSeconds) * time.Second
	}
	return &XiaoVideoService{
		repo:          repo,
		accountRepo:   accountRepo,
		openAIGateway: openAIGateway,
		httpUpstream:  httpUpstream,
		cfg:           cfg,
		client:        &http.Client{Timeout: timeout},
		authCache:     authCache,
		billingCache:  billingCache,
	}
}

func (s *XiaoVideoService) ActiveForGroup(ctx context.Context, groupID *int64) bool {
	if s == nil || s.cfg == nil || !s.cfg.VideoAPI.Active() || groupID == nil || *groupID <= 0 || s.accountRepo == nil {
		return false
	}
	accounts, err := s.accountRepo.ListModelAvailabilityCandidates(ctx, groupID, []string{PlatformXiaoAPI}, true)
	if err != nil {
		return false
	}
	for i := range accounts {
		if accounts[i].Platform == PlatformXiaoAPI && accounts[i].Type == AccountTypeAPIKey {
			return true
		}
	}
	return false
}

func (s *XiaoVideoService) Enabled() bool {
	return s != nil && s.cfg != nil && s.cfg.VideoAPI.Active()
}

func (s *XiaoVideoService) PublicBaseURL() string {
	if s == nil || s.cfg == nil {
		return ""
	}
	return s.cfg.VideoAPI.PublicBaseURL
}

// ListModels merges video models from every eligible account in the API key's group.
func (s *XiaoVideoService) ListModels(ctx context.Context, owner VideoOwner) ([]map[string]any, error) {
	accounts, err := s.videoAccounts(ctx, owner.GroupID, false)
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return nil, ErrVideoUpstreamUnavailable
	}
	models := make(map[string]map[string]any)
	var firstErr error
	for i := range accounts {
		account := &accounts[i]
		pricing, pricingErr := account.XiaoVideoPricingRules()
		if pricingErr != nil {
			if firstErr == nil {
				firstErr = ErrVideoPricingUnavailable.WithCause(pricingErr)
			}
			continue
		}
		resp, requestErr := s.upstreamWithAccount(ctx, account, http.MethodGet, "/v1/models", "", nil, "", "")
		if requestErr != nil {
			if firstErr == nil {
				firstErr = requestErr
			}
			continue
		}
		raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		_ = resp.Body.Close()
		if readErr != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
			if firstErr == nil {
				if readErr != nil {
					firstErr = ErrVideoUpstreamUnavailable.WithCause(readErr)
				} else {
					firstErr = upstreamVideoError(resp, raw)
				}
			}
			continue
		}
		var envelope struct {
			Data []map[string]any `json:"data"`
		}
		if json.Unmarshal(raw, &envelope) != nil {
			if firstErr == nil {
				firstErr = ErrVideoUpstreamUnavailable
			}
			continue
		}
		accountModels := pricedVideoModelsForAccount(account, pricing, envelope.Data)
		if len(accountModels) == 0 && firstErr == nil {
			firstErr = ErrVideoPricingUnavailable
		}
		for id, model := range accountModels {
			if existing, ok := models[id]; ok {
				mergePricedVideoModel(existing, model)
				continue
			}
			models[id] = model
		}
	}
	if len(models) == 0 && firstErr != nil {
		return nil, firstErr
	}
	keys := make([]string, 0, len(models))
	for key := range models {
		keys = append(keys, key)
	}
	sortStrings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, models[key])
	}
	return out, nil
}

// ListCapabilities is a compatibility view for the browser workbench. Model
// responses already carry the effective, priced capability contract, so
// returning the same records keeps both endpoints consistent.
func (s *XiaoVideoService) ListCapabilities(ctx context.Context, owner VideoOwner) (int, []map[string]any, error) {
	models, err := s.ListModels(ctx, owner)
	if err != nil {
		return 1, nil, err
	}
	return 1, models, nil
}

// Upload accepts an optional idempotency key for compatibility with the
// browser workbench. Durable idempotency records are owned by the generation
// endpoint; the upload key is reserved for a future media-level idempotency
// implementation.
func (s *XiaoVideoService) Upload(ctx context.Context, owner VideoOwner, body io.Reader, contentType string, idempotencyKeys ...string) (*VideoMedia, error) {
	_ = idempotencyKeys
	account, release, err := s.selectVideoAccountForUpload(ctx, owner)
	if err != nil {
		return nil, err
	}
	defer release()
	resp, err := s.upstreamWithAccount(ctx, account, http.MethodPost, "/v1/videos/uploads", contentType, body, "", "")
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if readErr != nil {
		return nil, ErrVideoUpstreamUnavailable.WithCause(readErr)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, upstreamVideoError(resp, raw)
	}
	var upstream struct {
		MediaID          string    `json:"media_id"`
		URL              string    `json:"url"`
		Type             string    `json:"type"`
		MediaContentType string    `json:"media_type"`
		MIMEType         string    `json:"mime_type"`
		Container        string    `json:"container"`
		DurationUS       int64     `json:"duration_us"`
		ExpiresAt        time.Time `json:"expires_at"`
	}
	if err := json.Unmarshal(raw, &upstream); err != nil || strings.TrimSpace(upstream.MediaID) == "" || strings.TrimSpace(upstream.URL) == "" {
		return nil, ErrVideoUpstreamUnavailable.WithCause(errors.New("invalid upload response"))
	}
	if upstream.ExpiresAt.IsZero() {
		upstream.ExpiresAt = time.Now().Add(24 * time.Hour)
	}
	mediaID, err := newVideoID("vidmedia_")
	if err != nil {
		return nil, err
	}
	mediaType := upstream.Type
	if mediaType == "" {
		mediaType = "UPLOADED"
	}
	media := &VideoMedia{
		MediaID:          mediaID,
		UpstreamMediaID:  upstream.MediaID,
		AccountID:        account.ID,
		UserID:           owner.UserID,
		APIKeyID:         owner.APIKeyID,
		UpstreamURL:      upstream.URL,
		MediaType:        mediaType,
		MediaContentType: upstream.MediaContentType,
		MIMEType:         upstream.MIMEType,
		Container:        upstream.Container,
		DurationUS:       upstream.DurationUS,
		ExpiresAt:        upstream.ExpiresAt,
		CreatedAt:        time.Now(),
	}
	if err := s.repo.CreateMedia(ctx, media); err != nil {
		return nil, err
	}
	return media, nil
}

func (s *XiaoVideoService) selectVideoAccountForUpload(ctx context.Context, owner VideoOwner) (*Account, func(), error) {
	accounts, err := s.videoAccounts(ctx, owner.GroupID, true)
	if err != nil {
		return nil, func() {}, ErrVideoUpstreamUnavailable.WithCause(err)
	}
	excluded := make(map[int64]struct{})
	hasUploadCapableAccount := false
	for i := range accounts {
		if accounts[i].XiaoVideoProtocol() == XiaoVideoProtocolOpenAISora {
			excluded[accounts[i].ID] = struct{}{}
			continue
		}
		hasUploadCapableAccount = true
	}
	if !hasUploadCapableAccount {
		return nil, func() {}, ErrVideoUploadUnsupported
	}
	return s.selectVideoAccount(ctx, owner, "", 0, excluded)
}

func (s *XiaoVideoService) OpenMedia(ctx context.Context, owner VideoOwner, mediaID, rangeHeader, rawQuery string) (*http.Response, error) {
	media, err := s.repo.GetMediaForOwner(ctx, mediaID, owner.APIKeyID)
	if err != nil {
		return nil, err
	}
	account, err := s.accountByID(ctx, media.AccountID)
	if err != nil {
		return nil, err
	}
	path := "/v1/videos/uploads/" + url.PathEscape(media.UpstreamMediaID) + "/content"
	if strings.TrimSpace(rawQuery) != "" {
		path += "?" + rawQuery
	}
	return s.upstreamWithAccount(ctx, account, http.MethodGet, path, "", nil, rangeHeader, "")
}

func (s *XiaoVideoService) Create(ctx context.Context, owner VideoOwner, body []byte, idempotencyKey string) (*VideoJob, error) {
	if !s.Enabled() {
		return nil, ErrVideoExecutionDisabled
	}
	if owner.GroupID == nil {
		return nil, ErrVideoGenerationDisabled
	}
	if !validVideoIdempotencyKey(idempotencyKey) {
		return nil, ErrVideoIdempotencyInvalid
	}
	rewritten, meta, requestHash, fixedAccountID, err := s.rewriteGenerationRequest(ctx, owner, body)
	if err != nil {
		return nil, err
	}
	jobID, err := newVideoID("vidjob_")
	if err != nil {
		return nil, err
	}
	excluded := make(map[int64]struct{})
	pricingUnavailable := false
	for attempt := 0; attempt < 3; attempt++ {
		account, release, selectErr := s.selectVideoAccount(ctx, owner, meta.Model, fixedAccountID, excluded)
		if selectErr != nil {
			return nil, selectErr
		}
		hasAPIKey := strings.TrimSpace(account.GetCredential("api_key")) != ""
		hasBaseURL := strings.TrimSpace(account.GetCredential("base_url")) != ""
		if !hasAPIKey || !hasBaseURL {
			slog.WarnContext(ctx, "xiao_video.account_credentials_incomplete",
				"account_id", account.ID,
				"attempt", attempt+1,
				"has_api_key", hasAPIKey,
				"has_base_url", hasBaseURL,
			)
			release()
			excluded[account.ID] = struct{}{}
			continue
		}
		preauthorizationAmount, resolvedResolution, resolvedDuration, pricingOK := account.XiaoVideoPriceWithReferenceVideo(
			meta.Model, meta.Resolution, meta.Duration, meta.Audio,
			meta.ReferenceVideo,
		)
		if !pricingOK {
			slog.WarnContext(ctx, "xiao_video.account_pricing_unavailable",
				"account_id", account.ID,
				"attempt", attempt+1,
			)
			release()
			pricingUnavailable = true
			if fixedAccountID != 0 {
				return nil, ErrVideoPricingUnavailable
			}
			excluded[account.ID] = struct{}{}
			continue
		}
		resolvedMeta := meta
		resolvedMeta.Resolution = resolvedResolution
		resolvedMeta.Duration = resolvedDuration
		upstreamBody, rewriteErr := rewriteVideoRequestForAccount(
			account,
			rewritten,
			resolveVideoUpstreamModel(account, meta.Model),
			resolvedMeta.Resolution,
			resolvedMeta.Duration,
		)
		if rewriteErr != nil {
			release()
			return nil, rewriteErr
		}
		reserved, created, reserveErr := s.repo.ReserveJob(ctx, VideoJobReservation{
			JobID:                  jobID,
			AccountID:              account.ID,
			Owner:                  owner,
			IdempotencyKey:         idempotencyKey,
			RequestHash:            requestHash,
			Model:                  meta.Model,
			Resolution:             resolvedMeta.Resolution,
			Duration:               resolvedMeta.Duration,
			AspectRatio:            meta.AspectRatio,
			PreauthorizationAmount: preauthorizationAmount,
		})
		if reserveErr != nil {
			release()
			if created {
				_ = s.repo.ReleaseJobReservation(context.Background(), jobID)
				s.invalidateBalance(ctx, owner.UserID)
			}
			return nil, reserveErr
		}
		if !created {
			release()
			if reserved.RequestHash != requestHash {
				return nil, ErrVideoIdempotencyConflict
			}
			if !strings.HasPrefix(reserved.UpstreamJobID, videoCreatingUpstreamPrefix) &&
				!strings.HasPrefix(reserved.UpstreamJobID, videoRetryableUpstreamPrefix) {
				return reserved, nil
			}
			retryAccount, accountErr := s.accountByID(ctx, reserved.AccountID)
			if accountErr != nil {
				return nil, accountErr
			}
			account, release, accountErr = s.acquireVideoSlot(ctx, retryAccount)
			if accountErr != nil {
				return nil, accountErr
			}
			claimed, claimErr := s.repo.ClaimJobReservationRetry(ctx, reserved.JobID, time.Now().Add(-s.videoReservationStaleAfter()))
			if claimErr != nil {
				release()
				return nil, claimErr
			}
			if !claimed {
				release()
				return nil, ErrVideoRequestInProgress
			}
			jobID = reserved.JobID
			resolvedMeta.Resolution = reserved.Resolution
			resolvedMeta.Duration = reserved.Duration
			resolvedMeta.AspectRatio = reserved.AspectRatio
			upstreamBody, rewriteErr = rewriteVideoRequestForAccount(
				account,
				rewritten,
				resolveVideoUpstreamModel(account, meta.Model),
				resolvedMeta.Resolution,
				resolvedMeta.Duration,
			)
			if rewriteErr != nil {
				release()
				s.markVideoReservationRetryable(jobID)
				return nil, rewriteErr
			}
		}
		resp, requestErr := s.upstreamWithAccount(ctx, account, http.MethodPost, videoCreatePath(account), "application/json", bytes.NewReader(upstreamBody), "", videoUpstreamIdempotencyScope(owner.APIKeyID, idempotencyKey))
		release()
		if requestErr != nil {
			if idempotencyKey != "" {
				s.markVideoReservationRetryable(jobID)
				if ctx.Err() != nil {
					return nil, ctx.Err()
				}
				return nil, requestErr
			}
			_ = s.repo.ReleaseJobReservation(context.Background(), jobID)
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			if fixedAccountID == 0 && isRetryableVideoTransportError(requestErr) {
				excluded[account.ID] = struct{}{}
				continue
			}
			return nil, requestErr
		}
		raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		_ = resp.Body.Close()
		if readErr != nil {
			if idempotencyKey != "" {
				s.markVideoReservationRetryable(jobID)
			} else {
				_ = s.repo.ReleaseJobReservation(context.Background(), jobID)
			}
			return nil, ErrVideoUpstreamUnavailable.WithCause(readErr)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			if idempotencyKey != "" && resp.StatusCode >= 500 {
				s.markVideoReservationRetryable(jobID)
				return nil, upstreamVideoError(resp, raw)
			}
			_ = s.repo.ReleaseJobReservation(context.Background(), jobID)
			if fixedAccountID == 0 && retryableVideoStatus(resp.StatusCode) {
				excluded[account.ID] = struct{}{}
				continue
			}
			return nil, upstreamVideoError(resp, raw)
		}
		upstream, err := decodeVideoUpstreamResponse(account, raw)
		if err != nil {
			if idempotencyKey != "" {
				s.markVideoReservationRetryable(jobID)
			} else {
				_ = s.repo.ReleaseJobReservation(context.Background(), jobID)
			}
			return nil, ErrVideoUpstreamUnavailable.WithCause(err)
		}
		upstreamID := videoStringValue(upstream["job_id"])
		status := defaultString(videoStringValue(upstream["status"]), "pending")
		amountText := videoStringValue(upstream["amount"])
		if upstreamID == "" {
			if idempotencyKey != "" {
				s.markVideoReservationRetryable(jobID)
			} else {
				_ = s.repo.ReleaseJobReservation(context.Background(), jobID)
			}
			return nil, ErrVideoUpstreamUnavailable.WithCause(errors.New("upstream job_id missing"))
		}
		if amountText == "" {
			refreshed, refreshedRaw, refreshErr := s.fetchUpstreamJob(ctx, account, upstreamID)
			if refreshErr != nil {
				if idempotencyKey != "" {
					s.markVideoReservationRetryable(jobID)
					return nil, refreshErr
				}
				_ = s.cancelUpstream(context.Background(), account, upstreamID)
				_ = s.repo.ReleaseJobReservation(context.Background(), jobID)
				return nil, refreshErr
			}
			upstream = refreshed
			raw = refreshedRaw
			status = defaultString(videoStringValue(upstream["status"]), status)
			amountText = videoStringValue(upstream["amount"])
		}
		if !isVideoStatus(status) {
			_ = s.cancelUpstream(context.Background(), account, upstreamID)
			_ = s.repo.ReleaseJobReservation(context.Background(), jobID)
			return nil, ErrVideoUpstreamUnavailable.WithCause(errors.New("invalid upstream video status"))
		}
		amount, amountErr := strconv.ParseFloat(amountText, 64)
		if amountErr != nil || amount < 0 {
			_ = s.cancelUpstream(context.Background(), account, upstreamID)
			_ = s.repo.ReleaseJobReservation(context.Background(), jobID)
			return nil, ErrVideoUpstreamUnavailable
		}
		upstreamCurrency := defaultString(videoStringValue(upstream["currency"]), "USD")
		resolvedMeta = resolveVideoGenerationMeta(resolvedMeta, upstream)
		job, finalizeErr := s.repo.FinalizeJobAndReconcileHold(ctx, VideoJobFinalization{
			JobID:            jobID,
			UpstreamJobID:    upstreamID,
			Status:           status,
			UpstreamAmount:   amount,
			UpstreamCurrency: upstreamCurrency,
			Resolution:       resolvedMeta.Resolution,
			Duration:         resolvedMeta.Duration,
			AspectRatio:      resolvedMeta.AspectRatio,
			UpstreamResponse: raw,
		})
		if finalizeErr != nil {
			_ = s.cancelUpstream(context.Background(), account, upstreamID)
			_ = s.repo.ReleaseJobReservation(context.Background(), jobID)
			return nil, finalizeErr
		}
		s.invalidateBalance(ctx, owner.UserID)
		return job, nil
	}
	if pricingUnavailable {
		return nil, ErrVideoPricingUnavailable
	}
	slog.WarnContext(ctx, "xiao_video.account_attempts_exhausted",
		"group_id", derefGroupID(owner.GroupID),
		"model", meta.Model,
		"excluded_accounts", len(excluded),
	)
	return nil, ErrVideoCapacityExhausted
}

type videoGenerationMeta struct {
	Model          string
	Resolution     string
	AspectRatio    string
	Duration       int
	Audio          bool
	ReferenceVideo bool
}

func (s *XiaoVideoService) rewriteGenerationRequest(ctx context.Context, owner VideoOwner, body []byte) ([]byte, videoGenerationMeta, string, int64, error) {
	var request map[string]any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&request); err != nil {
		return nil, videoGenerationMeta{}, "", 0, ErrVideoRequestInvalid
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, videoGenerationMeta{}, "", 0, ErrVideoRequestInvalid
	}
	allowed := map[string]struct{}{"model": {}, "prompt": {}, "resolution": {}, "duration": {}, "aspect_ratio": {}, "audio": {}, "prompt_enhance": {}, "image_url": {}, "start_frame_url": {}, "end_frame_url": {}, "guidances": {}}
	for key := range request {
		if _, ok := allowed[key]; !ok {
			return nil, videoGenerationMeta{}, "", 0, ErrVideoRequestInvalid
		}
	}
	for _, key := range []string{"model", "prompt", "resolution", "aspect_ratio", "prompt_enhance", "image_url", "start_frame_url", "end_frame_url"} {
		if value, exists := request[key]; exists && value != nil {
			if _, ok := value.(string); !ok {
				return nil, videoGenerationMeta{}, "", 0, ErrVideoRequestInvalid
			}
		}
	}
	meta := videoGenerationMeta{
		Model:          videoStringValue(request["model"]),
		Resolution:     videoStringValue(request["resolution"]),
		AspectRatio:    videoStringValue(request["aspect_ratio"]),
		ReferenceVideo: requestContainsReferenceVideo(request),
	}
	if prompt := videoStringValue(request["prompt"]); meta.Model == "" || prompt == "" {
		return nil, meta, "", 0, ErrVideoRequestInvalid
	}
	if err := validateVideoPromptParameters(videoStringValue(request["prompt"]), meta.AspectRatio, request["duration"]); err != nil {
		return nil, meta, "", 0, err
	}
	if rawDuration, exists := request["duration"]; exists && rawDuration != nil {
		n, ok := rawDuration.(json.Number)
		if !ok {
			return nil, meta, "", 0, ErrVideoRequestInvalid
		}
		meta.Duration, _ = strconv.Atoi(n.String())
		if meta.Duration <= 0 {
			return nil, meta, "", 0, ErrVideoRequestInvalid
		}
	}
	if rawAudio, exists := request["audio"]; exists {
		audio, ok := rawAudio.(bool)
		if !ok {
			return nil, meta, "", 0, ErrVideoRequestInvalid
		}
		meta.Audio = audio
	}
	// Normalize omitted audio to false so idempotency and retries use one stable
	// representation while preserving the legacy no-audio behavior.
	request["audio"] = meta.Audio
	if videoStringValue(request["image_url"]) != "" && videoStringValue(request["start_frame_url"]) != "" {
		return nil, meta, "", 0, ErrVideoRequestInvalid
	}
	if rawGuidances, exists := request["guidances"]; exists && rawGuidances != nil {
		guidances, ok := rawGuidances.(map[string]any)
		if !ok {
			return nil, meta, "", 0, ErrVideoRequestInvalid
		}
		if videoStringValue(request["start_frame_url"]) != "" || videoStringValue(request["end_frame_url"]) != "" {
			if images, ok := guidances["image_reference"].([]any); ok && len(images) > 0 {
				return nil, meta, "", 0, ErrVideoOptionUnsupported
			}
		}
		if err := normalizeVideoGuidances(guidances); err != nil {
			return nil, meta, "", 0, err
		}
	}
	var fixedAccountID int64
	mapURL := func(value string) (string, error) {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(value)), "data:") {
			return "", ErrVideoRequestInvalid
		}
		mapped, accountID, err := s.mapMediaURL(ctx, owner, value)
		if err != nil {
			return "", err
		}
		if accountID > 0 {
			if fixedAccountID > 0 && fixedAccountID != accountID {
				return "", ErrVideoMediaAccountMismatch
			}
			fixedAccountID = accountID
		}
		return mapped, nil
	}
	for _, key := range []string{"image_url", "start_frame_url", "end_frame_url"} {
		if value := videoStringValue(request[key]); value != "" {
			mapped, err := mapURL(value)
			if err != nil {
				return nil, meta, "", 0, err
			}
			request[key] = mapped
		}
	}
	if guidances, ok := request["guidances"].(map[string]any); ok {
		for _, listKey := range []string{"image_reference", "video_reference_base", "audio_reference"} {
			items, _ := guidances[listKey].([]any)
			for _, rawItem := range items {
				item, _ := rawItem.(map[string]any)
				for _, mediaKey := range []string{"image", "video", "audio"} {
					media, _ := item[mediaKey].(map[string]any)
					if media == nil {
						continue
					}
					if value := videoStringValue(media["url"]); value != "" {
						mapped, err := mapURL(value)
						if err != nil {
							return nil, meta, "", 0, err
						}
						media["url"] = mapped
					}
				}
			}
		}
	}
	rewritten, err := json.Marshal(request)
	if err != nil {
		return nil, meta, "", 0, ErrVideoRequestInvalid
	}
	digest := sha256.Sum256(rewritten)
	return rewritten, meta, hex.EncodeToString(digest[:]), fixedAccountID, nil
}

func validateVideoPromptParameters(prompt, aspectRatio string, rawDuration any) error {
	if aspectRatio != "" {
		for _, ratio := range []string{"21:9", "16:9", "9:21", "9:16", "4:3", "3:4", "1:1"} {
			if strings.Contains(prompt, ratio) && ratio != aspectRatio {
				return ErrVideoPromptAspectRatioConflict
			}
		}
	}
	duration := 0
	if number, ok := rawDuration.(json.Number); ok {
		duration, _ = strconv.Atoi(number.String())
	}
	if duration > 0 {
		for _, marker := range []string{"总时长", "视频时长", "片长", "时长"} {
			index := strings.Index(prompt, marker)
			if index < 0 {
				continue
			}
			tail := prompt[index+len(marker):]
			for len(tail) > 0 && strings.ContainsRune(" \t:：为是", rune(tail[0])) {
				tail = tail[1:]
			}
			value := 0
			for len(tail) > 0 && tail[0] >= '0' && tail[0] <= '9' {
				value = value*10 + int(tail[0]-'0')
				tail = tail[1:]
			}
			if value > 0 && value != duration && (strings.HasPrefix(strings.TrimSpace(tail), "秒") || strings.HasPrefix(strings.TrimSpace(tail), "s")) {
				return ErrVideoPromptDurationConflict
			}
		}
	}
	return nil
}

// normalizeVideoGuidances keeps legacy clients compatible while ensuring every
// reference image sent upstream has a strict strength and deterministic order.
func normalizeVideoGuidances(guidances map[string]any) error {
	items, exists := guidances["image_reference"]
	if !exists || items == nil {
		return nil
	}
	list, ok := items.([]any)
	if !ok {
		return ErrVideoRequestInvalid
	}
	for index, rawItem := range list {
		item, ok := rawItem.(map[string]any)
		if !ok {
			return ErrVideoRequestInvalid
		}
		rawStrength := strings.ToUpper(strings.TrimSpace(videoStringValue(item["strength"])))
		if rawStrength == "" || rawStrength == "AUTO" {
			rawStrength = "MID"
		}
		switch rawStrength {
		case "LOW", "MID", "HIGH":
		default:
			return ErrVideoReferenceImageStrengthInvalid
		}
		item["strength"] = rawStrength
		item["order"] = index
	}
	return nil
}

// requestContainsReferenceVideo identifies the AIStartLab reference-video
// shape without treating start/end frame images as video references.
func requestContainsReferenceVideo(request map[string]any) bool {
	guidances, ok := request["guidances"].(map[string]any)
	if !ok {
		return false
	}
	if items, ok := guidances["video_reference_base"].([]any); ok && len(items) > 0 {
		return true
	}
	for _, rawItems := range guidances {
		items, ok := rawItems.([]any)
		if !ok {
			continue
		}
		for _, rawItem := range items {
			item, ok := rawItem.(map[string]any)
			if !ok {
				continue
			}
			if _, ok := item["video"].(map[string]any); ok {
				return true
			}
		}
	}
	return false
}

func (s *XiaoVideoService) mapMediaURL(ctx context.Context, owner VideoOwner, value string) (string, int64, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Path == "" {
		return value, 0, nil
	}
	marker := "/v1/videos/uploads/"
	index := strings.Index(parsed.Path, marker)
	if index < 0 {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(value)), "http://") || strings.HasPrefix(strings.ToLower(strings.TrimSpace(value)), "https://") {
			return value, 0, nil
		}
		return "", 0, ErrVideoRequestInvalid
	}
	rest := parsed.Path[index+len(marker):]
	mediaID := strings.SplitN(rest, "/", 2)[0]
	if !strings.HasPrefix(mediaID, "vidmedia_") {
		return value, 0, nil
	}
	media, err := s.repo.GetMediaForOwner(ctx, mediaID, owner.APIKeyID)
	if err != nil {
		return "", 0, err
	}
	return media.UpstreamURL, media.AccountID, nil
}

func (s *XiaoVideoService) Get(ctx context.Context, owner VideoOwner, jobID string) (*VideoJob, error) {
	job, err := s.repo.GetJobForOwner(ctx, jobID, owner.APIKeyID)
	if err != nil {
		return nil, err
	}
	if isVideoTerminal(job.Status) {
		return job, nil
	}
	return s.refresh(ctx, job)
}

func (s *XiaoVideoService) List(ctx context.Context, owner VideoOwner, limit int) ([]*VideoJob, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.ListJobsForOwner(ctx, owner.APIKeyID, limit)
}

func (s *XiaoVideoService) Cancel(ctx context.Context, owner VideoOwner, jobID string) (*VideoJob, error) {
	job, err := s.repo.GetJobForOwner(ctx, jobID, owner.APIKeyID)
	if err != nil {
		return nil, err
	}
	if job.Status != "pending" && job.Status != "running" {
		return nil, ErrVideoJobNotCancelable
	}
	account, err := s.accountByID(ctx, job.AccountID)
	if err != nil {
		return nil, err
	}
	if account.XiaoVideoProtocol() == XiaoVideoProtocolOpenAISora {
		return nil, ErrVideoJobNotCancelable
	}
	resp, err := s.upstreamWithAccount(ctx, account, http.MethodDelete, videoJobPath(account, job.UpstreamJobID), "", nil, "", "")
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, upstreamVideoError(resp, raw)
	}
	var upstream map[string]any
	if json.Unmarshal(raw, &upstream) == nil {
		status := videoStringValue(upstream["status"])
		if isVideoStatus(status) {
			var finished *time.Time
			if isVideoTerminal(status) {
				now := time.Now()
				finished = &now
			}
			meta := resolveVideoGenerationMeta(videoGenerationMeta{
				Resolution:  job.Resolution,
				Duration:    job.Duration,
				AspectRatio: job.AspectRatio,
			}, upstream)
			updated, updateErr := s.repo.UpdateJobAndSettle(ctx, VideoJobUpdate{
				JobID:            job.JobID,
				Status:           status,
				Resolution:       meta.Resolution,
				Duration:         meta.Duration,
				AspectRatio:      meta.AspectRatio,
				UpstreamResponse: raw,
				FinishedAt:       finished,
			})
			if updateErr != nil {
				return nil, updateErr
			}
			if updated.SettlementStatus != job.SettlementStatus {
				s.invalidateBalance(ctx, updated.UserID)
			}
			return updated, nil
		}
	}

	// A successful DELETE only requests cancellation. If the response does not
	// carry a documented status, try one immediate refresh and otherwise leave
	// the held amount untouched for the reconciler to settle later.
	updated, refreshErr := s.refresh(ctx, job)
	if refreshErr == nil {
		return updated, nil
	}
	return job, nil
}

func (s *XiaoVideoService) OpenContent(ctx context.Context, owner VideoOwner, jobID, rangeHeader, rawQuery string) (*http.Response, error) {
	job, err := s.Get(ctx, owner, jobID)
	if err != nil {
		return nil, err
	}
	if job.Status != "completed" {
		return nil, ErrVideoResourceNotFound
	}
	account, err := s.accountByID(ctx, job.AccountID)
	if err != nil {
		return nil, err
	}
	if account.XiaoVideoProtocol() == XiaoVideoProtocolOpenAISora {
		if resultURL, ok := openAISoraResultURL(job.UpstreamResponse); ok {
			return s.openExternalVideoContent(ctx, account, resultURL, rangeHeader)
		}
	}
	path := videoContentPath(account, job.UpstreamJobID)
	if strings.TrimSpace(rawQuery) != "" {
		path += "?" + rawQuery
	}
	return s.upstreamWithAccount(ctx, account, http.MethodGet, path, "", nil, rangeHeader, "")
}

func openAISoraResultURL(raw []byte) (string, bool) {
	var envelope struct {
		Metadata struct {
			ResultURL string `json:"result_url"`
		} `json:"metadata"`
	}
	if json.Unmarshal(raw, &envelope) != nil {
		return "", false
	}
	resultURL, err := url.Parse(strings.TrimSpace(envelope.Metadata.ResultURL))
	if err != nil || resultURL.Host == "" || (resultURL.Scheme != "http" && resultURL.Scheme != "https") {
		return "", false
	}
	return resultURL.String(), true
}

func (s *XiaoVideoService) openExternalVideoContent(ctx context.Context, account *Account, resultURL, rangeHeader string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resultURL, nil)
	if err != nil {
		return nil, ErrVideoUpstreamUnavailable.WithCause(err)
	}
	if strings.TrimSpace(rangeHeader) != "" {
		req.Header.Set("Range", rangeHeader)
	}
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	if s.httpUpstream != nil {
		resp, requestErr := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
		if requestErr != nil {
			return nil, ErrVideoUpstreamUnavailable.WithCause(requestErr)
		}
		return resp, nil
	}
	resp, requestErr := s.client.Do(req)
	if requestErr != nil {
		return nil, ErrVideoUpstreamUnavailable.WithCause(requestErr)
	}
	return resp, nil
}

func (s *XiaoVideoService) Reconcile(ctx context.Context, limit int) error {
	jobs, err := s.repo.ListActiveJobs(ctx, limit)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		_, _ = s.refresh(ctx, job)
	}
	return nil
}

func (s *XiaoVideoService) refresh(ctx context.Context, job *VideoJob) (*VideoJob, error) {
	account, err := s.accountByID(ctx, job.AccountID)
	if err != nil {
		return nil, err
	}
	resp, err := s.upstreamWithAccount(ctx, account, http.MethodGet, videoJobPath(account, job.UpstreamJobID), "", nil, "", "")
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if readErr != nil {
		return nil, ErrVideoUpstreamUnavailable.WithCause(readErr)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, upstreamVideoError(resp, raw)
	}
	upstream, err := decodeVideoUpstreamResponse(account, raw)
	if err != nil {
		return nil, ErrVideoUpstreamUnavailable.WithCause(err)
	}
	status := defaultString(videoStringValue(upstream["status"]), job.Status)
	if !isVideoStatus(status) {
		return nil, ErrVideoUpstreamUnavailable.WithCause(errors.New("invalid upstream video status"))
	}
	var finished *time.Time
	if isVideoTerminal(status) {
		now := time.Now()
		finished = &now
	}
	meta := resolveVideoGenerationMeta(videoGenerationMeta{
		Resolution:  job.Resolution,
		Duration:    job.Duration,
		AspectRatio: job.AspectRatio,
	}, upstream)
	updated, err := s.repo.UpdateJobAndSettle(ctx, VideoJobUpdate{
		JobID:            job.JobID,
		Status:           status,
		Resolution:       meta.Resolution,
		Duration:         meta.Duration,
		AspectRatio:      meta.AspectRatio,
		UpstreamResponse: raw,
		FinishedAt:       finished,
	})
	if err == nil && updated.SettlementStatus != job.SettlementStatus {
		s.invalidateBalance(ctx, updated.UserID)
	}
	return updated, err
}

func (s *XiaoVideoService) fetchUpstreamJob(ctx context.Context, account *Account, upstreamID string) (map[string]any, []byte, error) {
	resp, err := s.upstreamWithAccount(ctx, account, http.MethodGet, videoJobPath(account, upstreamID), "", nil, "", "")
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if readErr != nil {
		return nil, nil, ErrVideoUpstreamUnavailable.WithCause(readErr)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, upstreamVideoError(resp, raw)
	}
	upstream, err := decodeVideoUpstreamResponse(account, raw)
	if err != nil {
		return nil, nil, ErrVideoUpstreamUnavailable.WithCause(err)
	}
	return upstream, raw, nil
}

func (s *XiaoVideoService) cancelUpstream(ctx context.Context, account *Account, upstreamID string) error {
	if account != nil && account.XiaoVideoProtocol() == XiaoVideoProtocolOpenAISora {
		return ErrVideoJobNotCancelable
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	resp, err := s.upstreamWithAccount(ctx, account, http.MethodDelete, videoJobPath(account, upstreamID), "", nil, "", "")
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	return nil
}

func (s *XiaoVideoService) selectVideoAccount(ctx context.Context, owner VideoOwner, model string, fixedAccountID int64, excluded map[int64]struct{}) (*Account, func(), error) {
	if fixedAccountID > 0 {
		account, err := s.accountByID(ctx, fixedAccountID)
		if err != nil {
			return nil, func() {}, err
		}
		if !s.accountEligibleForVideo(owner, account, model) {
			return nil, func() {}, ErrVideoUpstreamUnavailable
		}
		return s.acquireVideoSlot(ctx, account)
	}
	accounts, err := s.videoAccounts(ctx, owner.GroupID, true)
	if err != nil {
		return nil, func() {}, ErrVideoUpstreamUnavailable.WithCause(err)
	}
	sort.SliceStable(accounts, func(i, j int) bool {
		if accounts[i].Priority != accounts[j].Priority {
			return accounts[i].Priority < accounts[j].Priority
		}
		switch {
		case accounts[i].LastUsedAt == nil && accounts[j].LastUsedAt != nil:
			return true
		case accounts[i].LastUsedAt != nil && accounts[j].LastUsedAt == nil:
			return false
		case accounts[i].LastUsedAt != nil && accounts[j].LastUsedAt != nil && !accounts[i].LastUsedAt.Equal(*accounts[j].LastUsedAt):
			return accounts[i].LastUsedAt.Before(*accounts[j].LastUsedAt)
		default:
			return accounts[i].ID < accounts[j].ID
		}
	})

	eligible := 0
	for i := range accounts {
		account := &accounts[i]
		if _, skip := excluded[account.ID]; skip || !s.accountEligibleForVideo(owner, account, model) {
			continue
		}
		eligible++
		selected, release, acquireErr := s.acquireVideoSlot(ctx, account)
		if acquireErr == nil {
			return selected, release, nil
		}
		if !errors.Is(acquireErr, ErrVideoCapacityExhausted) {
			return nil, func() {}, acquireErr
		}
	}
	slog.WarnContext(ctx, "xiao_video.account_selection_exhausted",
		"group_id", derefGroupID(owner.GroupID),
		"model", model,
		"candidate_accounts", len(accounts),
		"eligible_accounts", eligible,
	)
	return nil, func() {}, ErrVideoCapacityExhausted
}

func (s *XiaoVideoService) acquireVideoSlot(ctx context.Context, account *Account) (*Account, func(), error) {
	if account == nil {
		return nil, func() {}, ErrVideoUpstreamUnavailable
	}
	if s.openAIGateway == nil {
		return nil, func() {}, ErrVideoUpstreamUnavailable
	}
	result, err := s.openAIGateway.tryAcquireAccountSlot(ctx, account.ID, account.Concurrency)
	if err != nil {
		return nil, func() {}, ErrVideoUpstreamUnavailable.WithCause(err)
	}
	if result == nil || !result.Acquired {
		slog.WarnContext(ctx, "xiao_video.account_slot_exhausted",
			"account_id", account.ID,
			"max_concurrency", account.Concurrency,
		)
		return nil, func() {}, ErrVideoCapacityExhausted
	}
	return account, result.ReleaseFunc, nil
}

func (s *XiaoVideoService) accountByID(ctx context.Context, id int64) (*Account, error) {
	if id <= 0 || s.accountRepo == nil {
		return nil, ErrVideoUpstreamUnavailable
	}
	account, err := s.accountRepo.GetByID(ctx, id)
	if err != nil || account == nil {
		return nil, ErrVideoUpstreamUnavailable
	}
	if account.Platform != PlatformXiaoAPI || account.Type != AccountTypeAPIKey || account.XiaoVideoProtocol() == "" {
		return nil, ErrVideoUpstreamUnavailable
	}
	if strings.TrimSpace(account.GetCredential("api_key")) == "" || strings.TrimSpace(account.GetCredential("base_url")) == "" {
		return nil, ErrVideoUpstreamUnavailable
	}
	return account, nil
}

func (s *XiaoVideoService) accountEligibleForVideo(owner VideoOwner, account *Account, model string) bool {
	if account == nil || account.Platform != PlatformXiaoAPI || account.Type != AccountTypeAPIKey || account.XiaoVideoProtocol() == "" || !account.IsSchedulable() {
		return false
	}
	if model != "" && !account.IsModelSupported(model) {
		return false
	}
	if owner.GroupID == nil {
		return len(account.GroupIDs) == 0 && len(account.AccountGroups) == 0
	}
	for _, id := range account.GroupIDs {
		if id == *owner.GroupID {
			return true
		}
	}
	for _, group := range account.AccountGroups {
		if group.GroupID == *owner.GroupID {
			return true
		}
	}
	return false
}

func (s *XiaoVideoService) videoAccounts(ctx context.Context, groupID *int64, includeTransient bool) ([]Account, error) {
	if s == nil || s.accountRepo == nil || groupID == nil {
		return nil, ErrVideoUpstreamUnavailable
	}
	accounts, err := s.accountRepo.ListModelAvailabilityCandidates(ctx, groupID, []string{PlatformXiaoAPI}, true)
	if err != nil {
		return nil, err
	}
	out := make([]Account, 0, len(accounts))
	for _, account := range accounts {
		if account.Platform != PlatformXiaoAPI || account.Type != AccountTypeAPIKey || account.XiaoVideoProtocol() == "" {
			continue
		}
		if !includeTransient && !account.IsActive() {
			continue
		}
		out = append(out, account)
	}
	return out, nil
}

func (s *XiaoVideoService) upstreamWithAccount(ctx context.Context, account *Account, method, path, contentType string, body io.Reader, rangeHeader, idempotencyKey string) (*http.Response, error) {
	if !s.Enabled() {
		return nil, ErrVideoExecutionDisabled
	}
	if account == nil {
		return nil, ErrVideoUpstreamUnavailable
	}
	endpoint, err := accountVideoEndpoint(account.GetCredential("base_url"), path)
	if err != nil {
		return nil, ErrVideoUpstreamUnavailable.WithCause(err)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, ErrVideoUpstreamUnavailable.WithCause(err)
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(account.GetCredential("api_key")))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if method == http.MethodPost && path == "/v1/videos/generations" {
		req.Header.Set("Prefer", "respond-async")
	}
	if strings.TrimSpace(rangeHeader) != "" {
		req.Header.Set("Range", rangeHeader)
	}
	if strings.TrimSpace(idempotencyKey) != "" {
		digest := sha256.Sum256([]byte(strconv.FormatInt(account.ID, 10) + ":" + idempotencyKey))
		req.Header.Set("Idempotency-Key", "sub2api-"+hex.EncodeToString(digest[:]))
	}
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	if s.httpUpstream != nil {
		resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
		if err != nil {
			return nil, ErrVideoUpstreamUnavailable.WithCause(err)
		}
		return resp, nil
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, ErrVideoUpstreamUnavailable.WithCause(err)
	}
	return resp, nil
}

func accountVideoEndpoint(rawBase, path string) (string, error) {
	base, err := url.Parse(strings.TrimSpace(rawBase))
	if err != nil || base.Scheme == "" || base.Host == "" || base.RawQuery != "" || base.Fragment != "" {
		return "", errors.New("invalid video account base_url")
	}
	requestURL, err := url.Parse(path)
	if err != nil || requestURL.IsAbs() || requestURL.Host != "" || requestURL.Fragment != "" {
		return "", errors.New("invalid video upstream path")
	}
	basePath := strings.TrimRight(base.Path, "/")
	requestPath := requestURL.Path
	if strings.HasSuffix(basePath, "/v1") && strings.HasPrefix(requestPath, "/v1/") {
		requestPath = strings.TrimPrefix(requestPath, "/v1")
	}
	base.Path = basePath + requestPath
	base.RawPath = ""
	base.RawQuery = requestURL.RawQuery
	return base.String(), nil
}

func rewriteVideoRequest(body []byte, model, resolution string, duration int) ([]byte, error) {
	model = strings.TrimSpace(model)
	resolution = strings.TrimSpace(resolution)
	if model == "" || resolution == "" || duration <= 0 {
		return nil, ErrVideoRequestInvalid
	}
	var request map[string]any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&request); err != nil {
		return nil, ErrVideoRequestInvalid
	}
	request["model"] = model
	request["resolution"] = resolution
	request["duration"] = duration
	rewritten, err := json.Marshal(request)
	if err != nil {
		return nil, ErrVideoRequestInvalid
	}
	return rewritten, nil
}

func rewriteVideoRequestForAccount(account *Account, body []byte, model, resolution string, duration int) ([]byte, error) {
	if account == nil || account.XiaoVideoProtocol() != XiaoVideoProtocolOpenAISora {
		return rewriteVideoRequest(body, model, resolution, duration)
	}
	return rewriteOpenAISoraVideoRequest(account, body, model, resolution, duration)
}

func rewriteOpenAISoraVideoRequest(account *Account, body []byte, model, resolution string, duration int) ([]byte, error) {
	var request map[string]any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&request); err != nil {
		return nil, ErrVideoRequestInvalid
	}
	if audio, _ := request["audio"].(bool); audio {
		return nil, ErrVideoOptionUnsupported
	}
	if videoStringValue(request["prompt_enhance"]) != "" {
		return nil, ErrVideoOptionUnsupported
	}
	if err := validateAIStartLabRequestCapabilities(account, model, request); err != nil {
		return nil, err
	}

	payload := map[string]any{
		"model":   strings.TrimSpace(model),
		"prompt":  videoStringValue(request["prompt"]),
		"seconds": strconv.Itoa(duration),
		"n":       1,
	}
	if payload["model"] == "" || payload["prompt"] == "" || duration <= 0 {
		return nil, ErrVideoRequestInvalid
	}
	if aspectRatio := videoStringValue(request["aspect_ratio"]); aspectRatio != "" {
		payload["size"] = aspectRatio
	}

	metadata := map[string]any{"resolution": strings.TrimSpace(resolution)}
	images := make([]string, 0)
	startFrame := videoStringValue(request["start_frame_url"])
	endFrame := videoStringValue(request["end_frame_url"])
	if imageURL := videoStringValue(request["image_url"]); imageURL != "" {
		images = append(images, imageURL)
	}
	if startFrame != "" {
		images = append(images, startFrame)
	}
	if endFrame != "" {
		if startFrame == "" {
			return nil, ErrVideoRequestInvalid
		}
		images = append(images, endFrame)
	}
	videos, audios := []string{}, []string{}
	if guidances, ok := request["guidances"].(map[string]any); ok {
		images = append(images, videoGuidanceURLs(guidances, "image_reference", "image")...)
		videos = append(videos, videoGuidanceURLs(guidances, "video_reference_base", "video")...)
		audios = append(audios, videoGuidanceURLs(guidances, "audio_reference", "audio")...)
	}
	if len(videos) > 0 || len(audios) > 0 {
		if len(images) == 0 {
			return nil, ErrVideoOptionUnsupported
		}
	}
	if endFrame != "" && len(images) != 2 {
		return nil, ErrVideoOptionUnsupported
	}
	switch {
	case endFrame != "":
		metadata["mode_type"] = "frames2video"
	case len(images) > 0:
		metadata["mode_type"] = "image2video"
	default:
		metadata["mode_type"] = "text2video"
	}
	if len(images) > 0 {
		metadata["images"] = images
	}
	if len(videos) > 0 {
		metadata["videos"] = videos
	}
	if len(audios) > 0 {
		metadata["audios"] = audios
	}
	payload["metadata"] = metadata
	return json.Marshal(payload)
}

func validateAIStartLabRequestCapabilities(account *Account, model string, request map[string]any) error {
	if account == nil {
		return nil
	}
	capabilities := videoCapabilitiesForAccount(account)
	capability := capabilities[model]
	if capability == nil {
		for id, candidate := range capabilities {
			if videoModelSuffix(id) == model {
				if capability != nil {
					return nil
				}
				capability = candidate
			}
		}
	}
	if capability == nil {
		return nil
	}
	if aspectRatio := videoStringValue(request["aspect_ratio"]); aspectRatio != "" {
		if values := videoStringSet(capability["aspect_ratios"]); len(values) > 0 {
			if _, ok := values[aspectRatio]; !ok {
				return ErrVideoOptionUnsupported
			}
		}
	}
	if start := videoStringValue(request["start_frame_url"]); start != "" {
		if supported, ok := capability["supports_start_frame"].(bool); ok && !supported {
			return ErrVideoOptionUnsupported
		}
	}
	if end := videoStringValue(request["end_frame_url"]); end != "" {
		if supported, ok := capability["supports_end_frame"].(bool); ok && !supported {
			return ErrVideoOptionUnsupported
		}
	}
	guidances, _ := request["guidances"].(map[string]any)
	if guidances == nil {
		return nil
	}
	if supported, ok := capability["supports_guidances"].(bool); ok && !supported {
		return ErrVideoOptionUnsupported
	}
	limits, _ := capability["max_references"].(map[string]any)
	for listKey, mediaKey := range map[string]string{"image_reference": "image", "video_reference_base": "video", "audio_reference": "audio"} {
		items, _ := guidances[listKey].([]any)
		if len(items) == 0 {
			continue
		}
		limit := 0
		hasLimit := false
		if raw, ok := limits[mediaKey]; ok {
			hasLimit = true
			limit, _ = strconv.Atoi(videoStringValue(raw))
		}
		if hasLimit && (limit <= 0 || len(items) > limit) {
			return ErrVideoOptionUnsupported
		}
	}
	return nil
}

func videoGuidanceURLs(guidances map[string]any, listKey, mediaKey string) []string {
	items, _ := guidances[listKey].([]any)
	urls := make([]string, 0, len(items))
	for _, rawItem := range items {
		item, _ := rawItem.(map[string]any)
		media, _ := item[mediaKey].(map[string]any)
		if value := videoStringValue(media["url"]); value != "" {
			urls = append(urls, value)
		}
	}
	return urls
}

func videoCreatePath(account *Account) string {
	if account != nil && account.XiaoVideoProtocol() == XiaoVideoProtocolOpenAISora {
		return "/v1/videos"
	}
	return "/v1/videos/generations"
}

func videoJobPath(account *Account, upstreamID string) string {
	prefix := "/v1/videos/jobs/"
	if account != nil && account.XiaoVideoProtocol() == XiaoVideoProtocolOpenAISora {
		prefix = "/v1/videos/"
	}
	return prefix + url.PathEscape(upstreamID)
}

func videoContentPath(account *Account, upstreamID string) string {
	return videoJobPath(account, upstreamID) + "/content"
}

func decodeVideoUpstreamResponse(account *Account, raw []byte) (map[string]any, error) {
	var upstream map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&upstream); err != nil {
		return nil, err
	}
	if account == nil || account.XiaoVideoProtocol() != XiaoVideoProtocolOpenAISora {
		return upstream, nil
	}
	upstream["job_id"] = videoStringValue(upstream["id"])
	switch videoStringValue(upstream["status"]) {
	case "queued":
		upstream["status"] = "pending"
	case "in_progress":
		upstream["status"] = "running"
	case "completed", "failed":
	default:
		return nil, errors.New("invalid OpenAI/Sora video status")
	}
	upstream["duration"] = videoStringValue(upstream["seconds"])
	upstream["aspect_ratio"] = videoStringValue(upstream["size"])
	if metadata, ok := upstream["metadata"].(map[string]any); ok {
		upstream["resolution"] = videoStringValue(metadata["resolution"])
	}
	// The compatibility API does not expose credit consumption. Downstream
	// settlement continues to use the configured selling price.
	upstream["amount"] = "0"
	upstream["currency"] = "CREDITS"
	return upstream, nil
}

func seedanceSupportsGeneratedAudio(model string) bool {
	switch strings.ToLower(strings.TrimSpace(model)) {
	case "seedance-2.0", "seedance-2.0-fast", "seedance-2.0-mini":
		return true
	default:
		return false
	}
}

const xiaoVideoCapabilitiesCredentialKey = "video_capabilities"

func videoCapabilitiesForAccount(account *Account) map[string]map[string]any {
	result := make(map[string]map[string]any)
	if account == nil || account.Credentials == nil {
		return result
	}
	raw := account.Credentials[xiaoVideoCapabilitiesCredentialKey]
	if typed, ok := raw.(map[string]map[string]any); ok {
		return typed
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return result
	}
	if err := json.Unmarshal(encoded, &result); err != nil {
		return make(map[string]map[string]any)
	}
	return result
}

func resolveVideoUpstreamModel(account *Account, publicModel string) string {
	if account == nil {
		return strings.TrimSpace(publicModel)
	}
	model := strings.TrimSpace(account.GetMappedModel(publicModel))
	if account.XiaoVideoProtocol() != XiaoVideoProtocolOpenAISora || strings.Contains(model, ":") {
		return model
	}
	candidates := make([]string, 0)
	for upstreamID := range videoCapabilitiesForAccount(account) {
		if videoModelSuffix(upstreamID) == model {
			candidates = append(candidates, upstreamID)
		}
	}
	if len(candidates) > 0 {
		return preferredVideoUpstreamModel(candidates)
	}
	return model
}

func upstreamVideoCapabilityByID(upstreamByID map[string]map[string]any, model string) map[string]any {
	if capability := upstreamByID[model]; capability != nil {
		return capability
	}
	candidates := make([]string, 0)
	for id := range upstreamByID {
		if strings.Contains(id, ":") && videoModelSuffix(id) == model {
			candidates = append(candidates, id)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	return upstreamByID[preferredVideoUpstreamModel(candidates)]
}

func preferredVideoUpstreamModel(candidates []string) string {
	sort.Strings(candidates)
	for _, candidate := range candidates {
		// AIStartLab marks channel 47 as its default video channel. Prefer it
		// when resolving a legacy bare model name, while keeping the fallback
		// deterministic for accounts whose catalog uses another channel set.
		if strings.HasPrefix(candidate, "47:") {
			return candidate
		}
	}
	return candidates[0]
}

func videoModelSuffix(model string) string {
	if index := strings.IndexByte(model, ':'); index >= 0 {
		return strings.TrimSpace(model[index+1:])
	}
	return strings.TrimSpace(model)
}

func pricedVideoModelsForAccount(account *Account, pricing []XiaoVideoPricingRule, upstream []map[string]any) map[string]map[string]any {
	upstreamByID := make(map[string]map[string]any, len(upstream))
	for _, item := range upstream {
		if id := videoStringValue(item["id"]); id != "" {
			upstreamByID[id] = item
		}
	}
	rulesByModel := make(map[string][]XiaoVideoPricingRule)
	for _, rule := range pricing {
		rulesByModel[rule.Model] = append(rulesByModel[rule.Model], rule)
	}
	out := make(map[string]map[string]any, len(rulesByModel))
	storedCapabilities := videoCapabilitiesForAccount(account)
	for publicModel, rules := range rulesByModel {
		upstreamModel := resolveVideoUpstreamModel(account, publicModel)
		capability := upstreamVideoCapabilityByID(upstreamByID, upstreamModel)
		if capability == nil {
			continue
		}
		storedCapability := storedCapabilities[upstreamModel]
		if storedCapability == nil {
			storedCapability = storedCapabilities[account.GetMappedModel(publicModel)]
		}
		supportedResolutions := videoStringSet(capability["resolutions"])
		resolutions := make([]string, 0, len(rules))
		defaultResolution := ""
		defaultDuration := 0
		for _, rule := range rules {
			if len(supportedResolutions) > 0 {
				if _, ok := supportedResolutions[rule.Resolution]; !ok {
					continue
				}
			}
			resolutions = append(resolutions, rule.Resolution)
			if rule.DefaultResolution {
				defaultResolution = rule.Resolution
				defaultDuration = rule.DefaultDuration
			}
		}
		if len(resolutions) == 0 {
			continue
		}
		sort.Strings(resolutions)
		if defaultResolution == "" && len(resolutions) == 1 {
			defaultResolution = resolutions[0]
			for _, rule := range rules {
				if rule.Resolution == defaultResolution {
					defaultDuration = rule.DefaultDuration
					break
				}
			}
		}
		model := map[string]any{
			"id":                publicModel,
			"object":            "model",
			"owned_by":          "video",
			"capability_source": account.XiaoVideoProtocol(),
			"resolutions":       resolutions,
		}
		if multiplier := account.XiaoVideoReferenceVideoMultiplier(); multiplier > 1 {
			model["reference_video_multiplier"] = multiplier
		}
		if defaultResolution != "" {
			model["default_resolution"] = defaultResolution
		}
		if defaultDuration > 0 {
			model["default_duration"] = defaultDuration
		}
		for _, key := range []string{"default_aspect_ratio", "supports_guidances", "supports_start_frame", "requires_start_frame", "supports_end_frame", "max_references", "durations", "aspect_ratios"} {
			if value, ok := storedCapability[key]; ok {
				model[key] = value
			} else if value, ok := capability[key]; ok {
				model[key] = value
			}
		}
		if account.XiaoVideoProtocol() != XiaoVideoProtocolOpenAISora &&
			(seedanceSupportsGeneratedAudio(publicModel) || seedanceSupportsGeneratedAudio(upstreamModel)) {
			model["supports_audio"] = true
		} else if account.XiaoVideoProtocol() == XiaoVideoProtocolOpenAISora {
			model["supports_audio"] = false
		} else if value, ok := capability["supports_audio"]; ok {
			model["supports_audio"] = value
		}
		out[publicModel] = model
	}
	return out
}

func videoStringSet(value any) map[string]struct{} {
	out := make(map[string]struct{})
	switch values := value.(type) {
	case []any:
		for _, item := range values {
			if text := videoStringValue(item); text != "" {
				out[text] = struct{}{}
			}
		}
	case []string:
		for _, item := range values {
			if text := strings.TrimSpace(item); text != "" {
				out[text] = struct{}{}
			}
		}
	}
	return out
}

func videoFloatValue(value any) float64 {
	switch number := value.(type) {
	case float64:
		return number
	case float32:
		return float64(number)
	case int:
		return float64(number)
	case int64:
		return float64(number)
	case json.Number:
		parsed, _ := number.Float64()
		return parsed
	case string:
		parsed, _ := strconv.ParseFloat(strings.TrimSpace(number), 64)
		return parsed
	default:
		return 0
	}
}

func mergePricedVideoModel(target, source map[string]any) {
	resolutions := videoStringSet(target["resolutions"])
	for resolution := range videoStringSet(source["resolutions"]) {
		resolutions[resolution] = struct{}{}
	}
	merged := make([]string, 0, len(resolutions))
	for resolution := range resolutions {
		merged = append(merged, resolution)
	}
	sort.Strings(merged)
	target["resolutions"] = merged
	targetSource := videoStringValue(target["capability_source"])
	sourceSource := videoStringValue(source["capability_source"])
	if targetSource == "" {
		target["capability_source"] = sourceSource
	} else if sourceSource != "" && sourceSource != targetSource {
		target["capability_source"] = "mixed"
	}
	for _, key := range []string{"default_resolution", "default_duration", "default_aspect_ratio", "supports_guidances", "supports_start_frame", "requires_start_frame", "supports_end_frame"} {
		if _, exists := target[key]; exists {
			continue
		}
		if value, exists := source[key]; exists {
			target[key] = value
		}
	}
	if sourceMultiplier := videoFloatValue(source["reference_video_multiplier"]); sourceMultiplier > videoFloatValue(target["reference_video_multiplier"]) {
		target["reference_video_multiplier"] = sourceMultiplier
	}
	for _, key := range []string{"durations", "aspect_ratios"} {
		if key == "durations" {
			values := videoNumericValues(target[key])
			for _, value := range videoNumericValues(source[key]) {
				if !containsFloat64(values, value) {
					values = append(values, value)
				}
			}
			sort.Float64s(values)
			if len(values) > 0 {
				items := make([]any, len(values))
				for index, value := range values {
					items[index] = value
				}
				target[key] = items
			}
			continue
		}
		if sourceValues := videoStringSet(source[key]); len(sourceValues) > 0 {
			existing := videoStringSet(target[key])
			for value := range sourceValues {
				existing[value] = struct{}{}
			}
			values := make([]string, 0, len(existing))
			for value := range existing {
				values = append(values, value)
			}
			sort.Strings(values)
			target[key] = values
		}
	}
	if sourceReferences, ok := source["max_references"].(map[string]any); ok {
		targetReferences, _ := target["max_references"].(map[string]any)
		if targetReferences == nil {
			targetReferences = make(map[string]any)
		}
		for key, value := range sourceReferences {
			if sourceCount, ok := value.(float64); ok {
				if targetCount, ok := targetReferences[key].(float64); !ok || sourceCount > targetCount {
					targetReferences[key] = sourceCount
				}
			}
		}
		target["max_references"] = targetReferences
	}
	targetAudio, targetHasAudio := target["supports_audio"]
	sourceAudio, sourceHasAudio := source["supports_audio"]
	if targetAudio == true || sourceAudio == true {
		target["supports_audio"] = true
	} else if !targetHasAudio && sourceHasAudio {
		target["supports_audio"] = sourceAudio
	}
}

func videoNumericValues(value any) []float64 {
	values := make([]float64, 0)
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if number, err := strconv.ParseFloat(videoStringValue(item), 64); err == nil {
				values = append(values, number)
			}
		}
	case []int:
		for _, item := range typed {
			values = append(values, float64(item))
		}
	case []float64:
		values = append(values, typed...)
	}
	return values
}

func containsFloat64(values []float64, needle float64) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func resolveVideoGenerationMeta(current videoGenerationMeta, upstream map[string]any) videoGenerationMeta {
	if resolution := videoStringValue(upstream["resolution"]); resolution != "" {
		current.Resolution = resolution
	}
	if duration, err := strconv.Atoi(videoStringValue(upstream["duration"])); err == nil && duration > 0 {
		current.Duration = duration
	}
	if aspectRatio := videoStringValue(upstream["aspect_ratio"]); aspectRatio != "" {
		current.AspectRatio = aspectRatio
	}
	return current
}

func isRetryableVideoTransportError(err error) bool {
	return err != nil && errors.Is(err, ErrVideoUpstreamUnavailable)
}

func retryableVideoStatus(status int) bool {
	return status == http.StatusRequestTimeout || status == http.StatusTooManyRequests || status == http.StatusBadGateway || status == http.StatusServiceUnavailable || status == http.StatusGatewayTimeout || status >= 500
}

func invalidateVideoBalance(authCache APIKeyAuthCacheInvalidator, billingCache *BillingCacheService, ctx context.Context, userID int64) {
	if authCache != nil {
		authCache.InvalidateAuthCacheByUserID(ctx, userID)
	}
	if billingCache != nil {
		_ = billingCache.InvalidateUserBalance(ctx, userID)
	}
}

func (s *XiaoVideoService) invalidateBalance(ctx context.Context, userID int64) {
	invalidateVideoBalance(s.authCache, s.billingCache, ctx, userID)
}

func (s *XiaoVideoService) markVideoReservationRetryable(jobID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = s.repo.MarkJobReservationRetryable(ctx, jobID)
}

func (s *XiaoVideoService) videoReservationStaleAfter() time.Duration {
	delay := 40 * time.Second
	if s != nil && s.client != nil && s.client.Timeout > 0 {
		delay = s.client.Timeout + 10*time.Second
	}
	if delay < 15*time.Second {
		return 15 * time.Second
	}
	return delay
}

type XiaoVideoRuntime struct {
	svc    *XiaoVideoService
	cfg    *config.Config
	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

func NewXiaoVideoRuntime(svc *XiaoVideoService, cfg *config.Config) *XiaoVideoRuntime {
	runtime := &XiaoVideoRuntime{svc: svc, cfg: cfg}
	runtime.Start()
	return runtime
}

func (r *XiaoVideoRuntime) Start() {
	if r == nil || r.svc == nil || r.cfg == nil || !r.cfg.VideoAPI.Active() {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	r.done = make(chan struct{})
	go func() {
		defer close(r.done)
		interval := time.Duration(r.cfg.VideoAPI.ReconcileIntervalSeconds) * time.Second
		if interval < 5*time.Second {
			interval = 30 * time.Second
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = r.svc.Reconcile(ctx, r.cfg.VideoAPI.ReconcileBatchSize)
			}
		}
	}()
}

func (r *XiaoVideoRuntime) Stop() {
	if r == nil {
		return
	}
	r.mu.Lock()
	cancel, done := r.cancel, r.done
	r.cancel = nil
	r.done = nil
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

func validVideoIdempotencyKey(value string) bool {
	if len(value) < 1 || len(value) > 128 {
		return false
	}
	for _, char := range value {
		if char < 0x20 || char > 0x7e {
			return false
		}
	}
	return true
}

func videoUpstreamIdempotencyScope(apiKeyID int64, idempotencyKey string) string {
	if idempotencyKey == "" {
		return ""
	}
	return strconv.FormatInt(apiKeyID, 10) + ":" + idempotencyKey
}

func newVideoID(prefix string) (string, error) {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(data[:]), nil
}

func isVideoTerminal(status string) bool {
	return status == "completed" || status == "failed" || status == "canceled"
}

func isVideoStatus(status string) bool {
	switch status {
	case "pending", "running", "settling", "completed", "failed", "canceled":
		return true
	default:
		return false
	}
}

func videoStringValue(value any) string {
	switch value := value.(type) {
	case string:
		return strings.TrimSpace(value)
	case json.Number:
		return value.String()
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	default:
		return ""
	}
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func upstreamVideoError(resp *http.Response, body []byte) error {
	return &VideoUpstreamError{Status: resp.StatusCode, Header: resp.Header.Clone(), Body: body}
}

func VideoUpstreamDiagnosticFromError(err *VideoUpstreamError) VideoUpstreamDiagnostic {
	if err == nil {
		return VideoUpstreamDiagnostic{}
	}
	var envelope struct {
		Error struct {
			Code      string `json:"code"`
			Message   string `json:"message"`
			RequestID string `json:"request_id"`
		} `json:"error"`
		Code      string `json:"code"`
		Message   string `json:"message"`
		RequestID string `json:"request_id"`
	}
	if json.Unmarshal(err.Body, &envelope) != nil {
		return VideoUpstreamDiagnostic{}
	}
	diagnostic := VideoUpstreamDiagnostic{
		Code:      strings.TrimSpace(envelope.Error.Code),
		Message:   strings.TrimSpace(envelope.Error.Message),
		RequestID: strings.TrimSpace(envelope.Error.RequestID),
	}
	if diagnostic.Code == "" {
		diagnostic.Code = strings.TrimSpace(envelope.Code)
	}
	if diagnostic.Message == "" {
		diagnostic.Message = strings.TrimSpace(envelope.Message)
	}
	if diagnostic.RequestID == "" {
		diagnostic.RequestID = strings.TrimSpace(envelope.RequestID)
	}
	return diagnostic
}

func LogVideoUpstreamErrorForRequest(err *VideoUpstreamError, requestID, path string) {
	if err == nil {
		return
	}
	diagnostic := VideoUpstreamDiagnosticFromError(err)
	bodyHash := sha256.Sum256(err.Body)
	upstreamRequestID := sanitizeVideoUpstreamRequestID(err.Header.Get("X-Request-Id"))
	if upstreamRequestID == "" {
		upstreamRequestID = sanitizeVideoUpstreamRequestID(diagnostic.RequestID)
	}
	fields := []any{
		"request_id", sanitizeVideoUpstreamRequestID(requestID),
		"upstream_request_id", upstreamRequestID,
		"status", err.Status,
		"path", path,
		"body_bytes", len(err.Body),
		"body_sha256", hex.EncodeToString(bodyHash[:]),
	}
	if code := sanitizeVideoUpstreamDiagnostic(diagnostic.Code); code != "" {
		fields = append(fields, "upstream_code", code)
	}
	if message := sanitizeVideoUpstreamDiagnostic(diagnostic.Message); message != "" {
		fields = append(fields, "upstream_message", message)
	}
	slog.Warn("xiao_video.upstream_error", fields...)
}

func sanitizeVideoUpstreamDiagnostic(value string) string {
	if strings.ContainsAny(value, "\r\n") {
		return ""
	}
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if value == "" || len(value) > 240 {
		return ""
	}
	lower := strings.ToLower(value)
	for _, marker := range []string{"http://", "https://", "bearer ", "api_key", "account_id", "account id", "internal", "secret", "token"} {
		if strings.Contains(lower, marker) {
			return ""
		}
	}
	return value
}

func sanitizeVideoUpstreamRequestID(value string) string {
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

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}
