package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type videoProviderOperation string

const (
	videoOperationModels        videoProviderOperation = "models"
	videoOperationCapabilities  videoProviderOperation = "capabilities"
	videoOperationUpload        videoProviderOperation = "upload"
	videoOperationUploadContent videoProviderOperation = "upload_content"
	videoOperationCreate        videoProviderOperation = "create"
	videoOperationStatus        videoProviderOperation = "status"
	videoOperationCancel        videoProviderOperation = "cancel"
	videoOperationContent       videoProviderOperation = "content"
)

type videoProviderEndpoint struct {
	Method             string `json:"method"`
	Path               string `json:"path"`
	PreferRespondAsync bool   `json:"prefer_respond_async,omitempty"`
}

type VideoAdapterAuthConfig struct {
	Type   string `json:"type"`
	Header string `json:"header,omitempty"`
	Prefix string `json:"prefix,omitempty"`
}

type VideoAdapterEndpointsConfig struct {
	Models        videoProviderEndpoint `json:"models"`
	Capabilities  videoProviderEndpoint `json:"capabilities,omitempty"`
	Upload        videoProviderEndpoint `json:"upload,omitempty"`
	UploadContent videoProviderEndpoint `json:"upload_content,omitempty"`
	Create        videoProviderEndpoint `json:"create"`
	Status        videoProviderEndpoint `json:"status"`
	Cancel        videoProviderEndpoint `json:"cancel,omitempty"`
	Content       videoProviderEndpoint `json:"content,omitempty"`
}

type VideoAdapterRequestConfig struct {
	PassThrough  bool              `json:"pass_through,omitempty"`
	Fields       map[string]string `json:"fields"`
	StringFields []string          `json:"string_fields,omitempty"`
	Static       map[string]any    `json:"static,omitempty"`
}

type VideoAdapterResponseConfig struct {
	DataPath  string            `json:"data_path,omitempty"`
	Fields    map[string]string `json:"fields"`
	Defaults  map[string]any    `json:"defaults,omitempty"`
	StatusMap map[string]string `json:"status_map,omitempty"`
	ResultURL string            `json:"result_url,omitempty"`
}

type VideoAdapterCollectionConfig struct {
	ItemsPath         string `json:"items_path,omitempty"`
	IDPath            string `json:"id_path,omitempty"`
	SchemaVersionPath string `json:"schema_version_path,omitempty"`
}

// VideoJSONAdapterConfig describes ordinary JSON/REST video providers without
// allowing executable templates. Providers with multipart creation, SSE, or
// webhook-only workflows should implement videoProviderAdapter directly.
type VideoJSONAdapterConfig struct {
	Version           int                          `json:"version"`
	Auth              VideoAdapterAuthConfig       `json:"auth"`
	StaticHeaders     map[string]string            `json:"static_headers,omitempty"`
	IdempotencyHeader string                       `json:"idempotency_header,omitempty"`
	Endpoints         VideoAdapterEndpointsConfig  `json:"endpoints"`
	Request           VideoAdapterRequestConfig    `json:"request"`
	Response          VideoAdapterResponseConfig   `json:"response"`
	UploadResponse    VideoAdapterResponseConfig   `json:"upload_response,omitempty"`
	Models            VideoAdapterCollectionConfig `json:"models,omitempty"`
	Capabilities      VideoAdapterCollectionConfig `json:"capabilities,omitempty"`
}

type videoProviderAdapter interface {
	Protocol() string
	Endpoint(videoProviderOperation, string) (videoProviderEndpoint, bool)
	Authorize(*http.Request, string)
	StaticHeaders() map[string]string
	IdempotencyHeader() string
	RewriteCreate(*Account, []byte, string, string, int) ([]byte, error)
	DecodeJob([]byte) (map[string]any, error)
	DecodeModels([]byte) ([]map[string]any, error)
	DecodeCapabilities([]byte) (int, []map[string]any, error)
	DecodeUpload([]byte) (map[string]any, error)
	ResultURL([]byte) (string, bool)
}

type nativeVideoAdapter struct{}

