//go:build unit

package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type recordingVideoObjectStore struct {
	headBucketCalls int
	uploadedKeys    []string
	openedKeys      []string
	object          []byte
}

func (s *recordingVideoObjectStore) Upload(_ context.Context, key string, body io.Reader, _ string, _ int64, _ int64) error {
	s.uploadedKeys = append(s.uploadedKeys, key)
	s.object, _ = io.ReadAll(body)
	return nil
}

func (s *recordingVideoObjectStore) Open(_ context.Context, key, _ string) (*http.Response, error) {
	s.openedKeys = append(s.openedKeys, key)
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"video/mp4"}},
		Body:       io.NopCloser(bytes.NewReader(s.object)),
	}, nil
}

func (s *recordingVideoObjectStore) HeadBucket(context.Context) error {
	s.headBucketCalls++
	return nil
}

func newVideoStorageFixture(t *testing.T) (*VideoStorageSettingService, *stubSettingRepo, *[]AliyunOSSConfig, *recordingVideoObjectStore) {
	t.Helper()
	repo := newStubSettingRepo()
	var built []AliyunOSSConfig
	store := &recordingVideoObjectStore{}
	factory := func(_ context.Context, cfg *AliyunOSSConfig) (VideoObjectStore, error) {
		built = append(built, *cfg)
		return store, nil
	}
	return NewVideoStorageSettingService(repo, reversibleEncryptor{}, true, factory), repo, &built, store
}

func TestVideoStorageSettingsPersistsSelectedUsersAndOwnSecret(t *testing.T) {
	svc, repo, built, _ := newVideoStorageFixture(t)
	ctx := context.Background()

	saved, err := svc.Update(ctx, VideoStorageSettings{
		Enabled: true, Bucket: "video-bucket", Endpoint: "https://oss-cn-hangzhou.aliyuncs.com",
		Region: "cn-hangzhou", AccessKeyID: "ak", AccessKeySecret: "secret", Prefix: "/archive/videos/",
		UserIDs: []int64{9, 3, 9, 0, -1},
	})
	require.NoError(t, err)
	require.Empty(t, saved.AccessKeySecret)
	require.Equal(t, []int64{3, 9}, saved.UserIDs)
	require.Equal(t, "archive/videos/", saved.Prefix)

	raw, err := repo.GetValue(ctx, settingKeyVideoStorageConfig)
	require.NoError(t, err)
	require.NotContains(t, raw, `"access_key_secret":"secret"`)
	require.Contains(t, raw, "enc:secret")

	require.True(t, svc.SelectedForUser(3))
	require.False(t, svc.SelectedForUser(4))
	store, prefix, maxBytes, ok := svc.StoreForUser(9)
	require.True(t, ok)
	require.NotNil(t, store)
	require.Equal(t, "archive/videos/", prefix)
	require.Positive(t, maxBytes)
	require.Len(t, *built, 1)
	require.Equal(t, "https://oss-cn-hangzhou.aliyuncs.com", (*built)[0].Endpoint)
	require.Equal(t, "cn-hangzhou", (*built)[0].Region)
	require.Equal(t, "video-bucket", (*built)[0].Bucket)
	require.Equal(t, "ak", (*built)[0].AccessKeyID)
	require.Equal(t, "secret", (*built)[0].AccessKeySecret)
}

func TestVideoStorageSettingsUsesIndependentAliyunOSSConfigAndTestConnection(t *testing.T) {
	svc, _, built, store := newVideoStorageFixture(t)
	ctx := context.Background()

	_, err := svc.Update(ctx, VideoStorageSettings{
		Enabled: true, Endpoint: "https://oss-cn-shanghai.aliyuncs.com", Region: "cn-shanghai",
		Bucket: "video-bucket", AccessKeyID: "video-ak", AccessKeySecret: "video-secret",
		Prefix: "videos", UserIDs: []int64{41},
	})
	require.NoError(t, err)
	require.True(t, svc.SelectedForUser(41))
	require.Len(t, *built, 1)
	require.Equal(t, "https://oss-cn-shanghai.aliyuncs.com", (*built)[0].Endpoint)
	require.Equal(t, "cn-shanghai", (*built)[0].Region)
	require.Equal(t, "video-bucket", (*built)[0].Bucket)
	require.Equal(t, "video-ak", (*built)[0].AccessKeyID)
	require.Equal(t, "video-secret", (*built)[0].AccessKeySecret)

	err = svc.TestConnection(ctx, VideoStorageSettings{
		Endpoint: "https://oss-cn-shanghai.aliyuncs.com", Region: "cn-shanghai",
		Bucket: "video-bucket", AccessKeyID: "video-ak", Prefix: "videos",
	})
	require.NoError(t, err)
	require.Equal(t, 1, store.headBucketCalls)
}

