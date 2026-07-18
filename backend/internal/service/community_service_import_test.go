package service

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestNormalizeImportedCredentialAcceptsSupportedOAuthShapes(t *testing.T) {
	openAI, err := normalizeImportedCredential("openai", "plus", "rt_abcdefghijklmnopqrstuvwxyz")
	require.NoError(t, err)
	require.JSONEq(t, `{"refresh_token":"rt_abcdefghijklmnopqrstuvwxyz"}`, openAI)

	claude, err := normalizeImportedCredential("anthropic", "oauth", "sk-ant-sid01-abcdefghijklmnopqrstuvwxyz")
	require.NoError(t, err)
	require.JSONEq(t, `{"session_key":"sk-ant-sid01-abcdefghijklmnopqrstuvwxyz"}`, claude)

	jsonCredential, err := normalizeImportedCredential("openai", "team", map[string]any{
		"refresh_token": "rt_team_abcdefghijklmnopqrstuvwxyz",
		"access_token":  "oauth-access-token",
	})
	require.NoError(t, err)
	require.Contains(t, jsonCredential, "refresh_token")
}

func TestNormalizeImportedCredentialRejectsReferenceSiteForbiddenInputs(t *testing.T) {
	tests := []struct {
		name, provider, tier string
		value                any
	}{
		{name: "openai api key", provider: "openai", tier: "plus", value: "sk-proj-secret-api-key"},
		{name: "url", provider: "openai", tier: "plus", value: "https://upstream.example/v1"},
		{name: "pro credential import", provider: "openai", tier: "pro", value: "rt_abcdefghijklmnopqrstuvwxyz"},
		{name: "nested cookie", provider: "openai", tier: "team", value: map[string]any{"refresh_token": "rt_abcdefghijklmnopqrstuvwxyz", "extra": map[string]any{"cookie": "session=secret"}}},
		{name: "upstream config", provider: "anthropic", tier: "oauth", value: map[string]any{"session_key": "sk-ant-sid01-abcdefghijklmnopqrstuvwxyz", "upstream": "custom"}},
		{name: "missing refresh token", provider: "openai", tier: "plus", value: map[string]any{"access_token": "only-access"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := normalizeImportedCredential(tt.provider, tt.tier, tt.value)
			require.ErrorIs(t, err, ErrCommunityInvalid)
		})
	}
}

func TestNormalizeAccountInputUsesEmptyPostgresArraysForOptionalLists(t *testing.T) {
	in := CommunityAccountInput{
		Name:            "OpenAI Plus",
		Provider:        "openai",
		OAuthCredential: `{"refresh_token":"rt_test"}`,
		AccountTier:     "plus",
	}

	require.NoError(t, normalizeAccountInput(&in))
	require.NotNil(t, in.Tags)
	require.Empty(t, in.Tags)
	require.NotNil(t, in.SupportedModels)
	require.Empty(t, in.SupportedModels)
}

func TestImportAccountsRejectsMoreThanOneHundredBeforeWriting(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	items := make([]CommunityAccountImportItem, 101)
	for i := range items {
		items[i].Credential = "rt_abcdefghijklmnopqrstuvwxyz"
	}

	service := NewCommunityService(db, communityTestEncryptor{})
	_, err = service.ImportAccounts(context.Background(), 17, CommunityAccountImportInput{Provider: "openai", AccountTier: "plus", Items: items})

	require.ErrorIs(t, err, ErrCommunityInvalid)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestImportAnthropicAccountsRequiresProxy(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	service := NewCommunityService(db, communityTestEncryptor{})
	_, err = service.ImportAccounts(context.Background(), 17, CommunityAccountImportInput{
		Provider: "anthropic",
		Items:    []CommunityAccountImportItem{{Credential: "sk-ant-sid01-example-session-key"}},
	})

	require.ErrorIs(t, err, ErrCommunityInvalid)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBatchUpdateAccountsScopesWriteToOwner(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	shareMode := "public"
	mock.ExpectExec(regexp.QuoteMeta("UPDATE community_accounts SET share_mode=$1,updated_at=NOW() WHERE owner_user_id=$2 AND id=ANY($3) AND deleted_at IS NULL")).
		WithArgs("public", int64(17), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 2))

	service := NewCommunityService(db, communityTestEncryptor{})
	count, err := service.BatchUpdateAccounts(context.Background(), 17, CommunityAccountBatchUpdateInput{IDs: []int64{1, 2, 99}, ShareMode: &shareMode})

	require.NoError(t, err)
	require.EqualValues(t, 2, count)
	require.NoError(t, mock.ExpectationsWereMet())
}
