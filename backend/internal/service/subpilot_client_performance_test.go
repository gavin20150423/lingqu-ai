package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestSubPilotRuntimeConfigRefreshIsCoalesced(t *testing.T) {
	var configCalls atomic.Int64
	var selectCalls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "shared-secret", r.Header.Get("X-SubPilot-Secret"))
		switch r.URL.Path {
		case subPilotRuntimeConfigPath:
			configCalls.Add(1)
			time.Sleep(30 * time.Millisecond)
			_ = json.NewEncoder(w).Encode(subPilotRuntimeConfig{
				DispatchEnabled: true, DispatchFailOpen: true, DispatchSelectTimeoutMS: 200,
				DispatchAutoBypassFailures: 3, DispatchAutoRecover: true,
			})
		case subPilotSelectPath:
			selectCalls.Add(1)
			_, _ = w.Write([]byte(`{"decision":"selected","account":{"id":"1"},"lease":{"id":"lease-1"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := newSubPilotClient(config.SubPilotConfig{
		Enabled: true, BaseURL: server.URL, TimeoutMS: 200, FailOpen: true, SharedSecret: "shared-secret",
	})
	require.NotNil(t, client)
	var wg sync.WaitGroup
	for index := 0; index < 64; index++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rec, err := client.recommendAccount(context.Background(), subPilotSelectRequest{
				RequestID: "req", Platform: PlatformOpenAI, GroupID: "1", Model: "gpt-test",
			})
			if err != nil || rec == nil {
				t.Errorf("recommendAccount() rec=%+v err=%v", rec, err)
			}
		}()
	}
	wg.Wait()
	require.Eventually(t, func() bool {
		return configCalls.Load() == 1
	}, time.Second, 10*time.Millisecond)
	require.Equal(t, int64(64), selectCalls.Load())
}

func TestSubPilotReportEnqueueDoesNotWaitForNetwork(t *testing.T) {
	called := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, subPilotReportSuccessPath, r.URL.Path)
		time.Sleep(150 * time.Millisecond)
		called <- struct{}{}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newSubPilotClient(config.SubPilotConfig{
		Enabled: true, BaseURL: server.URL, TimeoutMS: 200, FailOpen: true,
	})
	require.NotNil(t, client)
	started := time.Now()
	client.reportSuccess(context.Background(), subPilotReportSuccessRequest{RequestID: "req", LeaseID: "lease", AccountID: "1"})
	require.Less(t, time.Since(started), 20*time.Millisecond)
	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("queued report was not delivered")
	}
}

func TestSubPilotRuntimeRefreshDoesNotDelaySelect(t *testing.T) {
	configStarted := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case subPilotRuntimeConfigPath:
			configStarted <- struct{}{}
			time.Sleep(150 * time.Millisecond)
			_ = json.NewEncoder(w).Encode(subPilotRuntimeConfig{
				DispatchEnabled: true, DispatchFailOpen: true, DispatchSelectTimeoutMS: 200,
				DispatchAutoBypassFailures: 3, DispatchAutoRecover: true,
			})
		case subPilotSelectPath:
			_, _ = w.Write([]byte(`{"decision":"selected","account":{"id":"1"},"lease":{"id":"lease-1"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := newSubPilotClient(config.SubPilotConfig{
		Enabled: true, BaseURL: server.URL, TimeoutMS: 300, FailOpen: true, SharedSecret: "shared-secret",
	})
	require.NotNil(t, client)
	started := time.Now()
	rec, err := client.recommendAccount(context.Background(), subPilotSelectRequest{
		RequestID: "req", Platform: PlatformOpenAI, GroupID: "1", Model: "gpt-test",
	})
	require.NoError(t, err)
	require.NotNil(t, rec)
	require.Less(t, time.Since(started), 100*time.Millisecond)
	select {
	case <-configStarted:
	case <-time.After(time.Second):
		t.Fatal("runtime refresh did not start")
	}
}

func TestSubPilotCircuitBypassesAfterConfiguredFailures(t *testing.T) {
	var selectCalls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case subPilotRuntimeConfigPath:
			_ = json.NewEncoder(w).Encode(subPilotRuntimeConfig{
				DispatchEnabled: true, DispatchFailOpen: true, DispatchSelectTimeoutMS: 200,
				DispatchAutoBypassFailures: 2, DispatchAutoRecover: false,
			})
		case subPilotSelectPath:
			selectCalls.Add(1)
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := newSubPilotClient(config.SubPilotConfig{
		Enabled: true, BaseURL: server.URL, TimeoutMS: 200, FailOpen: true, SharedSecret: "shared-secret",
	})
	require.NotNil(t, client)
	_ = client.runtimeConfig(context.Background())
	require.Eventually(t, func() bool {
		client.state.mu.RLock()
		defer client.state.mu.RUnlock()
		return client.state.runtime.DispatchAutoBypassFailures == 2
	}, time.Second, 10*time.Millisecond)
	for index := 0; index < 3; index++ {
		rec, err := client.recommendAccount(context.Background(), subPilotSelectRequest{
			RequestID: "req", Platform: PlatformOpenAI, GroupID: "1", Model: "gpt-test",
		})
		require.NoError(t, err)
		require.Nil(t, rec)
	}
	require.Equal(t, int64(2), selectCalls.Load())
	require.True(t, client.isBypassed())
}

func TestSubPilotCircuitAutomaticallyRecoversAfterRuntimeRefresh(t *testing.T) {
	var recoverEnabled atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case subPilotRuntimeConfigPath:
			_ = json.NewEncoder(w).Encode(subPilotRuntimeConfig{
				DispatchEnabled: true, DispatchFailOpen: true, DispatchSelectTimeoutMS: 200,
				DispatchAutoBypassFailures: 1, DispatchAutoRecover: recoverEnabled.Load(),
			})
		case subPilotSelectPath:
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := newSubPilotClient(config.SubPilotConfig{
		Enabled: true, BaseURL: server.URL, TimeoutMS: 200, FailOpen: true, SharedSecret: "shared-secret",
	})
	require.NotNil(t, client)
	_ = client.runtimeConfig(context.Background())
	require.Eventually(t, func() bool {
		client.state.mu.RLock()
		defer client.state.mu.RUnlock()
		return client.state.runtime.DispatchAutoBypassFailures == 1
	}, time.Second, 10*time.Millisecond)

	rec, err := client.recommendAccount(context.Background(), subPilotSelectRequest{
		RequestID: "req", Platform: PlatformOpenAI, GroupID: "1", Model: "gpt-test",
	})
	require.NoError(t, err)
	require.Nil(t, rec)
	require.True(t, client.isBypassed())

	recoverEnabled.Store(true)
	client.state.mu.Lock()
	client.state.lastRefresh = time.Now().Add(-subPilotConfigCacheTTL)
	client.state.mu.Unlock()
	_ = client.runtimeConfig(context.Background())
	require.Eventually(t, func() bool {
		return !client.isBypassed()
	}, time.Second, 10*time.Millisecond)
}

func TestSubPilotSharedHTTPClientKeepsEnoughIdleConnections(t *testing.T) {
	client := newSubPilotSharedHTTPClient()
	transport, ok := client.Transport.(*http.Transport)
	require.True(t, ok)
	require.Equal(t, subPilotMaxIdleConns, transport.MaxIdleConns)
	require.Equal(t, subPilotMaxIdleConns, transport.MaxIdleConnsPerHost)
}