func (nativeVideoAdapter) Protocol() string { return XiaoVideoProtocolNative }
func (nativeVideoAdapter) Endpoint(operation videoProviderOperation, id string) (videoProviderEndpoint, bool) {
	switch operation {
	case videoOperationModels:
		return videoProviderEndpoint{Method: http.MethodGet, Path: "/v1/models"}, true
	case videoOperationCapabilities:
		return videoProviderEndpoint{Method: http.MethodGet, Path: "/v1/videos/capabilities"}, true
	case videoOperationUpload:
		return videoProviderEndpoint{Method: http.MethodPost, Path: "/v1/videos/uploads"}, true
	case videoOperationUploadContent:
		return videoProviderEndpoint{Method: http.MethodGet, Path: "/v1/videos/uploads/" + url.PathEscape(id) + "/content"}, true
	case videoOperationCreate:
		return videoProviderEndpoint{Method: http.MethodPost, Path: "/v1/videos/generations", PreferRespondAsync: true}, true
	case videoOperationStatus:
		return videoProviderEndpoint{Method: http.MethodGet, Path: "/v1/videos/jobs/" + url.PathEscape(id)}, true
	case videoOperationCancel:
		return videoProviderEndpoint{Method: http.MethodDelete, Path: "/v1/videos/jobs/" + url.PathEscape(id)}, true
	case videoOperationContent:
		return videoProviderEndpoint{Method: http.MethodGet, Path: "/v1/videos/jobs/" + url.PathEscape(id) + "/content"}, true
	default:
		return videoProviderEndpoint{}, false
	}
}
func (nativeVideoAdapter) Authorize(req *http.Request, apiKey string) {
	req.Header.Set("Authorization", "Bearer "+apiKey)
}
func (nativeVideoAdapter) StaticHeaders() map[string]string { return nil }
func (nativeVideoAdapter) IdempotencyHeader() string        { return "Idempotency-Key" }
func (nativeVideoAdapter) RewriteCreate(_ *Account, body []byte, model, resolution string, duration int) ([]byte, error) {
	return rewriteVideoRequest(body, model, resolution, duration)
}
func (nativeVideoAdapter) DecodeJob(raw []byte) (map[string]any, error) {
	return decodeVideoJSONObject(raw)
}
func (nativeVideoAdapter) DecodeModels(raw []byte) ([]map[string]any, error) {
	return decodeVideoCollection(raw, "data", "id")
}
func (nativeVideoAdapter) DecodeCapabilities(raw []byte) (int, []map[string]any, error) {
	return decodeVideoCapabilityCollection(raw, "data", "id", "schema_version")
}
func (nativeVideoAdapter) DecodeUpload(raw []byte) (map[string]any, error) {
	return decodeVideoJSONObject(raw)
}
func (nativeVideoAdapter) ResultURL([]byte) (string, bool) { return "", false }

type openAISoraVideoAdapter struct{ nativeVideoAdapter }

func (openAISoraVideoAdapter) Protocol() string { return XiaoVideoProtocolOpenAISora }
func (openAISoraVideoAdapter) Endpoint(operation videoProviderOperation, id string) (videoProviderEndpoint, bool) {
	switch operation {
	case videoOperationModels:
		return videoProviderEndpoint{Method: http.MethodGet, Path: "/v1/models"}, true
	case videoOperationCapabilities:
		return videoProviderEndpoint{Method: http.MethodGet, Path: "/v1/videos/capabilities"}, true
	case videoOperationCreate:
		return videoProviderEndpoint{Method: http.MethodPost, Path: "/v1/videos"}, true
	case videoOperationStatus:
		return videoProviderEndpoint{Method: http.MethodGet, Path: "/v1/videos/" + url.PathEscape(id)}, true
	default:
		return videoProviderEndpoint{}, false
	}
}
func (openAISoraVideoAdapter) RewriteCreate(account *Account, body []byte, model, resolution string, duration int) ([]byte, error) {
	return rewriteOpenAISoraVideoRequest(account, body, model, resolution, duration)
}
func (openAISoraVideoAdapter) DecodeJob(raw []byte) (map[string]any, error) {
	upstream, err := decodeVideoJSONObject(raw)
	if err != nil {
		return nil, err
	}
	upstream["job_id"] = videoStringValue(upstream["id"])
	switch videoStringValue(upstream["status"]) {
	case "queued":
		upstream["status"] = "pending"
	case "in_progress":
		upstream["status"] = "running"
	case "completed", "failed":
	default:
		return nil, errors.New("invalid OpenAI/Sora video status")
	}
	upstream["duration"] = videoStringValue(upstream["seconds"])
	upstream["aspect_ratio"] = videoStringValue(upstream["size"])
	if metadata, ok := upstream["metadata"].(map[string]any); ok {
		upstream["resolution"] = videoStringValue(metadata["resolution"])
	}
	upstream["amount"] = "0"
	upstream["currency"] = "CREDITS"
	return upstream, nil
}
func (openAISoraVideoAdapter) ResultURL(raw []byte) (string, bool) {
	return safeVideoResultURL(jsonPathValueFromBytes(raw, "metadata.result_url"))
}

// ctmoaiVideoAdapter implements CTMOAI's Seedance and MiniMax H3 gateway.
// Both model families share the asynchronous /v1/videos lifecycle, but H3
// uses different reference and first/last-frame fields, so it cannot be
// represented safely by the generic custom_json mapper.
type ctmoaiVideoAdapter struct{}

