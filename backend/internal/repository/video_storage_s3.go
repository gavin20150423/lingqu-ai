package repository

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/Wei-Shaw/sub2api/internal/pkg/servertiming"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type s3VideoObjectStore struct {
	client *s3.Client
	bucket string
}

func NewS3VideoObjectStoreFactory() service.VideoObjectStoreFactory {
	return func(ctx context.Context, cfg *service.BackupS3Config) (service.VideoObjectStore, error) {
		client, err := newS3Client(ctx, s3ClientParams{
			Endpoint: cfg.Endpoint, Region: cfg.Region, AccessKeyID: cfg.AccessKeyID,
			SecretAccessKey: cfg.SecretAccessKey, ForcePathStyle: cfg.ForcePathStyle,
		})
		if err != nil {
			return nil, err
		}
		return &s3VideoObjectStore{client: client, bucket: cfg.Bucket}, nil
	}
}

func (s *s3VideoObjectStore) Upload(ctx context.Context, key string, body io.Reader, contentType string, contentLength, maxBytes int64) error {
	if contentLength > maxBytes && maxBytes > 0 {
		return fmt.Errorf("video object exceeds %d bytes", maxBytes)
	}
	if contentLength < 0 && maxBytes > 0 {
		body = &maxBytesReader{reader: body, max: maxBytes}
	}
	input := &s3.PutObjectInput{Bucket: &s.bucket, Key: &key, Body: body, ContentType: &contentType}
	if contentLength >= 0 {
		input.ContentLength = &contentLength
	}
	finish := servertiming.ObserveDependency(ctx, "s3")
	_, err := s.client.PutObject(ctx, input)
	finish()
	if err != nil {
		return fmt.Errorf("OSS PutObject: %w", err)
	}
	return nil
}

func (s *s3VideoObjectStore) Open(ctx context.Context, key, rangeHeader string) (*http.Response, error) {
	input := &s3.GetObjectInput{Bucket: &s.bucket, Key: &key}
	if rangeHeader != "" {
		input.Range = &rangeHeader
	}
	finish := servertiming.ObserveDependency(ctx, "s3")
	result, err := s.client.GetObject(ctx, input)
	finish()
	if err != nil {
		return nil, fmt.Errorf("OSS GetObject: %w", err)
	}
	header := make(http.Header)
	setObjectHeader(header, "Content-Type", result.ContentType)
	setObjectHeader(header, "Content-Range", result.ContentRange)
	setObjectHeader(header, "Accept-Ranges", result.AcceptRanges)
	setObjectHeader(header, "Cache-Control", result.CacheControl)
	setObjectHeader(header, "ETag", result.ETag)
	if result.ContentLength != nil {
		header.Set("Content-Length", strconv.FormatInt(*result.ContentLength, 10))
	}
	if result.LastModified != nil {
		header.Set("Last-Modified", result.LastModified.UTC().Format(http.TimeFormat))
	}
	status := http.StatusOK
	if result.ContentRange != nil && *result.ContentRange != "" {
		status = http.StatusPartialContent
	}
	return &http.Response{StatusCode: status, Header: header, Body: result.Body}, nil
}

func (s *s3VideoObjectStore) HeadBucket(ctx context.Context) error {
	finish := servertiming.ObserveDependency(ctx, "s3")
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: &s.bucket})
	finish()
	if err != nil {
		return fmt.Errorf("OSS HeadBucket: %w", err)
	}
	return nil
}

func setObjectHeader(header http.Header, name string, value *string) {
	if value != nil && *value != "" {
		header.Set(name, *value)
	}
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

var _ service.VideoObjectStore = (*s3VideoObjectStore)(nil)
