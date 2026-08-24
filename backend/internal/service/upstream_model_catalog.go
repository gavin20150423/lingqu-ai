package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

const (
	upstreamModelsBodyLimit                  = 1 << 20
	upstreamPricingSourceNone                = "none"
	upstreamPricingSourceModelList           = "model_list"
	upstreamPricingSourceAIStartLabConfig    = "aistartlab_config"
	upstreamPricingNoteAIStartLabUnavailable = "aistartlab_config_unavailable"
	upstreamPricingNoteIncomplete            = "incomplete_pricing"
)

// UpstreamModelSpec is one billable model and resolution combination exposed by an upstream.
// Cost metadata remains optional because the OpenAI model-list format does not define pricing.
type UpstreamModelSpec struct {
	ID                 string         `json:"id"`
	Resolution         string         `json:"resolution,omitempty"`
	UpstreamCost       *float64       `json:"upstream_cost,omitempty"`
	CostCurrency       string         `json:"cost_currency,omitempty"`
	CostUnit           string         `json:"cost_unit,omitempty"`
	DefaultDuration    int            `json:"default_duration,omitempty"`
	DefaultResolution  bool           `json:"default_resolution,omitempty"`
	Durations          []int          `json:"durations,omitempty"`
	AspectRatios       []string       `json:"aspect_ratios,omitempty"`
	DefaultAspectRatio string         `json:"default_aspect_ratio,omitempty"`
	SupportsAudio      bool           `json:"supports_audio,omitempty"`
	SupportsGuidances  bool           `json:"supports_guidances,omitempty"`
	SupportsStartFrame bool           `json:"supports_start_frame,omitempty"`
	RequiresStartFrame bool           `json:"requires_start_frame,omitempty"`
	SupportsEndFrame   bool           `json:"supports_end_frame,omitempty"`
	MaxReferences      map[string]int `json:"max_references,omitempty"`
}

// UpstreamModelCatalog extends the common model list with optional pricing metadata.
type UpstreamModelCatalog struct {
	Models        []string            `json:"models"`
	ModelSpecs    []UpstreamModelSpec `json:"model_specs,omitempty"`
	PricingSource string              `json:"pricing_source"`
	PricingNote   string              `json:"pricing_note,omitempty"`
}

// FetchUpstreamModelCatalog keeps the common model sync contract while enriching XiaoAPI video accounts.
func (s *AccountTestService) FetchUpstreamModelCatalog(ctx context.Context, account *Account) (*UpstreamModelCatalog, error) {
	if account == nil || account.Platform != PlatformXiaoAPI {
		models, err := s.FetchUpstreamSupportedModels(ctx, account)
		if err != nil {
			return nil, err
		}
		return &UpstreamModelCatalog{Models: models, PricingSource: upstreamPricingSourceNone}, nil
	}

	// AIStartLab's OpenAI-compatible model endpoint intentionally omits pricing.
	// Its OpenAPI config endpoint is richer, but a deployment may not expose it,
	// so preserve model import by falling back to the standard model list.
	if strings.EqualFold(strings.TrimSpace(account.GetCredential("video_protocol")), XiaoVideoProtocolOpenAISora) {
		if catalog, err := s.fetchAIStartLabModelCatalog(ctx, account); err == nil && len(catalog.Models) > 0 {
			markIncompleteCatalogPricing(catalog)
			return catalog, nil
		}

		catalog, err := s.fetchXiaoVideoModelListCatalog(ctx, account)
		if err != nil {
			return nil, err
		}
		catalog.PricingNote = upstreamPricingNoteAIStartLabUnavailable
		return catalog, nil
	}

	catalog, err := s.fetchXiaoVideoModelListCatalog(ctx, account)
	if err != nil {
		return nil, err
	}
	markIncompleteCatalogPricing(catalog)
	return catalog, nil
}