func TestVideoStorageSettingsRejectsIncompleteEnabledConfig(t *testing.T) {
	svc, _, _, _ := newVideoStorageFixture(t)
	_, err := svc.Update(context.Background(), VideoStorageSettings{Enabled: true, Bucket: "only-a-bucket"})
	require.ErrorIs(t, err, ErrVideoStorageIncomplete)
}

func TestXiaoVideoPersistsCompletedContentToOSSBeforeServingIt(t *testing.T) {
	repo := newVideoRepositoryStub()
	repo.jobs["vidjob_oss"] = &VideoJob{
		JobID: "vidjob_oss", UpstreamJobID: "upstream-oss", AccountID: 42, UserID: 11, APIKeyID: 22,
		Status: "completed", Model: "video-public", Resolution: "720p", Duration: 4, AspectRatio: "16:9",
		Amount: 1, Currency: "USD", SettlementStatus: "captured", StorageRequested: true,
		UpstreamResponse: []byte(`{"status":"completed"}`), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	accounts := &videoAccountRepoStub{accounts: []Account{videoTestAccount(42, 7, "https://upstream.example.test/v1")}}
	upstreamCalls := 0
	upstream := &videoHTTPUpstreamStub{do: func(req *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
		require.Equal(t, int64(42), accountID)
		upstreamCalls++
		require.Equal(t, http.MethodGet, req.Method)
		require.Contains(t, []string{
			"/v1/videos/jobs/upstream-oss/content",
			"/v1/videos/jobs/upstream-oss-late/content",
		}, req.URL.Path)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"video/mp4"}},
			Body:       io.NopCloser(bytes.NewReader([]byte("upstream-video"))),
		}, nil
	}}
	svc := newVideoServiceForTest(repo, accounts, upstream)
	settings, _, _, store := newVideoStorageFixture(t)
	_, err := settings.Update(context.Background(), VideoStorageSettings{
		Enabled: true, Bucket: "video-bucket", Endpoint: "https://oss-cn-hangzhou.aliyuncs.com",
		Region: "cn-hangzhou", AccessKeyID: "ak", AccessKeySecret: "secret", UserIDs: []int64{11},
	})
	require.NoError(t, err)
	svc.SetVideoStorageSettingService(settings)

	resp, err := svc.OpenContent(context.Background(), VideoOwner{UserID: 11, APIKeyID: 22, GroupID: videoInt64Ptr(7)}, "vidjob_oss", "", "")
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, []byte("upstream-video"), body)
	require.Equal(t, 1, upstreamCalls)
	require.Equal(t, []string{"videos/vidjob_oss.mp4"}, store.uploadedKeys)
	require.Equal(t, []string{"videos/vidjob_oss.mp4"}, store.openedKeys)
	require.Equal(t, "oss", repo.jobs["vidjob_oss"].StorageProvider)
	require.Equal(t, "videos/vidjob_oss.mp4", repo.jobs["vidjob_oss"].StorageKey)

	_, err = settings.Update(context.Background(), VideoStorageSettings{
		Enabled: false, Bucket: "video-bucket", Endpoint: "https://oss-cn-hangzhou.aliyuncs.com",
		AccessKeyID: "ak", UserIDs: []int64{11},
	})
	require.NoError(t, err)
	resp, err = svc.OpenContent(context.Background(), VideoOwner{UserID: 11, APIKeyID: 22, GroupID: videoInt64Ptr(7)}, "vidjob_oss", "", "")
	require.NoError(t, err)
	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, []byte("upstream-video"), body)
	require.Equal(t, 1, upstreamCalls)

	// Turning off future uploads must not cancel a job that already recorded
	// StorageRequested=true before it finished.
	repo.jobs["vidjob_oss_late"] = &VideoJob{
		JobID: "vidjob_oss_late", UpstreamJobID: "upstream-oss-late", AccountID: 42, UserID: 11, APIKeyID: 22,
		Status: "completed", Model: "video-public", Resolution: "720p", Duration: 4, AspectRatio: "16:9",
		Amount: 1, Currency: "USD", SettlementStatus: "captured", StorageRequested: true,
		UpstreamResponse: []byte(`{"status":"completed"}`), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	resp, err = svc.OpenContent(context.Background(), VideoOwner{UserID: 11, APIKeyID: 22, GroupID: videoInt64Ptr(7)}, "vidjob_oss_late", "", "")
	require.NoError(t, err)
	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, []byte("upstream-video"), body)
	require.Equal(t, 2, upstreamCalls)
	require.Equal(t, []string{"videos/vidjob_oss.mp4", "videos/vidjob_oss_late.mp4"}, store.uploadedKeys)
	require.Equal(t, []string{"videos/vidjob_oss.mp4", "videos/vidjob_oss.mp4", "videos/vidjob_oss_late.mp4"}, store.openedKeys)
	require.Equal(t, "oss", repo.jobs["vidjob_oss_late"].StorageProvider)
	require.Equal(t, "videos/vidjob_oss_late.mp4", repo.jobs["vidjob_oss_late"].StorageKey)
}
