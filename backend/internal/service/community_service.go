package service

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/lib/pq"
)

var (
	ErrCommunityNotFound     = errors.New("resource not found")
	ErrCommunityForbidden    = errors.New("operation not permitted")
	ErrCommunityInvalid      = errors.New("invalid request")
	ErrCommunityInsufficient = errors.New("insufficient balance")
	ErrCommunityConflict     = errors.New("resource state conflict")
)

type CommunityService struct {
	db        *sql.DB
	encryptor SecretEncryptor
}

func NewCommunityService(db *sql.DB, encryptor SecretEncryptor) *CommunityService {
	return &CommunityService{db: db, encryptor: encryptor}
}

type CommunityAccountInput struct {
	Name            string         `json:"name"`
	Provider        string         `json:"provider"`
	OAuthCredential string         `json:"oauth_credential"`
	Capacity        int            `json:"capacity"`
	Concurrency     int            `json:"concurrency"`
	Tags            []string       `json:"tags"`
	SupportedModels []string       `json:"supported_models"`
	ShareMode       string         `json:"share_mode"`
	AccountTier     string         `json:"account_tier"`
	ExpiresAt       *time.Time     `json:"expires_at"`
	ProviderOptions map[string]any `json:"provider_options"`
	Notes           string         `json:"notes"`
	ProxyID         *int64         `json:"proxy_id"`
}

type CommunityListingInput struct {
	AccountID          int64   `json:"account_id"`
	Title              string  `json:"title"`
	Description        string  `json:"description"`
	SeatLimit          int     `json:"seat_limit"`
	PerUserConcurrency int     `json:"per_user_concurrency"`
	MinimumBalance     float64 `json:"minimum_balance"`
	HourlyPrice        float64 `json:"hourly_price"`
	HourlyMinimumSpend float64 `json:"hourly_minimum_spend"`
	UsageMultiplier    float64 `json:"usage_multiplier"`
	IdleTimeoutMinutes int     `json:"idle_timeout_minutes"`
	Publish            bool    `json:"publish"`
}

type CommunityAccount struct {
	ID              int64      `json:"id"`
	OwnerUserID     int64      `json:"owner_user_id"`
	Name            string     `json:"name"`
	Provider        string     `json:"provider"`
	Status          string     `json:"status"`
	ReviewStatus    string     `json:"review_status"`
	ReviewNote      string     `json:"review_note"`
	ShareMode       string     `json:"share_mode"`
	AccountTier     string     `json:"account_tier"`
	GroupName       string     `json:"group_name"`
	Capacity        int        `json:"capacity"`
	Concurrency     int        `json:"concurrency"`
	Schedulable     bool       `json:"schedulable"`
	Priority        int        `json:"priority"`
	TodayRequests   int64      `json:"today_requests"`
	TodayTokens     int64      `json:"today_tokens"`
	Usage5hPercent  float64    `json:"usage_5h_percent"`
	Usage7dPercent  float64    `json:"usage_7d_percent"`
	Tags            []string   `json:"tags"`
	SupportedModels []string   `json:"supported_models"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	LastUsedAt      *time.Time `json:"last_used_at,omitempty"`
	Notes           string     `json:"notes"`
	CreatedAt       time.Time  `json:"created_at"`
	ProxyID         *int64     `json:"proxy_id,omitempty"`
}

type CommunityAccountImportItem struct {
	Name       string `json:"name"`
	Credential any    `json:"credential"`
}

type CommunityAccountImportInput struct {
	Provider    string                       `json:"provider"`
	AccountTier string                       `json:"account_tier"`
	ProxyID     *int64                       `json:"proxy_id"`
	ShareMode   string                       `json:"share_mode"`
	Concurrency int                          `json:"concurrency"`
	Items       []CommunityAccountImportItem `json:"items"`
}

type CommunityAccountExportItem struct {
	Name            string         `json:"name"`
	Provider        string         `json:"provider"`
	AccountTier     string         `json:"account_tier"`
	ShareMode       string         `json:"share_mode"`
	Concurrency     int            `json:"concurrency"`
	Tags            []string       `json:"tags"`
	SupportedModels []string       `json:"supported_models"`
	ProviderOptions map[string]any `json:"provider_options"`
	Notes           string         `json:"notes"`
	ProxyID         *int64         `json:"proxy_id,omitempty"`
	Credential      any            `json:"credential"`
}

type CommunityAccountBatchUpdateInput struct {
	IDs          []int64    `json:"ids"`
	ShareMode    *string    `json:"share_mode"`
	Concurrency  *int       `json:"concurrency"`
	Priority     *int       `json:"priority"`
	Schedulable  *bool      `json:"schedulable"`
	ProxyID      *int64     `json:"proxy_id"`
	ClearProxy   bool       `json:"clear_proxy"`
	Notes        *string    `json:"notes"`
	ExpiresAt    *time.Time `json:"expires_at"`
	ClearExpires bool       `json:"clear_expires"`
}

type CommunityListing struct {
	ID                 int64     `json:"id"`
	AccountID          int64     `json:"account_id"`
	OwnerUserID        int64     `json:"owner_user_id"`
	OwnerName          string    `json:"owner_name"`
	Title              string    `json:"title"`
	Description        string    `json:"description"`
	Provider           string    `json:"provider"`
	AccountTier        string    `json:"account_tier"`
	Tags               []string  `json:"tags"`
	SupportedModels    []string  `json:"supported_models"`
	SeatLimit          int       `json:"seat_limit"`
	SeatsUsed          int       `json:"seats_used"`
	PerUserConcurrency int       `json:"per_user_concurrency"`
	MinimumBalance     float64   `json:"minimum_balance"`
	HourlyPrice        float64   `json:"hourly_price"`
	HourlyMinimumSpend float64   `json:"hourly_minimum_spend"`
	UsageMultiplier    float64   `json:"usage_multiplier"`
	Usage5hPercent     float64   `json:"usage_5h_percent"`
	Usage7dPercent     float64   `json:"usage_7d_percent"`
	IdleTimeoutMinutes int       `json:"idle_timeout_minutes"`
	CommissionRate     float64   `json:"commission_rate"`
	Status             string    `json:"status"`
	Score              float64   `json:"score"`
	RatingCount        int       `json:"rating_count"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type CommunityProxyInput struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	IPType   string `json:"ip_type"`
	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type CommunityProxy struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	IPType       string    `json:"ip_type"`
	Protocol     string    `json:"protocol"`
	Host         string    `json:"host"`
	Port         int       `json:"port"`
	Username     string    `json:"username"`
	HasPassword  bool      `json:"has_password"`
	AccountCount int       `json:"account_count"`
	CreatedAt    time.Time `json:"created_at"`
}

func normalizeCommunityProxy(in *CommunityProxyInput) error {
	in.Name = strings.TrimSpace(in.Name)
	in.Host = strings.TrimSpace(in.Host)
	in.Username = strings.TrimSpace(in.Username)
	in.Protocol = strings.ToLower(strings.TrimSpace(in.Protocol))
	in.IPType = strings.ToLower(strings.TrimSpace(in.IPType))
	if in.IPType == "" {
		in.IPType = "ipv4"
	}
	if in.Name == "" {
		in.Name = fmt.Sprintf("%s:%d", in.Host, in.Port)
	}
	if in.Host == "" || strings.ContainsAny(in.Host, " \t\r\n") || in.Port < 1 || in.Port > 65535 {
		return ErrCommunityInvalid
	}
	switch in.Protocol {
	case "http", "https", "socks5", "socks5h":
	default:
		return ErrCommunityInvalid
	}
	if in.IPType != "ipv4" && in.IPType != "ipv6" {
		return ErrCommunityInvalid
	}
	return nil
}

func (s *CommunityService) ListProxies(ctx context.Context, userID int64) ([]CommunityProxy, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT p.id,p.name,p.ip_type,p.protocol,p.host,p.port,COALESCE(p.username,''),COALESCE(p.password,'')<>'',(SELECT COUNT(*) FROM community_accounts a WHERE a.proxy_id=p.id AND a.deleted_at IS NULL),p.created_at FROM proxies p WHERE p.owner_user_id=$1 AND p.deleted_at IS NULL ORDER BY p.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]CommunityProxy, 0)
	for rows.Next() {
		var item CommunityProxy
		if err = rows.Scan(&item.ID, &item.Name, &item.IPType, &item.Protocol, &item.Host, &item.Port, &item.Username, &item.HasPassword, &item.AccountCount, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *CommunityService) SaveProxy(ctx context.Context, userID int64, in CommunityProxyInput) (*CommunityProxy, error) {
	if err := normalizeCommunityProxy(&in); err != nil {
		return nil, err
	}
	var id int64
	if in.ID == 0 {
		err := s.db.QueryRowContext(ctx, `INSERT INTO proxies(name,ip_type,protocol,host,port,username,password,status,owner_user_id) VALUES($1,$2,$3,$4,$5,NULLIF($6,''),NULLIF($7,''),'active',$8) RETURNING id`, in.Name, in.IPType, in.Protocol, in.Host, in.Port, in.Username, in.Password, userID).Scan(&id)
		if err != nil {
			return nil, err
		}
	} else {
		res, err := s.db.ExecContext(ctx, `UPDATE proxies SET name=$1,ip_type=$2,protocol=$3,host=$4,port=$5,username=NULLIF($6,''),password=CASE WHEN $7='' THEN password ELSE $7 END,updated_at=NOW() WHERE id=$8 AND owner_user_id=$9 AND deleted_at IS NULL`, in.Name, in.IPType, in.Protocol, in.Host, in.Port, in.Username, in.Password, in.ID, userID)
		if err != nil {
			return nil, err
		}
		affected, _ := res.RowsAffected()
		if affected == 0 {
			return nil, ErrCommunityNotFound
		}
		id = in.ID
	}
	items, err := s.ListProxies(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].ID == id {
			return &items[i], nil
		}
	}
	return nil, ErrCommunityNotFound
}

func (s *CommunityService) DeleteProxy(ctx context.Context, userID, id int64) error {
	res, err := s.db.ExecContext(ctx, `UPDATE proxies p SET deleted_at=NOW(),status='disabled',updated_at=NOW() WHERE p.id=$1 AND p.owner_user_id=$2 AND p.deleted_at IS NULL AND NOT EXISTS(SELECT 1 FROM community_accounts a WHERE a.proxy_id=p.id AND a.deleted_at IS NULL)`, id, userID)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ErrCommunityConflict
	}
	return nil
}

func (s *CommunityService) CanUseProxy(ctx context.Context, userID int64, proxyID *int64) error {
	if proxyID == nil {
		return nil
	}
	var exists int
	if err := s.db.QueryRowContext(ctx, `SELECT 1 FROM proxies WHERE id=$1 AND owner_user_id=$2 AND status='active' AND deleted_at IS NULL`, *proxyID, userID).Scan(&exists); errors.Is(err, sql.ErrNoRows) {
		return ErrCommunityForbidden
	} else {
		return err
	}
}

func normalizeAccountInput(in *CommunityAccountInput) error {
	in.Name = strings.TrimSpace(in.Name)
	in.Provider = strings.ToLower(strings.TrimSpace(in.Provider))
	if in.Name == "" || (in.Provider != "openai" && in.Provider != "anthropic") || strings.TrimSpace(in.OAuthCredential) == "" {
		return ErrCommunityInvalid
	}
	if in.Capacity <= 0 {
		in.Capacity = 1
	}
	if in.Concurrency <= 0 {
		in.Concurrency = 1
	}
	if in.Capacity > 1000 || in.Concurrency > 1000 {
		return ErrCommunityInvalid
	}
	if in.ShareMode == "" {
		in.ShareMode = "private"
	}
	if in.ShareMode != "private" && in.ShareMode != "public" {
		return ErrCommunityInvalid
	}
	in.AccountTier = strings.ToLower(strings.TrimSpace(in.AccountTier))
	if in.Provider == "openai" {
		switch in.AccountTier {
		case "free", "plus", "pro", "team", "k12":
		default:
			return ErrCommunityInvalid
		}
		if in.AccountTier == "pro" && in.ProxyID == nil {
			return ErrCommunityInvalid
		}
	} else if in.AccountTier == "" {
		in.AccountTier = "oauth"
	}
	if in.Provider == "anthropic" && in.ProxyID == nil {
		return ErrCommunityInvalid
	}
	if in.Tags == nil {
		in.Tags = []string{}
	}
	if in.SupportedModels == nil {
		in.SupportedModels = []string{}
	}
	return nil
}