func (ctmoaiVideoAdapter) Protocol() string { return XiaoVideoProtocolCTMOAI }
func (ctmoaiVideoAdapter) Endpoint(operation videoProviderOperation, id string) (videoProviderEndpoint, bool) {
	switch operation {
	case videoOperationModels:
		return videoProviderEndpoint{Method: http.MethodGet, Path: "/v1/models"}, true
	case videoOperationUpload:
		return videoProviderEndpoint{Method: http.MethodPost, Path: "/api/sd-media/upload"}, true
	case videoOperationCreate:
		return videoProviderEndpoint{Method: http.MethodPost, Path: "/v1/videos"}, true
	case videoOperationStatus:
		return videoProviderEndpoint{Method: http.MethodGet, Path: "/v1/videos/" + url.PathEscape(id)}, true
	case videoOperationContent:
		return videoProviderEndpoint{Method: http.MethodGet, Path: "/v1/videos/" + url.PathEscape(id) + "/content"}, true
	default:
		return videoProviderEndpoint{}, false
	}
}
func (ctmoaiVideoAdapter) Authorize(req *http.Request, apiKey string) {
	req.Header.Set("Authorization", "Bearer "+apiKey)
}
func (ctmoaiVideoAdapter) StaticHeaders() map[string]string { return nil }
func (ctmoaiVideoAdapter) IdempotencyHeader() string        { return "Idempotency-Key" }
func (ctmoaiVideoAdapter) RewriteCreate(account *Account, raw []byte, model, _ string, duration int) ([]byte, error) {
	var canonical map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&canonical); err != nil {
		return nil, ErrVideoRequestInvalid
	}
	model = strings.TrimSpace(model)
	prompt := videoStringValue(canonical["prompt"])
	if model == "" || prompt == "" || duration <= 0 {
		return nil, ErrVideoRequestInvalid
	}
	if err := validateCTMOAIRequestCapabilities(account, model, canonical, duration); err != nil {
		return nil, err
	}

	payload := map[string]any{
		"model":   model,
		"prompt":  prompt,
		"seconds": duration,
	}
	if aspectRatio := videoStringValue(canonical["aspect_ratio"]); aspectRatio != "" {
		payload["aspect_ratio"] = aspectRatio
	}

	images := make([]string, 0)
	if imageURL := videoStringValue(canonical["image_url"]); imageURL != "" {
		images = append(images, imageURL)
	}
	startFrame := videoStringValue(canonical["start_frame_url"])
	endFrame := videoStringValue(canonical["end_frame_url"])
	if endFrame != "" && startFrame == "" {
		return nil, ErrVideoRequestInvalid
	}
	if startFrame != "" {
		images = append(images, startFrame)
		if endFrame != "" {
			images = append(images, endFrame)
			payload["workflow_id"] = "fl2v"
		}
	} else if endFrame != "" {
		return nil, ErrVideoRequestInvalid
	}

	videos, audios := make([]string, 0), make([]string, 0)
	if guidances, ok := canonical["guidances"].(map[string]any); ok {
		images = append(images, videoGuidanceURLs(guidances, "image_reference", "image")...)
		videos = append(videos, videoGuidanceURLs(guidances, "video_reference_base", "video")...)
		audios = append(audios, videoGuidanceURLs(guidances, "audio_reference", "audio")...)
	}

	if isCTMOAICFModel(model) {
		if len(images) == 0 {
			return nil, ErrVideoOptionUnsupported
		}
		// CTMOAI accepts input_reference for one CF image and images[] for
		// multi-reference/first-last-frame requests.
		if len(images) == 1 && payload["workflow_id"] == nil {
			payload["input_reference"] = images[0]
		} else {
			payload["images"] = images
		}
	} else if len(images) > 0 {
		payload["images"] = images
	}
	if len(videos) > 0 {
		payload["reference_videos"] = videos
	}
	if len(audios) > 0 {
		payload["reference_audios"] = audios
	}
	return json.Marshal(payload)
}
func (ctmoaiVideoAdapter) DecodeJob(raw []byte) (map[string]any, error) {
	upstream, err := decodeVideoJSONObject(raw)
	if err != nil {
		return nil, err
	}
	jobID := videoStringValue(upstream["task_id"])
	if jobID == "" {
		jobID = videoStringValue(upstream["id"])
	}
	if jobID == "" {
		return nil, errors.New("CTMOAI response has no task id")
	}
	upstream["job_id"] = jobID
	status := strings.ToLower(strings.TrimSpace(videoStringValue(upstream["status"])))
	switch status {
	case "queued", "unknown", "pending":
		upstream["status"] = "pending"
	case "in_progress", "running", "processing":
		upstream["status"] = "running"
	case "completed", "succeeded", "success":
		upstream["status"] = "completed"
	case "failed", "error", "timeout", "timed_out", "expired":
		upstream["status"] = "failed"
	case "cancelled", "canceled":
		upstream["status"] = "canceled"
	default:
		return nil, fmt.Errorf("invalid CTMOAI video status %q", status)
	}
	if duration := videoStringValue(upstream["seconds"]); duration != "" {
		upstream["duration"] = duration
	}
	if ratio := videoStringValue(upstream["aspect_ratio"]); ratio != "" {
		upstream["aspect_ratio"] = ratio
	}
	upstream["amount"] = "0"
	upstream["currency"] = "CREDITS"
	return upstream, nil
}
func (ctmoaiVideoAdapter) DecodeModels(raw []byte) ([]map[string]any, error) {
	var root map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&root); err != nil {
		return nil, err
	}
	rawItems, ok := root["data"].([]any)
	if !ok {
		return nil, errors.New("CTMOAI model list data is not an array")
	}
	items := make([]map[string]any, 0, len(rawItems))
	for _, rawItem := range rawItems {
		item, ok := rawItem.(map[string]any)
		if !ok {
			continue
		}
		id := firstVideoString(item, "id", "model", "model_id", "name")
		if id == "" {
			continue
		}
		copy := make(map[string]any, len(item)+1)
		for key, value := range item {
			copy[key] = value
		}
		copy["id"] = id
		ctmoaiNormalizeModel(copy)
		items = append(items, copy)
	}
	return items, nil
}
func (ctmoaiVideoAdapter) DecodeCapabilities(raw []byte) (int, []map[string]any, error) {
	return 0, nil, errors.New("CTMOAI does not expose a separate capabilities endpoint")
}
func (ctmoaiVideoAdapter) DecodeUpload(raw []byte) (map[string]any, error) {
	result, err := decodeVideoJSONObject(raw)
	if err != nil {
		return nil, err
	}
	value := videoStringValue(result["url"])
	if value == "" {
		return nil, errors.New("CTMOAI upload response has no url")
	}
	if _, ok := safeVideoResultURL(value); !ok {
		return nil, errors.New("CTMOAI upload response url is invalid")
	}
	if videoStringValue(result["media_id"]) == "" {
		parsed, _ := url.Parse(value)
		id := strings.Trim(strings.TrimSpace(parsed.Path), "/")
		if slash := strings.LastIndexByte(id, '/'); slash >= 0 {
			id = id[slash+1:]
		}
		if id == "" {
			id = value
		}
		result["media_id"] = id
	}
	result["type"] = defaultString(videoStringValue(result["type"]), "UPLOADED")
	return result, nil
}
func (ctmoaiVideoAdapter) ResultURL([]byte) (string, bool) { return "", false }

