package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFinalizeOperationalOrDegradedKeepsSlowSuccessfulCheckHealthy(t *testing.T) {
	result := finalizeOperationalOrDegraded(&CheckResult{}, 7*time.Second, 7000)

	require.Equal(t, MonitorStatusOperational, result.Status)
	require.Equal(t, "slow response: 7000ms", result.Message)
}
