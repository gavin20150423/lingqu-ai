package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

type subPilotSoftCooldownAccountRepo struct {
	AccountRepository
	account *Account
}

type subPilotStickyTrackingCache struct {
	schedulerTestGatewayCache
	setCalls     int
	refreshCalls int
}

func (c *subPilotStickyTrackingCache) SetSessionAccountID(ctx context.Context, groupID int64, sessionHash string, accountID int64, ttl time.Duration) error {
	c.setCalls++
	return c.schedulerTestGatewayCache.SetSessionAccountID(ctx, groupID, sessionHash, accountID, ttl)
}

func (c *subPilotStickyTrackingCache) RefreshSessionTTL(ctx context.Context, groupID int64, sessionHash string, ttl time.Duration) error {
	c.refreshCalls++
	return c.schedulerTestGatewayCache.RefreshSessionTTL(ctx, groupID, sessionHash, ttl)
}

func (r subPilotSoftCooldownAccountRepo) GetByID(_ context.Context, id int64) (*Account, error) {
	if r.account != nil && r.account.ID == id {
		return r.account, nil
	}
	return nil, errors.New("account not found")
}

func (r subPilotSoftCooldownAccountRepo) ListSchedulableByGroupIDAndPlatform(context.Context, int64, string) ([]Account, error) {
	return nil, nil
}

func TestNewSubPilotSelectRequestIncludesAPIKeyID(t *testing.T) {
	ctx := WithSubPilotAPIKeyID(context.Background(), 123)

	req := newSubPilotSelectRequest(ctx, PlatformOpenAI, 45, "gpt-5.4", "session-1")

	require.Equal(t, "123", req.APIKeyID)
	require.Equal(t, "45", req.GroupID)
	require.Equal(t, PlatformOpenAI, req.Platform)
	require.Equal(t, "gpt-5.4", req.Model)
	require.Equal(t, "session-1", req.SessionKey)
	require.NotEmpty(t, req.RequestID)
}

func TestSubPilotReportFailureFallsBackToContextAPIKeyID(t *testing.T) {
	requests := make(chan subPilotReportFailureRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/dispatch/report-failure", r.URL.Path)
		var req subPilotReportFailureRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		requests <- req
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	svc := &OpenAIGatewayService{cfg: &config.Config{
		Gateway: config.GatewayConfig{
			SubPilot: config.SubPilotConfig{
				Enabled:   true,
				BaseURL:   server.URL,
				TimeoutMS: 500,
			},
		},
	}}
	ctx := WithSubPilotAPIKeyID(context.Background(), 789)
	svc.ReportSubPilotFailure(ctx, SubPilotFailureInput{
		LeaseID:     "lease-1",
		Account:     &Account{ID: 11},
		RequestID:   "request-1",
		Platform:    PlatformOpenAI,
		GroupID:     "22",
		Model:       "gpt-5.4",
		SessionKey:  "session-1",
		StatusCode:  http.StatusBadRequest,
		ErrorCode:   "upstream_400",
		RequestType: "stream",
	})

	select {
	case req := <-requests:
		require.Equal(t, "789", req.APIKeyID)
		require.Equal(t, "22", req.GroupID)
		require.Equal(t, "11", req.AccountID)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for SubPilot failure report")
	}
}

func TestOpenAISubPilotSuccessReportCarriesLeaseAndSession(t *testing.T) {
	requests := make(chan subPilotReportSuccessRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, subPilotReportSuccessPath, r.URL.Path)
		var req subPilotReportSuccessRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		requests <- req
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	svc := &OpenAIGatewayService{cfg: &config.Config{
		Gateway: config.GatewayConfig{
			SubPilot: config.SubPilotConfig{
				Enabled:   true,
				BaseURL:   server.URL,
				TimeoutMS: 500,
			},
		},
	}}
	svc.reportSubPilotSuccess(
		context.Background(),
		&UsageLog{RequestID: "request-success", APIKeyID: 456, AccountID: 11, RequestedModel: "gpt-5.6"},
		&OpenAIRecordUsageInput{
			SubPilotLeaseID:    "lease-success",
			SubPilotSessionKey: "session-success",
			QuotaPlatform:      PlatformOpenAI,
		},
		nil,
		1,
	)

	select {
	case req := <-requests:
		require.Equal(t, "lease-success", req.LeaseID)
		require.Equal(t, "session-success", req.SessionKey)
		require.Equal(t, "11", req.AccountID)
		require.Equal(t, "gpt-5.6", req.Model)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for SubPilot success report")
	}
}

func TestSubPilotReportUsesLongerMinimumTimeoutThanSelect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(150 * time.Millisecond)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newSubPilotClient(config.SubPilotConfig{
		Enabled:   true,
		BaseURL:   server.URL,
		TimeoutMS: 80,
	})
	require.NotNil(t, client)

	err := client.postJSONWithTimeout(context.Background(), "/v1/dispatch/report-failure", subPilotReportFailureRequest{
		RequestID:  "request-slow-report",
		LeaseID:    "lease-slow-report",
		AccountID:  "11",
		Platform:   PlatformOpenAI,
		GroupID:    "22",
		Model:      "gpt-5.4",
		StatusCode: http.StatusBadRequest,
	}, nil, client.reportTimeout())
	require.NoError(t, err)
}

