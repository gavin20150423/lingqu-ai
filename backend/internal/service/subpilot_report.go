package service

import (
	"context"
	"strconv"
	"strings"
)

type SubPilotFailureInput struct {
	LeaseID       string
	APIKey        *APIKey
	Account       *Account
	RequestID     string
	Platform      string
	GroupID       string
	Model         string
	UpstreamModel string
	SessionKey    string
	StatusCode    int
	ErrorCode     string
	ErrorMessage  string
	RequestType   string
	Stream        bool
	OpenAIWSMode  bool
	QuotaPlatform string
}

func (s *GatewayService) reportSubPilotSuccess(ctx context.Context, usageLog *UsageLog, input *recordUsageCoreInput, cost *CostBreakdown, accountRateMultiplier float64) {
	client := s.subPilotClient()
	if client == nil || usageLog == nil || input == nil || input.SubPilotLeaseID == "" {
		return
	}
	stream := usageLog.Stream
	firstTokenMS := 0
	if usageLog.FirstTokenMs != nil {
		firstTokenMS = *usageLog.FirstTokenMs
	}
	latencyMS := 0
	if usageLog.DurationMs != nil {
		latencyMS = *usageLog.DurationMs
	}
	totalCost := 0.0
	if cost != nil {
		totalCost = cost.TotalCost
	}
	client.reportSuccess(ctx, subPilotReportSuccessRequest{
		RequestID:       subPilotReportRequestID(ctx, usageLog.RequestID),
		LeaseID:         input.SubPilotLeaseID,
		APIKeyID:        strconv.FormatInt(usageLog.APIKeyID, 10),
		AccountID:       strconv.FormatInt(usageLog.AccountID, 10),
		Platform:        subPilotPlatformFromAPIKey(input.APIKey, input.QuotaPlatform),
		GroupID:         subPilotGroupIDString(input.APIKey),
		Model:           subPilotReportModel(usageLog.RequestedModel, valueOfStringPtr(usageLog.UpstreamModel)),
		LatencyMS:       latencyMS,
		FirstTokenMS:    firstTokenMS,
		RequestType:     subPilotRequestType(usageLog.Stream, usageLog.OpenAIWSMode),
		Stream:          &stream,
		OfficialUSDUsed: subPilotOfficialUSD(totalCost, accountRateMultiplier),
		SessionKey:      input.SubPilotSessionKey,
		StickyTTLMS:     stickySessionTTL.Milliseconds(),
	})
}

func (s *OpenAIGatewayService) reportSubPilotSuccess(ctx context.Context, usageLog *UsageLog, input *OpenAIRecordUsageInput, cost *CostBreakdown, accountRateMultiplier float64) {
	client := s.subPilotClient()
	if client == nil || usageLog == nil || input == nil || input.SubPilotLeaseID == "" {
		return
	}
	stream := usageLog.Stream
	firstTokenMS := 0
	if usageLog.FirstTokenMs != nil {
		firstTokenMS = *usageLog.FirstTokenMs
	}
	latencyMS := 0
	if usageLog.DurationMs != nil {
		latencyMS = *usageLog.DurationMs
	}
	totalCost := 0.0
	if cost != nil {
		totalCost = cost.TotalCost
	}
	client.reportSuccess(ctx, subPilotReportSuccessRequest{
		RequestID:       subPilotReportRequestID(ctx, usageLog.RequestID),
		LeaseID:         input.SubPilotLeaseID,
		APIKeyID:        strconv.FormatInt(usageLog.APIKeyID, 10),
		AccountID:       strconv.FormatInt(usageLog.AccountID, 10),
		Platform:        subPilotPlatformFromAPIKey(input.APIKey, input.QuotaPlatform),
		GroupID:         subPilotGroupIDString(input.APIKey),
		Model:           subPilotReportModel(usageLog.RequestedModel, valueOfStringPtr(usageLog.UpstreamModel)),
		LatencyMS:       latencyMS,
		FirstTokenMS:    firstTokenMS,
		RequestType:     subPilotRequestType(usageLog.Stream, usageLog.OpenAIWSMode),
		Stream:          &stream,
		OfficialUSDUsed: subPilotOfficialUSD(totalCost, accountRateMultiplier),
		SessionKey:      input.SubPilotSessionKey,
		StickyTTLMS:     openaiStickySessionTTL.Milliseconds(),
	})
}

