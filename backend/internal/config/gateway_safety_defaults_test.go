package config

import (
	"testing"

	"github.com/spf13/viper"
)

func TestGatewayCompatibilitySwitchesDefaultToSafeProductionSemantics(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	setDefaults()

	if viper.GetBool("gateway.ignore_account_cooldowns") {
		t.Fatal("gateway must respect persisted account cooldowns by default")
	}
	if viper.GetBool("gateway.passthrough_upstream_errors") {
		t.Fatal("gateway must retain normalized client errors by default")
	}
}
