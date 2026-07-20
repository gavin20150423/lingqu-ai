package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
)

const (
	subPilotSelectPath          = "/v1/dispatch/select"
	subPilotReleaseLeasePath    = "/v1/dispatch/release-lease"
	subPilotReportSuccessPath   = "/v1/dispatch/report-success"
	subPilotReportFailurePath   = "/v1/dispatch/report-failure"
	subPilotRuntimeConfigPath   = "/v1/dispatch/runtime-config"
	subPilotDefaultTimeout      = 80 * time.Millisecond
	subPilotReportTimeout       = 500 * time.Millisecond
	subPilotConfigCacheTTL      = 5 * time.Second
	subPilotReportQueueSize     = 16384
	subPilotReportWorkers       = 8
	subPilotMaxIdleConns        = 256
	subPilotMaxResponseBodySize = 64 << 10
)

var subPilotClientCache sync.Map
var subPilotSharedHTTPClient = newSubPilotSharedHTTPClient()
var subPilotReportQueue = make(chan subPilotReportJob, subPilotReportQueueSize)
var subPilotReportWorkersOnce sync.Once
var subPilotDroppedReports atomic.Uint64

type subPilotClient struct {
	baseURL      string
	timeout      time.Duration
	sharedSecret string
	client       *http.Client
	state        *subPilotCircuitState
}

type subPilotRuntimeConfig struct {
	DispatchEnabled            bool `json:"dispatchEnabled"`
	DispatchFailOpen           bool `json:"dispatchFailOpen"`
	DispatchSelectTimeoutMS    int  `json:"dispatchSelectTimeoutMs"`
	DispatchAutoBypassFailures int  `json:"dispatchAutoBypassFailures"`
	DispatchAutoRecover        bool `json:"dispatchAutoRecover"`
}

type subPilotCircuitState struct {
	mu          sync.RWMutex
	runtime     subPilotRuntimeConfig
	lastRefresh time.Time
	refreshing  bool
	failures    atomic.Int64
	bypassed    atomic.Bool
}

type subPilotReportJob struct {
	client  *subPilotClient
	path    string
	payload any
}

type subPilotSelectRequest struct {
	RequestID          string   `json:"request_id"`
	APIKeyID           string   `json:"api_key_id,omitempty"`
	Platform           string   `json:"platform"`
	GroupID            string   `json:"group_id,omitempty"`
	Model              string   `json:"model"`
	SessionKey         string   `json:"session_key,omitempty"`
	ExcludedAccountIDs []string `json:"excluded_account_ids,omitempty"`
}

type subPilotSelectResponse struct {
	Decision string `json:"decision"`
	GroupID  string `json:"group_id,omitempty"`
	Account  struct {
		ID string `json:"id"`
	} `json:"account"`
	Lease struct {
		ID string `json:"id"`
	} `json:"lease"`
}

type subPilotSelectResult struct {
	AccountID int64
	LeaseID   string
	RequestID string
}

type subPilotReleaseLeaseRequest struct {
	RequestID string `json:"request_id"`
	LeaseID   string `json:"lease_id"`
	AccountID string `json:"account_id"`
}

type subPilotReportSuccessRequest struct {
	RequestID       string  `json:"request_id"`
	LeaseID         string  `json:"lease_id"`
	APIKeyID        string  `json:"api_key_id,omitempty"`
	AccountID       string  `json:"account_id"`
	Platform        string  `json:"platform"`
	GroupID         string  `json:"group_id"`
	Model           string  `json:"model"`
	SessionKey      string  `json:"session_key,omitempty"`
	LatencyMS       int     `json:"latency_ms,omitempty"`
	FirstTokenMS    int     `json:"first_token_ms,omitempty"`
	RequestType     string  `json:"request_type,omitempty"`
	Stream          *bool   `json:"stream,omitempty"`
	OfficialUSDUsed float64 `json:"official_usd_used,omitempty"`
	StickyTTLMS     int64   `json:"sticky_ttl_ms,omitempty"`
}

