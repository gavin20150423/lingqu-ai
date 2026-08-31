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

var ErrVideoStorageIncomplete = errors.New("video OSS storage is enabled but bucket/access_key_id/secret_access_key are incomplete")

// VideoObjectStore is the private object-store surface used by generated videos.
// Objects are intentionally served through the authenticated video endpoint.
type VideoObjectStore interface {
	Upload(context.Context, string, io.Reader, string, int64, int64) error
	Open(context.Context, string, string) (*http.Response, error)
	HeadBucket(context.Context) error
}

type VideoObjectStoreFactory func(context.Context, *BackupS3Config) (VideoObjectStore, error)

type VideoStorageSettings struct {
	Enabled       bool `json:"enabled"`
	ReuseBackupS3 bool `json:"reuse_backup_s3"`

	Bucket  string  `json:"bucket"`
	Prefix  string  `json:"prefix"`
	UserIDs []int64 `json:"user_ids"`

	Endpoint        string `json:"endpoint"`
	Region          string `json:"region"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key,omitempty"` //nolint:revive // field name follows AWS convention
	ForcePathStyle  bool   `json:"force_path_style"`
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
	settingRepo SettingRepository
	encryptor   SecretEncryptor
	backup      *BackupService
	factory     VideoObjectStoreFactory

	mu       sync.Mutex
	resolved bool
	storage  *resolvedVideoStorage
}

func NewVideoStorageSettingService(
	settingRepo SettingRepository,
	encryptor SecretEncryptor,
	backup *BackupService,
	factory VideoObjectStoreFactory,
) *VideoStorageSettingService {
	return &VideoStorageSettingService{settingRepo: settingRepo, encryptor: encryptor, backup: backup, factory: factory}
}

func (s *VideoStorageSettingService) Get(ctx context.Context) (*VideoStorageSettings, error) {
	settings, err := s.load(ctx)
	if err != nil {
		return nil, err
	}
	if settings == nil {
		settings = &VideoStorageSettings{Prefix: defaultVideoStoragePrefix, Region: "auto", ReuseBackupS3: true, UserIDs: []int64{}}
	}
	settings.SecretAccessKey = ""
	return settings, nil
}

func (s *VideoStorageSettingService) SecretConfigured(ctx context.Context) bool {
	settings, err := s.load(ctx)
	if err != nil || settings == nil || !settings.Enabled {
		return false
	}
	if settings.ReuseBackupS3 {
		cfg, loadErr := s.backupCredentials(ctx)
		return loadErr == nil && cfg != nil && cfg.SecretAccessKey != ""
	}
	return settings.SecretAccessKey != ""
}

func (s *VideoStorageSettingService) Update(ctx context.Context, in VideoStorageSettings) (*VideoStorageSettings, error) {
	normalizeVideoStorageSettings(&in)
	if in.ReuseBackupS3 {
		in.Endpoint, in.Region, in.AccessKeyID, in.SecretAccessKey = "", "", "", ""
		in.ForcePathStyle = false
	} else if in.SecretAccessKey == "" {
		if old, err := s.load(ctx); err == nil && old != nil && !old.ReuseBackupS3 {
			in.SecretAccessKey = old.SecretAccessKey
		}
	} else {
		if s.backup == nil || !s.backup.EncryptionKeyConfigured() {
			return nil, ErrSecretEncryptionKeyNotConfigured
		}
		encrypted, err := s.encryptor.Encrypt(in.SecretAccessKey)
		if err != nil {
			return nil, fmt.Errorf("encrypt video storage secret: %w", err)
		}
		in.SecretAccessKey = encrypted
	}
	if in.Enabled {
		cfg, err := s.toS3Config(ctx, &in)
		if err != nil {
			return nil, err
		}
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
	in.SecretAccessKey = ""
	return &in, nil
}

func (s *VideoStorageSettingService) TestConnection(ctx context.Context, in VideoStorageSettings) error {
	normalizeVideoStorageSettings(&in)
	if !in.ReuseBackupS3 && in.SecretAccessKey == "" {
		if old, err := s.load(ctx); err == nil && old != nil && !old.ReuseBackupS3 {
			in.SecretAccessKey = old.SecretAccessKey
		}
	}
	cfg, err := s.toS3Config(ctx, &in)
	if err != nil {
		return err
	}
	if !cfg.IsConfigured() {
		return ErrVideoStorageIncomplete
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

func (s *VideoStorageSettingService) Store() (VideoObjectStore, string, int64, bool) {
	storage := s.resolve()
	if storage == nil || !storage.enabled {
		return nil, "", 0, false
	}
	return storage.store, storage.prefix, storage.maxObjectBytes, true
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
	cfg, err := s.toS3Config(context.Background(), settings)
	if err != nil || !cfg.IsConfigured() {
		logger.L().Warn("video_storage.settings_invalid", zap.Error(err))
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

func (s *VideoStorageSettingService) toS3Config(ctx context.Context, in *VideoStorageSettings) (*BackupS3Config, error) {
	cfg := &BackupS3Config{
		Endpoint: in.Endpoint, Region: in.Region, Bucket: in.Bucket, AccessKeyID: in.AccessKeyID,
		SecretAccessKey: in.SecretAccessKey, Prefix: in.Prefix, ForcePathStyle: in.ForcePathStyle,
	}
	if in.ReuseBackupS3 {
		backupCfg, err := s.backupCredentials(ctx)
		if err != nil {
			return nil, err
		}
		if backupCfg == nil {
			return nil, errors.New("video storage is set to reuse backup S3, but backup S3 is not configured")
		}
		cfg.Endpoint, cfg.Region = backupCfg.Endpoint, backupCfg.Region
		cfg.AccessKeyID, cfg.SecretAccessKey = backupCfg.AccessKeyID, backupCfg.SecretAccessKey
		cfg.ForcePathStyle = backupCfg.ForcePathStyle
		if cfg.Bucket == "" {
			cfg.Bucket = backupCfg.Bucket
		}
	} else if cfg.SecretAccessKey != "" {
		decrypted, err := s.encryptor.Decrypt(cfg.SecretAccessKey)
		if err == nil {
			cfg.SecretAccessKey = decrypted
		} else {
			logger.L().Warn("video_storage secret decrypt failed; treating stored value as plaintext", zap.Error(err))
		}
	}
	return cfg, nil
}

func (s *VideoStorageSettingService) backupCredentials(ctx context.Context) (*BackupS3Config, error) {
	if s.backup == nil {
		return nil, errors.New("backup service is unavailable")
	}
	return s.backup.loadS3Config(ctx)
}

func normalizeVideoStorageSettings(in *VideoStorageSettings) {
	in.Endpoint = strings.TrimSpace(in.Endpoint)
	in.Region = strings.TrimSpace(in.Region)
	in.Bucket = strings.TrimSpace(in.Bucket)
	in.AccessKeyID = strings.TrimSpace(in.AccessKeyID)
	in.SecretAccessKey = strings.TrimSpace(in.SecretAccessKey)
	in.Prefix = strings.Trim(strings.TrimSpace(in.Prefix), "/")
	if in.Prefix == "" {
		in.Prefix = strings.TrimSuffix(defaultVideoStoragePrefix, "/")
	}
	in.Prefix += "/"
	if in.Region == "" {
		in.Region = "auto"
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