func (s *CommunityService) CreateAccount(ctx context.Context, userID int64, in CommunityAccountInput) (*CommunityAccount, error) {
	if err := normalizeAccountInput(&in); err != nil {
		return nil, err
	}
	if err := s.CanUseProxy(ctx, userID, in.ProxyID); err != nil {
		return nil, err
	}
	credential, err := s.encryptor.Encrypt(strings.TrimSpace(in.OAuthCredential))
	if err != nil {
		return nil, fmt.Errorf("encrypt oauth credential: %w", err)
	}
	options, err := json.Marshal(in.ProviderOptions)
	if err != nil {
		return nil, ErrCommunityInvalid
	}
	row := s.db.QueryRowContext(ctx, `INSERT INTO community_accounts
		(owner_user_id,name,provider,credential_encrypted,share_mode,account_tier,capacity,concurrency,tags,supported_models,expires_at,provider_options,notes,proxy_id)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING `+communityAccountSelect,
		userID, in.Name, in.Provider, credential, in.ShareMode, in.AccountTier, in.Capacity, in.Concurrency, pq.Array(in.Tags), pq.Array(in.SupportedModels), in.ExpiresAt, string(options), strings.TrimSpace(in.Notes), in.ProxyID)
	return scanCommunityAccount(row)
}

func normalizeImportedCredential(provider, tier string, value any) (string, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	tier = strings.ToLower(strings.TrimSpace(tier))
	if provider != "openai" && provider != "anthropic" {
		return "", ErrCommunityInvalid
	}
	if provider == "openai" && tier == "pro" {
		return "", ErrCommunityInvalid
	}

	if token, ok := value.(string); ok {
		token = strings.TrimSpace(token)
		lower := strings.ToLower(token)
		if len(token) < 12 || strings.ContainsAny(token, " \t\r\n") || strings.Contains(lower, "://") || (provider == "openai" && strings.HasPrefix(lower, "sk-")) {
			return "", ErrCommunityInvalid
		}
		field := "refresh_token"
		if provider == "anthropic" {
			field = "session_key"
		}
		payload, _ := json.Marshal(map[string]string{field: token})
		return string(payload), nil
	}

	object, ok := value.(map[string]any)
	if !ok || len(object) == 0 || importedCredentialHasForbiddenData(object) {
		return "", ErrCommunityInvalid
	}
	if provider == "openai" {
		if _, ok = nonEmptyImportedString(object, "refresh_token"); !ok {
			return "", ErrCommunityInvalid
		}
	} else {
		_, hasSession := nonEmptyImportedString(object, "session_key")
		_, hasRefresh := nonEmptyImportedString(object, "refresh_token")
		if !hasSession && !hasRefresh {
			return "", ErrCommunityInvalid
		}
	}
	payload, err := json.Marshal(object)
	if err != nil {
		return "", ErrCommunityInvalid
	}
	return string(payload), nil
}

func nonEmptyImportedString(object map[string]any, key string) (string, bool) {
	value, ok := object[key].(string)
	value = strings.TrimSpace(value)
	return value, ok && value != ""
}

func importedCredentialHasForbiddenData(value any) bool {
	switch current := value.(type) {
	case map[string]any:
		for key, child := range current {
			lower := strings.ToLower(strings.TrimSpace(key))
			if strings.Contains(lower, "api_key") || strings.Contains(lower, "apikey") || strings.Contains(lower, "cookie") || strings.Contains(lower, "upstream") || strings.Contains(lower, "base_url") || strings.Contains(lower, "baseurl") {
				return true
			}
			if importedCredentialHasForbiddenData(child) {
				return true
			}
		}
	case []any:
		for _, child := range current {
			if importedCredentialHasForbiddenData(child) {
				return true
			}
		}
	case string:
		lower := strings.ToLower(strings.TrimSpace(current))
		return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
	}
	return false
}

