package service

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type staticReceiptCodeStorageConfigProvider struct {
	cfg *config.Config
}

func NewReceiptCodeStorageConfigProvider(cfg *config.Config) ReceiptCodeStorageConfigProvider {
	return &staticReceiptCodeStorageConfigProvider{cfg: cfg}
}

func (p *staticReceiptCodeStorageConfigProvider) GetReceiptCodeStorageConfig(context.Context) (config.ReceiptCodeStorageConfig, error) {
	if p == nil || p.cfg == nil {
		return config.ReceiptCodeStorageConfig{}, nil
	}
	return p.cfg.ReceiptCodeStorage, nil
}
