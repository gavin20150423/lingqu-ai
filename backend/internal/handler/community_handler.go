package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type CommunityHandler struct {
	service        *service.CommunityService
	anthropicOAuth *service.OAuthService
	openaiOAuth    *service.OpenAIOAuthService
	billing        *service.BillingService
	payment        *service.PaymentService
}

func NewCommunityHandler(s *service.CommunityService, anthropicOAuth *service.OAuthService, openaiOAuth *service.OpenAIOAuthService, billing *service.BillingService, paymentService *service.PaymentService) *CommunityHandler {
	if paymentService != nil {
		paymentService.SetCommunityStoreFulfiller(s)
	}
	return &CommunityHandler{service: s, anthropicOAuth: anthropicOAuth, openaiOAuth: openaiOAuth, billing: billing, payment: paymentService}
}

func communitySubject(c *gin.Context) (int64, bool) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "User not authenticated")
		return 0, false
	}
	return subject.UserID, true
}
func communityID(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid "+name)
		return 0, false
	}
	return id, true
}
func communityError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrCommunityInvalid):
		response.BadRequest(c, "请求参数不正确")
	case errors.Is(err, service.ErrCommunityNotFound):
		response.NotFound(c, "记录不存在")
	case errors.Is(err, service.ErrCommunityForbidden):
		response.Forbidden(c, "无权执行此操作")
	case errors.Is(err, service.ErrCommunityInsufficient):
		response.Error(c, http.StatusPaymentRequired, "余额不足")
	case errors.Is(err, service.ErrCommunityConflict):
		response.Error(c, http.StatusConflict, "当前状态不允许此操作")
	default:
		subject, _ := middleware2.GetAuthSubjectFromContext(c)
		requestID, _ := c.Request.Context().Value(ctxkey.RequestID).(string)
		slog.Error("community operation failed",
			"error", err,
			"method", c.Request.Method,
			"route", c.FullPath(),
			"user_id", subject.UserID,
			"request_id", requestID,
		)
		response.InternalError(c, "操作失败")
	}
}

func (h *CommunityHandler) ListAccounts(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	v, err := h.service.ListAccounts(c.Request.Context(), uid)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, v)
}
func (h *CommunityHandler) CreateAccount(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	var in service.CommunityAccountInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	v, err := h.service.CreateAccount(c.Request.Context(), uid, in)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Created(c, v)
}

func (h *CommunityHandler) ImportAccounts(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	var in service.CommunityAccountImportInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid import request")
		return
	}
	items, err := h.service.ImportAccounts(c.Request.Context(), uid, in)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Created(c, gin.H{"count": len(items), "accounts": items})
}

func (h *CommunityHandler) ExportAccounts(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	var in struct {
		IDs []int64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid export request")
		return
	}
	items, err := h.service.ExportAccounts(c.Request.Context(), uid, in.IDs)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, items)
}

func (h *CommunityHandler) BatchUpdateAccounts(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	var in service.CommunityAccountBatchUpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid batch update request")
		return
	}
	count, err := h.service.BatchUpdateAccounts(c.Request.Context(), uid, in)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, gin.H{"updated": count})
}

