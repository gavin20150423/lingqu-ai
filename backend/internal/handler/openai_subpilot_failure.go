package handler

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *OpenAIGatewayHandler) reportSubPilotForwardFailure(c *gin.Context, apiKey *service.APIKey, account *service.Account, selection *service.AccountSelectionResult, model string, sessionKey string, stream bool, failoverErr *service.UpstreamFailoverError, err error) {
	h.reportSubPilotForwardFailureWithCode(c, apiKey, account, selection, model, sessionKey, stream, failoverErr, err, "")
}

func (h *OpenAIGatewayHandler) reportSubPilotForwardFailureWithCode(c *gin.Context, apiKey *service.APIKey, account *service.Account, selection *service.AccountSelectionResult, model string, sessionKey string, stream bool, failoverErr *service.UpstreamFailoverError, err error, errorCode string) {
	if h == nil || h.gatewayService == nil || selection == nil || selection.SubPilotLeaseID == "" || account == nil || c == nil || c.Request == nil {
		storeSubPilotRetryDirective(c, service.SubPilotRetryDirective{})
		return
	}
	statusCode := 0
	errorMessage := ""
	if failoverErr != nil {
		statusCode = failoverErr.StatusCode
		errorMessage = service.ExtractUpstreamErrorMessage(failoverErr.ResponseBody)
	}
	if strings.TrimSpace(errorMessage) == "" && err != nil {
		errorMessage = err.Error()
	}
	directive := h.gatewayService.ReportSubPilotFailure(c.Request.Context(), service.SubPilotFailureInput{
		LeaseID: selection.SubPilotLeaseID, APIKey: apiKey, Account: account,
		RequestID: selection.SubPilotRequestID, Model: model, SessionKey: sessionKey,
		StatusCode: statusCode, ErrorCode: errorCode, ErrorMessage: errorMessage, Stream: stream,
		QuotaPlatform: service.QuotaPlatform(c.Request.Context(), apiKey),
	})
	storeSubPilotRetryDirective(c, directive)
}

func subPilotLeaseID(selection *service.AccountSelectionResult) string {
	if selection == nil {
		return ""
	}
	return selection.SubPilotLeaseID
}
