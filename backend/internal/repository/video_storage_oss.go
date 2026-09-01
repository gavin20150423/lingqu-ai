package repository

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	aliyunoss "github.com/aliyun/aliyun-oss-go-sdk/oss"

	"github.com/Wei-Shaw/sub2api/internal/pkg/servertiming"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// aliyunOSSVideoObjectStore is deliberately backed by Alibaba Cloud's native
// OSS SDK. Video persistence must not inherit the backup S3 client or its
// S3-compatible endpoint behavior.
type aliyunOSSVideoObjectStore struct {
	bucket *aliyunoss.Bucket
}

func NewAliyunOSSVideoObjectStoreFactory() service.VideoObjectStoreFactory {
	return func(ctx context.Context, cfg *service.AliyunOSSConfig) (service.VideoObjectStore, error) {
		if cfg == nil || !cfg.IsConfigured() {
			return nil, service.ErrVideoStorageIncomplete
		}
		client, err := aliyunoss.New(
			cfg.Endpoint,
			cfg.AccessKeyID,
			cfg.AccessKeySecret,
			aliyunoss.Region(cfg.Region),
			aliyunoss.UserAgent("SubPilot-video-aliyun-oss"),
		)
		if err != nil {
			return nil, fmt.Errorf("create Alibaba Cloud OSS client: %w", err)
		}
		bucket, err := client.Bucket(cfg.Bucket)
		if err != nil {
			return nil, fmt.Errorf("open Alibaba Cloud OSS bucket: %w", err)
		}
		return &aliyunOSSVideoObjectStore{bucket: bucket}, nil
	}
}

func (s *aliyunOSSVideoObjectStore) Upload(ctx context.Context, key string, body io.Reader, contentType string, contentLength, maxBytes int64) error {
	if contentLength > maxBytes && maxBytes > 0 {
		return fmt.Errorf("video object exceeds %d bytes", maxBytes)
	}
	if contentLength < 0 && maxBytes > 0 {
		body = &maxBytesReader{reader: body, max: maxBytes}
	}
	options := []aliyunoss.Option{
		aliyunoss.WithContext(ctx),
		aliyunoss.ContentType(contentType),
		aliyunoss.ACL(aliyunoss.ACLPrivate),
	}
	if contentLength >= 0 {
		options = append(options, aliyunoss.ContentLength(contentLength))
	}
	finish := servertiming.ObserveDependency(ctx, "aliyun_oss")
	err := s.bucket.PutObject(key, body, options...)
	finish()
	if err != nil {
		return fmt.Errorf("Alibaba Cloud OSS PutObject: %w", err)
	}
	return nil
}

func (s *aliyunOSSVideoObjectStore) Open(ctx context.Context, key, rangeHeader string) (*http.Response, error) {
	options := []aliyunoss.Option{aliyunoss.WithContext(ctx)}
	rangeValue, err := normalizeAliyunOSSRange(rangeHeader)
	if err != nil {
		return nil, err
	}
	if rangeValue != "" {
		options = append(options, aliyunoss.NormalizedRange(rangeValue))
	}
	finish := servertiming.ObserveDependency(ctx, "aliyun_oss")
	result, err := s.bucket.DoGetObject(&aliyunoss.GetObjectRequest{ObjectKey: key}, options)
	finish()
	if err != nil {
		return nil, fmt.Errorf("Alibaba Cloud OSS GetObject: %w", err)
	}
	if result == nil || result.Response == nil || result.Response.Body == nil {
		return nil, errors.New("Alibaba Cloud OSS returned an empty object response")
	}
	header := cloneObjectHeaders(result.Response.Headers)
	contentLength := int64(-1)
	if value := strings.TrimSpace(header.Get("Content-Length")); value != "" {
		if parsed, parseErr := strconv.ParseInt(value, 10, 64); parseErr == nil && parsed >= 0 {
			contentLength = parsed
		}
	}
	return &http.Response{
		StatusCode:    result.Response.StatusCode,
		Header:        header,
		ContentLength: contentLength,
		Body:          result.Response.Body,
	}, nil
}

func (s *aliyunOSSVideoObjectStore) HeadBucket(ctx context.Context) error {
	finish := servertiming.ObserveDependency(ctx, "aliyun_oss")
	_, err := s.bucket.Client.GetBucketInfo(s.bucket.BucketName, aliyunoss.WithContext(ctx))
	finish()
	if err != nil {
		return fmt.Errorf("Alibaba Cloud OSS GetBucketInfo: %w", err)
	}
	return nil
}

func cloneObjectHeaders(source http.Header) http.Header {
	header := make(http.Header, len(source))
	for key, values := range source {
		header[key] = append([]string(nil), values...)
	}
	return header
}

// normalizeAliyunOSSRange converts the HTTP Range header form (bytes=0-99)
// to the Alibaba SDK form (0-99). The SDK option only supports one range;
// reject malformed or multipart ranges instead of silently downloading a full
// object.
func normalizeAliyunOSSRange(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	if len(raw) >= len("bytes=") && strings.EqualFold(raw[:len("bytes=")], "bytes=") {
		raw = strings.TrimSpace(raw[len("bytes="):])
	}
	parts := strings.Split(raw, "-")
	if len(parts) != 2 || (parts[0] == "" && parts[1] == "") {
		return "", errors.New("invalid video range; Alibaba Cloud OSS supports one byte range")
	}
	for _, part := range parts {
		for _, char := range part {
			if char < '0' || char > '9' {
				return "", errors.New("invalid video range; Alibaba Cloud OSS expects bytes=start-end")
			}
		}
	}
	return raw, nil
}

type maxBytesReader struct {
	reader io.Reader
	read   int64
	max    int64
}

func (r *maxBytesReader) Read(p []byte) (int, error) {
	if r.read < r.max {
		remaining := r.max - r.read
		if int64(len(p)) > remaining {
			p = p[:remaining]
		}
		n, err := r.reader.Read(p)
		r.read += int64(n)
		return n, err
	}
	var extra [1]byte
	n, err := r.reader.Read(extra[:])
	if n > 0 || err == nil {
		return 0, errors.New("video object exceeds configured size limit")
	}
	return 0, err
}

var _ service.VideoObjectStore = (*aliyunOSSVideoObjectStore)(nil)
