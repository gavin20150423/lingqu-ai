package service

import (
	"context"
	"log"
	"sync"
	"time"
)

type CommunityBillingSweeper interface {
	SweepCommunityBilling(ctx context.Context, now time.Time) error
	SweepCommunityStoreReservations(ctx context.Context, now time.Time) error
}

type CommunityBillingService struct {
	repo     CommunityBillingSweeper
	interval time.Duration
	stopCh   chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

func NewCommunityBillingService(repo CommunityBillingSweeper, interval time.Duration) *CommunityBillingService {
	return &CommunityBillingService{repo: repo, interval: interval, stopCh: make(chan struct{})}
}

func (s *CommunityBillingService) Start() {
	if s == nil || s.repo == nil || s.interval <= 0 {
		return
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		s.runOnce()
		for {
			select {
			case <-ticker.C:
				s.runOnce()
			case <-s.stopCh:
				return
			}
		}
	}()
}

func (s *CommunityBillingService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() { close(s.stopCh) })
	s.wg.Wait()
}

func (s *CommunityBillingService) runOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := s.repo.SweepCommunityBilling(ctx, time.Now()); err != nil {
		log.Printf("[CommunityBilling] sweep failed: %v", err)
	}
	if err := s.repo.SweepCommunityStoreReservations(ctx, time.Now()); err != nil {
		log.Printf("[CommunityStore] reservation sweep failed: %v", err)
	}
}
