package repository

import (
	"context"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
)

func TestUniqueInt64sPreserveOrder(t *testing.T) {
	got := uniqueInt64sPreserveOrder([]int64{16, 4, 16, 9, 4, 9})
	want := []int64{16, 4, 9}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("uniqueInt64sPreserveOrder() = %v, want %v", got, want)
	}
}

func TestUniqueInt64sPreserveOrderReturnsShortInputUnchanged(t *testing.T) {
	input := []int64{7}
	got := uniqueInt64sPreserveOrder(input)
	if !reflect.DeepEqual(got, input) {
		t.Fatalf("uniqueInt64sPreserveOrder() = %v, want %v", got, input)
	}
}

func TestBindGroupsUsesIdempotentUpsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	t.Cleanup(func() { _ = client.Close() })

	mock.ExpectBegin()
	// BindGroups 先对本轮涉及的 group 打 FOR SHARE 活体锁（与 group 删除方的
	// FOR UPDATE 互斥），再串行化账号侧改动。返回行数必须等于去重后的 group 数，
	// 否则 lockLiveGroups 会判定 ErrGroupNotFound。
	mock.ExpectQuery(regexp.QuoteMeta(`/* account_group_live_group_lock */`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(2)).AddRow(int64(3)))
	mock.ExpectQuery(`(?s)SELECT .* FROM "accounts".*FOR UPDATE`).
		WithArgs(int64(27)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(27)))
	mock.ExpectQuery(`(?s)SELECT .* FROM "account_groups".*account_id.*`).
		WithArgs(int64(27)).
		WillReturnRows(sqlmock.NewRows([]string{"account_id", "group_id", "priority", "created_at", "auto_managed"}).
			AddRow(int64(27), int64(2), 1, time.Now(), false))
	mock.ExpectExec(`(?s)DELETE FROM "account_groups".*group_id.*NOT IN.*`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(`ON CONFLICT ("account_id", "group_id") DO UPDATE SET "priority" = "excluded"."priority"`)).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectCommit()

	repo := newAccountRepositoryWithSQL(client, nil, nil)
	require.NoError(t, repo.BindGroups(context.Background(), 27, []int64{2, 3, 2}))
	require.NoError(t, mock.ExpectationsWereMet())
}
