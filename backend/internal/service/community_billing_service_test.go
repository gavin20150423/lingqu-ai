package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type communityBillingSweeperStub struct {
	mu    sync.Mutex
	calls int
	done  chan struct{}
}

func (s *communityBillingSweeperStub) SweepCommunityBilling(context.Context, time.Time) error {
	s.mu.Lock()
	s.calls++
	s.mu.Unlock()
	select {
	case s.done <- struct{}{}:
	default:
	}
	return nil
}

func (s *communityBillingSweeperStub) SweepCommunityStoreReservations(context.Context, time.Time) error {
	return nil
}

func TestCommunityBillingServiceRunsImmediatelyAndStops(t *testing.T) {
	repo := &communityBillingSweeperStub{done: make(chan struct{}, 1)}
	service := NewCommunityBillingService(repo, time.Hour)
	service.Start()

	select {
	case <-repo.done:
	case <-time.After(time.Second):
		t.Fatal("community billing sweep did not run on startup")
	}
	service.Stop()
	repo.mu.Lock()
	require.Equal(t, 1, repo.calls)
	repo.mu.Unlock()
}