func isCTMOAICFModel(model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	return strings.Contains(model, "-cf-2k") || strings.Contains(model, "-cf-4k")
}

func firstVideoString(item map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := videoStringValue(item[key]); value != "" {
			return value
		}
	}
	return ""
}

func validateCTMOAIRequestCapabilities(account *Account, model string, request map[string]any, duration int) error {
	if account == nil {
		return nil
	}
	capabilities := videoCapabilitiesForAccount(account)
	// CTMOAI's quantized H3 model has a stricter live contract than older
	// administrator-saved capability records. Keep the server-side guard in
	// sync with the provider while the model catalogue is refreshed.
	capability := ctmoaiKnownCapability(model)
	if stored := capabilities[model]; capability == nil {
		capability = stored
	}
	if capability == nil {
		for id, candidate := range capabilities {
			if videoModelSuffix(id) == model {
				capability = candidate
				break
			}
		}
	}
	if capability == nil {
		return nil
	}
	if duration > 0 {
		if durations := videoIntSet(capability["durations"]); len(durations) > 0 {
			if _, ok := durations[duration]; !ok {
				return ErrVideoOptionUnsupported
			}
		}
	}
	if ratio := videoStringValue(request["aspect_ratio"]); ratio != "" {
		if ratios := videoStringSet(capability["aspect_ratios"]); len(ratios) > 0 {
			if _, ok := ratios[ratio]; !ok {
				return ErrVideoOptionUnsupported
			}
		}
	}
	if start := videoStringValue(request["start_frame_url"]); start != "" {
		if supported, ok := capability["supports_start_frame"].(bool); ok && !supported {
			return ErrVideoOptionUnsupported
		}
	}
	if end := videoStringValue(request["end_frame_url"]); end != "" {
		if supported, ok := capability["supports_end_frame"].(bool); ok && !supported {
			return ErrVideoOptionUnsupported
		}
	}
	// Frame URLs are image references too, but they are represented outside
	// guidances by the public workbench contract. Count them against the same
	// provider-declared image limit so a caller cannot bypass the limit by
	// switching between reference forms.
	imageCount := 0
	if videoStringValue(request["image_url"]) != "" {
		imageCount++
	}
	if videoStringValue(request["start_frame_url"]) != "" {
		imageCount++
	}
	if videoStringValue(request["end_frame_url"]) != "" {
		imageCount++
	}
	guidances, _ := request["guidances"].(map[string]any)
	limits := capabilityReferenceLimits(capability["max_references"])
	guidanceSupported, guidanceSupportedDeclared := capability["supports_guidances"].(bool)
	for listKey, mediaKey := range map[string]string{"image_reference": "image", "video_reference_base": "video", "audio_reference": "audio"} {
		if guidances == nil {
			break
		}
		items, _ := guidances[listKey].([]any)
		if len(items) == 0 {
			continue
		}
		if guidanceSupportedDeclared && !guidanceSupported {
			return ErrVideoOptionUnsupported
		}
		limit, hasLimit := videoIntValue(limits[mediaKey])
		if hasLimit && (limit <= 0 || len(items) > limit) {
			return ErrVideoOptionUnsupported
		}
		if mediaKey == "image" {
			imageCount += len(items)
		}
	}
	if imageCount > 0 {
		if limit, ok := videoIntValue(limits["image"]); ok && (limit <= 0 || imageCount > limit) {
			return ErrVideoOptionUnsupported
		}
		if guidanceSupportedDeclared && !guidanceSupported && videoStringValue(request["start_frame_url"]) == "" && videoStringValue(request["image_url"]) == "" {
			return ErrVideoOptionUnsupported
		}
	}
	return nil
}

