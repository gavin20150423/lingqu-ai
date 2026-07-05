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
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
)

const subPilotDefaultTimeout = 80 * time.Millisecond

type subPilotClient struct {
	baseURL  string
	timeout  time.Duration
	failOpen bool
	client   *http.Client
}

type subPilotSelectRequest struct {
	RequestID  string `json:"request_id"`
	APIKeyID   string `json:"api_key_id,omitempty"`
	Platform   string `json:"platform"`
	GroupID    string `json:"group_id,omitempty"`
	Model      string `json:"model"`
	SessionKey string `json:"session_key,omitempty"`
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
	return &subPilotClient{
		baseURL:  strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		timeout:  timeout,
		failOpen: cfg.FailOpen,
		client:   &http.Client{Timeout: timeout},
	}
}

func subPilotConfigFrom(cfg *config.Config) config.SubPilotConfig {
	if cfg == nil {
		return config.SubPilotConfig{}
	}
	return cfg.Gateway.SubPilot
}

func (c *subPilotClient) recommendAccount(ctx context.Context, req subPilotSelectRequest) (*subPilotSelectResult, error) {
	if c == nil {
		return nil, nil
	}
	var resp subPilotSelectResponse
	if err := c.postJSON(ctx, "/v1/dispatch/select", req, &resp); err != nil {
		return nil, c.handleError("subpilot select failed", err)
	}
	if resp.Decision != "selected" {
		return nil, nil
	}
	accountID, err := strconv.ParseInt(strings.TrimSpace(resp.Account.ID), 10, 64)
	if err != nil || accountID <= 0 || strings.TrimSpace(resp.Lease.ID) == "" {
		return nil, c.handleError("subpilot select returned invalid account", err)
	}
	return &subPilotSelectResult{AccountID: accountID, LeaseID: strings.TrimSpace(resp.Lease.ID)}, nil
}

func (c *subPilotClient) reportSuccess(ctx context.Context, req subPilotReportSuccessRequest) {
	if c == nil {
		return
	}
	if err := c.postJSON(ctx, "/v1/dispatch/report-success", req, nil); err != nil {
		slog.Warn("subpilot report success failed", "error", err)
	}
}

func (c *subPilotClient) reportFailure(ctx context.Context, req subPilotReportFailureRequest) {
	if c == nil {
		return
	}
	if err := c.postJSON(ctx, "/v1/dispatch/report-failure", req, nil); err != nil {
		slog.Warn("subpilot report failure failed", "error", err)
	}
}

func (c *subPilotClient) postJSON(ctx context.Context, path string, in any, out any) error {
	if c == nil {
		return nil
	}
	body, err := json.Marshal(in)
	if err != nil {
		return err
	}
	reqCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		limited, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("subpilot status %d: %s", resp.StatusCode, strings.TrimSpace(string(limited)))
	}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *subPilotClient) handleError(message string, err error) error {
	if c == nil || c.failOpen {
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