func (s *CommunityService) ImportAccounts(ctx context.Context, userID int64, in CommunityAccountImportInput) ([]CommunityAccount, error) {
	in.Provider = strings.ToLower(strings.TrimSpace(in.Provider))
	in.AccountTier = strings.ToLower(strings.TrimSpace(in.AccountTier))
	if in.ShareMode == "" {
		in.ShareMode = "private"
	}
	if len(in.Items) == 0 || len(in.Items) > 100 || (in.Provider != "openai" && in.Provider != "anthropic") || (in.ShareMode != "private" && in.ShareMode != "public") {
		return nil, ErrCommunityInvalid
	}
	if in.Concurrency <= 0 {
		in.Concurrency = 1
	}
	if in.Concurrency > 1000 || (in.Provider == "openai" && in.AccountTier == "pro") || (in.Provider == "anthropic" && in.ProxyID == nil) {
		return nil, ErrCommunityInvalid
	}
	if in.Provider == "openai" {
		switch in.AccountTier {
		case "free", "plus", "team", "k12":
		default:
			return nil, ErrCommunityInvalid
		}
	}
	if err := s.CanUseProxy(ctx, userID, in.ProxyID); err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	items := make([]CommunityAccount, 0, len(in.Items))
	for index, item := range in.Items {
		credential, normalizeErr := normalizeImportedCredential(in.Provider, in.AccountTier, item.Credential)
		if normalizeErr != nil {
			return nil, normalizeErr
		}
		encrypted, encryptErr := s.encryptor.Encrypt(credential)
		if encryptErr != nil {
			return nil, fmt.Errorf("encrypt imported oauth credential: %w", encryptErr)
		}
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = fmt.Sprintf("%s OAuth %d", strings.ToUpper(in.Provider[:1])+in.Provider[1:], index+1)
		}
		row := tx.QueryRowContext(ctx, `INSERT INTO community_accounts
			(owner_user_id,name,provider,credential_encrypted,share_mode,account_tier,capacity,concurrency,provider_options,notes,proxy_id)
			VALUES($1,$2,$3,$4,$5,$6,$7,$7,'{}','批量导入',$8) RETURNING `+communityAccountSelect,
			userID, name, in.Provider, encrypted, in.ShareMode, in.AccountTier, in.Concurrency, in.ProxyID)
		created, scanErr := scanCommunityAccount(row)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, *created)
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *CommunityService) ExportAccounts(ctx context.Context, userID int64, ids []int64) ([]CommunityAccountExportItem, error) {
	if len(ids) > 1000 {
		return nil, ErrCommunityInvalid
	}
	args := []any{userID}
	where := "owner_user_id=$1 AND deleted_at IS NULL"
	if len(ids) > 0 {
		where += " AND id=ANY($2)"
		args = append(args, pq.Array(ids))
	}
	rows, err := s.db.QueryContext(ctx, `SELECT name,provider,account_tier,share_mode,concurrency,tags,supported_models,provider_options,notes,proxy_id,credential_encrypted FROM community_accounts WHERE `+where+` ORDER BY created_at`, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]CommunityAccountExportItem, 0)
	for rows.Next() {
		var item CommunityAccountExportItem
		var optionsJSON []byte
		var encrypted string
		if err = rows.Scan(&item.Name, &item.Provider, &item.AccountTier, &item.ShareMode, &item.Concurrency, pq.Array(&item.Tags), pq.Array(&item.SupportedModels), &optionsJSON, &item.Notes, &item.ProxyID, &encrypted); err != nil {
			return nil, err
		}
		plain, decryptErr := s.encryptor.Decrypt(encrypted)
		if decryptErr != nil {
			return nil, decryptErr
		}
		if err = json.Unmarshal(optionsJSON, &item.ProviderOptions); err != nil {
			item.ProviderOptions = map[string]any{}
		}
		if err = json.Unmarshal([]byte(plain), &item.Credential); err != nil {
			item.Credential = plain
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *CommunityService) BatchUpdateAccounts(ctx context.Context, userID int64, in CommunityAccountBatchUpdateInput) (int64, error) {
	if len(in.IDs) == 0 || len(in.IDs) > 1000 {
		return 0, ErrCommunityInvalid
	}
	if in.ShareMode != nil && *in.ShareMode != "private" && *in.ShareMode != "public" {
		return 0, ErrCommunityInvalid
	}
	if in.Concurrency != nil && (*in.Concurrency < 1 || *in.Concurrency > 1000) {
		return 0, ErrCommunityInvalid
	}
	if in.Priority != nil && (*in.Priority < 0 || *in.Priority > 1000) {
		return 0, ErrCommunityInvalid
	}
	if in.ProxyID != nil {
		if err := s.CanUseProxy(ctx, userID, in.ProxyID); err != nil {
			return 0, err
		}
	}
	sets := make([]string, 0, 8)
	args := make([]any, 0, 10)
	add := func(column string, value any) {
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s=$%d", column, len(args)))
	}
	if in.ShareMode != nil {
		add("share_mode", *in.ShareMode)
	}
	if in.Concurrency != nil {
		add("concurrency", *in.Concurrency)
		add("capacity", *in.Concurrency)
	}
	if in.Priority != nil {
		add("priority", *in.Priority)
	}
	if in.Schedulable != nil {
		add("schedulable", *in.Schedulable)
	}
	if in.ProxyID != nil {
		add("proxy_id", *in.ProxyID)
	} else if in.ClearProxy {
		sets = append(sets, "proxy_id=NULL")
	}
	if in.Notes != nil {
		add("notes", strings.TrimSpace(*in.Notes))
	}
	if in.ExpiresAt != nil {
		add("expires_at", *in.ExpiresAt)
	} else if in.ClearExpires {
		sets = append(sets, "expires_at=NULL")
	}
	if len(sets) == 0 {
		return 0, ErrCommunityInvalid
	}
	sets = append(sets, "updated_at=NOW()")
	args = append(args, userID, pq.Array(in.IDs))
	query := fmt.Sprintf("UPDATE community_accounts SET %s WHERE owner_user_id=$%d AND id=ANY($%d) AND deleted_at IS NULL", strings.Join(sets, ","), len(args)-1, len(args))
	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

type rowScanner interface{ Scan(...any) error }

func scanCommunityAccount(row rowScanner) (*CommunityAccount, error) {
	var out CommunityAccount
	err := row.Scan(&out.ID, &out.OwnerUserID, &out.Name, &out.Provider, &out.Status, &out.ReviewStatus, &out.ReviewNote, &out.ShareMode, &out.AccountTier, &out.Capacity, &out.Concurrency, &out.Schedulable, &out.Priority, &out.TodayRequests, &out.TodayTokens, &out.Usage5hPercent, &out.Usage7dPercent, pq.Array(&out.Tags), pq.Array(&out.SupportedModels), &out.ExpiresAt, &out.LastUsedAt, &out.Notes, &out.CreatedAt, &out.ProxyID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCommunityNotFound
	}
	if err == nil {
		out.GroupName = map[string]string{"openai": "OpenAI账号模式", "anthropic": "Anthropic账号模式"}[out.Provider]
	}
	return &out, err
}

func communitySchedulerConfig(provider string, credentials map[string]any, options map[string]any, communityAccountID int64) (map[string]any, map[string]any) {
	credentialCopy := make(map[string]any, len(credentials)+1)
	for key, value := range credentials {
		credentialCopy[key] = value
	}
	extra := map[string]any{"community_account_id": communityAccountID, "community_owner_managed": true}
	if mapping, ok := options["model_mapping"].(map[string]any); ok && len(mapping) > 0 {
		credentialCopy["model_mapping"] = mapping
	}
	if provider != "anthropic" {
		return credentialCopy, extra
	}
	if value, ok := options["intercept_warmup"].(bool); ok {
		credentialCopy["intercept_warmup_requests"] = value
	}
	if value, ok := options["temp_unschedulable_enabled"].(bool); ok {
		credentialCopy["temp_unschedulable_enabled"] = value
	}
	if value, ok := options["temp_unschedulable_rules"].([]any); ok {
		credentialCopy["temp_unschedulable_rules"] = value
	}
	optionMap := map[string]string{
		"session_limit":                "max_sessions",
		"session_idle_timeout_minutes": "session_idle_timeout_minutes",
		"rpm_limit":                    "base_rpm",
		"rpm_strategy":                 "rpm_strategy",
		"rpm_sticky_buffer":            "rpm_sticky_buffer",
		"tls_fingerprint":              "enable_tls_fingerprint",
		"session_affinity":             "session_id_masking_enabled",
		"cache_ttl_override":           "cache_ttl_override_enabled",
		"user_msg_queue_mode":          "user_msg_queue_mode",
		"window_cost_limit":            "window_cost_limit",
		"window_cost_sticky_reserve":   "window_cost_sticky_reserve",
	}
	for source, target := range optionMap {
		if value, exists := options[source]; exists {
			extra[target] = value
		}
	}
	if enabled, _ := options["cache_ttl_override"].(bool); enabled {
		target, _ := options["cache_ttl_target"].(string)
		if target != "1h" {
			target = "5m"
		}
		extra["cache_ttl_override_target"] = target
	}
	return credentialCopy, extra
}

const communityAccountSelect = `id,owner_user_id,name,provider,status,review_status,review_note,share_mode,account_tier,capacity,concurrency,schedulable,priority,today_requests,today_tokens,usage_5h_percent,usage_7d_percent,tags,supported_models,expires_at,last_used_at,notes,created_at,proxy_id`

func (s *CommunityService) ListAccounts(ctx context.Context, userID int64) ([]CommunityAccount, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+communityAccountSelect+` FROM community_accounts WHERE owner_user_id=$1 AND deleted_at IS NULL ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]CommunityAccount, 0)
	for rows.Next() {
		item, err := scanCommunityAccount(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (s *CommunityService) ListAdminAccounts(ctx context.Context) ([]CommunityAccount, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+communityAccountSelect+` FROM community_accounts WHERE deleted_at IS NULL ORDER BY CASE review_status WHEN 'pending' THEN 0 WHEN 'rejected' THEN 1 ELSE 2 END,created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]CommunityAccount, 0)
	for rows.Next() {
		item, scanErr := scanCommunityAccount(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func decodeCommunityCredential(raw string) (map[string]any, error) {
	credentials := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &credentials); err == nil && len(credentials) > 0 {
		return credentials, nil
	}
	if strings.TrimSpace(raw) == "" {
		return nil, ErrCommunityInvalid
	}
	return map[string]any{"refresh_token": strings.TrimSpace(raw)}, nil
}

func (s *CommunityService) ReviewAccount(ctx context.Context, adminID, id int64, decision, note string) error {
	if decision != "approved" && decision != "rejected" {
		return ErrCommunityInvalid
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var name, provider, encrypted, notes string
	var concurrency, priority int
	var expiresAt *time.Time
	var proxyID *int64
	var schedulerAccountID *int64
	var optionsJSON []byte
	err = tx.QueryRowContext(ctx, `SELECT name,provider,credential_encrypted,concurrency,priority,expires_at,scheduler_account_id,proxy_id,provider_options,notes FROM community_accounts WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, id).Scan(&name, &provider, &encrypted, &concurrency, &priority, &expiresAt, &schedulerAccountID, &proxyID, &optionsJSON, &notes)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrCommunityNotFound
	}
	if err != nil {
		return err
	}
	if decision == "rejected" {
		if schedulerAccountID != nil {
			if _, err = tx.ExecContext(ctx, `UPDATE accounts SET status='disabled',schedulable=FALSE,updated_at=NOW() WHERE id=$1`, *schedulerAccountID); err != nil {
				return err
			}
			if _, err = tx.ExecContext(ctx, `INSERT INTO scheduler_outbox(event_type,account_id) VALUES('account_changed',$1)`, *schedulerAccountID); err != nil {
				return err
			}
		}
		if _, err = tx.ExecContext(ctx, `UPDATE community_accounts SET review_status='rejected',review_note=$2,reviewed_by=$3,reviewed_at=NOW(),status='rejected',schedulable=FALSE,updated_at=NOW() WHERE id=$1`, id, strings.TrimSpace(note), adminID); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE community_listings SET status='rejected',updated_at=NOW() WHERE account_id=$1`, id); err != nil {
			return err
		}
		return tx.Commit()
	}
	if schedulerAccountID == nil {
		plain, decryptErr := s.encryptor.Decrypt(encrypted)
		if decryptErr != nil {
			return fmt.Errorf("decrypt oauth credential: %w", decryptErr)
		}
		credentials, decodeErr := decodeCommunityCredential(plain)
		if decodeErr != nil {
			return decodeErr
		}
		providerOptions := make(map[string]any)
		if len(optionsJSON) > 0 {
			if decodeErr = json.Unmarshal(optionsJSON, &providerOptions); decodeErr != nil {
				return fmt.Errorf("decode provider options: %w", decodeErr)
			}
		}
		credentials, extra := communitySchedulerConfig(provider, credentials, providerOptions, id)
		credentialsJSON, marshalErr := json.Marshal(credentials)
		if marshalErr != nil {
			return marshalErr
		}
		extraJSON, marshalErr := json.Marshal(extra)
		if marshalErr != nil {
			return marshalErr
		}
		var accountID int64
		err = tx.QueryRowContext(ctx, `INSERT INTO accounts(name,notes,platform,type,credentials,extra,concurrency,priority,status,schedulable,expires_at,proxy_id) VALUES($1,$2,$3,'oauth',$4::jsonb,$5::jsonb,$6,$7,'active',TRUE,$8,$9) RETURNING id`, "[共享] "+name, notes, provider, string(credentialsJSON), string(extraJSON), concurrency, priority, expiresAt, proxyID).Scan(&accountID)
		if err != nil {
			return err
		}
		schedulerAccountID = &accountID
	}
	if _, err = tx.ExecContext(ctx, `UPDATE community_accounts SET scheduler_account_id=$2,review_status='approved',review_note=$3,reviewed_by=$4,reviewed_at=NOW(),status='active',schedulable=TRUE,updated_at=NOW() WHERE id=$1`, id, *schedulerAccountID, strings.TrimSpace(note), adminID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO scheduler_outbox(event_type,account_id) VALUES('account_changed',$1)`, *schedulerAccountID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *CommunityService) DeleteAccount(ctx context.Context, userID, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var schedulerAccountID *int64
	if err = tx.QueryRowContext(ctx, `SELECT scheduler_account_id FROM community_accounts WHERE id=$1 AND owner_user_id=$2 AND deleted_at IS NULL FOR UPDATE`, id, userID).Scan(&schedulerAccountID); errors.Is(err, sql.ErrNoRows) {
		return ErrCommunityNotFound
	} else if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `UPDATE community_accounts SET deleted_at=NOW(),status='paused',schedulable=FALSE,updated_at=NOW() WHERE id=$1 AND owner_user_id=$2 AND deleted_at IS NULL`, id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrCommunityNotFound
	}
	if _, err = tx.ExecContext(ctx, `UPDATE community_listings SET status='paused',updated_at=NOW() WHERE account_id=$1 AND owner_user_id=$2`, id, userID); err != nil {
		return err
	}
	if schedulerAccountID != nil {
		if _, err = tx.ExecContext(ctx, `UPDATE accounts SET status='disabled',schedulable=FALSE,updated_at=NOW() WHERE id=$1`, *schedulerAccountID); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO scheduler_outbox(event_type,account_id) VALUES('account_changed',$1)`, *schedulerAccountID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *CommunityService) commissionRate(ctx context.Context) float64 {
	var raw string
	if err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key='community_marketplace_commission_percent'`).Scan(&raw); err == nil {
		var v float64
		if _, err = fmt.Sscan(raw, &v); err == nil && v >= 0 && v <= 100 {
			return v
		}
	}
	return 10
}

func (s *CommunityService) CreateListing(ctx context.Context, userID int64, in CommunityListingInput) (*CommunityListing, error) {
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" || in.AccountID <= 0 {
		return nil, ErrCommunityInvalid
	}
	if in.SeatLimit <= 0 {
		in.SeatLimit = 1
	}
	if in.PerUserConcurrency <= 0 {
		in.PerUserConcurrency = 1
	}
	if in.UsageMultiplier < 0 || in.MinimumBalance < 0 || in.HourlyPrice < 0 || in.HourlyMinimumSpend < 0 {
		return nil, ErrCommunityInvalid
	}
	if in.IdleTimeoutMinutes <= 0 {
		in.IdleTimeoutMinutes = 30
	}
	status := "draft"
	share := "private"
	if in.Publish {
		status = "published"
		share = "public"
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	var exists int
	if err = tx.QueryRowContext(ctx, `SELECT 1 FROM community_accounts WHERE id=$1 AND owner_user_id=$2 AND review_status='approved' AND status='active' AND deleted_at IS NULL`, in.AccountID, userID).Scan(&exists); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCommunityForbidden
	} else if err != nil {
		return nil, err
	}
	rate := s.commissionRate(ctx)
	var id int64
	err = tx.QueryRowContext(ctx, `INSERT INTO community_listings(account_id,owner_user_id,title,description,seat_limit,per_user_concurrency,minimum_balance,hourly_price,hourly_minimum_spend,usage_multiplier,idle_timeout_minutes,commission_rate,status) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) ON CONFLICT(account_id) DO UPDATE SET title=EXCLUDED.title,description=EXCLUDED.description,seat_limit=EXCLUDED.seat_limit,per_user_concurrency=EXCLUDED.per_user_concurrency,minimum_balance=EXCLUDED.minimum_balance,hourly_price=EXCLUDED.hourly_price,hourly_minimum_spend=EXCLUDED.hourly_minimum_spend,usage_multiplier=EXCLUDED.usage_multiplier,idle_timeout_minutes=EXCLUDED.idle_timeout_minutes,commission_rate=EXCLUDED.commission_rate,status=EXCLUDED.status,updated_at=NOW() RETURNING id`, in.AccountID, userID, in.Title, strings.TrimSpace(in.Description), in.SeatLimit, in.PerUserConcurrency, in.MinimumBalance, in.HourlyPrice, in.HourlyMinimumSpend, in.UsageMultiplier, in.IdleTimeoutMinutes, rate, status).Scan(&id)
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE community_accounts SET share_mode=$1,updated_at=NOW() WHERE id=$2`, share, in.AccountID); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetListing(ctx, id)
}

// UpdateListing edits an existing owner's terms without exposing account ownership.
func (s *CommunityService) UpdateListing(ctx context.Context, userID, listingID int64, in CommunityListingInput) (*CommunityListing, error) {
	var accountID int64
	if err := s.db.QueryRowContext(ctx, `SELECT account_id FROM community_listings WHERE id=$1 AND owner_user_id=$2`, listingID, userID).Scan(&accountID); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCommunityForbidden
	} else if err != nil {
		return nil, err
	}
	in.AccountID = accountID
	return s.CreateListing(ctx, userID, in)
}

func (s *CommunityService) GetListing(ctx context.Context, id int64) (*CommunityListing, error) {
	row := s.db.QueryRowContext(ctx, listingSelect+` WHERE l.id=$1`, id)
	return scanListing(row)
}