func capabilityReferenceLimits(value any) map[string]any {
	result := make(map[string]any)
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			result[key] = item
		}
	case map[string]int:
		for key, item := range typed {
			result[key] = item
		}
	case map[string]float64:
		for key, item := range typed {
			result[key] = item
		}
	case map[string]json.Number:
		for key, item := range typed {
			result[key] = item
		}
	}
	return result
}

func ctmoaiKnownCapability(model string) map[string]any {
	normalized := strings.ToLower(strings.TrimSpace(model))
	if !strings.Contains(normalized, "minimax-h3-quantized-768p") {
		return nil
	}
	return map[string]any{
		"durations":            []any{json.Number("4"), json.Number("5"), json.Number("6"), json.Number("7"), json.Number("8"), json.Number("9"), json.Number("10")},
		"aspect_ratios":        []any{"16:9", "9:16", "1:1", "2:3", "3:2", "3:4", "4:3", "21:9"},
		"supports_start_frame": true,
		"supports_end_frame":   true,
		"supports_audio":       false,
		"max_references": map[string]any{
			"image": json.Number("4"),
			"video": json.Number("0"),
			"audio": json.Number("0"),
		},
		"supports_guidances": true,
	}
}

func videoIntSet(value any) map[int]struct{} {
	result := make(map[int]struct{})
	switch values := value.(type) {
	case []any:
		for _, item := range values {
			if parsed, ok := videoIntValue(item); ok && parsed > 0 {
				result[parsed] = struct{}{}
			}
		}
	case []int:
		for _, item := range values {
			if item > 0 {
				result[item] = struct{}{}
			}
		}
	}
	return result
}

func videoIntValue(value any) (int, bool) {
	switch typed := value.(type) {
	case json.Number:
		parsed, err := strconv.Atoi(typed.String())
		return parsed, err == nil
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), typed == float64(int(typed))
	default:
		parsed, err := strconv.Atoi(strings.TrimSpace(videoStringValue(value)))
		return parsed, err == nil
	}
}

func ctmoaiNormalizeModel(item map[string]any) {
	if item == nil {
		return
	}
	if _, ok := item["resolutions"]; !ok {
		if resolution := videoStringValue(item["resolution"]); resolution != "" {
			item["resolutions"] = []any{resolution}
		}
	}
	if _, ok := item["durations"]; !ok {
		if values, ok := item["durations_seconds"].([]any); ok {
			item["durations"] = values
		}
	}
	if _, ok := item["aspect_ratios"]; !ok {
		if values, ok := item["ratios"].([]any); ok {
			item["aspect_ratios"] = values
		}
	}
	maxRefs := map[string]any{}
	for key, target := range map[string]string{"max_images": "image", "max_videos": "video", "max_audios": "audio"} {
		if value := item[key]; value != nil {
			maxRefs[target] = value
		}
	}
	if len(maxRefs) > 0 {
		item["max_references"] = maxRefs
		item["supports_guidances"] = true
	}
}

type customJSONVideoAdapter struct{ config VideoJSONAdapterConfig }

