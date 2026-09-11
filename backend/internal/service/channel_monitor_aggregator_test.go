package service

import (
	"testing"
	"time"
)

func TestAvailabilityPercentDistinguishesNoSamplesFromZeroPercent(t *testing.T) {
	if got := availabilityPercent(nil); got != nil {
		t.Fatalf("nil row returned %v, want nil", *got)
	}
	if got := availabilityPercent(&ChannelMonitorAvailability{}); got != nil {
		t.Fatalf("zero samples returned %v, want nil", *got)
	}

	got := availabilityPercent(&ChannelMonitorAvailability{
		TotalChecks:     4,
		AvailabilityPct: 0,
	})
	if got == nil || *got != 0 {
		t.Fatalf("measured zero percent returned %v, want pointer to 0", got)
	}
}

func TestBuildStatusSummaryLeavesAvailabilityEmptyWithoutWindowSamples(t *testing.T) {
	summary := buildStatusSummary(nil, nil, "gpt-5.4", nil)
	if summary.Availability7d != nil {
		t.Fatalf("availability = %v, want nil", *summary.Availability7d)
	}
}

func TestBuildUserViewFallsBackToLatestProbeWhenTimelineIsEmpty(t *testing.T) {
	checkedAt := time.Date(2026, 9, 9, 8, 30, 0, 0, time.UTC)
	latency := 842
	ping := 17
	latest := &ChannelMonitorLatest{
		Model:         "gpt-5.5",
		Status:        MonitorStatusOperational,
		LatencyMs:     &latency,
		PingLatencyMs: &ping,
		CheckedAt:     checkedAt,
	}

	view := buildUserViewFromSummary(
		&ChannelMonitor{ID: 7, PrimaryModel: "gpt-5.5"},
		MonitorStatusSummary{PrimaryStatus: MonitorStatusOperational},
		latest,
		nil,
	)

	if len(view.Timeline) != 1 {
		t.Fatalf("timeline length = %d, want 1 latest-probe fallback", len(view.Timeline))
	}
	point := view.Timeline[0]
	if point.Status != latest.Status || !point.CheckedAt.Equal(checkedAt) {
		t.Fatalf("timeline fallback = %#v, want latest probe %#v", point, latest)
	}
	if point.LatencyMs == nil || *point.LatencyMs != latency || point.PingLatencyMs == nil || *point.PingLatencyMs != ping {
		t.Fatalf("timeline fallback lost latency data: %#v", point)
	}
}