const listingSelect = `SELECT l.id,l.account_id,l.owner_user_id,COALESCE(NULLIF(u.username,''),u.email),l.title,l.description,a.provider,a.account_tier,a.tags,a.supported_models,l.seat_limit,(SELECT COUNT(*) FROM community_memberships m WHERE m.listing_id=l.id AND m.status='active'),l.per_user_concurrency,l.minimum_balance,l.hourly_price,l.hourly_minimum_spend,l.usage_multiplier,a.usage_5h_percent,a.usage_7d_percent,l.idle_timeout_minutes,l.commission_rate,l.status,COALESCE((SELECT AVG(r.score) FROM community_reviews r WHERE r.listing_id=l.id),l.score)::double precision,(SELECT COUNT(*) FROM community_reviews r WHERE r.listing_id=l.id),l.updated_at FROM community_listings l JOIN community_accounts a ON a.id=l.account_id JOIN users u ON u.id=l.owner_user_id`

func scanListing(row rowScanner) (*CommunityListing, error) {
	var v CommunityListing
	err := row.Scan(&v.ID, &v.AccountID, &v.OwnerUserID, &v.OwnerName, &v.Title, &v.Description, &v.Provider, &v.AccountTier, pq.Array(&v.Tags), pq.Array(&v.SupportedModels), &v.SeatLimit, &v.SeatsUsed, &v.PerUserConcurrency, &v.MinimumBalance, &v.HourlyPrice, &v.HourlyMinimumSpend, &v.UsageMultiplier, &v.Usage5hPercent, &v.Usage7dPercent, &v.IdleTimeoutMinutes, &v.CommissionRate, &v.Status, &v.Score, &v.RatingCount, &v.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCommunityNotFound
	}
	return &v, err
}

