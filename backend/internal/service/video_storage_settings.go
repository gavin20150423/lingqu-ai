package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

const (
	settingKeyVideoStorageConfig = "video_storage_config"
	defaultVideoStoragePrefix    = "videos/"
	defaultVideoMaxObjectBytes   = int64(4 << 30)
)

var ErrVideoStorageIncomplete = errors.New("Alibaba Cloud OSS storage is enabled but endpoint/region/bucket/access_key_id/access_key_secret are incomplete")

// VideoObjectStore is the private object-store surface used by generated videos.
// Objects are intentionally served through the authenticated video endpoint.
type VideoObjectStore interface {
	Upload(context.Context, string, io.Reader, string, int64, int64) error
	Open(context.Context, string, string) (*http.Response, error)
	HeadBucket(context.Context) error
}

// AliyunOSSConfig is the independent Alibaba Cloud OSS configuration used by
// generated video persistence. It intentionally has no relationship to the
// backup S3 configuration.
type AliyunOSSConfig struct {
	Endpoint        string `json:"endpoint"`
	Region          string `json:"region"`
	Bucket          string `json:"bucket"`
	AccessKeyID     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret,omitempty"`
	Prefix          string `json:"prefix"`
}

func (c *AliyunOSSConfig) IsConfigured() bool {
	return c != nil && c.Endpoint != "" && c.Region != "" && c.Bucket != "" && c.AccessKeyID != "" && c.AccessKeySecret != ""
}

type VideoObjectStoreFactory func(context.Context, *AliyunOSSConfig) (VideoObjectStore, error)

type VideoStorageSettings struct {
	Enabled bool `json:"enabled"`

	Bucket  string  `json:"bucket"`
	Prefix  string  `json:"prefix"`
	UserIDs []int64 `json:"user_ids"`

	Endpoint        string `json:"endpoint"`
	Region          string `json:"region"`
	AccessKeyID     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret,omitempty"`
}

type resolvedVideoStorage struct {
	store          VideoObjectStore
	prefix         string
	maxObjectBytes int64
	userIDs        []int64
	selected       map[int64]struct{}
	enabled        bool
}

// VideoStorageSettingService stores the OSS credentials and the users whose
// newly completed videos should be copied to OSS.
type VideoStorageSettingService struct {
	settingRepo             SettingRepository
	encryptor               SecretEncryptor
	encryptionKeyConfigured bool
	factory                 VideoObjectStoreFactory

	mu       sync.Mutex
	resolved bool
	storage  *resolvedVideoStorage
}

func NewVideoStorageSettingService(
	settingRepo SettingRepository,
	encryptor SecretEncryptor,
	encryptionKeyConfigured bool,
	factory VideoObjectStoreFactory,
) *VideoStorageSettingService {
	return &VideoStorageSettingService{
		settingRepo: settingRepo, encryptor: encryptor,
		encryptionKeyConfigured: encryptionKeyConfigured, factory: factory,
	}
}

func (s *VideoStorageSettingService) Get(ctx context.Context) (*VideoStorageSettings, error) {
	settings, err := s.load(ctx)
	if err != nil {
		return nil, err
	}
	if settings == nil {
		settings = &VideoStorageSettings{Prefix: defaultVideoStoragePrefix, Region: "cn-hangzhou", UserIDs: []int64{}}
	}
	settings.AccessKeySecret = ""
	return settings, nil
}

func (s *VideoStorageSettingService) SecretConfigured(ctx context.Context) bool {
	settings, err := s.load(ctx)
	if err != nil || settings == nil || !settings.Enabled {
		return false
	}
	return settings.AccessKeySecret != ""
}

func (s *VideoStorageSettingService) Update(ctx context.Context, in VideoStorageSettings) (*VideoStorageSettings, error) {
	normalizeVideoStorageSettings(&in)
	if in.AccessKeySecret == "" {
		if old, err := s.load(ctx); err == nil && old != nil {
			in.AccessKeySecret = old.AccessKeySecret
		}
	} else {
		if !s.encryptionKeyConfigured {
			return nil, ErrSecretEncryptionKeyNotConfigured
		}
		encrypted, err := s.encryptor.Encrypt(in.AccessKeySecret)
		if err != nil {
			return nil, fmt.Errorf("encrypt video OSS access key secret: %w", err)
		}
		in.AccessKeySecret = encrypted
	}
	if in.Enabled {
		cfg := s.toAliyunOSSConfig(&in)
		if !cfg.IsConfigured() {
			return nil, ErrVideoStorageIncomplete
		}
	}

	data, err := json.Marshal(in)
	if err != nil {
		return nil, fmt.Errorf("marshal video storage settings: %w", err)
	}
	if err := s.settingRepo.Set(ctx, settingKeyVideoStorageConfig, string(data)); err != nil {
		return nil, fmt.Errorf("save video storage settings: %w", err)
	}
	s.Invalidate()
	in.AccessKeySecret = ""
	return &in, nil
}

func (s *VideoStorageSettingService) TestConnection(ctx context.Context, in VideoStorageSettings) error {
	normalizeVideoStorageSettings(&in)
	if in.AccessKeySecret == "" {
		if old, err := s.load(ctx); err == nil && old != nil {
			in.AccessKeySecret = old.AccessKeySecret
		}
	}
	cfg := s.toAliyunOSSConfig(&in)
	if !cfg.IsConfigured() {
		return ErrVideoStorageIncomplete
	}
	if s.factory == nil {
		return errors.New("Alibaba Cloud OSS client factory is not configured")
	}
	store, err := s.factory(ctx, cfg)
	if err != nil {
		return err
	}
	return store.HeadBucket(ctx)
}

