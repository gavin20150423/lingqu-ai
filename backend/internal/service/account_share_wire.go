package service

import "github.com/Wei-Shaw/sub2api/internal/config"

func ProvideAccountShareModeService(
	cfg *config.Config,
	repo AccountShareModeRepository,
	accountRepo AccountRepository,
	apiKeyRepo APIKeyRepository,
	userRepo UserRepository,
	proxyRepo ProxyRepository,
	openAIOAuth *OpenAIOAuthService,
	oauth *OAuthService,
	concurrency *ConcurrencyService,
	invalidator APIKeyAuthCacheInvalidator,
	accountTest *AccountTestService,
	rateLimit *RateLimitService,
	billingCache *BillingCacheService,
	billing *BillingService,
	pricing *ModelPricingResolver,
	settings SettingRepository,
	settingService *SettingService,
	openAIGateway *OpenAIGatewayService,
) *AccountShareModeService {
	service := NewAccountShareModeService(repo, accountRepo, apiKeyRepo, userRepo, proxyRepo, openAIOAuth, oauth)
	if cfg != nil {
		service.SetActionTokenSecret(cfg.JWT.Secret)
	}
	service.SetRuntimeDependencies(concurrency, invalidator, accountTest, rateLimit)
	service.SetBillingCacheService(billingCache)
	service.SetRecommendationPricingDependencies(billing, pricing)
	service.SetSettingService(settingService)
	service.SetReviewModerationSettingRepository(settings)
	openAIGateway.SetAccountShareModeService(service)
	service.StartSeatBillingWorker()
	service.StartReviewModerationWorker()
	return service
}