func (s *CommunityService) ListMarketplace(ctx context.Context, provider, search string) ([]CommunityListing, error) {
	q := listingSelect + ` WHERE l.status='published' AND a.share_mode='public' AND a.review_status='approved' AND a.status='active' AND a.schedulable=TRUE AND a.deleted_at IS NULL`
	args := []any{}
	if provider != "" {
		args = append(args, provider)
		q += fmt.Sprintf(" AND a.provider=$%d", len(args))
	}
	if search != "" {
		args = append(args, "%"+search+"%")
		q += fmt.Sprintf(" AND (l.title ILIKE $%d OR l.description ILIKE $%d OR COALESCE(NULLIF(u.username,''),u.email) ILIKE $%d OR $%d=ANY(a.supported_models))", len(args), len(args), len(args), len(args))
	}
	q += ` ORDER BY l.score DESC,l.updated_at DESC`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []CommunityListing{}
	for rows.Next() {
		v, err := scanListing(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

func (s *CommunityService) ListOwnerListings(ctx context.Context, userID int64, provider string) ([]CommunityListing, error) {
	q := listingSelect + ` WHERE l.owner_user_id=$1 AND a.deleted_at IS NULL`
	args := []any{userID}
	if provider != "" {
		args = append(args, provider)
		q += ` AND a.provider=$2`
	}
	q += ` ORDER BY l.updated_at DESC`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]CommunityListing, 0)
	for rows.Next() {
		item, scanErr := scanListing(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (s *CommunityService) JoinListing(ctx context.Context, userID, listingID, apiKeyID int64, idleTimeoutMinutes int) (int64, error) {
	if apiKeyID <= 0 {
		return 0, ErrCommunityInvalid
	}
	if idleTimeoutMinutes == 0 {
		idleTimeoutMinutes = 10
	}
	if idleTimeoutMinutes < 1 || idleTimeoutMinutes > 1440 {
		return 0, ErrCommunityInvalid
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	var owner int64
	var provider string
	var minimumBalance float64
	err = tx.QueryRowContext(ctx, `SELECT l.owner_user_id,a.provider,l.minimum_balance FROM community_listings l JOIN community_accounts a ON a.id=l.account_id WHERE l.id=$1 AND l.status='published' FOR UPDATE`, listingID).Scan(&owner, &provider, &minimumBalance)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrCommunityNotFound
	}
	if err != nil {
		return 0, err
	}
	var balance float64
	var keyMode *string
	if err = tx.QueryRowContext(ctx, `SELECT u.balance,k.account_mode_platform FROM api_keys k JOIN users u ON u.id=k.user_id WHERE k.id=$1 AND k.user_id=$2 AND k.status='active' AND k.deleted_at IS NULL FOR UPDATE OF k,u`, apiKeyID, userID).Scan(&balance, &keyMode); err != nil {
		return 0, ErrCommunityForbidden
	}
	if keyMode == nil || *keyMode != provider {
		if keyMode != nil {
			return 0, ErrCommunityForbidden
		}
		if _, err = tx.ExecContext(ctx, `UPDATE api_keys SET account_mode_platform=$1,updated_at=NOW() WHERE id=$2 AND account_mode_platform IS NULL`, provider, apiKeyID); err != nil {
			return 0, err
		}
	}
	if owner != userID && balance < minimumBalance {
		return 0, ErrCommunityInsufficient
	}
	var reservationOrder int
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE(MIN(slot),0) FROM generate_series(1,5) AS slot WHERE NOT EXISTS (SELECT 1 FROM community_memberships WHERE api_key_id=$1 AND reservation_order=slot AND status IN ('reserved','active'))`, apiKeyID).Scan(&reservationOrder); err != nil {
		return 0, err
	}
	if reservationOrder == 0 {
		return 0, ErrCommunityConflict
	}
	var id int64
	err = tx.QueryRowContext(ctx, `INSERT INTO community_memberships(listing_id,member_user_id,api_key_id,status,reservation_order,idle_timeout_minutes) VALUES($1,$2,$3,'reserved',$4,$5) ON CONFLICT(listing_id,api_key_id) WHERE status IN ('reserved','active') AND api_key_id IS NOT NULL DO UPDATE SET status='reserved',reservation_order=EXCLUDED.reservation_order,idle_timeout_minutes=EXCLUDED.idle_timeout_minutes,activated_at=NULL,last_request_at=NULL,idle_expires_at=NULL,ended_at=NULL RETURNING id`, listingID, userID, apiKeyID, reservationOrder, idleTimeoutMinutes).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

func (s *CommunityService) LeaveListing(ctx context.Context, userID, listingID int64) error {
	res, err := s.db.ExecContext(ctx, `UPDATE community_memberships SET status='left',ended_at=NOW() WHERE listing_id=$1 AND member_user_id=$2 AND status IN ('reserved','active')`, listingID, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrCommunityNotFound
	}
	return nil
}

func (s *CommunityService) ReviewMembership(ctx context.Context, userID, membershipID int64, score float64, content string) error {
	if membershipID <= 0 || score < 1 || score > 10 {
		return ErrCommunityInvalid
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO community_reviews(listing_id,membership_id,reviewer_user_id,owner_user_id,score,content)
		SELECT m.listing_id,m.id,m.member_user_id,l.owner_user_id,$3,$4
		FROM community_memberships m JOIN community_listings l ON l.id=m.listing_id
		WHERE m.id=$1 AND m.member_user_id=$2 AND m.activated_at IS NOT NULL AND l.owner_user_id<>$2
		ON CONFLICT(membership_id,reviewer_user_id) DO UPDATE SET score=EXCLUDED.score,content=EXCLUDED.content,updated_at=NOW()`, membershipID, userID, score, strings.TrimSpace(content))
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrCommunityForbidden
	}
	return nil
}

type CommunityMembership struct {
	ID                 int64      `json:"id"`
	ListingID          int64      `json:"listing_id"`
	APIKeyID           *int64     `json:"api_key_id,omitempty"`
	ReservationOrder   int        `json:"reservation_order"`
	Status             string     `json:"status"`
	IdleTimeoutMinutes int        `json:"idle_timeout_minutes"`
	JoinedAt           time.Time  `json:"joined_at"`
	ActivatedAt        *time.Time `json:"activated_at,omitempty"`
	LastUsedAt         *time.Time `json:"last_used_at,omitempty"`
	LastRequestAt      *time.Time `json:"last_request_at,omitempty"`
	IdleExpiresAt      *time.Time `json:"idle_expires_at,omitempty"`
}

type CommunityConsumptionAccount struct {
	ListingID     int64      `json:"listing_id"`
	MembershipID  int64      `json:"membership_id"`
	Title         string     `json:"title"`
	Provider      string     `json:"provider"`
	Status        string     `json:"status"`
	ActivatedAt   *time.Time `json:"activated_at,omitempty"`
	LastRequestAt *time.Time `json:"last_request_at,omitempty"`
}

type CommunityConsumptionSummary struct {
	Scope            string  `json:"scope"`
	RequestSpend     float64 `json:"request_spend"`
	HourlyPrecharged float64 `json:"hourly_precharged"`
	HourlyRefunded   float64 `json:"hourly_refunded"`
	Total            float64 `json:"total"`
}

type CommunitySelectionInput struct {
	Provider            string  `json:"provider"`
	APIKeyID            int64   `json:"api_key_id"`
	Model               string  `json:"model"`
	RequestCount        int     `json:"request_count"`
	Hours               float64 `json:"hours"`
	InputTokens         int     `json:"input_tokens"`
	OutputTokens        int     `json:"output_tokens"`
	CacheCreationTokens int     `json:"cache_creation_tokens"`
	CacheReadTokens     int     `json:"cache_read_tokens"`
}

type CommunitySelectionResult struct {
	Listing          CommunityListing `json:"listing"`
	BaseRequestCost  float64          `json:"base_request_cost"`
	RequestSpend     float64          `json:"request_spend"`
	HourlyFee        float64          `json:"hourly_fee"`
	HourlyFeeWaived  bool             `json:"hourly_fee_waived"`
	RequiredSpend    float64          `json:"required_spend"`
	EstimatedTotal   float64          `json:"estimated_total"`
	EstimatedPerHour float64          `json:"estimated_per_hour"`
	RemainingSeats   int              `json:"remaining_seats"`
}

type CommunityRecentUsageAverage struct {
	RequestCount        int64   `json:"request_count"`
	InputTokens         float64 `json:"input_tokens"`
	OutputTokens        float64 `json:"output_tokens"`
	CacheCreationTokens float64 `json:"cache_creation_tokens"`
	CacheReadTokens     float64 `json:"cache_read_tokens"`
}

type CommunityOwnerSummary struct {
	PublishedListings int     `json:"published_listings"`
	ActiveMembers     int     `json:"active_members"`
	GrossRevenue      float64 `json:"gross_revenue"`
	PlatformFees      float64 `json:"platform_fees"`
	NetRevenue        float64 `json:"net_revenue"`
}

type CommunityAccountModeKey struct {
	ID       int64   `json:"id"`
	Name     string  `json:"name"`
	Status   string  `json:"status"`
	Platform *string `json:"account_mode_platform,omitempty"`
}

func (s *CommunityService) ListAccountModeKeys(ctx context.Context, userID int64, provider string) ([]CommunityAccountModeKey, error) {
	if provider != "openai" && provider != "anthropic" {
		return nil, ErrCommunityInvalid
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,status,account_mode_platform FROM api_keys WHERE user_id=$1 AND status='active' AND deleted_at IS NULL AND (account_mode_platform=$2 OR account_mode_platform IS NULL) ORDER BY account_mode_platform NULLS LAST,created_at DESC`, userID, provider)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]CommunityAccountModeKey, 0)
	for rows.Next() {
		var item CommunityAccountModeKey
		if err = rows.Scan(&item.ID, &item.Name, &item.Status, &item.Platform); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *CommunityService) SetAccountModeKey(ctx context.Context, userID, keyID int64, provider string) error {
	if provider != "openai" && provider != "anthropic" {
		return ErrCommunityInvalid
	}
	res, err := s.db.ExecContext(ctx, `UPDATE api_keys SET account_mode_platform=$1,updated_at=NOW() WHERE id=$2 AND user_id=$3 AND status='active' AND deleted_at IS NULL AND (account_mode_platform IS NULL OR account_mode_platform=$1)`, provider, keyID, userID)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ErrCommunityConflict
	}
	return nil
}

func (s *CommunityService) ListMemberships(ctx context.Context, userID int64) ([]CommunityMembership, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,listing_id,api_key_id,reservation_order,status,idle_timeout_minutes,joined_at,activated_at,last_used_at,last_request_at,idle_expires_at FROM community_memberships WHERE member_user_id=$1 ORDER BY api_key_id NULLS LAST,reservation_order,joined_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]CommunityMembership, 0)
	for rows.Next() {
		var item CommunityMembership
		if err = rows.Scan(&item.ID, &item.ListingID, &item.APIKeyID, &item.ReservationOrder, &item.Status, &item.IdleTimeoutMinutes, &item.JoinedAt, &item.ActivatedAt, &item.LastUsedAt, &item.LastRequestAt, &item.IdleExpiresAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *CommunityService) ListConsumptionAccounts(ctx context.Context, userID int64) ([]CommunityConsumptionAccount, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT l.id,m.id,l.title,a.provider,m.status,m.activated_at,m.last_request_at
		FROM community_memberships m
		JOIN community_listings l ON l.id=m.listing_id
		JOIN community_accounts a ON a.id=l.account_id
		WHERE m.member_user_id=$1
		ORDER BY COALESCE(m.last_request_at,m.activated_at,m.joined_at) DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]CommunityConsumptionAccount, 0)
	for rows.Next() {
		var item CommunityConsumptionAccount
		if err = rows.Scan(&item.ListingID, &item.MembershipID, &item.Title, &item.Provider, &item.Status, &item.ActivatedAt, &item.LastRequestAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *CommunityService) GetConsumptionSummary(ctx context.Context, userID, membershipID int64, scope string) (*CommunityConsumptionSummary, error) {
	if scope != "session" && scope != "today" && scope != "7d" {
		return nil, ErrCommunityInvalid
	}
	var exists int
	if err := s.db.QueryRowContext(ctx, `SELECT 1 FROM community_memberships WHERE id=$1 AND member_user_id=$2`, membershipID, userID).Scan(&exists); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCommunityNotFound
	} else if err != nil {
		return nil, err
	}
	startClause := ""
	args := []any{membershipID, userID}
	switch scope {
	case "today":
		startClause = " AND created_at>=CURRENT_DATE"
	case "7d":
		startClause = " AND created_at>=NOW()-INTERVAL '7 days'"
	}
	var requestSpend float64
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(gross_amount),0) FROM community_settlements WHERE membership_id=$1 AND payer_user_id=$2 AND settlement_type='request_usage'`+startClause, args...).Scan(&requestSpend); err != nil {
		return nil, err
	}
	windowClause := ""
	switch scope {
	case "today":
		windowClause = " AND created_at>=CURRENT_DATE"
	case "7d":
		windowClause = " AND created_at>=NOW()-INTERVAL '7 days'"
	}
	var precharged, refunded float64
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(hourly_fee_precharged),0),COALESCE(SUM(hourly_fee_refunded),0) FROM community_billing_windows WHERE membership_id=$1 AND payer_user_id=$2`+windowClause, args...).Scan(&precharged, &refunded); err != nil {
		return nil, err
	}
	return &CommunityConsumptionSummary{Scope: scope, RequestSpend: requestSpend, HourlyPrecharged: precharged, HourlyRefunded: refunded, Total: requestSpend + precharged - refunded}, nil
}

func (s *CommunityService) RecommendListings(ctx context.Context, userID int64, in CommunitySelectionInput, baseRequestCost float64) ([]CommunitySelectionResult, error) {
	in.Provider = strings.ToLower(strings.TrimSpace(in.Provider))
	in.Model = strings.TrimSpace(in.Model)
	if (in.Provider != "openai" && in.Provider != "anthropic") || in.APIKeyID <= 0 || in.Model == "" || in.RequestCount < 1 || in.RequestCount > 100000 || in.Hours <= 0 || in.Hours > 720 || baseRequestCost < 0 || in.InputTokens < 0 || in.OutputTokens < 0 || in.CacheCreationTokens < 0 || in.CacheReadTokens < 0 {
		return nil, ErrCommunityInvalid
	}
	var keyMode *string
	if err := s.db.QueryRowContext(ctx, `SELECT account_mode_platform FROM api_keys WHERE id=$1 AND user_id=$2 AND status='active' AND deleted_at IS NULL`, in.APIKeyID, userID).Scan(&keyMode); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCommunityForbidden
	} else if err != nil {
		return nil, err
	}
	if keyMode != nil && *keyMode != in.Provider {
		return nil, ErrCommunityForbidden
	}
	listings, err := s.ListMarketplace(ctx, in.Provider, "")
	if err != nil {
		return nil, err
	}
	results := make([]CommunitySelectionResult, 0, len(listings))
	for _, listing := range listings {
		if len(listing.SupportedModels) > 0 {
			matched := false
			for _, model := range listing.SupportedModels {
				if strings.EqualFold(model, in.Model) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		requestSpend := baseRequestCost * float64(in.RequestCount) * listing.UsageMultiplier
		requiredSpend := listing.HourlyMinimumSpend * in.Hours
		waived := listing.HourlyPrice > 0 && listing.HourlyMinimumSpend > 0 && requestSpend >= requiredSpend
		hourlyFee := listing.HourlyPrice * in.Hours
		if waived || listing.OwnerUserID == userID {
			hourlyFee = 0
		}
		total := requestSpend + hourlyFee
		results = append(results, CommunitySelectionResult{
			Listing: listing, BaseRequestCost: baseRequestCost, RequestSpend: requestSpend,
			HourlyFee: hourlyFee, HourlyFeeWaived: waived, RequiredSpend: requiredSpend,
			EstimatedTotal: total, EstimatedPerHour: total / in.Hours,
			RemainingSeats: int(math.Max(0, float64(listing.SeatLimit-listing.SeatsUsed))),
		})
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].EstimatedPerHour == results[j].EstimatedPerHour {
			return results[i].Listing.Score > results[j].Listing.Score
		}
		return results[i].EstimatedPerHour < results[j].EstimatedPerHour
	})
	return results, nil
}

func (s *CommunityService) GetRecentUsageAverage(ctx context.Context, userID, apiKeyID int64) (*CommunityRecentUsageAverage, error) {
	var exists int
	if err := s.db.QueryRowContext(ctx, `SELECT 1 FROM api_keys WHERE id=$1 AND user_id=$2 AND status='active' AND deleted_at IS NULL`, apiKeyID, userID).Scan(&exists); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCommunityForbidden
	} else if err != nil {
		return nil, err
	}
	var out CommunityRecentUsageAverage
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*),COALESCE(AVG(input_tokens),0),COALESCE(AVG(output_tokens),0),COALESCE(AVG(cache_creation_tokens),0),COALESCE(AVG(cache_read_tokens),0) FROM usage_logs WHERE user_id=$1 AND api_key_id=$2 AND created_at>=NOW()-INTERVAL '3 days'`, userID, apiKeyID).
		Scan(&out.RequestCount, &out.InputTokens, &out.OutputTokens, &out.CacheCreationTokens, &out.CacheReadTokens); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *CommunityService) GetOwnerSummary(ctx context.Context, userID int64) (*CommunityOwnerSummary, error) {
	var out CommunityOwnerSummary
	if err := s.db.QueryRowContext(ctx, `SELECT
		COUNT(*) FILTER (WHERE status='published'),
		COALESCE((SELECT COUNT(*) FROM community_memberships m JOIN community_listings ml ON ml.id=m.listing_id WHERE ml.owner_user_id=$1 AND m.status='active'),0)
		FROM community_listings WHERE owner_user_id=$1`, userID).Scan(&out.PublishedListings, &out.ActiveMembers); err != nil {
		return nil, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(gross_amount),0),COALESCE(SUM(platform_fee),0),COALESCE(SUM(owner_amount),0) FROM community_settlements WHERE owner_user_id=$1`, userID).
		Scan(&out.GrossRevenue, &out.PlatformFees, &out.NetRevenue); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *CommunityService) SetListingStatus(ctx context.Context, userID, listingID int64, status string) error {
	if status != "published" && status != "paused" {
		return ErrCommunityInvalid
	}
	result, err := s.db.ExecContext(ctx, `UPDATE community_listings l SET status=$3,updated_at=NOW() FROM community_accounts a WHERE l.id=$1 AND l.owner_user_id=$2 AND a.id=l.account_id AND a.owner_user_id=$2 AND a.review_status='approved' AND a.status='active' AND a.schedulable=TRUE AND a.share_mode='public' AND a.deleted_at IS NULL`, listingID, userID, status)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrCommunityForbidden
	}
	return nil
}

type Ticket struct {
	ID          int64           `json:"id"`
	UserID      int64           `json:"user_id"`
	Subject     string          `json:"subject"`
	Category    string          `json:"category"`
	Priority    string          `json:"priority"`
	Status      string          `json:"status"`
	UserUnread  int             `json:"user_unread"`
	AdminUnread int             `json:"admin_unread"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	Messages    []TicketMessage `json:"messages,omitempty"`
}
type TicketMessage struct {
	ID            int64     `json:"id"`
	AuthorUserID  int64     `json:"author_user_id"`
	AuthorRole    string    `json:"author_role"`
	Content       string    `json:"content"`
	AttachmentURL *string   `json:"attachment_url,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

func createCommunitySystemTicketTx(ctx context.Context, tx *sql.Tx, userID int64, subject, category, content string) error {
	var ticketID int64
	if err := tx.QueryRowContext(ctx, `INSERT INTO support_tickets(user_id,subject,category,priority,status,user_unread,admin_unread) VALUES($1,$2,$3,'normal','open',1,0) RETURNING id`, userID, subject, category).Scan(&ticketID); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO support_ticket_messages(ticket_id,author_user_id,author_role,content) VALUES($1,$2,'system',$3)`, ticketID, userID, content)
	return err
}

