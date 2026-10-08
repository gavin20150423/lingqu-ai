//go:build unit

package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

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
		// 账号本身是"可用"的，只是处在**运行时**压力下（限流窗口内）。
		// 这才是单账号强制路径的原意：分组只绑了一个账号时，操作员已经用绑定
		// 做出了选择，不该因为瞬态运行时状态就硬失败——等待计划会兜住。
		// ⚠️ 这里刻意**不用** disabled/schedulable=false 当例子：那属于操作员显式禁用，
		// 必须无条件生效（见下一个用例，也是 2026-10-06 由 804/kiro-95 事故定案的修法）。
		rateLimitedUntil := time.Now().Add(30 * time.Minute)
		account := Account{
			ID:               786,
			Name:             "only-account",
			Platform:         PlatformAnthropic,
			Status:           StatusActive,
			Schedulable:      true,
			Concurrency:      5,
			RateLimitResetAt: &rateLimitedUntil,
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

	// 2026-10-06 由 804/kiro-95 事故定案：操作员把账号 disabled / schedulable=false
	// 之后，单账号分组的强制路径仍会把它推去调度并真实发出流量。
	// 修法是这类显式禁用无条件生效：不走强制路径，交回正常调度。
	t.Run("skips explicitly disabled single account", func(t *testing.T) {
		groupID := int64(904)
		account := Account{
			ID:          921,
			Name:        "disabled-only-account",
			Platform:    PlatformAnthropic,
			Status:      StatusDisabled,
			Schedulable: false,
			Concurrency: 5,
		}
		svc := &GatewayService{
			accountRepo: &mockAccountRepoForPlatform{
				accountsByID: map[int64]*Account{account.ID: &account},
			},
			groupRepo: &mockGroupRepoForGateway{
				singleAccounts: map[int64]int64{groupID: account.ID},
			},
			cfg:                testConfig(),
			concurrencyService: NewConcurrencyService(&mockConcurrencyCache{}),
		}

		result, forced, err := svc.tryForceSingleAccountGroup(ctx, &groupID)

		require.NoError(t, err)
		require.False(t, forced, "禁用账号不得被单账号强制路径选中")
		require.Nil(t, result)
	})

	t.Run("skips account with manual schedulable off", func(t *testing.T) {
		groupID := int64(905)
		account := Account{
			ID:          922,
			Platform:    PlatformAnthropic,
			Status:      StatusActive,
			Schedulable: false, // 只是手动关了可调度开关，状态仍是 active
			Concurrency: 5,
		}
		svc := &GatewayService{
			accountRepo: &mockAccountRepoForPlatform{
				accountsByID: map[int64]*Account{account.ID: &account},
			},
			groupRepo: &mockGroupRepoForGateway{
				singleAccounts: map[int64]int64{groupID: account.ID},
			},
			cfg:                testConfig(),
			concurrencyService: NewConcurrencyService(&mockConcurrencyCache{}),
		}

		result, forced, err := svc.tryForceSingleAccountGroup(ctx, &groupID)

		require.NoError(t, err)
		require.False(t, forced, "手动关掉 schedulable 的账号同样不得被强制选中")
		require.Nil(t, result)
	})

	t.Run("retains concurrency guard", func(t *testing.T) {
		groupID := int64(902)
		account := Account{ID: 787, Platform: PlatformAnthropic, Status: StatusActive, Schedulable: true, Concurrency: 1}
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
