//go:build unit

package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestGatewaySingleAccountGroupForce(t *testing.T) {
	ctx := context.Background()

	t.Run("bypasses runtime status and exclusions", func(t *testing.T) {
		var subPilotCalls atomic.Int64
		subPilot := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			subPilotCalls.Add(1)
			http.Error(w, "single-account force must bypass SubPilot", http.StatusServiceUnavailable)
		}))
		defer subPilot.Close()

		groupID := int64(901)
		account := Account{
			ID:          786,
			Name:        "only-account",
			Platform:    PlatformAnthropic,
			Status:      StatusDisabled,
			Schedulable: false,
			Concurrency: 5,
		}
		repo := &mockAccountRepoForPlatform{
			accountsByID: map[int64]*Account{account.ID: &account},
		}
		groupRepo := &mockGroupRepoForGateway{
			groups: map[int64]*Group{
				groupID: {ID: groupID, Platform: PlatformAnthropic, Status: StatusActive, Hydrated: true},
			},
			singleAccounts: map[int64]int64{groupID: account.ID},
		}
		concurrencyCache := &mockConcurrencyCache{}
		cfg := testConfig()
		cfg.Gateway.SubPilot = config.SubPilotConfig{Enabled: true, BaseURL: subPilot.URL, TimeoutMS: 500}
		svc := &GatewayService{
			accountRepo:        repo,
			groupRepo:          groupRepo,
			cfg:                cfg,
			concurrencyService: NewConcurrencyService(concurrencyCache),
		}

		result, err := svc.SelectAccountWithLoadAwareness(
			ctx,
			&groupID,
			"",
			"claude-sonnet-4-6",
			map[int64]struct{}{account.ID: {}},
			"",
			0,
		)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, account.ID, result.Account.ID)
		require.True(t, result.Acquired)
		require.Nil(t, result.WaitPlan)
		require.Equal(t, 1, concurrencyCache.acquireAccountCalls)
		require.Zero(t, subPilotCalls.Load())
	})

	t.Run("retains concurrency guard", func(t *testing.T) {
		groupID := int64(902)
		account := Account{ID: 787, Platform: PlatformAnthropic, Concurrency: 1}
		repo := &mockAccountRepoForPlatform{
			accountsByID: map[int64]*Account{account.ID: &account},
		}
		groupRepo := &mockGroupRepoForGateway{
			singleAccounts: map[int64]int64{groupID: account.ID},
		}
		concurrencyCache := &mockConcurrencyCache{
			acquireResults: map[int64]bool{account.ID: false},
		}
		svc := &GatewayService{
			accountRepo:        repo,
			groupRepo:          groupRepo,
			cfg:                testConfig(),
			concurrencyService: NewConcurrencyService(concurrencyCache),
		}

		result, forced, err := svc.tryForceSingleAccountGroup(ctx, &groupID)

		require.NoError(t, err)
		require.True(t, forced)
		require.False(t, result.Acquired)
		require.NotNil(t, result.WaitPlan)
		require.Equal(t, account.ID, result.WaitPlan.AccountID)
	})

	t.Run("does not affect multi-account groups", func(t *testing.T) {
		groupID := int64(903)
		svc := &GatewayService{
			accountRepo: &mockAccountRepoForPlatform{accountsByID: map[int64]*Account{}},
			groupRepo:   &mockGroupRepoForGateway{},
			cfg:         testConfig(),
		}

		result, forced, err := svc.tryForceSingleAccountGroup(ctx, &groupID)

		require.NoError(t, err)
		require.False(t, forced)
		require.Nil(t, result)
	})
}