type subPilotReportFailureRequest struct {
	RequestID    string `json:"request_id"`
	LeaseID      string `json:"lease_id"`
	APIKeyID     string `json:"api_key_id,omitempty"`
	AccountID    string `json:"account_id"`
	Platform     string `json:"platform"`
	GroupID      string `json:"group_id"`
	Model        string `json:"model"`
	SessionKey   string `json:"session_key,omitempty"`
	StatusCode   int    `json:"status_code,omitempty"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	RequestType  string `json:"request_type,omitempty"`
	Stream       *bool  `json:"stream,omitempty"`
}

func newSubPilotClient(cfg config.SubPilotConfig) *subPilotClient {
	if !cfg.Enabled || strings.TrimSpace(cfg.BaseURL) == "" {
		return nil
	}
	timeout := time.Duration(cfg.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = subPilotDefaultTimeout
	}
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	sharedSecret := strings.TrimSpace(cfg.SharedSecret)
	key := strings.Join([]string{
		baseURL,
		timeout.String(),
		strconv.FormatBool(cfg.FailOpen),
		sharedSecret,
	}, "\x00")
	if cached, ok := subPilotClientCache.Load(key); ok {
		if client, ok := cached.(*subPilotClient); ok {
			return client
		}
	}
	client := &subPilotClient{
		baseURL:      baseURL,
		timeout:      timeout,
		sharedSecret: sharedSecret,
		client:       subPilotSharedHTTPClient,
		state: &subPilotCircuitState{runtime: subPilotRuntimeConfig{
			DispatchEnabled:            true,
			DispatchFailOpen:           cfg.FailOpen,
			DispatchSelectTimeoutMS:    int(timeout / time.Millisecond),
			DispatchAutoBypassFailures: 3,
			DispatchAutoRecover:        true,
		}},
	}
	actual, _ := subPilotClientCache.LoadOrStore(key, client)
	if cached, ok := actual.(*subPilotClient); ok {
		return cached
	}
	return client
}

func newSubPilotSharedHTTPClient() *http.Client {
	transport := http.DefaultTransport
	if defaultTransport, ok := http.DefaultTransport.(*http.Transport); ok {
		clonedTransport := defaultTransport.Clone()
		clonedTransport.MaxIdleConns = subPilotMaxIdleConns
		clonedTransport.MaxIdleConnsPerHost = subPilotMaxIdleConns
		transport = clonedTransport
	}
	return &http.Client{Transport: transport}
}

func subPilotConfigFrom(cfg *config.Config) config.SubPilotConfig {
	if cfg == nil {
		return config.SubPilotConfig{}
	}
	return cfg.Gateway.SubPilot
}

func (c *subPilotClient) recommendAccount(ctx context.Context, req subPilotSelectRequest) (*subPilotSelectResult, error) {
	recommendation, _, err := c.recommendAccountWithOwnership(ctx, req)
	return recommendation, err
}

func (c *subPilotClient) recommendAccountWithOwnership(ctx context.Context, req subPilotSelectRequest) (*subPilotSelectResult, bool, error) {
	if c == nil {
		return nil, false, nil
	}
	runtime := c.runtimeConfig(ctx)
	if !runtime.DispatchEnabled {
		return nil, false, nil
	}
	if c.isBypassed() {
		if runtime.DispatchFailOpen {
			return nil, false, nil
		}
		return nil, true, errors.New("subpilot dispatch is temporarily bypassed after repeated failures")
	}
	var resp subPilotSelectResponse
	timeout := time.Duration(runtime.DispatchSelectTimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = c.timeout
	}
	if err := c.postJSONWithTimeout(ctx, subPilotSelectPath, req, &resp, timeout); err != nil {
		c.recordFailure(runtime)
		handledErr := c.handleRuntimeError("subpilot select failed", err, runtime.DispatchFailOpen)
		return nil, handledErr != nil, handledErr
	}
	c.recordSuccess()
	if resp.Decision != "selected" {
		return nil, true, nil
	}
	accountID, err := strconv.ParseInt(strings.TrimSpace(resp.Account.ID), 10, 64)
	if err != nil || accountID <= 0 || strings.TrimSpace(resp.Lease.ID) == "" {
		if err == nil {
			err = errors.New("missing account or lease")
		}
		c.recordFailure(runtime)
		handledErr := c.handleRuntimeError("subpilot select returned invalid account", err, runtime.DispatchFailOpen)
		return nil, handledErr != nil, handledErr
	}
	return &subPilotSelectResult{AccountID: accountID, LeaseID: strings.TrimSpace(resp.Lease.ID), RequestID: req.RequestID}, true, nil
}

func (c *subPilotClient) takeoverActive(ctx context.Context) bool {
	if c == nil {
		return false
	}
	runtime := c.runtimeConfig(ctx)
	if !runtime.DispatchEnabled {
		return false
	}
	return !c.isBypassed() || !runtime.DispatchFailOpen
}

func (c *subPilotClient) reportSuccess(ctx context.Context, req subPilotReportSuccessRequest) {
	if c == nil {
		return
	}
	c.enqueueReport(subPilotReportSuccessPath, req)
}

func (c *subPilotClient) reportFailure(ctx context.Context, req subPilotReportFailureRequest) {
	if c == nil {
		return
	}
	c.enqueueReport(subPilotReportFailurePath, req)
}

func (c *subPilotClient) releaseLease(ctx context.Context, req subPilotReleaseLeaseRequest) {
	if c == nil {
		return
	}
	c.enqueueReport(subPilotReleaseLeasePath, req)
}

func (c *subPilotClient) enqueueReport(path string, payload any) {
	subPilotReportWorkersOnce.Do(startSubPilotReportWorkers)
	select {
	case subPilotReportQueue <- subPilotReportJob{client: c, path: path, payload: payload}:
	default:
		dropped := subPilotDroppedReports.Add(1)
		if dropped == 1 || dropped%1000 == 0 {
			slog.Warn("subpilot report queue full", "dropped", dropped)
		}
	}
}

func startSubPilotReportWorkers() {
	for index := 0; index < subPilotReportWorkers; index++ {
		go func() {
			for job := range subPilotReportQueue {
				timeout := job.client.reportTimeout()
				ctx, cancel := context.WithTimeout(context.Background(), timeout)
				err := job.client.postJSONWithTimeout(ctx, job.path, job.payload, nil, timeout)
				cancel()
				if err != nil {
					slog.Debug("subpilot async report failed", "path", job.path, "error", err)
				}
			}
		}()
	}
}

func (c *subPilotClient) runtimeConfig(_ context.Context) subPilotRuntimeConfig {
	if c == nil || c.state == nil {
		return subPilotRuntimeConfig{DispatchEnabled: true, DispatchFailOpen: true, DispatchSelectTimeoutMS: int(subPilotDefaultTimeout / time.Millisecond)}
	}
	c.state.mu.RLock()
	cached := c.state.runtime
	stale := c.state.lastRefresh.IsZero() || time.Since(c.state.lastRefresh) >= subPilotConfigCacheTTL
	refreshing := c.state.refreshing
	c.state.mu.RUnlock()
	if c.sharedSecret == "" || !stale || refreshing {
		return cached
	}

	c.state.mu.Lock()
	if c.state.refreshing || (!c.state.lastRefresh.IsZero() && time.Since(c.state.lastRefresh) < subPilotConfigCacheTTL) {
		cached = c.state.runtime
		c.state.mu.Unlock()
		return cached
	}
	c.state.refreshing = true
	c.state.mu.Unlock()
	go c.refreshRuntimeConfig()
	return cached
}

func (c *subPilotClient) refreshRuntimeConfig() {
	var next subPilotRuntimeConfig
	err := c.getJSONWithTimeout(context.Background(), subPilotRuntimeConfigPath, &next, c.timeout)
	c.state.mu.Lock()
	c.state.refreshing = false
	c.state.lastRefresh = time.Now()
	if err != nil {
		c.state.mu.Unlock()
		return
	}
	if next.DispatchSelectTimeoutMS <= 0 {
		next.DispatchSelectTimeoutMS = int(subPilotDefaultTimeout / time.Millisecond)
	}
	if next.DispatchAutoBypassFailures <= 0 {
		next.DispatchAutoBypassFailures = 3
	}
	c.state.runtime = next
	if c.state.bypassed.Load() && next.DispatchAutoRecover {
		c.state.bypassed.Store(false)
		c.state.failures.Store(0)
	}
	c.state.mu.Unlock()
}

func (c *subPilotClient) isBypassed() bool {
	return c != nil && c.state != nil && c.state.bypassed.Load()
}

func (c *subPilotClient) recordFailure(runtime subPilotRuntimeConfig) {
	if c == nil || c.state == nil {
		return
	}
	threshold := runtime.DispatchAutoBypassFailures
	if threshold <= 0 {
		threshold = 3
	}
	if c.state.failures.Add(1) >= int64(threshold) {
		c.state.bypassed.Store(true)
	}
}

func (c *subPilotClient) recordSuccess() {
	if c == nil || c.state == nil {
		return
	}
	c.state.failures.Store(0)
	c.state.bypassed.Store(false)
}

func (c *subPilotClient) postJSONWithTimeout(ctx context.Context, path string, in any, out any, timeout time.Duration) error {
	if c == nil {
		return nil
	}
	body, err := json.Marshal(in)
	if err != nil {
		return err
	}
	reqCtx, cancel := context.WithTimeout(contextOrBackground(ctx), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	c.setSharedSecret(req)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			slog.Debug("subpilot response close failed", "error", closeErr)
		}
	}()
	raw, err := readSubPilotResponse(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("subpilot status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if out == nil || resp.StatusCode == http.StatusNoContent || len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, out)
}

func (c *subPilotClient) getJSONWithTimeout(ctx context.Context, path string, out any, timeout time.Duration) error {
	reqCtx, cancel := context.WithTimeout(contextOrBackground(ctx), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	c.setSharedSecret(req)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := readSubPilotResponse(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("subpilot status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return json.Unmarshal(raw, out)
}

func readSubPilotResponse(body io.Reader) ([]byte, error) {
	raw, err := io.ReadAll(io.LimitReader(body, subPilotMaxResponseBodySize+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > subPilotMaxResponseBodySize {
		return nil, fmt.Errorf("subpilot response exceeds %d bytes", subPilotMaxResponseBodySize)
	}
	return raw, nil
}

func contextOrBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func (c *subPilotClient) reportTimeout() time.Duration {
	if c != nil && c.timeout > subPilotReportTimeout {
		return c.timeout
	}
	return subPilotReportTimeout
}

func (c *subPilotClient) setSharedSecret(req *http.Request) {
	if c != nil && req != nil && c.sharedSecret != "" {
		req.Header.Set("X-SubPilot-Secret", c.sharedSecret)
	}
}

func (c *subPilotClient) handleRuntimeError(message string, err error, failOpen bool) error {
	if c == nil || failOpen {
		if err != nil {
			slog.Warn(message, "error", err)
		} else {
			slog.Warn(message)
		}
		return nil
	}
	if err == nil {
		return errors.New(message)
	}
	return fmt.Errorf("%s: %w", message, err)
}

func subPilotRequestID(ctx context.Context) string {
	if ctx != nil {
		if v, _ := ctx.Value(ctxkey.ClientRequestID).(string); strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
		if v, _ := ctx.Value(ctxkey.RequestID).(string); strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return generateRequestID()
}

func subPilotRequestType(stream bool, openAIWSMode bool) string {
	if openAIWSMode {
		return "ws_v2"
	}
	if stream {
		return "stream"
	}
	return "sync"
}