func markIncompleteCatalogPricing(catalog *UpstreamModelCatalog) {
	if catalog == nil || len(catalog.Models) == 0 {
		return
	}
	completeModels := make(map[string]bool, len(catalog.Models))
	seenModels := make(map[string]bool, len(catalog.Models))
	for _, model := range catalog.Models {
		completeModels[model] = true
	}
	for _, spec := range catalog.ModelSpecs {
		if _, exists := completeModels[spec.ID]; !exists {
			continue
		}
		seenModels[spec.ID] = true
		completeModels[spec.ID] = completeModels[spec.ID] && hasCompleteCatalogPricing(spec)
	}
	for _, model := range catalog.Models {
		if !seenModels[model] || !completeModels[model] {
			catalog.PricingNote = upstreamPricingNoteIncomplete
			return
		}
	}
}

func hasCompleteCatalogPricing(spec UpstreamModelSpec) bool {
	if spec.UpstreamCost == nil || *spec.UpstreamCost < 0 || strings.TrimSpace(spec.CostCurrency) == "" {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(spec.CostUnit)) {
	case "second":
		return spec.DefaultDuration > 0
	case "request":
		return spec.DefaultDuration > 0
	default:
		return false
	}
}

func (s *AccountTestService) fetchAIStartLabModelCatalog(ctx context.Context, account *Account) (*UpstreamModelCatalog, error) {
	req, err := s.buildAIStartLabConfigRequest(ctx, account)
	if err != nil {
		return nil, err
	}
	body, err := s.fetchUpstreamCatalogBody(req, account, "AIStartLab generation config")
	if err != nil {
		return nil, err
	}
	specs, err := extractAIStartLabVideoModelSpecs(body)
	if err != nil {
		return nil, newUpstreamModelSyncUpstreamError("AIStartLab generation config was not valid JSON", err)
	}
	applyCatalogCostDefaults(specs, "CREDITS", true, true)
	models := modelIDsFromSpecs(specs)
	if len(models) == 0 {
		return nil, newUpstreamModelSyncUpstreamError("AIStartLab generation config returned no supported models", nil)
	}
	return &UpstreamModelCatalog{Models: models, ModelSpecs: specs, PricingSource: upstreamPricingSourceAIStartLabConfig}, nil
}

func (s *AccountTestService) fetchXiaoVideoModelListCatalog(ctx context.Context, account *Account) (*UpstreamModelCatalog, error) {
	req, err := s.buildXiaoVideoUpstreamModelsRequest(ctx, account)
	if err != nil {
		return nil, err
	}
	body, err := s.fetchUpstreamCatalogBody(req, account, "upstream model list")
	if err != nil {
		return nil, err
	}
	models, err := extractUpstreamModelIDs(body)
	if err != nil {
		return nil, newUpstreamModelSyncUpstreamError("Upstream model list response was not valid JSON", err)
	}
	if len(models) == 0 {
		return nil, newUpstreamModelSyncUpstreamError("Upstream returned no supported models", nil)
	}
	specs, _ := extractXiaoVideoModelSpecs(body)
	specs = filterSpecsByModelIDs(specs, models)
	applyCatalogCostDefaults(specs, "USD", false, false)
	return &UpstreamModelCatalog{Models: models, ModelSpecs: specs, PricingSource: upstreamPricingSourceModelList}, nil
}

func (s *AccountTestService) buildAIStartLabConfigRequest(ctx context.Context, account *Account) (*http.Request, error) {
	modelReq, err := s.buildXiaoVideoUpstreamModelsRequest(ctx, account)
	if err != nil {
		return nil, err
	}
	configURL, err := buildAIStartLabConfigURL(modelReq.URL.String())
	if err != nil {
		return nil, newUpstreamModelSyncConfigError("Invalid AIStartLab generation config URL", err)
	}
	req := modelReq.Clone(ctx)
	req.URL = configURL
	return req, nil
}

func buildAIStartLabConfigURL(rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("missing URL scheme or host")
	}
	parsed.Path = "/openapi/generation/config"
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed, nil
}

