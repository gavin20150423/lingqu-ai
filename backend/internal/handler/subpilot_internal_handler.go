package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type SubPilotInternalHandler struct {
	accountTestService *service.AccountTestService
	cfg                *config.Config
}

type subPilotProbeRequest struct {
	ModelID string `json:"model_id"`
	Prompt  string `json:"prompt"`
}

type subPilotProbeResponse struct {
	Success      bool   `json:"success"`
	AccountID    int64  `json:"account_id"`
	LatencyMS    int64  `json:"latency_ms"`
	ModelID      string `json:"model_id"`
	ErrorMessage string `json:"error_message,omitempty"`
}

func NewSubPilotInternalHandler(accountTestService *service.AccountTestService, cfg *config.Config) *SubPilotInternalHandler {
	return &SubPilotInternalHandler{
		accountTestService: accountTestService,
		cfg:                cfg,
	}
}

func (h *SubPilotInternalHandler) ProbeAccount(c *gin.Context) {
	if h == nil || h.accountTestService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "subpilot probe unavailable"})
		return
	}
	secret := ""
	if h.cfg != nil {
		secret = strings.TrimSpace(h.cfg.Gateway.SubPilot.ProbeSecret)
		if secret == "" {
			secret = strings.TrimSpace(h.cfg.Gateway.SubPilot.SharedSecret)
		}
	}
	if secret == "" || c.GetHeader("X-SubPilot-Secret") != secret {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	accountID, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || accountID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
		return
	}

	var req subPilotProbeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	result, err := h.accountTestService.RunTestBackgroundWithPrompt(c.Request.Context(), accountID, strings.TrimSpace(req.ModelID), strings.TrimSpace(req.Prompt))
	if err != nil && result == nil {
		c.JSON(http.StatusOK, subPilotProbeResponse{
			Success:      false,
			AccountID:    accountID,
			ModelID:      strings.TrimSpace(req.ModelID),
			ErrorMessage: err.Error(),
		})
		return
	}

	resp := subPilotProbeResponse{
		Success:   result != nil && result.Status == "success",
		AccountID: accountID,
		ModelID:   strings.TrimSpace(req.ModelID),
	}
	if result != nil {
		resp.LatencyMS = result.LatencyMs
		resp.ErrorMessage = result.ErrorMessage
	}
	c.JSON(http.StatusOK, resp)
}
