package service

import (
	"context"
	"errors"
	"time"
)

var (
	ErrOpenAIImageAsyncTaskNotFound  = errors.New("openai image async task not found")
	ErrOpenAIImageAsyncTaskStoreFull = errors.New("openai image async task store full")
)

type OpenAIImageAsyncTaskStatus string

const (
	OpenAIImageAsyncTaskQueued    OpenAIImageAsyncTaskStatus = "queued"
	OpenAIImageAsyncTaskRunning   OpenAIImageAsyncTaskStatus = "running"
	OpenAIImageAsyncTaskCompleted OpenAIImageAsyncTaskStatus = "completed"
	OpenAIImageAsyncTaskFailed    OpenAIImageAsyncTaskStatus = "failed"
)

type OpenAIImageAsyncTask struct {
	ID             string
	UserID         int64
	APIKeyID       int64
	Status         OpenAIImageAsyncTaskStatus
	Request        *OpenAIImageAsyncTaskRequest
	ResultStatus   int
	ResultBody     []byte
	ResultMimeType string
	ErrorMessage   string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	FinishedAt     time.Time
}

type OpenAIImageAsyncTaskRequest struct {
	Method       string
	URL          string
	Header       map[string][]string
	Body         []byte
	APIKey       *APIKey
	Subject      OpenAIImageAsyncTaskSubject
	Role         string
	Subscription *UserSubscription
	Endpoint     string
}

type OpenAIImageAsyncTaskSubject struct {
	UserID      int64
	Concurrency int
}

type OpenAIImageAsyncTaskStore interface {
	Enqueue(ctx context.Context, request *OpenAIImageAsyncTaskRequest) (*OpenAIImageAsyncTask, error)
	Dequeue(ctx context.Context, wait time.Duration) (*OpenAIImageAsyncTask, error)
	RequeueStaleRunning(ctx context.Context, staleAfter time.Duration) (int64, error)
	MarkRunning(ctx context.Context, taskID string, owner string, leaseTTL time.Duration) (bool, error)
	RenewRunning(ctx context.Context, taskID string, owner string, leaseTTL time.Duration) error
	MarkCompleted(ctx context.Context, taskID string, statusCode int, body []byte, contentType string) error
	MarkFailed(ctx context.Context, taskID string, message string) error
	Get(ctx context.Context, taskID string) (*OpenAIImageAsyncTask, error)
}