func (s *AccountTestService) fetchUpstreamCatalogBody(req *http.Request, account *Account, label string) ([]byte, error) {
	resp, err := s.doUpstreamModelsRequest(req, upstreamModelsProxyURL(account), account)
	if err != nil {
		return nil, newUpstreamModelSyncUpstreamError("Failed to request "+label, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, upstreamModelsBodyLimit+1))
	if err != nil {
		return nil, newUpstreamModelSyncUpstreamError("Failed to read "+label, err)
	}
	if int64(len(body)) > upstreamModelsBodyLimit {
		return nil, newUpstreamModelSyncUpstreamError(label+" response is too large", fmt.Errorf("response exceeds %d bytes", upstreamModelsBodyLimit))
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, newUpstreamModelSyncUpstreamError(
			fmt.Sprintf("%s request failed with HTTP %d", label, resp.StatusCode),
			fmt.Errorf("%s returned HTTP %d", label, resp.StatusCode),
		)
	}
	return body, nil
}

func extractXiaoVideoModelSpecs(body []byte) ([]UpstreamModelSpec, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var root any
	if err := decoder.Decode(&root); err != nil {
		return nil, fmt.Errorf("parse upstream model pricing: %w", err)
	}
	specs := make([]UpstreamModelSpec, 0)
	walkXiaoVideoModelSpecs(root, "", "", &specs)
	return dedupeAndSortModelSpecs(specs), nil
}

type aiStartLabSpecCandidate struct {
	spec         UpstreamModelSpec
	defaultGroup bool
}

// extractAIStartLabVideoModelSpecs understands AIStartLab's generation-config
// shape. The generic catalog walker cannot associate a model's nested pricing
// object with its quality and duration, and it also includes imageConfig.
func extractAIStartLabVideoModelSpecs(body []byte) ([]UpstreamModelSpec, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var root any
	if err := decoder.Decode(&root); err != nil {
		return nil, fmt.Errorf("parse AIStartLab generation config: %w", err)
	}

	videoConfig, found := aiStartLabVideoConfig(root)
	if !found {
		// Keep compatibility with older deployments that expose a generic
		// priced model list from the same endpoint.
		return extractXiaoVideoModelSpecs(body)
	}

	candidates := make([]aiStartLabSpecCandidate, 0)
	modelsWithDefaultGroup := make(map[string]bool)
	for _, rawGroup := range videoConfig {
		group, ok := rawGroup.(map[string]any)
		if !ok {
			continue
		}
		defaultGroup, _ := group["defaultOption"].(bool)
		rawModels, _ := group["models"].([]any)
		for _, rawModel := range rawModels {
			model, ok := rawModel.(map[string]any)
			if !ok {
				continue
			}
			modelID := firstCatalogString(model, "model", "model_id", "modelId", "id")
			if modelID == "" {
				continue
			}
			upstreamID := aiStartLabUpstreamModelID(firstCatalogString(group, "channel", "channel_code", "channelCode"), modelID)
			if defaultGroup {
				modelsWithDefaultGroup[modelID] = true
			}
			duration := aiStartLabDefaultDuration(model)
			capability := aiStartLabModelCapability(model)
			rawQualities, _ := model["qualities"].([]any)
			for _, rawQuality := range rawQualities {
				quality, ok := rawQuality.(map[string]any)
				if !ok {
					continue
				}
				resolution := firstCatalogString(quality, "quality", "resolution", "size")
				pricing, _ := quality["pricing"].(map[string]any)
				if pricing == nil {
					continue
				}
				cost, _ := catalogCost(pricing)
				if cost == nil {
					continue
				}
				unit := normalizeCatalogCostUnit(firstCatalogString(pricing, "type", "billing_unit", "billingUnit", "unit"))
				currency := normalizeCatalogCurrency(firstCatalogString(pricing, "currency", "currency_code", "currencyCode"))
				if currency == "" {
					currency = "CREDITS"
				}
				candidates = append(candidates, aiStartLabSpecCandidate{
					spec: UpstreamModelSpec{
						ID:                 upstreamID,
						Resolution:         resolution,
						UpstreamCost:       cost,
						CostCurrency:       currency,
						CostUnit:           unit,
						DefaultDuration:    duration,
						Durations:          capability.durations,
						AspectRatios:       capability.aspectRatios,
						DefaultAspectRatio: capability.defaultAspectRatio,
						SupportsAudio:      capability.supportsAudio,
						SupportsGuidances:  capability.supportsGuidances,
						SupportsStartFrame: capability.supportsStartFrame,
						RequiresStartFrame: capability.requiresStartFrame,
						SupportsEndFrame:   capability.supportsEndFrame,
						MaxReferences:      capability.maxReferences,
					},
					defaultGroup: defaultGroup,
				})
			}
		}
	}

	byKey := make(map[string]UpstreamModelSpec, len(candidates))
	for _, candidate := range candidates {
		// Bare IDs from legacy config need the default channel to avoid duplicate
		// rows. Current AIStartLab IDs include the channel and are distinct models,
		// so keep every channel-specific option returned by the upstream.
		if !strings.Contains(candidate.spec.ID, ":") && modelsWithDefaultGroup[candidate.spec.ID] && !candidate.defaultGroup {
			continue
		}
		key := candidate.spec.ID + "\x00" + candidate.spec.Resolution
		if _, exists := byKey[key]; !exists {
			byKey[key] = candidate.spec
		}
	}

	specs := make([]UpstreamModelSpec, 0, len(byKey))
	for _, spec := range byKey {
		specs = append(specs, spec)
	}
	specs = dedupeAndSortModelSpecs(specs)
	markDefaultCatalogResolutions(specs)
	return specs, nil
}