func (h *CommunityHandler) GenerateAccountOAuthURL(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	var in struct {
		Provider    string `json:"provider"`
		ProxyID     *int64 `json:"proxy_id"`
		RedirectURI string `json:"redirect_uri"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	if err := h.service.CanUseProxy(c.Request.Context(), uid, in.ProxyID); err != nil {
		communityError(c, err)
		return
	}
	if in.Provider == "anthropic" && in.ProxyID == nil {
		response.BadRequest(c, "Anthropic OAuth requires a proxy")
		return
	}
	switch in.Provider {
	case "openai":
		result, err := h.openaiOAuth.GenerateAuthURL(c.Request.Context(), in.ProxyID, in.RedirectURI, service.PlatformOpenAI)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		response.Success(c, result)
	case "anthropic":
		result, err := h.anthropicOAuth.GenerateAuthURL(c.Request.Context(), in.ProxyID)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		response.Success(c, result)
	default:
		response.BadRequest(c, "Unsupported OAuth provider")
	}
}

func (h *CommunityHandler) ExchangeAccountOAuth(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	var in struct {
		Account     service.CommunityAccountInput `json:"account"`
		SessionID   string                        `json:"session_id"`
		Code        string                        `json:"code"`
		State       string                        `json:"state"`
		RedirectURI string                        `json:"redirect_uri"`
		ProxyID     *int64                        `json:"proxy_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.SessionID == "" || in.Code == "" {
		response.BadRequest(c, "Invalid OAuth callback")
		return
	}
	if err := h.service.CanUseProxy(c.Request.Context(), uid, in.ProxyID); err != nil {
		communityError(c, err)
		return
	}
	if in.Account.Provider == "anthropic" && in.ProxyID == nil {
		response.BadRequest(c, "Anthropic OAuth requires a proxy")
		return
	}
	var credential any
	var err error
	switch in.Account.Provider {
	case "openai":
		credential, err = h.openaiOAuth.ExchangeCode(c.Request.Context(), &service.OpenAIExchangeCodeInput{SessionID: in.SessionID, Code: in.Code, State: in.State, RedirectURI: in.RedirectURI, ProxyID: in.ProxyID})
	case "anthropic":
		credential, err = h.anthropicOAuth.ExchangeCode(c.Request.Context(), &service.ExchangeCodeInput{SessionID: in.SessionID, Code: in.Code, ProxyID: in.ProxyID})
	default:
		response.BadRequest(c, "Unsupported OAuth provider")
		return
	}
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	payload, err := json.Marshal(credential)
	if err != nil {
		response.InternalError(c, "Failed to protect OAuth credential")
		return
	}
	in.Account.OAuthCredential = string(payload)
	in.Account.ProxyID = in.ProxyID
	account, err := h.service.CreateAccount(c.Request.Context(), uid, in.Account)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Created(c, account)
}

func (h *CommunityHandler) ListProxies(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	items, err := h.service.ListProxies(c.Request.Context(), uid)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, items)
}

func (h *CommunityHandler) SaveProxy(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	var in service.CommunityProxyInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	if c.Param("id") != "" {
		id, valid := communityID(c, "id")
		if !valid {
			return
		}
		in.ID = id
	}
	item, err := h.service.SaveProxy(c.Request.Context(), uid, in)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, item)
}

func (h *CommunityHandler) DeleteProxy(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	id, ok := communityID(c, "id")
	if !ok {
		return
	}
	if err := h.service.DeleteProxy(c.Request.Context(), uid, id); err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}
func (h *CommunityHandler) DeleteAccount(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	id, ok := communityID(c, "id")
	if !ok {
		return
	}
	if err := h.service.DeleteAccount(c.Request.Context(), uid, id); err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}
func (h *CommunityHandler) CreateListing(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	var in service.CommunityListingInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	v, err := h.service.CreateListing(c.Request.Context(), uid, in)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Created(c, v)
}
func (h *CommunityHandler) ListMarketplace(c *gin.Context) {
	v, err := h.service.ListMarketplace(c.Request.Context(), c.Query("provider"), c.Query("search"))
	if err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, v)
}
func (h *CommunityHandler) ListOwnerListings(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	items, err := h.service.ListOwnerListings(c.Request.Context(), uid, c.Query("provider"))
	if err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, items)
}
func (h *CommunityHandler) JoinListing(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	id, ok := communityID(c, "id")
	if !ok {
		return
	}
	var in struct {
		APIKeyID           int64 `json:"api_key_id"`
		IdleTimeoutMinutes int   `json:"idle_timeout_minutes"`
	}
	_ = c.ShouldBindJSON(&in)
	membershipID, err := h.service.JoinListing(c.Request.Context(), uid, id, in.APIKeyID, in.IdleTimeoutMinutes)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Created(c, gin.H{"membership_id": membershipID})
}
func (h *CommunityHandler) LeaveListing(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	id, ok := communityID(c, "id")
	if !ok {
		return
	}
	if err := h.service.LeaveListing(c.Request.Context(), uid, id); err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, gin.H{"left": true})
}

