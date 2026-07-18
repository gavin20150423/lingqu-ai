package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// RegisterUserRoutes 注册用户相关路由（需要认证）
func RegisterUserRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth middleware.JWTAuthMiddleware,
	auditLog middleware.AuditLogMiddleware,
	settingService *service.SettingService,
) {
	authenticated := v1.Group("")
	authenticated.Use(gin.HandlerFunc(jwtAuth))
	authenticated.Use(middleware.BackendModeUserGuard(settingService))
	// 用户管理面变更类操作入审计（含 TOTP 启用/禁用、step-up 验证、密码修改等安全事件）
	authenticated.Use(gin.HandlerFunc(auditLog))
	{
		// 用户接口
		user := authenticated.Group("/user")
		{
			user.GET("/profile", h.User.GetProfile)
			user.PUT("/password", h.User.ChangePassword)
			user.PUT("", h.User.UpdateProfile)
			user.GET("/aff", h.User.GetAffiliate)
			user.POST("/aff/transfer", h.User.TransferAffiliateQuota)
			user.POST("/account-bindings/email/send-code", h.User.SendEmailBindingCode)
			user.POST("/account-bindings/email", h.User.BindEmailIdentity)
			user.DELETE("/account-bindings/:provider", h.User.UnbindIdentity)
			user.POST("/auth-identities/bind/start", h.User.StartIdentityBinding)
			user.GET("/api-keys/:id/usage/daily", h.Usage.GetMyAPIKeyDailyUsage)
			user.GET("/platform-quotas", h.User.GetMyPlatformQuotas)

			// 通知邮箱管理
			notifyEmail := user.Group("/notify-email")
			{
				notifyEmail.POST("/send-code", h.User.SendNotifyEmailCode)
				notifyEmail.POST("/verify", h.User.VerifyNotifyEmail)
				notifyEmail.PUT("/toggle", h.User.ToggleNotifyEmail)
				notifyEmail.DELETE("", h.User.RemoveNotifyEmail)
			}

			// TOTP 双因素认证
			totp := user.Group("/totp")
			{
				totp.GET("/status", h.Totp.GetStatus)
				totp.GET("/verification-method", h.Totp.GetVerificationMethod)
				totp.POST("/send-code", h.Totp.SendVerifyCode)
				totp.POST("/setup", h.Totp.InitiateSetup)
				totp.POST("/enable", h.Totp.Enable)
				totp.POST("/disable", h.Totp.Disable)
				// 敏感操作二次验证：授予当前会话一段时间的 step-up 权限
				totp.POST("/step-up", h.Totp.StepUp)
			}
		}

		// API Key管理
		keys := authenticated.Group("/keys")
		{
			keys.GET("", h.APIKey.List)
			keys.GET("/:id", h.APIKey.GetByID)
			keys.POST("", h.APIKey.Create)
			keys.PUT("/:id", h.APIKey.Update)
			keys.DELETE("/:id", h.APIKey.Delete)
		}

		// 用户可用分组（非管理员接口）
		groups := authenticated.Group("/groups")
		{
			groups.GET("/available", h.APIKey.GetAvailableGroups)
			groups.GET("/rates", h.APIKey.GetUserGroupRates)
		}

		// 用户可用渠道（非管理员接口）
		channels := authenticated.Group("/channels")
		{
			channels.GET("/available", h.AvailableChannel.List)
		}

		// 使用记录
		usage := authenticated.Group("/usage")
		{
			usage.GET("", h.Usage.List)
			usage.GET("/errors", h.Usage.ListErrors)
			usage.GET("/errors/:id", h.Usage.GetErrorDetail)
			usage.GET("/:id", h.Usage.GetByID)
			usage.GET("/stats", h.Usage.Stats)
			// User dashboard endpoints
			usage.GET("/dashboard/stats", h.Usage.DashboardStats)
			usage.GET("/dashboard/trend", h.Usage.DashboardTrend)
			usage.GET("/dashboard/models", h.Usage.DashboardModels)
			usage.GET("/dashboard/snapshot-v2", h.Usage.DashboardSnapshotV2)
			usage.POST("/dashboard/api-keys-usage", h.Usage.DashboardAPIKeysUsage)
		}

		// 公告（用户可见）
		announcements := authenticated.Group("/announcements")
		{
			announcements.GET("", h.Announcement.List)
			announcements.POST("/:id/read", h.Announcement.MarkRead)
		}

		// 卡密兑换
		redeem := authenticated.Group("/redeem")
		{
			redeem.POST("", h.Redeem.Redeem)
			redeem.GET("/history", h.Redeem.GetHistory)
		}

		// 用户订阅
		subscriptions := authenticated.Group("/subscriptions")
		{
			subscriptions.GET("", h.Subscription.List)
			subscriptions.GET("/active", h.Subscription.GetActive)
			subscriptions.GET("/progress", h.Subscription.GetProgress)
			subscriptions.GET("/summary", h.Subscription.GetSummary)
		}

		// 渠道监控（用户只读）
		monitors := authenticated.Group("/channel-monitors")
		{
			monitors.GET("", h.ChannelMonitor.List)
			monitors.GET("/:id/status", h.ChannelMonitor.GetStatus)
		}

		community := authenticated.Group("/community")
		{
			community.GET("/accounts", h.Community.ListAccounts)
			community.POST("/accounts", h.Community.CreateAccount)
			community.POST("/accounts/import", h.Community.ImportAccounts)
			community.POST("/accounts/export", h.Community.ExportAccounts)
			community.PATCH("/accounts/batch", h.Community.BatchUpdateAccounts)
			community.POST("/accounts/oauth/auth-url", h.Community.GenerateAccountOAuthURL)
			community.POST("/accounts/oauth/exchange", h.Community.ExchangeAccountOAuth)
			community.GET("/proxies", h.Community.ListProxies)
			community.POST("/proxies", h.Community.SaveProxy)
			community.PUT("/proxies/:id", h.Community.SaveProxy)
			community.DELETE("/proxies/:id", h.Community.DeleteProxy)
			community.DELETE("/accounts/:id", h.Community.DeleteAccount)
			community.POST("/listings", h.Community.CreateListing)
			community.GET("/marketplace", h.Community.ListMarketplace)
			community.GET("/owner/listings", h.Community.ListOwnerListings)
			community.GET("/memberships", h.Community.ListMemberships)
			community.GET("/consumption/accounts", h.Community.ListConsumptionAccounts)
			community.GET("/consumption/:membership_id", h.Community.GetConsumptionSummary)
			community.POST("/selection-assistant", h.Community.RecommendListings)
			community.GET("/selection-assistant/recent/:api_key_id", h.Community.GetRecentUsageAverage)
			community.GET("/owner/summary", h.Community.GetOwnerSummary)
			community.PUT("/listings/:id/status", h.Community.SetListingStatus)
			community.GET("/account-mode-keys", h.Community.ListAccountModeKeys)
			community.PUT("/account-mode-keys/:id", h.Community.SetAccountModeKey)
			community.POST("/marketplace/:id/join", h.Community.JoinListing)
			community.DELETE("/marketplace/:id/membership", h.Community.LeaveListing)
			community.POST("/memberships/:id/review", h.Community.ReviewMembership)
			community.GET("/tickets", h.Community.ListTickets)
			community.POST("/tickets", h.Community.CreateTicket)
			community.GET("/tickets/:id", h.Community.GetTicket)
			community.POST("/tickets/:id/messages", h.Community.ReplyTicket)
			community.POST("/tickets/:id/read", h.Community.MarkTicketRead)
			community.POST("/tickets/:id/close", h.Community.CloseTicket)
			community.GET("/payout-methods", h.Community.ListPayoutMethods)
			community.PUT("/payout-methods", h.Community.SavePayoutMethod)
			community.DELETE("/payout-methods/:method", h.Community.DeletePayoutMethod)
			community.GET("/withdrawals", h.Community.ListWithdrawals)
			community.POST("/withdrawals", h.Community.CreateWithdrawal)
			community.POST("/withdrawals/:id/cancel", h.Community.CancelWithdrawal)
			community.GET("/store/products", h.Community.ListProducts)
			community.GET("/store/wallet", h.Community.GetStoreWallet)
			community.POST("/store/products/:id/buy", h.Community.BuyProduct)
			community.POST("/store/products/:id/platform-order", h.Community.CreatePlatformStoreOrder)
			community.GET("/store/orders", h.Community.ListStoreOrders)
		}
	}
}