func aiStartLabUpstreamModelID(channel, model string) string {
	model = strings.TrimSpace(model)
	channel = strings.TrimSpace(channel)
	if model == "" || strings.Contains(model, ":") || channel == "" {
		return model
	}
	return channel + ":" + model
}

type aiStartLabCapability struct {
	durations          []int
	aspectRatios       []string
	defaultAspectRatio string
	supportsAudio      bool
	supportsGuidances  bool
	supportsStartFrame bool
	requiresStartFrame bool
	supportsEndFrame   bool
	maxReferences      map[string]int
}

func aiStartLabModelCapability(model map[string]any) aiStartLabCapability {
	capability := aiStartLabCapability{}
	for _, key := range []string{"aspectRatios", "aspect_ratios", "aspectRatio"} {
		if values, ok := model[key].([]any); ok {
			for _, value := range values {
				if ratio := catalogString(value); ratio != "" && !aiStartContainsString(capability.aspectRatios, ratio) {
					capability.aspectRatios = append(capability.aspectRatios, ratio)
				}
			}
		}
	}
	duration, _ := model["duration"].(map[string]any)
	if duration != nil {
		if values, ok := duration["options"].([]any); ok {
			for _, value := range values {
				if seconds, ok := catalogFloat(value); ok && seconds > 0 && seconds <= 3600 && !aiStartContainsInt(capability.durations, int(seconds)) {
					capability.durations = append(capability.durations, int(seconds))
				}
			}
		}
		if len(capability.durations) == 0 {
			min, _ := catalogFloat(duration["min"])
			max, _ := catalogFloat(duration["max"])
			if min > 0 && max >= min && max-min <= 120 {
				for seconds := int(min); seconds <= int(max); seconds++ {
					capability.durations = append(capability.durations, seconds)
				}
			}
		}
	}
	if len(capability.aspectRatios) > 0 {
		capability.defaultAspectRatio = capability.aspectRatios[0]
	}
	modes, _ := model["modes"].([]any)
	for _, value := range modes {
		switch strings.ToLower(catalogString(value)) {
		case "first-frame-to-video", "first_frame_to_video", "frames2video":
			capability.supportsStartFrame = true
			if strings.Contains(strings.ToLower(catalogString(value)), "frames2video") {
				capability.supportsEndFrame = true
			}
		case "last-frame-to-video", "last_frame_to_video":
			capability.supportsEndFrame = true
		}
	}
	imageMax := firstCatalogInt(model, "inputImagesMax", "input_images_max")
	videoMax := firstCatalogInt(model, "inputVideosMax", "input_videos_max")
	audioMax := firstCatalogInt(model, "inputAudiosMax", "input_audios_max")
	capability.maxReferences = map[string]int{"image": imageMax, "video": videoMax, "audio": audioMax}
	capability.supportsGuidances = imageMax > 0 || videoMax > 0 || audioMax > 0
	return capability
}

func aiStartContainsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func aiStartContainsInt(values []int, needle int) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func aiStartLabVideoConfig(root any) ([]any, bool) {
	rootMap, ok := root.(map[string]any)
	if !ok {
		return nil, false
	}
	data, ok := rootMap["data"].(map[string]any)
	if !ok {
		return nil, false
	}
	for _, key := range []string{"videoConfig", "video_config"} {
		value, exists := data[key]
		if !exists {
			continue
		}
		groups, ok := value.([]any)
		return groups, ok
	}
	return nil, false
}

func aiStartLabDefaultDuration(model map[string]any) int {
	if duration := firstCatalogInt(model, "default_duration", "defaultDuration", "default_seconds", "defaultSeconds"); duration > 0 {
		return duration
	}
	if duration, ok := catalogFloat(model["duration"]); ok && duration > 0 && duration <= 3600 {
		return int(duration)
	}
	duration, _ := model["duration"].(map[string]any)
	if duration == nil {
		return 0
	}
	if value := firstCatalogInt(duration, "default", "default_duration", "defaultDuration", "default_seconds", "defaultSeconds"); value > 0 {
		return value
	}
	if options, ok := duration["options"].([]any); ok {
		for _, option := range options {
			if value, ok := catalogFloat(option); ok && value > 0 && value <= 3600 {
				return int(value)
			}
		}
	}
	return firstCatalogInt(duration, "min", "minimum")
}

func markDefaultCatalogResolutions(specs []UpstreamModelSpec) {
	byModel := make(map[string][]int)
	for index := range specs {
		byModel[specs[index].ID] = append(byModel[specs[index].ID], index)
	}
	for _, indexes := range byModel {
		defaultIndex := indexes[0]
		for _, index := range indexes {
			if strings.EqualFold(specs[index].Resolution, "720p") {
				defaultIndex = index
				break
			}
		}
		specs[defaultIndex].DefaultResolution = true
	}
}

func walkXiaoVideoModelSpecs(value any, inheritedModel, contextKey string, specs *[]UpstreamModelSpec) {
	switch node := value.(type) {
	case []any:
		for _, item := range node {
			walkXiaoVideoModelSpecs(item, inheritedModel, contextKey, specs)
		}
	case map[string]any:
		modelID := modelIDFromCatalogMap(node, contextKey)
		if modelID == "" {
			modelID = inheritedModel
		}
		if modelID != "" && catalogMapHasModelMetadata(node, contextKey) {
			appendSpecsFromCatalogMap(node, modelID, contextKey, specs)
		}
		for key, child := range node {
			if childMap, ok := child.(map[string]any); ok && isScalarMetadataMap(childMap) {
				continue
			}
			childModel := modelID
			if childModel == "" && strings.Contains(strings.ToLower(contextKey), "model") && !isCatalogContainerKey(key) {
				childModel = strings.TrimSpace(key)
			}
			walkXiaoVideoModelSpecs(child, childModel, key, specs)
		}
	}
}

func modelIDFromCatalogMap(node map[string]any, contextKey string) string {
	for _, key := range []string{"model", "model_id", "modelId", "model_code", "modelCode"} {
		if value := catalogString(node[key]); value != "" {
			return strings.TrimPrefix(value, "models/")
		}
	}
	context := strings.ToLower(contextKey)
	if strings.Contains(context, "model") || catalogMapHasPriceOrCapability(node) {
		for _, key := range []string{"id", "name"} {
			if value := catalogString(node[key]); value != "" {
				return strings.TrimPrefix(value, "models/")
			}
		}
	}
	return ""
}

func catalogMapHasModelMetadata(node map[string]any, contextKey string) bool {
	return strings.Contains(strings.ToLower(contextKey), "model") || catalogMapHasPriceOrCapability(node)
}