func (s *VideoStorageSettingService) StoreForUser(userID int64) (VideoObjectStore, string, int64, bool) {
	storage := s.resolve()
	if storage == nil || !storage.enabled {
		return nil, "", 0, false
	}
	if _, ok := storage.selected[userID]; !ok {
		return nil, "", 0, false
	}
	return storage.store, storage.prefix, storage.maxObjectBytes, true
}

// SelectedForUser determines whether a newly created job should be persisted.
// The decision is saved on the job so later selection changes do not alter work
// that was already in progress.
func (s *VideoStorageSettingService) SelectedForUser(userID int64) bool {
	storage := s.resolve()
	if storage == nil || !storage.enabled {
		return false
	}
	_, selected := storage.selected[userID]
	return selected
}

// StoredStore returns the configured object store even when new uploads are
// disabled. Completed jobs already referencing an OSS key must remain readable
// after an administrator changes the selection or turns off future uploads.
func (s *VideoStorageSettingService) StoredStore() (VideoObjectStore, bool) {
	storage := s.resolve()
	if storage == nil {
		return nil, false
	}
	return storage.store, true
}

// StoreForRequestedJob returns the configured store and upload limits for a job
// that already recorded StorageRequested=true. The current Enabled flag only
// controls new jobs and must not interrupt an in-flight persistence attempt.
func (s *VideoStorageSettingService) StoreForRequestedJob() (VideoObjectStore, string, int64, bool) {
	storage := s.resolve()
	if storage == nil {
		return nil, "", 0, false
	}
	return storage.store, storage.prefix, storage.maxObjectBytes, true
}

func (s *VideoStorageSettingService) SelectedUserIDs() []int64 {
	storage := s.resolve()
	if storage == nil {
		return nil
	}
	return append([]int64(nil), storage.userIDs...)
}

func (s *VideoStorageSettingService) resolve() *resolvedVideoStorage {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.resolved {
		return s.storage
	}
	s.resolved = true

	settings, err := s.load(context.Background())
	if err != nil || settings == nil {
		if err != nil {
			logger.L().Warn("video_storage.settings_load_failed", zap.Error(err))
		}
		return nil
	}
	cfg := s.toAliyunOSSConfig(settings)
	if !cfg.IsConfigured() {
		logger.L().Warn("video_storage.settings_invalid", zap.String("provider", "aliyun_oss"), zap.Error(ErrVideoStorageIncomplete))
		return nil
	}
	if s.factory == nil {
		logger.L().Error("video_storage.client_factory_missing")
		return nil
	}
	store, err := s.factory(context.Background(), cfg)
	if err != nil {
		logger.L().Error("video_storage.client_build_failed", zap.Error(err))
		return nil
	}
	selected := make(map[int64]struct{}, len(settings.UserIDs))
	for _, userID := range settings.UserIDs {
		selected[userID] = struct{}{}
	}
	s.storage = &resolvedVideoStorage{
		store: store, prefix: settings.Prefix, maxObjectBytes: defaultVideoMaxObjectBytes,
		userIDs: append([]int64(nil), settings.UserIDs...), selected: selected,
		enabled: settings.Enabled,
	}
	return s.storage
}

func (s *VideoStorageSettingService) Invalidate() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.resolved = false
	s.storage = nil
	s.mu.Unlock()
}

func (s *VideoStorageSettingService) load(ctx context.Context) (*VideoStorageSettings, error) {
	if s == nil || s.settingRepo == nil {
		return nil, nil //nolint:nilnil // an absent setting means disabled
	}
	raw, err := s.settingRepo.GetValue(ctx, settingKeyVideoStorageConfig)
	if err != nil || strings.TrimSpace(raw) == "" {
		return nil, nil //nolint:nilnil // an absent setting means disabled
	}
	var settings VideoStorageSettings
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		return nil, fmt.Errorf("parse video storage settings: %w", err)
	}
	normalizeVideoStorageSettings(&settings)
	return &settings, nil
}

func (s *VideoStorageSettingService) toAliyunOSSConfig(in *VideoStorageSettings) *AliyunOSSConfig {
	cfg := &AliyunOSSConfig{
		Endpoint: in.Endpoint, Region: in.Region, Bucket: in.Bucket,
		AccessKeyID: in.AccessKeyID, AccessKeySecret: in.AccessKeySecret,
		Prefix: in.Prefix,
	}
	if cfg.AccessKeySecret != "" {
		decrypted, err := s.encryptor.Decrypt(cfg.AccessKeySecret)
		if err == nil {
			cfg.AccessKeySecret = decrypted
		} else {
			logger.L().Warn("video_storage access key secret decrypt failed; treating stored value as plaintext", zap.Error(err))
		}
	}
	return cfg
}

func normalizeVideoStorageSettings(in *VideoStorageSettings) {
	in.Endpoint = strings.TrimSpace(in.Endpoint)
	in.Region = strings.TrimSpace(in.Region)
	in.Bucket = strings.TrimSpace(in.Bucket)
	in.AccessKeyID = strings.TrimSpace(in.AccessKeyID)
	in.AccessKeySecret = strings.TrimSpace(in.AccessKeySecret)
	in.Prefix = strings.Trim(strings.TrimSpace(in.Prefix), "/")
	if in.Prefix == "" {
		in.Prefix = strings.TrimSuffix(defaultVideoStoragePrefix, "/")
	}
	in.Prefix += "/"
	if in.Region == "" {
		in.Region = "cn-hangzhou"
	}
	seen := make(map[int64]struct{}, len(in.UserIDs))
	userIDs := make([]int64, 0, len(in.UserIDs))
	for _, userID := range in.UserIDs {
		if userID <= 0 {
			continue
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		userIDs = append(userIDs, userID)
	}
	sort.Slice(userIDs, func(i, j int) bool { return userIDs[i] < userIDs[j] })
	in.UserIDs = userIDs
}