func (s *CommunityService) CreateTicket(ctx context.Context, userID int64, subject, category, priority, content string) (*Ticket, error) {
	subject = strings.TrimSpace(subject)
	content = strings.TrimSpace(content)
	if subject == "" || len([]rune(subject)) > 180 || content == "" || len([]rune(content)) > 10000 {
		return nil, ErrCommunityInvalid
	}
	if category == "" {
		category = "general"
	}
	switch category {
	case "general", "support", "account", "billing", "store", "suggestion":
	default:
		return nil, ErrCommunityInvalid
	}
	if priority == "" {
		priority = "normal"
	}
	switch priority {
	case "low", "normal", "high", "urgent":
	default:
		return nil, ErrCommunityInvalid
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	var id int64
	err = tx.QueryRowContext(ctx, `INSERT INTO support_tickets(user_id,subject,category,priority) VALUES($1,$2,$3,$4) RETURNING id`, userID, subject, category, priority).Scan(&id)
	if err != nil {
		return nil, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO support_ticket_messages(ticket_id,author_user_id,author_role,content) VALUES($1,$2,'user',$3)`, id, userID, content)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetTicket(ctx, userID, id, false)
}

func scanTicket(row rowScanner) (*Ticket, error) {
	var t Ticket
	err := row.Scan(&t.ID, &t.UserID, &t.Subject, &t.Category, &t.Priority, &t.Status, &t.UserUnread, &t.AdminUnread, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCommunityNotFound
	}
	return &t, err
}
func (s *CommunityService) ListTickets(ctx context.Context, userID int64, admin bool) ([]Ticket, error) {
	q := `SELECT id,user_id,subject,category,priority,status,user_unread,admin_unread,created_at,updated_at FROM support_tickets`
	args := []any{}
	if !admin {
		q += ` WHERE user_id=$1`
		args = append(args, userID)
	}
	q += ` ORDER BY updated_at DESC`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []Ticket{}
	for rows.Next() {
		v, err := scanTicket(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}
func (s *CommunityService) GetTicket(ctx context.Context, userID, id int64, admin bool) (*Ticket, error) {
	q := `SELECT id,user_id,subject,category,priority,status,user_unread,admin_unread,created_at,updated_at FROM support_tickets WHERE id=$1`
	args := []any{id}
	if !admin {
		q += ` AND user_id=$2`
		args = append(args, userID)
	}
	t, err := scanTicket(s.db.QueryRowContext(ctx, q, args...))
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,author_user_id,author_role,content,attachment_url,created_at FROM support_ticket_messages WHERE ticket_id=$1 ORDER BY created_at`, id)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var m TicketMessage
		if err = rows.Scan(&m.ID, &m.AuthorUserID, &m.AuthorRole, &m.Content, &m.AttachmentURL, &m.CreatedAt); err != nil {
			return nil, err
		}
		t.Messages = append(t.Messages, m)
	}
	if admin {
		_, _ = s.db.ExecContext(ctx, `UPDATE support_tickets SET admin_unread=0 WHERE id=$1`, id)
	} else {
		_, _ = s.db.ExecContext(ctx, `UPDATE support_tickets SET user_unread=0 WHERE id=$1`, id)
	}
	return t, rows.Err()
}
func (s *CommunityService) ReplyTicket(ctx context.Context, userID, id int64, admin bool, content string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return ErrCommunityInvalid
	}
	role := "user"
	where := ` AND user_id=$2`
	args := []any{id, userID}
	status := "waiting_admin"
	if admin {
		role = "admin"
		where = ""
		args = []any{id}
		status = "waiting_user"
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var owner int64
	var currentStatus string
	if err = tx.QueryRowContext(ctx, `SELECT user_id,status FROM support_tickets WHERE id=$1`+where, args...).Scan(&owner, &currentStatus); errors.Is(err, sql.ErrNoRows) {
		return ErrCommunityNotFound
	} else if err != nil {
		return err
	}
	if currentStatus == "closed" {
		return ErrCommunityConflict
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO support_ticket_messages(ticket_id,author_user_id,author_role,content) VALUES($1,$2,$3,$4)`, id, userID, role, content)
	if err != nil {
		return err
	}
	if admin {
		_, err = tx.ExecContext(ctx, `UPDATE support_tickets SET status=$2,user_unread=user_unread+1,updated_at=NOW() WHERE id=$1`, id, status)
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE support_tickets SET status=$2,admin_unread=admin_unread+1,updated_at=NOW() WHERE id=$1`, id, status)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}
func (s *CommunityService) MarkTicketRead(ctx context.Context, userID, id int64) error {
	result, err := s.db.ExecContext(ctx, `UPDATE support_tickets SET user_unread=0 WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrCommunityNotFound
	}
	return nil
}
func (s *CommunityService) CloseUserTicket(ctx context.Context, userID, id int64) error {
	result, err := s.db.ExecContext(ctx, `UPDATE support_tickets SET status='closed',closed_at=NOW(),updated_at=NOW() WHERE id=$1 AND user_id=$2 AND status<>'closed'`, id, userID)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrCommunityConflict
	}
	return nil
}
func (s *CommunityService) UpdateTicketStatus(ctx context.Context, id int64, status string) error {
	switch status {
	case "open", "waiting_user", "waiting_admin", "resolved", "closed":
	default:
		return ErrCommunityInvalid
	}
	_, err := s.db.ExecContext(ctx, `UPDATE support_tickets SET status=$2,closed_at=CASE WHEN $2='closed' THEN NOW() ELSE NULL END,updated_at=NOW() WHERE id=$1`, id, status)
	return err
}

type PayoutMethod struct {
	Method      string    `json:"method"`
	QRCodeData  string    `json:"qr_code_data"`
	DisplayName string    `json:"display_name"`
	UpdatedAt   time.Time `json:"updated_at"`
}
type Withdrawal struct {
	ID               int64      `json:"id"`
	UserID           int64      `json:"user_id"`
	UserEmail        string     `json:"user_email,omitempty"`
	PayoutMethod     string     `json:"payout_method"`
	PayoutSnapshot   string     `json:"payout_snapshot"`
	Amount           float64    `json:"amount"`
	Fee              float64    `json:"fee"`
	Status           string     `json:"status"`
	UserNote         string     `json:"user_note"`
	AdminNote        string     `json:"admin_note"`
	PaymentReference string     `json:"payment_reference"`
	CreatedAt        time.Time  `json:"created_at"`
	ReviewedAt       *time.Time `json:"reviewed_at,omitempty"`
	PaidAt           *time.Time `json:"paid_at,omitempty"`
}

func (s *CommunityService) SavePayoutMethod(ctx context.Context, userID int64, method, qr, name string) (*PayoutMethod, error) {
	if (method != "alipay" && method != "wechat") || !validPayoutQRCode(qr) {
		return nil, ErrCommunityInvalid
	}
	var p PayoutMethod
	err := s.db.QueryRowContext(ctx, `INSERT INTO payout_methods(user_id,method,qr_code_data,display_name)VALUES($1,$2,$3,$4) ON CONFLICT(user_id,method)DO UPDATE SET qr_code_data=EXCLUDED.qr_code_data,display_name=EXCLUDED.display_name,updated_at=NOW() RETURNING method,qr_code_data,display_name,updated_at`, userID, method, qr, strings.TrimSpace(name)).Scan(&p.Method, &p.QRCodeData, &p.DisplayName, &p.UpdatedAt)
	return &p, err
}

func validPayoutQRCode(value string) bool {
	header, encoded, ok := strings.Cut(strings.TrimSpace(value), ",")
	if !ok {
		return false
	}
	switch strings.ToLower(header) {
	case "data:image/png;base64", "data:image/jpeg;base64", "data:image/webp;base64":
	default:
		return false
	}
	if encoded == "" || base64.StdEncoding.DecodedLen(len(encoded)) > 2*1024*1024 {
		return false
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(decoded) == 0 || len(decoded) > 2*1024*1024 {
		return false
	}
	declaredType := strings.TrimSuffix(strings.TrimPrefix(strings.ToLower(header), "data:"), ";base64")
	return strings.EqualFold(http.DetectContentType(decoded), declaredType)
}
func (s *CommunityService) ListPayoutMethods(ctx context.Context, userID int64) ([]PayoutMethod, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT method,qr_code_data,display_name,updated_at FROM payout_methods WHERE user_id=$1 ORDER BY method`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []PayoutMethod{}
	for rows.Next() {
		var p PayoutMethod
		if err = rows.Scan(&p.Method, &p.QRCodeData, &p.DisplayName, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
func (s *CommunityService) DeletePayoutMethod(ctx context.Context, userID int64, method string) error {
	if method != "alipay" && method != "wechat" {
		return ErrCommunityInvalid
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM payout_methods p WHERE p.user_id=$1 AND p.method=$2 AND NOT EXISTS(SELECT 1 FROM withdrawals w WHERE w.user_id=p.user_id AND w.payout_method=p.method AND w.status IN ('pending','approved'))`, userID, method)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ErrCommunityConflict
	}
	return nil
}
func scanWithdrawal(row rowScanner, admin bool) (*Withdrawal, error) {
	var w Withdrawal
	var err error
	if admin {
		err = row.Scan(&w.ID, &w.UserID, &w.UserEmail, &w.PayoutMethod, &w.PayoutSnapshot, &w.Amount, &w.Fee, &w.Status, &w.UserNote, &w.AdminNote, &w.PaymentReference, &w.CreatedAt, &w.ReviewedAt, &w.PaidAt)
	} else {
		err = row.Scan(&w.ID, &w.UserID, &w.PayoutMethod, &w.PayoutSnapshot, &w.Amount, &w.Fee, &w.Status, &w.UserNote, &w.AdminNote, &w.PaymentReference, &w.CreatedAt, &w.ReviewedAt, &w.PaidAt)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCommunityNotFound
	}
	return &w, err
}
func (s *CommunityService) CreateWithdrawal(ctx context.Context, userID int64, method string, amount float64, note string) (*Withdrawal, error) {
	roundedAmount := math.Round(amount*100) / 100
	if amount < 1 || math.Abs(amount-roundedAmount) > 1e-9 || (method != "alipay" && method != "wechat") {
		return nil, ErrCommunityInvalid
	}
	amount = roundedAmount
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	var snapshot string
	if err = tx.QueryRowContext(ctx, `SELECT qr_code_data FROM payout_methods WHERE user_id=$1 AND method=$2`, userID, method).Scan(&snapshot); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCommunityInvalid
	} else if err != nil {
		return nil, err
	}
	var count int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM withdrawals WHERE user_id=$1 AND created_at>=NOW()-INTERVAL '7 days'`, userID).Scan(&count); err != nil {
		return nil, err
	}
	if count >= 3 {
		return nil, ErrCommunityConflict
	}
	var pendingCount int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM withdrawals WHERE user_id=$1 AND status IN ('pending','approved')`, userID).Scan(&pendingCount); err != nil {
		return nil, err
	}
	if pendingCount > 0 {
		return nil, ErrCommunityConflict
	}
	fee := 0.0
	if err = tx.QueryRowContext(ctx, `SELECT CASE WHEN EXISTS(SELECT 1 FROM withdrawals WHERE user_id=$1) THEN 0 ELSE 0.10 END`, userID).Scan(&fee); err != nil {
		return nil, err
	}
	total := amount + fee
	res, err := tx.ExecContext(ctx, `UPDATE users SET balance=balance-$1,frozen_balance=frozen_balance+$1,updated_at=NOW() WHERE id=$2 AND balance>=$1`, total, userID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, ErrCommunityInsufficient
	}
	var id int64
	err = tx.QueryRowContext(ctx, `INSERT INTO withdrawals(user_id,payout_method,payout_snapshot,amount,fee,user_note)VALUES($1,$2,$3,$4,$5,$6)RETURNING id`, userID, method, snapshot, amount, fee, strings.TrimSpace(note)).Scan(&id)
	if err != nil {
		return nil, err
	}
	if err = createCommunitySystemTicketTx(ctx, tx, userID, "提现申请已提交", "billing", fmt.Sprintf("你的 %.2f 元提现申请已提交，当前等待管理员审核和线下打款。", amount)); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetWithdrawal(ctx, userID, id, false)
}
func (s *CommunityService) GetWithdrawal(ctx context.Context, userID, id int64, admin bool) (*Withdrawal, error) {
	base := `SELECT w.id,w.user_id,`
	args := []any{id}
	if admin {
		base += `u.email,`
	}
	base += `w.payout_method,w.payout_snapshot,w.amount,w.fee,w.status,w.user_note,w.admin_note,w.payment_reference,w.created_at,w.reviewed_at,w.paid_at FROM withdrawals w`
	if admin {
		base += ` JOIN users u ON u.id=w.user_id`
	}
	base += ` WHERE w.id=$1`
	if !admin {
		base += ` AND w.user_id=$2`
		args = append(args, userID)
	}
	return scanWithdrawal(s.db.QueryRowContext(ctx, base, args...), admin)
}
func (s *CommunityService) ListWithdrawals(ctx context.Context, userID int64, admin bool) ([]Withdrawal, error) {
	base := `SELECT w.id,w.user_id,`
	args := []any{}
	if admin {
		base += `u.email,`
	}
	base += `w.payout_method,w.payout_snapshot,w.amount,w.fee,w.status,w.user_note,w.admin_note,w.payment_reference,w.created_at,w.reviewed_at,w.paid_at FROM withdrawals w`
	if admin {
		base += ` JOIN users u ON u.id=w.user_id`
	} else {
		base += ` WHERE w.user_id=$1`
		args = append(args, userID)
	}
	base += ` ORDER BY w.created_at DESC`
	rows, err := s.db.QueryContext(ctx, base, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []Withdrawal{}
	for rows.Next() {
		w, err := scanWithdrawal(rows, admin)
		if err != nil {
			return nil, err
		}
		out = append(out, *w)
	}
	return out, rows.Err()
}
func (s *CommunityService) CancelWithdrawal(ctx context.Context, userID, id int64) error {
	return s.reviewWithdrawal(ctx, userID, id, "cancelled", "", "", false)
}
func (s *CommunityService) ReviewWithdrawal(ctx context.Context, adminID, id int64, status, note, reference string) error {
	return s.reviewWithdrawal(ctx, adminID, id, status, note, reference, true)
}
func (s *CommunityService) reviewWithdrawal(ctx context.Context, actorID, id int64, status, note, reference string, admin bool) error {
	if admin {
		if status != "approved" && status != "rejected" && status != "paid" {
			return ErrCommunityInvalid
		}
		if status == "paid" && strings.TrimSpace(reference) == "" {
			return ErrCommunityInvalid
		}
	} else if status != "cancelled" {
		return ErrCommunityInvalid
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var userID int64
	var amount, fee float64
	var current string
	q := `SELECT user_id,amount,fee,status FROM withdrawals WHERE id=$1 FOR UPDATE`
	args := []any{id}
	if !admin {
		q = `SELECT user_id,amount,fee,status FROM withdrawals WHERE id=$1 AND user_id=$2 FOR UPDATE`
		args = append(args, actorID)
	}
	if err = tx.QueryRowContext(ctx, q, args...).Scan(&userID, &amount, &fee, &current); errors.Is(err, sql.ErrNoRows) {
		return ErrCommunityNotFound
	} else if err != nil {
		return err
	}
	if admin {
		validTransition := (current == "pending" && (status == "approved" || status == "rejected")) || (current == "approved" && status == "paid")
		if !validTransition {
			return ErrCommunityConflict
		}
	} else if current != "pending" {
		return ErrCommunityConflict
	}
	total := amount + fee
	switch status {
	case "cancelled", "rejected":
		res, err := tx.ExecContext(ctx, `UPDATE users SET balance=balance+$1,frozen_balance=frozen_balance-$1,updated_at=NOW() WHERE id=$2 AND frozen_balance>=$1`, total, userID)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return ErrCommunityConflict
		}
	case "paid":
		res, err := tx.ExecContext(ctx, `UPDATE users SET frozen_balance=frozen_balance-$1,updated_at=NOW() WHERE id=$2 AND frozen_balance>=$1`, total, userID)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return ErrCommunityConflict
		}
	}
	_, err = tx.ExecContext(ctx, `UPDATE withdrawals SET status=$2::varchar,admin_note=$3,payment_reference=$4,reviewed_by=CASE WHEN $5 THEN $6 ELSE reviewed_by END,reviewed_at=CASE WHEN $5 THEN NOW() ELSE reviewed_at END,paid_at=CASE WHEN $2::varchar='paid' THEN NOW() ELSE paid_at END,updated_at=NOW() WHERE id=$1`, id, status, strings.TrimSpace(note), strings.TrimSpace(reference), admin, actorID)
	if err != nil {
		return err
	}
	if admin {
		message := map[string]string{
			"approved": "你的提现申请已审核通过，管理员将按收款码快照线下打款。",
			"rejected": "你的提现申请已驳回，冻结金额和手续费已退回余额。",
			"paid":     "你的提现申请已完成线下打款，请核对收款账户。",
		}[status]
		if note = strings.TrimSpace(note); note != "" {
			message += " 备注：" + note
		}
		if err = createCommunitySystemTicketTx(ctx, tx, userID, "提现状态更新", "billing", message); err != nil {
			return err
		}
	}
	return tx.Commit()
}

type StoreProduct struct {
	ID               int64    `json:"id"`
	Category         string   `json:"category"`
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	Price            float64  `json:"price"`
	PointsPrice      *float64 `json:"points_price,omitempty"`
	FulfillmentType  string   `json:"fulfillment_type"`
	FulfillmentValue float64  `json:"fulfillment_value"`
	Status           string   `json:"status"`
	Stock            int      `json:"stock"`
	SortOrder        int      `json:"sort_order"`
}
type StoreOrder struct {
	ID            int64           `json:"id"`
	OrderNo       string          `json:"order_no"`
	UserID        int64           `json:"user_id"`
	ProductID     int64           `json:"product_id"`
	ProductName   string          `json:"product_name"`
	Quantity      int             `json:"quantity"`
	UnitPrice     float64         `json:"unit_price"`
	TotalAmount   float64         `json:"total_amount"`
	PaymentSource string          `json:"payment_source"`
	Status        string          `json:"status"`
	Delivery      json.RawMessage `json:"delivery"`
	CreatedAt     time.Time       `json:"created_at"`
}

type PlatformStoreOrderPreparation struct {
	StoreOrderID int64     `json:"store_order_id"`
	OrderNo      string    `json:"order_no"`
	Amount       float64   `json:"amount"`
	ExpiresAt    time.Time `json:"expires_at"`
}

func (s *CommunityService) ListProducts(ctx context.Context, admin bool) ([]StoreProduct, error) {
	q := `SELECT p.id,p.category,p.name,p.description,p.price,p.points_price,p.fulfillment_type,p.fulfillment_value,p.status,(SELECT COUNT(*) FROM store_inventory i WHERE i.product_id=p.id AND i.status='available'),p.sort_order FROM store_products p`
	if !admin {
		q += ` WHERE p.status='active'`
	}
	q += ` ORDER BY p.sort_order,p.id`
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []StoreProduct{}
	for rows.Next() {
		var p StoreProduct
		if err = rows.Scan(&p.ID, &p.Category, &p.Name, &p.Description, &p.Price, &p.PointsPrice, &p.FulfillmentType, &p.FulfillmentValue, &p.Status, &p.Stock, &p.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
func (s *CommunityService) UpsertProduct(ctx context.Context, p StoreProduct) (*StoreProduct, error) {
	validFulfillment := p.FulfillmentType == "" || p.FulfillmentType == "card" || p.FulfillmentType == "redeem_code" || p.FulfillmentType == "balance" || p.FulfillmentType == "entitlement"
	validStatus := p.Status == "" || p.Status == "draft" || p.Status == "active" || p.Status == "inactive"
	if strings.TrimSpace(p.Name) == "" || p.Price < 0 || (p.PointsPrice != nil && *p.PointsPrice < 0) || !validFulfillment || !validStatus {
		return nil, ErrCommunityInvalid
	}
	if p.Category == "" {
		p.Category = "兑换码"
	}
	if p.FulfillmentType == "" {
		p.FulfillmentType = "card"
	}
	if p.Status == "" {
		p.Status = "draft"
	}
	if (p.FulfillmentType == "balance" || p.FulfillmentType == "entitlement") && p.FulfillmentValue <= 0 {
		return nil, ErrCommunityInvalid
	}
	if p.FulfillmentType == "entitlement" && math.Trunc(p.FulfillmentValue) != p.FulfillmentValue {
		return nil, ErrCommunityInvalid
	}
	if p.ID == 0 {
		err := s.db.QueryRowContext(ctx, `INSERT INTO store_products(category,name,description,price,points_price,fulfillment_type,fulfillment_value,status,sort_order)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)RETURNING id`, p.Category, p.Name, p.Description, p.Price, p.PointsPrice, p.FulfillmentType, p.FulfillmentValue, p.Status, p.SortOrder).Scan(&p.ID)
		if err != nil {
			return nil, err
		}
	} else {
		_, err := s.db.ExecContext(ctx, `UPDATE store_products SET category=$2,name=$3,description=$4,price=$5,points_price=$6,fulfillment_type=$7,fulfillment_value=$8,status=$9,sort_order=$10,updated_at=NOW() WHERE id=$1`, p.ID, p.Category, p.Name, p.Description, p.Price, p.PointsPrice, p.FulfillmentType, p.FulfillmentValue, p.Status, p.SortOrder)
		if err != nil {
			return nil, err
		}
	}
	items, err := s.ListProducts(ctx, true)
	if err != nil {
		return nil, err
	}
	for _, v := range items {
		if v.ID == p.ID {
			return &v, nil
		}
	}
	return nil, ErrCommunityNotFound
}
func (s *CommunityService) AddInventory(ctx context.Context, productID int64, values []string) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	var fulfillmentType string
	if err = tx.QueryRowContext(ctx, `SELECT fulfillment_type FROM store_products WHERE id=$1 FOR UPDATE`, productID).Scan(&fulfillmentType); errors.Is(err, sql.ErrNoRows) {
		return 0, ErrCommunityNotFound
	} else if err != nil {
		return 0, err
	}
	if fulfillmentType != "card" && fulfillmentType != "redeem_code" {
		return 0, ErrCommunityInvalid
	}
	count := 0
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		encrypted, encryptErr := s.encryptor.Encrypt(v)
		if encryptErr != nil {
			return 0, fmt.Errorf("encrypt inventory: %w", encryptErr)
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO store_inventory(product_id,secret_value)VALUES($1,$2)`, productID, encrypted); err != nil {
			return 0, err
		}
		count++
	}
	if count == 0 {
		return 0, ErrCommunityInvalid
	}
	return count, tx.Commit()
}

type StoreWallet struct {
	Balance float64 `json:"balance"`
	Points  float64 `json:"points"`
}

func (s *CommunityService) GetStoreWallet(ctx context.Context, userID int64) (*StoreWallet, error) {
	var wallet StoreWallet
	if err := s.db.QueryRowContext(ctx, `SELECT balance,points FROM users WHERE id=$1 AND deleted_at IS NULL`, userID).Scan(&wallet.Balance, &wallet.Points); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCommunityNotFound
		}
		return nil, err
	}
	return &wallet, nil
}