func TestSubPilotReportAPIKeyIDPrefersExplicitAPIKey(t *testing.T) {
	ctx := WithSubPilotAPIKeyID(context.Background(), 789)
	require.Equal(t, "456", subPilotReportAPIKeyID(ctx, &APIKey{ID: 456}))
}

func TestGatewaySubPilotAcceptsExplicitLastResortExcludedAccount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, subPilotSelectPath, r.URL.Path)
		var req subPilotSelectRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.Equal(t, []string{"36201"}, req.ExcludedAccountIDs)
		_, _ = w.Write([]byte(`{"decision":"selected","reason":"last_resort","account":{"id":"36201"},"lease":{"id":"lease-36201"}}`))
	}))
	defer server.Close()

	groupID := int64(10121)
	account := Account{
		ID: 36201, Platform: PlatformAnthropic, Type: AccountTypeAPIKey,
		Status: StatusActive, Schedulable: true, Concurrency: 1,
	}
	cache := &subPilotStickyTrackingCache{}
	svc := &GatewayService{cache: cache, cfg: &config.Config{Gateway: config.GatewayConfig{
		SubPilot: config.SubPilotConfig{Enabled: true, BaseURL: server.URL, TimeoutMS: 500},
	}}}

	selection, handled, err := svc.trySubPilotRecommend(
		context.Background(), &groupID, PlatformAnthropic, "gateway-last-resort-session", "claude-opus-4-6",
		map[int64]struct{}{account.ID: {}}, []Account{account}, false,
	)
	require.NoError(t, err)
	require.True(t, handled)
	require.Equal(t, account.ID, selection.Account.ID)
	require.Equal(t, "lease-36201", selection.SubPilotLeaseID)
	require.Zero(t, cache.setCalls)
	require.Zero(t, cache.refreshCalls)
}

func TestGatewaySubPilotLastResortReloadsSoftCooldownAccount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, subPilotSelectPath, r.URL.Path)
		_, _ = w.Write([]byte(`{"decision":"selected","reason":"last_resort","account":{"id":"36202"},"lease":{"id":"lease-36202"}}`))
	}))
	defer server.Close()

	groupID := int64(10122)
	cooldownUntil := time.Now().Add(time.Minute)
	account := &Account{
		ID: 36202, Platform: PlatformAnthropic, Type: AccountTypeAPIKey,
		Status: StatusActive, Schedulable: true, Concurrency: 1,
		TempUnschedulableUntil: &cooldownUntil,
		AccountGroups:          []AccountGroup{{GroupID: groupID}},
	}
	svc := &GatewayService{
		accountRepo: subPilotSoftCooldownAccountRepo{account: account},
		cfg: &config.Config{Gateway: config.GatewayConfig{SubPilot: config.SubPilotConfig{
			Enabled: true, BaseURL: server.URL, TimeoutMS: 500,
		}}},
	}

	selection, handled, err := svc.trySubPilotRecommend(
		context.Background(), &groupID, PlatformAnthropic, "", "claude-opus-4-6",
		nil, nil, false,
	)
	require.NoError(t, err)
	require.True(t, handled)
	require.Equal(t, account.ID, selection.Account.ID)
	require.Equal(t, "lease-36202", selection.SubPilotLeaseID)
}

