package service

import "testing"

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