func (s *CommunityService) PreparePlatformStoreOrder(ctx context.Context, userID, productID int64, quantity int, paymentMethod string) (*PlatformStoreOrderPreparation, error) {
	if quantity < 1 || quantity > 100 || strings.TrimSpace(paymentMethod) == "" {
		return nil, ErrCommunityInvalid
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	var product StoreProduct
	if err = tx.QueryRowContext(ctx, `SELECT id,category,name,description,price,points_price,fulfillment_type,fulfillment_value,status,0,sort_order FROM store_products WHERE id=$1 AND status='active' FOR UPDATE`, productID).
		Scan(&product.ID, &product.Category, &product.Name, &product.Description, &product.Price, &product.PointsPrice, &product.FulfillmentType, &product.FulfillmentValue, &product.Status, &product.Stock, &product.SortOrder); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCommunityNotFound
	} else if err != nil {
		return nil, err
	}
	total := math.Round(product.Price*float64(quantity)*1e8) / 1e8
	expiresAt := time.Now().Add(30 * time.Minute)
	orderNo := fmt.Sprintf("SP%d%06d", time.Now().UnixMilli(), userID%1000000)
	var orderID int64
	if err = tx.QueryRowContext(ctx, `INSERT INTO store_orders(order_no,user_id,product_id,quantity,unit_price,total_amount,payment_source,payment_method,status,expires_at) VALUES($1,$2,$3,$4,$5,$6,'platform',$7,'pending',$8) RETURNING id`, orderNo, userID, productID, quantity, product.Price, total, paymentMethod, expiresAt).Scan(&orderID); err != nil {
		return nil, err
	}
	if product.FulfillmentType == "card" || product.FulfillmentType == "redeem_code" {
		rows, queryErr := tx.QueryContext(ctx, `SELECT id FROM store_inventory WHERE product_id=$1 AND status='available' ORDER BY id LIMIT $2 FOR UPDATE SKIP LOCKED`, productID, quantity)
		if queryErr != nil {
			return nil, queryErr
		}
		ids := make([]int64, 0, quantity)
		for rows.Next() {
			var id int64
			if scanErr := rows.Scan(&id); scanErr != nil {
				_ = rows.Close()
				return nil, scanErr
			}
			ids = append(ids, id)
		}
		_ = rows.Close()
		if len(ids) != quantity {
			return nil, ErrCommunityConflict
		}
		if _, err = tx.ExecContext(ctx, `UPDATE store_inventory SET status='reserved',order_id=$1 WHERE id=ANY($2)`, orderID, pq.Array(ids)); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &PlatformStoreOrderPreparation{StoreOrderID: orderID, OrderNo: orderNo, Amount: total, ExpiresAt: expiresAt}, nil
}

func (s *CommunityService) BindPlatformStorePayment(ctx context.Context, userID, storeOrderID, paymentOrderID int64, expiresAt time.Time) error {
	result, err := s.db.ExecContext(ctx, `UPDATE store_orders SET payment_order_id=$3,expires_at=$4 WHERE id=$1 AND user_id=$2 AND payment_source='platform' AND status='pending' AND payment_order_id IS NULL`, storeOrderID, userID, paymentOrderID, expiresAt)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrCommunityConflict
	}
	return nil
}

