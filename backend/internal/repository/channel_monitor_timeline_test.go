package repository

import (
	"strings"
	"testing"
)

func TestRecentMonitorTimelineIsLimitedToOneHour(t *testing.T) {
	const predicate = "h.checked_at >= NOW() - INTERVAL '1 hour'"
	if !strings.Contains(listRecentHistoryForMonitorsQuery, predicate) {
		t.Fatalf("timeline query must contain %q", predicate)
	}
}
