package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractXiaoVideoModelSpecsSupportsFlatUSDPrices(t *testing.T) {
	t.Parallel()

	specs, err := extractXiaoVideoModelSpecs([]byte(`{"data":[{"id":"native-video","resolutions":["720p","1080p"],"price_per_second":0.2,"currency":"USD","default_resolution":"720p","default_duration":5}]}`))
	require.NoError(t, err)
	require.Len(t, specs, 2)
	require.Equal(t, "native-video", specs[0].ID)
	require.Equal(t, "1080p", specs[0].Resolution)
	require.NotNil(t, specs[0].UpstreamCost)
	require.Equal(t, 0.2, *specs[0].UpstreamCost)
	require.Equal(t, "USD", specs[0].CostCurrency)
	require.Equal(t, "second", specs[0].CostUnit)
	require.Equal(t, 5, specs[0].DefaultDuration)
	require.Equal(t, "720p", specs[1].Resolution)
	require.True(t, specs[1].DefaultResolution)
}

func TestExtractXiaoVideoModelSpecsSupportsNestedCreditPrices(t *testing.T) {
	t.Parallel()

	specs, err := extractXiaoVideoModelSpecs([]byte(`{"data":{"models":{"12:provider-video":{"configs":[{"resolution":"720p","points":500,"unit":"request","duration":5},{"resolution":"1080p","price":750,"currency":"credits","billing_unit":"request","default_duration":5}]}}}}`))
	require.NoError(t, err)
	require.Len(t, specs, 2)
	for _, spec := range specs {
		require.Equal(t, "12:provider-video", spec.ID)
		require.Equal(t, "request", spec.CostUnit)
		require.Equal(t, 5, spec.DefaultDuration)
	}
	require.Equal(t, "CREDITS", specs[0].CostCurrency)
}

func TestExtractAIStartLabVideoModelSpecsUsesNestedPricingAndFiltersImages(t *testing.T) {
	t.Parallel()

	specs, err := extractAIStartLabVideoModelSpecs([]byte(`{
		"data": {
			"imageConfig": [{"models":[{"model":"gpt-image-2","qualities":[{"quality":"1K","pricing":{"type":"fixed_total","credits":10}}]}]}],
			"videoConfig": [
				{"defaultOption":false,"models":[{"model":"seedance-2.0","duration":{"min":4,"max":15},"qualities":[{"quality":"720p","pricing":{"type":"per_second","credits":58}}]}]},
				{"defaultOption":true,"models":[{"model":"seedance-2.0","duration":{"min":4,"max":15},"qualities":[{"quality":"480p","pricing":{"type":"per_second","credits":36}},{"quality":"720p","pricing":{"type":"per_second","credits":52}}]}]},
				{"defaultOption":false,"models":[{"model":"gemini-omni-flash","duration":{"min":10,"max":10},"qualities":[{"quality":"720p","pricing":{"type":"fixed_total","credits":150}}]}]}
			]
		}
	}`))
	require.NoError(t, err)
	require.Len(t, specs, 3)
	require.Equal(t, []string{"gemini-omni-flash", "seedance-2.0"}, modelIDsFromSpecs(specs))

	omni := specs[0]
	require.Equal(t, "gemini-omni-flash", omni.ID)
	require.Equal(t, "720p", omni.Resolution)
	require.Equal(t, 150.0, *omni.UpstreamCost)
	require.Equal(t, "CREDITS", omni.CostCurrency)
	require.Equal(t, "request", omni.CostUnit)
	require.Equal(t, 10, omni.DefaultDuration)
	require.True(t, omni.DefaultResolution)

	seedance720 := specs[2]
	require.Equal(t, "seedance-2.0", seedance720.ID)
	require.Equal(t, "720p", seedance720.Resolution)
	require.Equal(t, 52.0, *seedance720.UpstreamCost)
	require.Equal(t, "second", seedance720.CostUnit)
	require.Equal(t, 4, seedance720.DefaultDuration)
	require.True(t, seedance720.DefaultResolution)
}

func TestExtractAIStartLabVideoModelSpecsPrefixesChannelAndKeepsCapabilities(t *testing.T) {
	t.Parallel()

	specs, err := extractAIStartLabVideoModelSpecs([]byte(`{
		"data": {"videoConfig": [{"channel":"47","defaultOption":true,"models":[{
			"model":"seedance-2.0","modes":["text2video","image2video"],
			"aspectRatios":["16:9","9:16"],"duration":{"min":4,"max":6},
			"inputImagesMax":9,"inputVideosMax":3,"inputAudiosMax":3,
			"qualities":[{"quality":"720p","pricing":{"type":"per_second","credits":52}}]
		}] }]}
	}`))
	require.NoError(t, err)
	require.Len(t, specs, 1)
	require.Equal(t, "47:seedance-2.0", specs[0].ID)
	require.Equal(t, []int{4, 5, 6}, specs[0].Durations)
	require.Equal(t, []string{"16:9", "9:16"}, specs[0].AspectRatios)
	require.Equal(t, "16:9", specs[0].DefaultAspectRatio)
	require.True(t, specs[0].SupportsGuidances)
	require.Equal(t, map[string]int{"image": 9, "video": 3, "audio": 3}, specs[0].MaxReferences)
}

