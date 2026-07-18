package service

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestCommunityProxyOwnershipIsolation(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	proxyID := int64(41)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT 1 FROM proxies WHERE id=$1 AND owner_user_id=$2 AND status='active' AND deleted_at IS NULL")).
		WithArgs(proxyID, int64(17)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}))

	service := NewCommunityService(db, communityTestEncryptor{})
	err = service.CanUseProxy(context.Background(), 17, &proxyID)

	require.ErrorIs(t, err, ErrCommunityForbidden)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCommunityProxyCannotDeleteWhileReferenced(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectExec(`UPDATE proxies p SET deleted_at=NOW\(\),status='disabled',updated_at=NOW\(\) WHERE`).
		WithArgs(int64(41), int64(17)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	service := NewCommunityService(db, communityTestEncryptor{})
	err = service.DeleteProxy(context.Background(), 17, 41)

	require.ErrorIs(t, err, ErrCommunityConflict)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCommunityProxyUpdateKeepsExistingPasswordWhenEmpty(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE proxies SET name=$1,ip_type=$2,protocol=$3,host=$4,port=$5,username=NULLIF($6,''),password=CASE WHEN $7='' THEN password ELSE $7 END,updated_at=NOW() WHERE id=$8 AND owner_user_id=$9 AND deleted_at IS NULL")).
		WithArgs("家庭代理", "ipv4", "socks5h", "127.0.0.1", 1080, "proxy-user", "", int64(41), int64(17)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT p\.id,p\.name,p\.ip_type,p\.protocol,p\.host,p\.port`).
		WithArgs(int64(17)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "ip_type", "protocol", "host", "port", "username", "has_password", "account_count", "created_at"}).
			AddRow(int64(41), "家庭代理", "ipv4", "socks5h", "127.0.0.1", 1080, "proxy-user", true, 0, now))

	service := NewCommunityService(db, communityTestEncryptor{})
	proxy, err := service.SaveProxy(context.Background(), 17, CommunityProxyInput{
		ID: 41, Name: "家庭代理", IPType: "ipv4", Protocol: "socks5h",
		Host: "127.0.0.1", Port: 1080, Username: "proxy-user",
	})

	require.NoError(t, err)
	require.True(t, proxy.HasPassword)
	require.NoError(t, mock.ExpectationsWereMet())
}