func (a customJSONVideoAdapter) Protocol() string { return XiaoVideoProtocolCustomJSON }
func (a customJSONVideoAdapter) Endpoint(operation videoProviderOperation, id string) (videoProviderEndpoint, bool) {
	var endpoint videoProviderEndpoint
	switch operation {
	case videoOperationModels:
		endpoint = a.config.Endpoints.Models
	case videoOperationCapabilities:
		endpoint = a.config.Endpoints.Capabilities
	case videoOperationUpload:
		endpoint = a.config.Endpoints.Upload
	case videoOperationUploadContent:
		endpoint = a.config.Endpoints.UploadContent
	case videoOperationCreate:
		endpoint = a.config.Endpoints.Create
	case videoOperationStatus:
		endpoint = a.config.Endpoints.Status
	case videoOperationCancel:
		endpoint = a.config.Endpoints.Cancel
	case videoOperationContent:
		endpoint = a.config.Endpoints.Content
	}
	if strings.TrimSpace(endpoint.Path) == "" {
		return videoProviderEndpoint{}, false
	}
	endpoint.Method = strings.ToUpper(strings.TrimSpace(endpoint.Method))
	endpoint.Path = strings.ReplaceAll(endpoint.Path, "{job_id}", url.PathEscape(id))
	endpoint.Path = strings.ReplaceAll(endpoint.Path, "{media_id}", url.PathEscape(id))
	return endpoint, true
}
func (a customJSONVideoAdapter) Authorize(req *http.Request, apiKey string) {
	switch a.config.Auth.Type {
	case "none":
		return
	case "header":
		req.Header.Set(a.config.Auth.Header, a.config.Auth.Prefix+apiKey)
	default:
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
}
func (a customJSONVideoAdapter) StaticHeaders() map[string]string { return a.config.StaticHeaders }
func (a customJSONVideoAdapter) IdempotencyHeader() string {
	return defaultString(a.config.IdempotencyHeader, "Idempotency-Key")
}
func (a customJSONVideoAdapter) RewriteCreate(_ *Account, body []byte, model, resolution string, duration int) ([]byte, error) {
	var canonical map[string]any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&canonical); err != nil {
		return nil, ErrVideoRequestInvalid
	}
	canonical["model"], canonical["resolution"], canonical["duration"] = model, resolution, duration
	payload := make(map[string]any)
	if a.config.Request.PassThrough {
		for key, value := range canonical {
			payload[key] = value
		}
	}
	for key, value := range a.config.Request.Static {
		setJSONPath(payload, key, value)
	}
	stringFields := make(map[string]struct{}, len(a.config.Request.StringFields))
	for _, key := range a.config.Request.StringFields {
		stringFields[key] = struct{}{}
	}
	for source, target := range a.config.Request.Fields {
		value := jsonPathValue(canonical, source)
		if value == nil || videoStringValue(value) == "" {
			continue
		}
		if _, ok := stringFields[source]; ok {
			value = videoStringValue(value)
		}
		setJSONPath(payload, target, value)
	}
	return json.Marshal(payload)
}
func (a customJSONVideoAdapter) DecodeJob(raw []byte) (map[string]any, error) {
	return normalizeVideoResponse(raw, a.config.Response)
}
func (a customJSONVideoAdapter) DecodeModels(raw []byte) ([]map[string]any, error) {
	return decodeVideoCollection(raw, a.config.Models.ItemsPath, defaultString(a.config.Models.IDPath, "id"))
}
func (a customJSONVideoAdapter) DecodeCapabilities(raw []byte) (int, []map[string]any, error) {
	return decodeVideoCapabilityCollection(raw, a.config.Capabilities.ItemsPath, defaultString(a.config.Capabilities.IDPath, "id"), a.config.Capabilities.SchemaVersionPath)
}
func (a customJSONVideoAdapter) DecodeUpload(raw []byte) (map[string]any, error) {
	return normalizeVideoResponse(raw, a.config.UploadResponse)
}
func (a customJSONVideoAdapter) ResultURL(raw []byte) (string, bool) {
	path := strings.TrimSpace(a.config.Response.ResultURL)
	if path == "" {
		path = strings.TrimSpace(a.config.Response.Fields["result_url"])
	}
	if path == "" {
		return "", false
	}
	var root any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&root) != nil {
		return "", false
	}
	// Field mappings are relative to response.data_path, while an explicit
	// result_url path may be either provider-rooted or data-rooted. Trying the
	// data envelope first keeps both compact provider payloads and nested
	// response envelopes compatible without adding another protocol switch.
	data := root
	if a.config.Response.DataPath != "" {
		if nested := jsonPathValue(root, a.config.Response.DataPath); nested != nil {
			data = nested
		}
	}
	if value, ok := safeVideoResultURL(jsonPathValue(data, path)); ok {
		return value, true
	}
	return safeVideoResultURL(jsonPathValue(root, path))
}

func videoProviderAdapterForAccount(account *Account) (videoProviderAdapter, error) {
	protocol := XiaoVideoProtocolNative
	if account != nil {
		protocol = account.XiaoVideoProtocol()
	}
	switch protocol {
	case XiaoVideoProtocolNative:
		return nativeVideoAdapter{}, nil
	case XiaoVideoProtocolOpenAISora:
		return openAISoraVideoAdapter{}, nil
	case XiaoVideoProtocolCTMOAI:
		return ctmoaiVideoAdapter{}, nil
	case XiaoVideoProtocolCustomJSON:
		config, err := videoJSONAdapterConfig(account)
		if err != nil {
			return nil, err
		}
		return customJSONVideoAdapter{config: config}, nil
	default:
		return nil, fmt.Errorf("unsupported video protocol %q", protocol)
	}
}