func (s *GatewayService) ReportSubPilotFailure(ctx context.Context, input SubPilotFailureInput) SubPilotRetryDirective {
	client := s.subPilotClient()
	if client == nil || input.LeaseID == "" || input.Account == nil {
		return SubPilotRetryDirective{}
	}
	// Content-policy refusals belong to this request, not to the selected
	// account. Do not feed them into SubPilot health metrics or retry/failover
	// feedback; the lease is still released by the report endpoint.
	if isNonRetryableRequestFailure(input.StatusCode, input.ErrorCode, input.ErrorMessage) {
		return SubPilotRetryDirective{}
	}
	stream := input.Stream
	return client.reportFailure(ctx, subPilotReportFailureRequest{
		RequestID:    subPilotReportRequestID(ctx, input.RequestID),
		LeaseID:      input.LeaseID,
		APIKeyID:     subPilotReportAPIKeyID(ctx, input.APIKey),
		AccountID:    strconv.FormatInt(input.Account.ID, 10),
		Platform:     input.reportPlatform(),
		GroupID:      input.reportGroupID(),
		Model:        subPilotReportModel(input.Model, input.UpstreamModel),
		SessionKey:   input.SessionKey,
		StatusCode:   input.StatusCode,
		ErrorCode:    input.ErrorCode,
		ErrorMessage: input.ErrorMessage,
		RequestType:  input.normalizedRequestType(),
		Stream:       &stream,
	})
}

func (s *OpenAIGatewayService) ReportSubPilotFailure(ctx context.Context, input SubPilotFailureInput) SubPilotRetryDirective {
	client := s.subPilotClient()
	if client == nil || input.LeaseID == "" || input.Account == nil {
		return SubPilotRetryDirective{}
	}
	if isNonRetryableRequestFailure(input.StatusCode, input.ErrorCode, input.ErrorMessage) {
		return SubPilotRetryDirective{}
	}
	stream := input.Stream
	return client.reportFailure(ctx, subPilotReportFailureRequest{
		RequestID:    subPilotReportRequestID(ctx, input.RequestID),
		LeaseID:      input.LeaseID,
		APIKeyID:     subPilotReportAPIKeyID(ctx, input.APIKey),
		AccountID:    strconv.FormatInt(input.Account.ID, 10),
		Platform:     input.reportPlatform(),
		GroupID:      input.reportGroupID(),
		Model:        subPilotReportModel(input.Model, input.UpstreamModel),
		SessionKey:   input.SessionKey,
		StatusCode:   input.StatusCode,
		ErrorCode:    input.ErrorCode,
		ErrorMessage: input.ErrorMessage,
		RequestType:  input.normalizedRequestType(),
		Stream:       &stream,
	})
}

func isNonRetryableRequestFailure(statusCode int, errorCode, message string) bool {
	if statusCode != 403 {
		return false
	}
	value := strings.ToLower(strings.TrimSpace(errorCode + " " + message))
	for _, marker := range []string{
		"content_policy_error",
		"content_policy_violation",
		"content policy",
		"content moderation",
		"usage policy",
		"safety policy",
	} {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return false
}

func (input SubPilotFailureInput) normalizedRequestType() string {
	if input.RequestType != "" {
		return input.RequestType
	}
	return subPilotRequestType(input.Stream, input.OpenAIWSMode)
}

func (input SubPilotFailureInput) reportPlatform() string {
	if input.Platform != "" {
		return input.Platform
	}
	return subPilotPlatformFromAPIKey(input.APIKey, input.QuotaPlatform)
}

func (input SubPilotFailureInput) reportGroupID() string {
	if input.GroupID != "" {
		return input.GroupID
	}
	return subPilotGroupIDString(input.APIKey)
}

func valueOfStringPtr(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