func (h *CommunityHandler) ReviewMembership(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	membershipID, ok := communityID(c, "id")
	if !ok {
		return
	}
	var in struct {
		Score   float64 `json:"score"`
		Content string  `json:"content"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid review")
		return
	}
	if err := h.service.ReviewMembership(c.Request.Context(), uid, membershipID, in.Score, in.Content); err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, gin.H{"reviewed": true})
}

func (h *CommunityHandler) ListMemberships(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	items, err := h.service.ListMemberships(c.Request.Context(), uid)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, items)
}

func (h *CommunityHandler) ListConsumptionAccounts(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	items, err := h.service.ListConsumptionAccounts(c.Request.Context(), uid)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, items)
}

func (h *CommunityHandler) GetConsumptionSummary(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	membershipID, valid := communityID(c, "membership_id")
	if !valid {
		return
	}
	item, err := h.service.GetConsumptionSummary(c.Request.Context(), uid, membershipID, c.DefaultQuery("scope", "session"))
	if err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, item)
}

func (h *CommunityHandler) RecommendListings(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	var in service.CommunitySelectionInput
	if err := c.ShouldBindJSON(&in); err != nil || h.billing == nil {
		response.BadRequest(c, "Invalid selection request")
		return
	}
	cost, err := h.billing.CalculateCost(in.Model, service.UsageTokens{
		InputTokens: in.InputTokens, OutputTokens: in.OutputTokens,
		CacheCreationTokens: in.CacheCreationTokens, CacheReadTokens: in.CacheReadTokens,
	}, 1)
	if err != nil {
		response.BadRequest(c, "无法计算该模型价格")
		return
	}
	items, err := h.service.RecommendListings(c.Request.Context(), uid, in, cost.ActualCost)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, items)
}

func (h *CommunityHandler) GetRecentUsageAverage(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	apiKeyID, valid := communityID(c, "api_key_id")
	if !valid {
		return
	}
	item, err := h.service.GetRecentUsageAverage(c.Request.Context(), uid, apiKeyID)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, item)
}

func (h *CommunityHandler) GetOwnerSummary(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	item, err := h.service.GetOwnerSummary(c.Request.Context(), uid)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, item)
}

func (h *CommunityHandler) SetListingStatus(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	id, valid := communityID(c, "id")
	if !valid {
		return
	}
	var in struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	if err := h.service.SetListingStatus(c.Request.Context(), uid, id, in.Status); err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, gin.H{"status": in.Status})
}

func (h *CommunityHandler) ListAccountModeKeys(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	items, err := h.service.ListAccountModeKeys(c.Request.Context(), uid, c.Query("provider"))
	if err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, items)
}

func (h *CommunityHandler) SetAccountModeKey(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	id, ok := communityID(c, "id")
	if !ok {
		return
	}
	var in struct {
		Provider string `json:"provider"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	if err := h.service.SetAccountModeKey(c.Request.Context(), uid, id, in.Provider); err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, gin.H{"configured": true})
}