func videoJSONAdapterConfig(account *Account) (VideoJSONAdapterConfig, error) {
	var config VideoJSONAdapterConfig
	if account == nil || account.Credentials == nil {
		return config, errors.New("video adapter configuration is required")
	}
	raw, exists := account.Credentials[XiaoVideoAdapterCredentialKey]
	if !exists {
		return config, errors.New("video_adapter is required")
	}
	encoded, err := json.Marshal(raw)
	if text, ok := raw.(string); ok {
		encoded = []byte(text)
	}
	if err != nil {
		return config, fmt.Errorf("encode video_adapter: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return config, fmt.Errorf("decode video_adapter: %w", err)
	}
	if err := validateVideoJSONAdapterConfig(config); err != nil {
		return config, err
	}
	return config, nil
}

func validateVideoJSONAdapterConfig(config VideoJSONAdapterConfig) error {
	if config.Version != 1 {
		return errors.New("video_adapter.version must be 1")
	}
	switch config.Auth.Type {
	case "bearer", "none":
	case "header":
		if !validVideoHeaderName(config.Auth.Header) {
			return errors.New("video_adapter.auth.header is invalid")
		}
	default:
		return errors.New("video_adapter.auth.type must be bearer, header, or none")
	}
	for name := range config.StaticHeaders {
		if !validVideoHeaderName(name) {
			return fmt.Errorf("video_adapter.static_headers contains invalid header %q", name)
		}
	}
	if config.IdempotencyHeader == "" {
		config.IdempotencyHeader = "Idempotency-Key"
	} else if !validVideoHeaderName(config.IdempotencyHeader) {
		return errors.New("video_adapter.idempotency_header is invalid")
	}
	endpoints := map[videoProviderOperation]videoProviderEndpoint{
		videoOperationModels: config.Endpoints.Models, videoOperationCapabilities: config.Endpoints.Capabilities,
		videoOperationUpload: config.Endpoints.Upload, videoOperationUploadContent: config.Endpoints.UploadContent,
		videoOperationCreate: config.Endpoints.Create, videoOperationStatus: config.Endpoints.Status,
		videoOperationCancel: config.Endpoints.Cancel, videoOperationContent: config.Endpoints.Content,
	}
	for operation, endpoint := range endpoints {
		required := operation == videoOperationModels || operation == videoOperationCreate || operation == videoOperationStatus
		if strings.TrimSpace(endpoint.Path) == "" {
			if required {
				return fmt.Errorf("video_adapter.endpoints.%s is required", operation)
			}
			continue
		}
		if err := validateVideoAdapterEndpoint(operation, endpoint); err != nil {
			return err
		}
	}
	if config.Endpoints.Content.Path == "" && config.Response.ResultURL == "" && config.Response.Fields["result_url"] == "" {
		return errors.New("video_adapter requires a content endpoint or response.result_url")
	}
	for source, target := range config.Request.Fields {
		if strings.TrimSpace(source) == "" || !validJSONPath(target) {
			return errors.New("video_adapter.request.fields contains an invalid mapping")
		}
	}
	for path := range config.Request.Static {
		if !validJSONPath(path) {
			return errors.New("video_adapter.request.static contains an invalid path")
		}
	}
	for canonical, path := range config.Response.Fields {
		if strings.TrimSpace(canonical) == "" || !validJSONPath(path) {
			return errors.New("video_adapter.response.fields contains an invalid mapping")
		}
	}
	if config.Response.Fields["job_id"] == "" || config.Response.Fields["status"] == "" {
		return errors.New("video_adapter.response.fields requires job_id and status")
	}
	for _, status := range config.Response.StatusMap {
		if !isVideoStatus(status) {
			return fmt.Errorf("video_adapter response status %q is invalid", status)
		}
	}
	if config.Endpoints.Upload.Path != "" {
		if config.Endpoints.UploadContent.Path == "" {
			return errors.New("video_adapter upload_content endpoint is required when upload is enabled")
		}
		if config.UploadResponse.Fields["media_id"] == "" || config.UploadResponse.Fields["url"] == "" {
			return errors.New("video_adapter.upload_response.fields requires media_id and url")
		}
	}
	for _, path := range []string{config.Models.ItemsPath, config.Models.IDPath, config.Capabilities.ItemsPath, config.Capabilities.IDPath, config.Capabilities.SchemaVersionPath, config.Response.DataPath, config.Response.ResultURL, config.UploadResponse.DataPath} {
		if path != "" && !validJSONPath(path) {
			return fmt.Errorf("video_adapter contains invalid JSON path %q", path)
		}
	}
	return nil
}

func validateVideoAdapterEndpoint(operation videoProviderOperation, endpoint videoProviderEndpoint) error {
	method := strings.ToUpper(strings.TrimSpace(endpoint.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return fmt.Errorf("video_adapter endpoint %s has an invalid method", operation)
	}
	parsed, err := url.Parse(strings.TrimSpace(endpoint.Path))
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.Fragment != "" || !strings.HasPrefix(parsed.Path, "/") {
		return fmt.Errorf("video_adapter endpoint %s must use an absolute relative path", operation)
	}
	path := endpoint.Path
	allowedJobID := operation == videoOperationStatus || operation == videoOperationCancel || operation == videoOperationContent
	allowedMediaID := operation == videoOperationUploadContent
	if strings.Contains(path, "{job_id}") && !allowedJobID {
		return fmt.Errorf("video_adapter endpoint %s cannot use {job_id}", operation)
	}
	if strings.Contains(path, "{media_id}") && !allowedMediaID {
		return fmt.Errorf("video_adapter endpoint %s cannot use {media_id}", operation)
	}
	cleaned := strings.ReplaceAll(strings.ReplaceAll(path, "{job_id}", ""), "{media_id}", "")
	if strings.ContainsAny(cleaned, "{}") {
		return fmt.Errorf("video_adapter endpoint %s contains an unsupported placeholder", operation)
	}
	if allowedJobID && !strings.Contains(path, "{job_id}") {
		return fmt.Errorf("video_adapter endpoint %s requires {job_id}", operation)
	}
	if allowedMediaID && !strings.Contains(path, "{media_id}") {
		return fmt.Errorf("video_adapter endpoint %s requires {media_id}", operation)
	}
	return nil
}

func validVideoHeaderName(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && http.CanonicalHeaderKey(value) != "" && !strings.ContainsAny(value, "\r\n:")
}

func validJSONPath(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" || strings.HasPrefix(path, ".") || strings.HasSuffix(path, ".") {
		return false
	}
	for _, segment := range strings.Split(path, ".") {
		if segment == "" || strings.ContainsAny(segment, "[]{}\r\n") {
			return false
		}
	}
	return true
}

func decodeVideoJSONObject(raw []byte) (map[string]any, error) {
	var value map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

func decodeVideoCollection(raw []byte, itemsPath, idPath string) ([]map[string]any, error) {
	var root any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&root); err != nil {
		return nil, err
	}
	items := root
	if strings.TrimSpace(itemsPath) != "" {
		items = jsonPathValue(root, itemsPath)
	}
	rawItems, ok := items.([]any)
	if !ok {
		return nil, errors.New("video adapter collection path is not an array")
	}
	result := make([]map[string]any, 0, len(rawItems))
	for _, rawItem := range rawItems {
		item, ok := rawItem.(map[string]any)
		if !ok {
			continue
		}
		id := videoStringValue(jsonPathValue(item, idPath))
		if id == "" {
			continue
		}
		copy := make(map[string]any, len(item)+1)
		for key, value := range item {
			copy[key] = value
		}
		copy["id"] = id
		result = append(result, copy)
	}
	return result, nil
}

func decodeVideoCapabilityCollection(raw []byte, itemsPath, idPath, schemaPath string) (int, []map[string]any, error) {
	items, err := decodeVideoCollection(raw, itemsPath, idPath)
	if err != nil {
		return 0, nil, err
	}
	version := 1
	if schemaPath != "" {
		if parsed, parseErr := strconv.Atoi(videoStringValue(jsonPathValueFromBytes(raw, schemaPath))); parseErr == nil && parsed > 0 {
			version = parsed
		}
	}
	return version, items, nil
}

func normalizeVideoResponse(raw []byte, config VideoAdapterResponseConfig) (map[string]any, error) {
	var root any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&root); err != nil {
		return nil, err
	}
	data := root
	if config.DataPath != "" {
		data = jsonPathValue(root, config.DataPath)
	}
	result := make(map[string]any, len(config.Fields)+len(config.Defaults))
	for key, value := range config.Defaults {
		result[key] = value
	}
	for canonical, path := range config.Fields {
		if value := jsonPathValue(data, path); value != nil {
			result[canonical] = value
		}
	}
	status := videoStringValue(result["status"])
	if mapped, ok := config.StatusMap[status]; ok {
		status = mapped
	} else if mapped, ok := config.StatusMap[strings.ToLower(status)]; ok {
		status = mapped
	}
	if status != "" {
		result["status"] = status
	}
	return result, nil
}

func jsonPathValueFromBytes(raw []byte, path string) any {
	var root any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&root) != nil {
		return nil
	}
	return jsonPathValue(root, path)
}

func jsonPathValue(root any, path string) any {
	current := root
	for _, segment := range strings.Split(strings.TrimSpace(path), ".") {
		switch typed := current.(type) {
		case map[string]any:
			current = typed[segment]
		case []any:
			index, err := strconv.Atoi(segment)
			if err != nil || index < 0 || index >= len(typed) {
				return nil
			}
			current = typed[index]
		default:
			return nil
		}
	}
	return current
}

func setJSONPath(root map[string]any, path string, value any) {
	segments := strings.Split(path, ".")
	current := root
	for _, segment := range segments[:len(segments)-1] {
		next, ok := current[segment].(map[string]any)
		if !ok {
			next = make(map[string]any)
			current[segment] = next
		}
		current = next
	}
	current[segments[len(segments)-1]] = value
}

func safeVideoResultURL(value any) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(videoStringValue(value)))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", false
	}
	return parsed.String(), true
}