func TestGatewaySubPilotLastResortRunsWhenSchedulableProjectionIsEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case subPilotRuntimeConfigPath:
			_ = json.NewEncoder(w).Encode(subPilotRuntimeConfig{
				DispatchEnabled: true, DispatchFailOpen: true, DispatchSelectTimeoutMS: 500,
				DispatchAutoBypassFailures: 3, DispatchAutoRecover: true,
			})
		case subPilotSelectPath:
			_, _ = w.Write([]byte(`{"decision":"selected","reason":"last_resort","account":{"id":"36205"},"lease":{"id":"lease-36205"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	groupID := int64(10125)
	cooldownUntil := time.Now().Add(time.Minute)
	account := &Account{
		ID: 36205, Platform: PlatformAnthropic, Type: AccountTypeAPIKey,
		Status: StatusActive, Schedulable: true, Concurrency: 1,
		TempUnschedulableUntil: &cooldownUntil,
		AccountGroups:          []AccountGroup{{GroupID: groupID}},
	}
	cfg := &config.Config{}
	cfg.Gateway.Scheduling.LoadBatchEnabled = true
	cfg.Gateway.SubPilot = config.SubPilotConfig{Enabled: true, BaseURL: server.URL, TimeoutMS: 500}
	svc := &GatewayService{
		accountRepo:        subPilotSoftCooldownAccountRepo{account: account},
		cfg:                cfg,
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}
	ctx := context.WithValue(context.Background(), ctxkey.ForcePlatform, PlatformAnthropic)

	selection, err := svc.SelectAccountWithLoadAwareness(
		ctx, &groupID, "", "claude-opus-4-6", nil, "", 0,
	)
	require.NoError(t, err)
	require.Equal(t, account.ID, selection.Account.ID)
	require.Equal(t, "lease-36205", selection.SubPilotLeaseID)
}

func TestOpenAISubPilotLastResortRunsWhenSchedulableProjectionIsEmpty(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case subPilotRuntimeConfigPath:
			_ = json.NewEncoder(w).Encode(subPilotRuntimeConfig{
				DispatchEnabled: true, DispatchFailOpen: true, DispatchSelectTimeoutMS: 500,
				DispatchAutoBypassFailures: 3, DispatchAutoRecover: true,
			})
		case subPilotSelectPath:
			_, _ = w.Write([]byte(`{"decision":"selected","reason":"last_resort","account":{"id":"36203"},"lease":{"id":"lease-36203"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	groupID := int64(10123)
	cooldownUntil := time.Now().Add(time.Minute)
	account := &Account{
		ID: 36203, Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Status: StatusActive, Schedulable: true, Concurrency: 1,
		TempUnschedulableUntil: &cooldownUntil,
		AccountGroups:          []AccountGroup{{GroupID: groupID}},
	}
	cfg := &config.Config{}
	cfg.Gateway.SubPilot = config.SubPilotConfig{Enabled: true, BaseURL: server.URL, TimeoutMS: 500}
	svc := &OpenAIGatewayService{
		accountRepo:        subPilotSoftCooldownAccountRepo{account: account},
		cache:              &schedulerTestGatewayCache{},
		cfg:                cfg,
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	selection, decision, err := svc.SelectAccountWithScheduler(
		context.Background(), &groupID, "", "", "gpt-5.4", nil,
		OpenAIUpstreamTransportAny, false,
	)
	require.NoError(t, err)
	require.Equal(t, account.ID, selection.Account.ID)
	require.Equal(t, "lease-36203", selection.SubPilotLeaseID)
	require.Equal(t, "subpilot", decision.Layer)
}

func TestSubPilotLastResortStillRejectsManuallyDisabledAccount(t *testing.T) {
	groupID := int64(10124)
	account := &Account{
		ID: 36204, Platform: PlatformAnthropic, Type: AccountTypeAPIKey,
		Status: StatusActive, Schedulable: false,
		AccountGroups: []AccountGroup{{GroupID: groupID}},
	}
	svc := &GatewayService{accountRepo: subPilotSoftCooldownAccountRepo{account: account}, cfg: &config.Config{}}

	selected := svc.validateSubPilotGatewayAccount(
		context.Background(), account.ID, &groupID, PlatformAnthropic,
		"claude-opus-4-6", nil, false, true,
	)
	require.Nil(t, selected)
}
