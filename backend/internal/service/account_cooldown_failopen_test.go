package service

import (
	"context"
	"testing"
	"time"
)

func TestAccountIsSchedulableIgnoringCooldownsKeepsTransientMarkersEligible(t *testing.T) {
	future := time.Now().Add(time.Hour)
	account := &Account{
		Status:                 StatusActive,
		Schedulable:            true,
		RateLimitResetAt:       &future,
		OverloadUntil:          &future,
		TempUnschedulableUntil: &future,
	}
	if account.IsSchedulable() {
		t.Fatal("test account must be blocked by the normal cooldown predicate")
	}
	if !account.IsSchedulableIgnoringCooldowns() {
		t.Fatal("transient cooldown markers must not block gateway dispatch")
	}
	if !account.IsSchedulableForModelIgnoringCooldowns(context.Background(), "gpt-5.5") {
		t.Fatal("model selection must use the fail-open account predicate")
	}
}