func catalogMapHasPriceOrCapability(node map[string]any) bool {
	for _, key := range []string{
		"price_per_second", "pricePerSecond", "cost_per_second", "costPerSecond",
		"unit_price", "unitPrice", "price", "cost", "credit", "credits", "point", "points",
		"resolution", "resolutions", "quality", "size",
		"default_duration", "defaultDuration", "duration", "seconds",
	} {
		if _, ok := node[key]; ok {
			return true
		}
	}
	return false
}

func appendSpecsFromCatalogMap(node map[string]any, modelID, contextKey string, specs *[]UpstreamModelSpec) {
	resolutions := catalogResolutions(node)
	if len(resolutions) == 0 && looksLikeResolution(contextKey) {
		resolutions = []string{strings.TrimSpace(contextKey)}
	}
	if len(resolutions) == 0 {
		resolutions = []string{""}
	}
	cost, costKey := catalogCost(node)
	currency := normalizeCatalogCurrency(firstCatalogString(node, "currency", "currency_code", "currencyCode"))
	unit := normalizeCatalogCostUnit(firstCatalogString(node, "price_unit", "priceUnit", "billing_unit", "billingUnit", "cost_unit", "costUnit", "unit"))
	if strings.Contains(strings.ToLower(costKey), "per_second") || strings.Contains(strings.ToLower(costKey), "persecond") {
		unit = "second"
	}
	if (costKey == "credit" || costKey == "credits" || costKey == "point" || costKey == "points") && currency == "" {
		currency = "CREDITS"
	}
	duration := firstCatalogInt(node, "default_duration", "defaultDuration", "duration", "seconds", "default_seconds", "defaultSeconds")
	defaultResolution := firstCatalogString(node, "default_resolution", "defaultResolution")
	defaultFlag, _ := node["is_default"].(bool)
	if value, ok := node["default"].(bool); ok {
		defaultFlag = defaultFlag || value
	}
	for index, resolution := range resolutions {
		*specs = append(*specs, UpstreamModelSpec{
			ID:                strings.TrimSpace(modelID),
			Resolution:        resolution,
			UpstreamCost:      cost,
			CostCurrency:      currency,
			CostUnit:          unit,
			DefaultDuration:   duration,
			DefaultResolution: defaultFlag || (defaultResolution != "" && strings.EqualFold(defaultResolution, resolution)) || (len(resolutions) == 1 && index == 0),
		})
	}
}

func applyCatalogCostDefaults(specs []UpstreamModelSpec, currency string, force, inferRequestUnit bool) {
	for index := range specs {
		if specs[index].UpstreamCost != nil && (force || strings.TrimSpace(specs[index].CostCurrency) == "") {
			specs[index].CostCurrency = currency
		}
		// AIStartLab config entries describe a priced generation option. When
		// they include a concrete duration but omit the billing-unit label, the
		// generic price is the cost of that option rather than a per-second rate.
		if inferRequestUnit && specs[index].UpstreamCost != nil && specs[index].CostUnit == "" && specs[index].DefaultDuration > 0 {
			specs[index].CostUnit = "request"
		}
	}
}

func isCatalogContainerKey(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "data", "items", "list", "models", "model_list", "modelconfig", "model_config", "configs", "options":
		return true
	default:
		return false
	}
}

func looksLikeResolution(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if strings.HasSuffix(normalized, "p") {
		if _, err := strconv.Atoi(strings.TrimSuffix(normalized, "p")); err == nil {
			return true
		}
	}
	parts := strings.Split(normalized, "x")
	if len(parts) != 2 {
		return false
	}
	_, leftErr := strconv.Atoi(parts[0])
	_, rightErr := strconv.Atoi(parts[1])
	return leftErr == nil && rightErr == nil
}

func catalogResolutions(node map[string]any) []string {
	values := make([]string, 0)
	for _, key := range []string{"resolution", "quality", "size"} {
		if value := catalogString(node[key]); value != "" {
			values = append(values, value)
		}
	}
	if list, ok := node["resolutions"].([]any); ok {
		for _, item := range list {
			if value := catalogString(item); value != "" {
				values = append(values, value)
			}
		}
	}
	return dedupeAndSortModelIDs(values)
}