func TestFetchAIStartLabModelCatalogUsesCreditDefaults(t *testing.T) {
	t.Parallel()

	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body: io.NopCloser(strings.NewReader(`{"data":{
			"imageConfig":[{"models":[{"model":"image-only","qualities":[{"quality":"1K","pricing":{"type":"fixed_total","credits":10}}]}]}],
			"videoConfig":[{"defaultOption":true,"models":[{"model":"12:provider-video","duration":{"min":5,"max":15},"qualities":[{"quality":"720p","pricing":{"type":"per_second","credits":50}}]}]}]
		}}`)),
	}}
	svc := &AccountTestService{httpUpstream: upstream, cfg: upstreamModelSyncTestConfig()}

	catalog, err := svc.FetchUpstreamModelCatalog(context.Background(), &Account{
		ID:       42,
		Platform: PlatformXiaoAPI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":        "video-key",
			"base_url":       "https://api.video.aistarslab.com/openai",
			"video_protocol": XiaoVideoProtocolOpenAISora,
		},
	})
	require.NoError(t, err)
	require.Equal(t, upstreamPricingSourceAIStartLabConfig, catalog.PricingSource)
	require.Equal(t, []string{"12:provider-video"}, catalog.Models)
	require.Len(t, catalog.ModelSpecs, 1)
	require.Equal(t, "CREDITS", catalog.ModelSpecs[0].CostCurrency)
	require.Equal(t, "second", catalog.ModelSpecs[0].CostUnit)
	require.Equal(t, 5, catalog.ModelSpecs[0].DefaultDuration)
	require.Equal(t, "https://api.video.aistarslab.com/openapi/generation/config", upstream.lastReq.URL.String())
	require.Equal(t, "Bearer video-key", upstream.lastReq.Header.Get("Authorization"))
}

func TestFetchNativeXiaoVideoCatalogDefaultsMissingCurrencyToUSD(t *testing.T) {
	t.Parallel()

	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"data":[{"id":"native-video","resolution":"720p","price_per_second":0.2}]}`)),
	}}
	svc := &AccountTestService{httpUpstream: upstream, cfg: upstreamModelSyncTestConfig()}

	catalog, err := svc.FetchUpstreamModelCatalog(context.Background(), &Account{
		ID:       43,
		Platform: PlatformXiaoAPI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "video-key",
			"base_url": "https://xiao.example.com",
		},
	})
	require.NoError(t, err)
	require.Equal(t, upstreamPricingSourceModelList, catalog.PricingSource)
	require.Len(t, catalog.ModelSpecs, 1)
	require.Equal(t, "USD", catalog.ModelSpecs[0].CostCurrency)
	require.Equal(t, "second", catalog.ModelSpecs[0].CostUnit)
}

func TestHasCompleteCatalogPricingRequiresCostUnitAndDuration(t *testing.T) {
	t.Parallel()
	cost := 0.2
	require.True(t, hasCompleteCatalogPricing(UpstreamModelSpec{
		UpstreamCost: &cost, CostCurrency: "USD", CostUnit: "second", DefaultDuration: 5,
	}))
	require.False(t, hasCompleteCatalogPricing(UpstreamModelSpec{
		UpstreamCost: &cost, CostCurrency: "USD", CostUnit: "second",
	}))
	require.False(t, hasCompleteCatalogPricing(UpstreamModelSpec{
		UpstreamCost: &cost, CostCurrency: "USD", DefaultDuration: 5,
	}))
}

func TestFetchAIStartLabCatalogMarksModelListFallback(t *testing.T) {
	t.Parallel()

	upstream := &httpUpstreamRecorder{responses: []*http.Response{
		{
			StatusCode: http.StatusNotFound,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("{\"error\":\"not found\"}")),
		},
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader("{\"data\":[{\"id\":\"model-without-price\"}]}")),
		},
	}}
	svc := &AccountTestService{httpUpstream: upstream, cfg: upstreamModelSyncTestConfig()}

	catalog, err := svc.FetchUpstreamModelCatalog(context.Background(), &Account{
		ID:       44,
		Platform: PlatformXiaoAPI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":        "video-key",
			"base_url":       "https://api.video.aistarslab.com/openai",
			"video_protocol": XiaoVideoProtocolOpenAISora,
		},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"model-without-price"}, catalog.Models)
	require.Equal(t, upstreamPricingSourceModelList, catalog.PricingSource)
	require.Equal(t, upstreamPricingNoteAIStartLabUnavailable, catalog.PricingNote)
	require.Len(t, upstream.requests, 2)
	require.Equal(t, "/openapi/generation/config", upstream.requests[0].URL.Path)
	require.Equal(t, "/openai/v1/models", upstream.requests[1].URL.Path)
}

func TestMarkIncompleteCatalogPricingChecksEveryResolution(t *testing.T) {
	t.Parallel()
	cost := 0.2
	catalog := &UpstreamModelCatalog{
		Models: []string{"video"},
		ModelSpecs: []UpstreamModelSpec{
			{ID: "video", Resolution: "720p", UpstreamCost: &cost, CostCurrency: "USD", CostUnit: "second", DefaultDuration: 5},
			{ID: "video", Resolution: "1080p"},
		},
	}
	markIncompleteCatalogPricing(catalog)
	require.Equal(t, upstreamPricingNoteIncomplete, catalog.PricingNote)
}