func (h *CommunityHandler) ListTickets(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	v, err := h.service.ListTickets(c.Request.Context(), uid, false)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, v)
}
func (h *CommunityHandler) CreateTicket(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	var in struct {
		Subject  string `json:"subject"`
		Category string `json:"category"`
		Priority string `json:"priority"`
		Content  string `json:"content"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	v, err := h.service.CreateTicket(c.Request.Context(), uid, in.Subject, in.Category, in.Priority, in.Content)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Created(c, v)
}
func (h *CommunityHandler) GetTicket(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	id, ok := communityID(c, "id")
	if !ok {
		return
	}
	v, err := h.service.GetTicket(c.Request.Context(), uid, id, false)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, v)
}
func (h *CommunityHandler) ReplyTicket(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	id, ok := communityID(c, "id")
	if !ok {
		return
	}
	var in struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	if err := h.service.ReplyTicket(c.Request.Context(), uid, id, false, in.Content); err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, gin.H{"sent": true})
}
func (h *CommunityHandler) MarkTicketRead(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	id, valid := communityID(c, "id")
	if !valid {
		return
	}
	if err := h.service.MarkTicketRead(c.Request.Context(), uid, id); err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, gin.H{"read": true})
}
func (h *CommunityHandler) CloseTicket(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	id, valid := communityID(c, "id")
	if !valid {
		return
	}
	if err := h.service.CloseUserTicket(c.Request.Context(), uid, id); err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, gin.H{"closed": true})
}

func (h *CommunityHandler) ListPayoutMethods(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	v, err := h.service.ListPayoutMethods(c.Request.Context(), uid)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, v)
}
func (h *CommunityHandler) SavePayoutMethod(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	var in struct {
		Method      string `json:"method"`
		QRCodeData  string `json:"qr_code_data"`
		DisplayName string `json:"display_name"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	v, err := h.service.SavePayoutMethod(c.Request.Context(), uid, in.Method, in.QRCodeData, in.DisplayName)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, v)
}
func (h *CommunityHandler) DeletePayoutMethod(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	if err := h.service.DeletePayoutMethod(c.Request.Context(), uid, c.Param("method")); err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}
func (h *CommunityHandler) ListWithdrawals(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	v, err := h.service.ListWithdrawals(c.Request.Context(), uid, false)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, v)
}
func (h *CommunityHandler) CreateWithdrawal(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	var in struct {
		Method string  `json:"method"`
		Amount float64 `json:"amount"`
		Note   string  `json:"note"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	v, err := h.service.CreateWithdrawal(c.Request.Context(), uid, in.Method, in.Amount, in.Note)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Created(c, v)
}
func (h *CommunityHandler) CancelWithdrawal(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	id, ok := communityID(c, "id")
	if !ok {
		return
	}
	if err := h.service.CancelWithdrawal(c.Request.Context(), uid, id); err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, gin.H{"cancelled": true})
}

func (h *CommunityHandler) ListProducts(c *gin.Context) {
	v, err := h.service.ListProducts(c.Request.Context(), false)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, v)
}
func (h *CommunityHandler) BuyProduct(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	id, ok := communityID(c, "id")
	if !ok {
		return
	}
	var in struct {
		Quantity      int    `json:"quantity"`
		PaymentSource string `json:"payment_source"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	if in.PaymentSource == "" {
		in.PaymentSource = "balance"
	}
	v, err := h.service.BuyProduct(c.Request.Context(), uid, id, in.Quantity, in.PaymentSource)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Created(c, v)
}

func (h *CommunityHandler) CreatePlatformStoreOrder(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	if h.payment == nil {
		response.Error(c, http.StatusServiceUnavailable, "平台支付暂不可用")
		return
	}
	productID, valid := communityID(c, "id")
	if !valid {
		return
	}
	var in struct {
		Quantity    int    `json:"quantity"`
		PaymentType string `json:"payment_type"`
		ReturnURL   string `json:"return_url"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	prepared, err := h.service.PreparePlatformStoreOrder(c.Request.Context(), uid, productID, in.Quantity, in.PaymentType)
	if err != nil {
		communityError(c, err)
		return
	}
	result, err := h.payment.CreateOrder(c.Request.Context(), service.CreateOrderRequest{
		UserID: uid, Amount: prepared.Amount, PaymentType: in.PaymentType,
		ClientIP: c.ClientIP(), IsMobile: isMobile(c), SrcHost: c.Request.Host,
		SrcURL: c.Request.Referer(), ReturnURL: in.ReturnURL, PaymentSource: "community_store",
		OrderType: payment.OrderTypeStore, StoreOrderID: prepared.StoreOrderID,
		Locale: c.GetHeader("Accept-Language"),
	})
	if err != nil {
		_ = h.service.CancelPreparedPlatformStoreOrder(c.Request.Context(), uid, prepared.StoreOrderID)
		response.ErrorFrom(c, err)
		return
	}
	if err = h.service.BindPlatformStorePayment(c.Request.Context(), uid, prepared.StoreOrderID, result.OrderID, result.ExpiresAt); err != nil {
		_, _ = h.payment.CancelOrder(c.Request.Context(), result.OrderID, uid)
		_ = h.service.CancelPreparedPlatformStoreOrder(c.Request.Context(), uid, prepared.StoreOrderID)
		communityError(c, err)
		return
	}
	response.Created(c, gin.H{"store_order_id": prepared.StoreOrderID, "payment": result})
}
func (h *CommunityHandler) GetStoreWallet(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	wallet, err := h.service.GetStoreWallet(c.Request.Context(), uid)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, wallet)
}
func (h *CommunityHandler) ListStoreOrders(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	v, err := h.service.ListStoreOrders(c.Request.Context(), uid, false)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, v)
}

func (h *CommunityHandler) AdminListTickets(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	v, err := h.service.ListTickets(c.Request.Context(), uid, true)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, v)
}

func (h *CommunityHandler) AdminListAccounts(c *gin.Context) {
	items, err := h.service.ListAdminAccounts(c.Request.Context())
	if err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, items)
}

func (h *CommunityHandler) AdminReviewAccount(c *gin.Context) {
	adminID, ok := communitySubject(c)
	if !ok {
		return
	}
	id, ok := communityID(c, "id")
	if !ok {
		return
	}
	var in struct {
		Decision string `json:"decision"`
		Note     string `json:"note"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	if err := h.service.ReviewAccount(c.Request.Context(), adminID, id, in.Decision, in.Note); err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, gin.H{"decision": in.Decision})
}
func (h *CommunityHandler) AdminGetTicket(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	id, ok := communityID(c, "id")
	if !ok {
		return
	}
	v, err := h.service.GetTicket(c.Request.Context(), uid, id, true)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, v)
}
func (h *CommunityHandler) AdminReplyTicket(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	id, ok := communityID(c, "id")
	if !ok {
		return
	}
	var in struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	if err := h.service.ReplyTicket(c.Request.Context(), uid, id, true, in.Content); err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, gin.H{"sent": true})
}
func (h *CommunityHandler) AdminUpdateTicket(c *gin.Context) {
	_, ok := communitySubject(c)
	if !ok {
		return
	}
	id, ok := communityID(c, "id")
	if !ok {
		return
	}
	var in struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	if err := h.service.UpdateTicketStatus(c.Request.Context(), id, in.Status); err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, gin.H{"status": in.Status})
}
func (h *CommunityHandler) AdminListWithdrawals(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	v, err := h.service.ListWithdrawals(c.Request.Context(), uid, true)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, v)
}
func (h *CommunityHandler) AdminReviewWithdrawal(c *gin.Context) {
	uid, ok := communitySubject(c)
	if !ok {
		return
	}
	id, ok := communityID(c, "id")
	if !ok {
		return
	}
	var in struct {
		Status           string `json:"status"`
		Note             string `json:"note"`
		PaymentReference string `json:"payment_reference"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	if err := h.service.ReviewWithdrawal(c.Request.Context(), uid, id, in.Status, in.Note, in.PaymentReference); err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, gin.H{"status": in.Status})
}
func (h *CommunityHandler) AdminListProducts(c *gin.Context) {
	v, err := h.service.ListProducts(c.Request.Context(), true)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, v)
}
func (h *CommunityHandler) AdminUpsertProduct(c *gin.Context) {
	var in service.StoreProduct
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	if raw := c.Param("id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			response.BadRequest(c, "Invalid id")
			return
		}
		in.ID = id
	}
	v, err := h.service.UpsertProduct(c.Request.Context(), in)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, v)
}
func (h *CommunityHandler) AdminAddInventory(c *gin.Context) {
	id, ok := communityID(c, "id")
	if !ok {
		return
	}
	var in struct {
		Values []string `json:"values"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	n, err := h.service.AddInventory(c.Request.Context(), id, in.Values)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, gin.H{"added": n})
}
func (h *CommunityHandler) AdminListStoreOrders(c *gin.Context) {
	v, err := h.service.ListStoreOrders(c.Request.Context(), 0, true)
	if err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, v)
}
func (h *CommunityHandler) AdminGetCommission(c *gin.Context) {
	v, _ := h.service.GetCommission(c.Request.Context())
	response.Success(c, gin.H{"commission_percent": v})
}
func (h *CommunityHandler) AdminSetCommission(c *gin.Context) {
	var in struct {
		CommissionPercent float64 `json:"commission_percent"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}
	if err := h.service.SetCommission(c.Request.Context(), in.CommissionPercent); err != nil {
		communityError(c, err)
		return
	}
	response.Success(c, gin.H{"commission_percent": in.CommissionPercent})
}