func catalogCost(node map[string]any) (*float64, string) {
	for _, key := range []string{"price_per_second", "pricePerSecond", "cost_per_second", "costPerSecond", "unit_price", "unitPrice", "price", "cost", "credit", "credits", "point", "points"} {
		if value, ok := catalogFloat(node[key]); ok && value >= 0 {
			return &value, key
		}
	}
	return nil, ""
}

func catalogFloat(value any) (float64, bool) {
	switch typed := value.(type) {
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	case float64:
		return typed, true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func catalogString(value any) string {
	if typed, ok := value.(string); ok {
		return strings.TrimSpace(typed)
	}
	return ""
}

func firstCatalogString(node map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := catalogString(node[key]); value != "" {
			return value
		}
	}
	return ""
}

func firstCatalogInt(node map[string]any, keys ...string) int {
	for _, key := range keys {
		if value, ok := catalogFloat(node[key]); ok && value > 0 && value <= 3600 {
			return int(value)
		}
	}
	return 0
}

func normalizeCatalogCurrency(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	switch value {
	case "$", "US DOLLAR", "US DOLLARS":
		return "USD"
	case "POINT", "POINTS", "CREDIT":
		return "CREDITS"
	default:
		return value
	}
}

func normalizeCatalogCostUnit(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.Trim(normalized, "/ ")
	switch normalized {
	case "s", "sec", "secs", "second", "seconds", "per_second", "per-second":
		return "second"
	case "request", "requests", "generation", "generations", "video", "videos", "per_request", "per-request", "fixed", "fixed_total", "fixed-total", "per_generation", "per-generation":
		return "request"
	default:
		return normalized
	}
}

func isScalarMetadataMap(node map[string]any) bool {
	if len(node) == 0 {
		return true
	}
	for _, value := range node {
		switch value.(type) {
		case map[string]any, []any:
			return false
		}
	}
	return !catalogMapHasPriceOrCapability(node)
}

func dedupeAndSortModelSpecs(specs []UpstreamModelSpec) []UpstreamModelSpec {
	byKey := make(map[string]UpstreamModelSpec, len(specs))
	for _, spec := range specs {
		spec.ID = strings.TrimSpace(spec.ID)
		spec.Resolution = strings.TrimSpace(spec.Resolution)
		if spec.ID == "" {
			continue
		}
		key := spec.ID + "\x00" + spec.Resolution
		current, exists := byKey[key]
		if !exists || modelSpecScore(spec) > modelSpecScore(current) {
			byKey[key] = spec
		}
	}
	result := make([]UpstreamModelSpec, 0, len(byKey))
	detailedModels := make(map[string]bool, len(byKey))
	for _, spec := range byKey {
		if modelSpecScore(spec) > 0 {
			detailedModels[spec.ID] = true
		}
	}
	for _, spec := range byKey {
		if detailedModels[spec.ID] && modelSpecScore(spec) == 0 {
			continue
		}
		result = append(result, spec)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].ID != result[j].ID {
			return result[i].ID < result[j].ID
		}
		return result[i].Resolution < result[j].Resolution
	})
	return result
}

func modelSpecScore(spec UpstreamModelSpec) int {
	score := 0
	if spec.Resolution != "" {
		score++
	}
	if spec.UpstreamCost != nil {
		score += 2
	}
	if spec.CostCurrency != "" {
		score++
	}
	if spec.CostUnit != "" {
		score++
	}
	if spec.DefaultDuration > 0 {
		score++
	}
	return score
}

func modelIDsFromSpecs(specs []UpstreamModelSpec) []string {
	models := make([]string, 0, len(specs))
	for _, spec := range specs {
		models = append(models, spec.ID)
	}
	return dedupeAndSortModelIDs(models)
}

func filterSpecsByModelIDs(specs []UpstreamModelSpec, models []string) []UpstreamModelSpec {
	allowed := make(map[string]struct{}, len(models))
	for _, model := range models {
		allowed[model] = struct{}{}
	}
	result := make([]UpstreamModelSpec, 0, len(specs))
	for _, spec := range specs {
		if _, ok := allowed[spec.ID]; ok {
			result = append(result, spec)
		}
	}
	return result
}