func (s *CommunityService) CancelPreparedPlatformStoreOrder(ctx context.Context, userID, storeOrderID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, `UPDATE store_orders SET status='failed' WHERE id=$1 AND user_id=$2 AND payment_source='platform' AND status='pending'`, storeOrderID, userID)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected > 0 {
		if _, err = tx.ExecContext(ctx, `UPDATE store_inventory SET status='available',order_id=NULL WHERE order_id=$1 AND status='reserved'`, storeOrderID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *CommunityService) FulfillPlatformStoreOrder(ctx context.Context, storeOrderID, paymentOrderID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var userID, productID int64
	var quantity int
	var status, fulfillmentType string
	var fulfillmentValue float64
	if err = tx.QueryRowContext(ctx, `SELECT o.user_id,o.product_id,o.quantity,o.status,p.fulfillment_type,p.fulfillment_value FROM store_orders o JOIN store_products p ON p.id=o.product_id WHERE o.id=$1 AND o.payment_order_id=$2 FOR UPDATE OF o,p`, storeOrderID, paymentOrderID).
		Scan(&userID, &productID, &quantity, &status, &fulfillmentType, &fulfillmentValue); errors.Is(err, sql.ErrNoRows) {
		return ErrCommunityNotFound
	} else if err != nil {
		return err
	}
	if status == "fulfilled" {
		return tx.Commit()
	}
	if status != "pending" && status != "paid" {
		return ErrCommunityConflict
	}
	delivery := make([]string, 0, quantity)
	switch fulfillmentType {
	case "card", "redeem_code":
		rows, queryErr := tx.QueryContext(ctx, `SELECT id,secret_value FROM store_inventory WHERE order_id=$1 AND status='reserved' ORDER BY id FOR UPDATE`, storeOrderID)
		if queryErr != nil {
			return queryErr
		}
		ids := make([]int64, 0, quantity)
		for rows.Next() {
			var id int64
			var encrypted string
			if scanErr := rows.Scan(&id, &encrypted); scanErr != nil {
				_ = rows.Close()
				return scanErr
			}
			plain, decryptErr := s.encryptor.Decrypt(encrypted)
			if decryptErr != nil {
				_ = rows.Close()
				return decryptErr
			}
			ids = append(ids, id)
			delivery = append(delivery, plain)
		}
		_ = rows.Close()
		if len(ids) != quantity {
			return ErrCommunityConflict
		}
		if _, err = tx.ExecContext(ctx, `UPDATE store_inventory SET status='sold',sold_at=NOW() WHERE id=ANY($1)`, pq.Array(ids)); err != nil {
			return err
		}
	case "balance":
		credit := fulfillmentValue * float64(quantity)
		if _, err = tx.ExecContext(ctx, `UPDATE users SET balance=balance+$1,updated_at=NOW() WHERE id=$2`, credit, userID); err != nil {
			return err
		}
		delivery = append(delivery, fmt.Sprintf("余额到账 %.2f", credit))
	case "entitlement":
		credit := int(fulfillmentValue) * quantity
		if _, err = tx.ExecContext(ctx, `UPDATE users SET concurrency=concurrency+$1,updated_at=NOW() WHERE id=$2`, credit, userID); err != nil {
			return err
		}
		delivery = append(delivery, fmt.Sprintf("负载额度到账 +%d", credit))
	default:
		return ErrCommunityInvalid
	}
	payload, _ := json.Marshal(delivery)
	if _, err = tx.ExecContext(ctx, `UPDATE store_orders SET status='fulfilled',delivery=$2,paid_at=COALESCE(paid_at,NOW()),fulfilled_at=NOW() WHERE id=$1`, storeOrderID, payload); err != nil {
		return err
	}
	if err = createCommunitySystemTicketTx(ctx, tx, userID, "商城订单已交付", "store", fmt.Sprintf("你的商城订单 #%d 已支付并自动交付，请到购买记录查看交付内容。", storeOrderID)); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *CommunityService) BuyProduct(ctx context.Context, userID, productID int64, quantity int, paymentSource string) (*StoreOrder, error) {
	if quantity <= 0 || quantity > 100 || (paymentSource != "balance" && paymentSource != "points") {
		return nil, ErrCommunityInvalid
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	var p StoreProduct
	err = tx.QueryRowContext(ctx, `SELECT id,category,name,description,price,points_price,fulfillment_type,fulfillment_value,status,0,sort_order FROM store_products WHERE id=$1 AND status='active' FOR UPDATE`, productID).Scan(&p.ID, &p.Category, &p.Name, &p.Description, &p.Price, &p.PointsPrice, &p.FulfillmentType, &p.FulfillmentValue, &p.Status, &p.Stock, &p.SortOrder)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCommunityNotFound
	} else if err != nil {
		return nil, err
	}
	unitPrice := p.Price
	column := "balance"
	if paymentSource == "points" {
		if p.PointsPrice == nil {
			return nil, ErrCommunityInvalid
		}
		unitPrice = *p.PointsPrice
		column = "points"
	}
	total := math.Round(unitPrice*float64(quantity)*100000000) / 100000000
	res, err := tx.ExecContext(ctx, `UPDATE users SET `+column+`=`+column+`-$1,updated_at=NOW() WHERE id=$2 AND `+column+`>=$1`, total, userID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, ErrCommunityInsufficient
	}
	orderNo := fmt.Sprintf("SC%d%06d", time.Now().UnixMilli(), userID%1000000)
	var orderID int64
	err = tx.QueryRowContext(ctx, `INSERT INTO store_orders(order_no,user_id,product_id,quantity,unit_price,total_amount,payment_source,status,paid_at)VALUES($1,$2,$3,$4,$5,$6,$7,'paid',NOW())RETURNING id`, orderNo, userID, productID, quantity, unitPrice, total, paymentSource).Scan(&orderID)
	if err != nil {
		return nil, err
	}
	delivery := []string{}
	switch p.FulfillmentType {
	case "card", "redeem_code":
		rows, err := tx.QueryContext(ctx, `SELECT id,secret_value FROM store_inventory WHERE product_id=$1 AND status='available' ORDER BY id LIMIT $2 FOR UPDATE SKIP LOCKED`, productID, quantity)
		if err != nil {
			return nil, err
		}
		ids := []int64{}
		for rows.Next() {
			var id int64
			var v string
			if err = rows.Scan(&id, &v); err != nil {
				_ = rows.Close()
				return nil, err
			}
			plain, decryptErr := s.encryptor.Decrypt(v)
			if decryptErr != nil {
				_ = rows.Close()
				return nil, fmt.Errorf("decrypt inventory: %w", decryptErr)
			}
			ids = append(ids, id)
			delivery = append(delivery, plain)
		}
		_ = rows.Close()
		if len(ids) != quantity {
			return nil, ErrCommunityConflict
		}
		for _, id := range ids {
			if _, err = tx.ExecContext(ctx, `UPDATE store_inventory SET status='sold',order_id=$2,sold_at=NOW() WHERE id=$1`, id, orderID); err != nil {
				return nil, err
			}
		}
	case "balance":
		credit := p.FulfillmentValue * float64(quantity)
		if _, err = tx.ExecContext(ctx, `UPDATE users SET balance=balance+$1 WHERE id=$2`, credit, userID); err != nil {
			return nil, err
		}
		delivery = []string{fmt.Sprintf("余额到账 %.2f", credit)}
	case "entitlement":
		credit := int(p.FulfillmentValue) * quantity
		if _, err = tx.ExecContext(ctx, `UPDATE users SET concurrency=concurrency+$1,updated_at=NOW() WHERE id=$2`, credit, userID); err != nil {
			return nil, err
		}
		delivery = []string{fmt.Sprintf("负载额度到账 +%d", credit)}
	default:
		return nil, ErrCommunityInvalid
	}
	payload, _ := json.Marshal(delivery)
	_, err = tx.ExecContext(ctx, `UPDATE store_orders SET status='fulfilled',delivery=$2,fulfilled_at=NOW() WHERE id=$1`, orderID, payload)
	if err != nil {
		return nil, err
	}
	if err = createCommunitySystemTicketTx(ctx, tx, userID, "商城订单已交付", "store", fmt.Sprintf("你的商城订单 #%d 已完成支付并自动交付，请到购买记录查看交付内容。", orderID)); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetStoreOrder(ctx, userID, orderID, false)
}
func scanStoreOrder(row rowScanner) (*StoreOrder, error) {
	var o StoreOrder
	err := row.Scan(&o.ID, &o.OrderNo, &o.UserID, &o.ProductID, &o.ProductName, &o.Quantity, &o.UnitPrice, &o.TotalAmount, &o.PaymentSource, &o.Status, &o.Delivery, &o.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrCommunityNotFound
	}
	return &o, err
}
func (s *CommunityService) GetStoreOrder(ctx context.Context, userID, id int64, admin bool) (*StoreOrder, error) {
	q := `SELECT o.id,o.order_no,o.user_id,o.product_id,p.name,o.quantity,o.unit_price,o.total_amount,o.payment_source,o.status,o.delivery,o.created_at FROM store_orders o JOIN store_products p ON p.id=o.product_id WHERE o.id=$1`
	args := []any{id}
	if !admin {
		q += ` AND o.user_id=$2`
		args = append(args, userID)
	}
	return scanStoreOrder(s.db.QueryRowContext(ctx, q, args...))
}
func (s *CommunityService) ListStoreOrders(ctx context.Context, userID int64, admin bool) ([]StoreOrder, error) {
	q := `SELECT o.id,o.order_no,o.user_id,o.product_id,p.name,o.quantity,o.unit_price,o.total_amount,o.payment_source,o.status,o.delivery,o.created_at FROM store_orders o JOIN store_products p ON p.id=o.product_id`
	args := []any{}
	if !admin {
		q += ` WHERE o.user_id=$1`
		args = append(args, userID)
	}
	q += ` ORDER BY o.created_at DESC`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []StoreOrder{}
	for rows.Next() {
		o, err := scanStoreOrder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *o)
	}
	return out, rows.Err()
}
func (s *CommunityService) GetCommission(ctx context.Context) (float64, error) {
	return s.commissionRate(ctx), nil
}
func (s *CommunityService) SetCommission(ctx context.Context, value float64) error {
	if value < 0 || value > 100 {
		return ErrCommunityInvalid
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO settings(key,value,updated_at)VALUES('community_marketplace_commission_percent',$1,NOW())ON CONFLICT(key)DO UPDATE SET value=EXCLUDED.value,updated_at=NOW()`, fmt.Sprintf("%.4f", value))
	return err
}
