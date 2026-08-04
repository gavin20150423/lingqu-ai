package migrations

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccountLeaseGroupBindingMigrationRemainsImmutable(t *testing.T) {
	content, err := FS.ReadFile("144_account_lease_group_binding.sql")
	require.NoError(t, err)

	sum := sha256.Sum256([]byte(strings.TrimSpace(string(content))))
	require.Equal(t, "f450fdf2ca73f95f2380f769a8c6f6c736bd29736321fed640ccd64afc2269ae", hex.EncodeToString(sum[:]))
}
