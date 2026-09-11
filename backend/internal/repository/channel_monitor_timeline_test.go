package repository

import (
	"strings"
	"testing"
)

func TestRecentMonitorTimelineKeepsLatestProbeResultsRegardlessOfAge(t *testing.T) {
	const timeCutoff = "NOW() - INTERVAL '1 hour'"
	if strings.Contains(listRecentHistoryForMonitorsQuery, timeCutoff) {
		t.Fatalf("timeline query must not expire probe results with %q", timeCutoff)
	}

	for _, clause := range []string{
		"ROW_NUMBER() OVER (PARTITION BY h.monitor_id ORDER BY h.checked_at DESC)",
		"WHERE rn <= $3",
		"ORDER BY monitor_id, checked_at DESC",
	} {
		if !strings.Contains(listRecentHistoryForMonitorsQuery, clause) {
			t.Fatalf("timeline query must contain %q", clause)
		}
	}
}
