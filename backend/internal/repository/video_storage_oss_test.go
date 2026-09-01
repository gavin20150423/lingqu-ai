//go:build unit

package repository

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestNormalizeAliyunOSSRange(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "http header", raw: "bytes=0-99", want: "0-99"},
		{name: "open ended", raw: "bytes=100-", want: "100-"},
		{name: "suffix", raw: "bytes=-500", want: "-500"},
		{name: "empty", raw: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeAliyunOSSRange(tt.raw)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestNormalizeAliyunOSSRangeRejectsUnsupportedRanges(t *testing.T) {
	for _, raw := range []string{"bytes=0-99,200-299", "bytes=", "bytes=abc-10", "bytes=-"} {
		_, err := normalizeAliyunOSSRange(raw)
		require.Error(t, err, raw)
	}
}

func TestNewAliyunOSSVideoObjectStoreFactoryRejectsIncompleteConfig(t *testing.T) {
	store, err := NewAliyunOSSVideoObjectStoreFactory()(context.Background(), &service.AliyunOSSConfig{})
	require.ErrorIs(t, err, service.ErrVideoStorageIncomplete)
	require.Nil(t, store)
}
