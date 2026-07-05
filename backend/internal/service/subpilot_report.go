package service

import (
	"context"
	"strconv"
)

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
	})
}

func valueOfStringPtr(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
