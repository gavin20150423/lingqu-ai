//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type updateServiceCacheStub struct {
	data string
}

func (s *updateServiceCacheStub) GetUpdateInfo(context.Context) (string, error) {
	if s.data == "" {
		return "", errors.New("cache miss")
	}
	return s.data, nil
}

func (s *updateServiceCacheStub) SetUpdateInfo(_ context.Context, data string, _ time.Duration) error {
	s.data = data
	return nil
}

type updateServiceGitHubClientStub struct {
	release     *GitHubRelease
	fetchedRepo string
}

func (s *updateServiceGitHubClientStub) FetchLatestRelease(_ context.Context, repo string) (*GitHubRelease, error) {
	s.fetchedRepo = repo
	return s.release, nil
}

func (s *updateServiceGitHubClientStub) DownloadFile(context.Context, string, string, int64) error {
	panic("DownloadFile should not be called when no update is available")
}

func (s *updateServiceGitHubClientStub) FetchChecksumFile(context.Context, string) ([]byte, error) {
	panic("FetchChecksumFile should not be called when no update is available")
}

func TestUpdateServicePerformUpdateNoUpdateReturnsSentinel(t *testing.T) {
	client := &updateServiceGitHubClientStub{
		release: &GitHubRelease{
			TagName: "v0.1.132",
			Name:    "v0.1.132",
		},
	}
	svc := NewUpdateServiceWithConfig(
		&updateServiceCacheStub{},
		client,
		"0.1.132",
		"release",
		true,
		"gavin20150423/lingqu-ai",
	)

	err := svc.PerformUpdate(context.Background())

	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNoUpdateAvailable))
	require.ErrorIs(t, err, ErrNoUpdateAvailable)
	require.Equal(t, "gavin20150423/lingqu-ai", client.fetchedRepo)
}

func TestUpdateServiceDisabledDoesNotFetchRelease(t *testing.T) {
	client := &updateServiceGitHubClientStub{}
	svc := NewUpdateServiceWithConfig(
		&updateServiceCacheStub{},
		client,
		"0.1.139-lingqu.3",
		"release",
		false,
		"gavin20150423/lingqu-ai",
	)

	info, err := svc.CheckUpdate(context.Background(), true)

	require.NoError(t, err)
	require.False(t, info.HasUpdate)
	require.False(t, info.UpdateEnabled)
	require.Equal(t, "0.1.139-lingqu.3", info.LatestVersion)
	require.Empty(t, client.fetchedRepo)
}

func TestCompareVersionsSupportsLingquSuffix(t *testing.T) {
	require.Equal(t, -1, compareVersions("0.1.139-lingqu.3", "0.1.139-lingqu.4"))
	require.Equal(t, 1, compareVersions("0.1.139-lingqu.4", "0.1.139-lingqu.3"))
	require.Equal(t, 0, compareVersions("v0.1.139-lingqu.4", "0.1.139-lingqu.4"))
}

func TestUpdateServiceIgnoresUpstreamLatestForLingquBuild(t *testing.T) {
	svc := NewUpdateServiceWithConfig(
		&updateServiceCacheStub{},
		&updateServiceGitHubClientStub{
			release: &GitHubRelease{
				TagName: "v0.1.140",
				Name:    "v0.1.140",
			},
		},
		"0.1.139-lingqu.3",
		"release",
		true,
		"gavin20150423/lingqu-ai",
	)

	info, err := svc.CheckUpdate(context.Background(), true)

	require.NoError(t, err)
	require.False(t, info.HasUpdate)
	require.Equal(t, "0.1.139-lingqu.3", info.LatestVersion)
}
