package service

import (
	"context"
	"database/sql/driver"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

type communityTestEncryptor struct{}

func (communityTestEncryptor) Encrypt(value string) (string, error) { return value, nil }
func (communityTestEncryptor) Decrypt(value string) (string, error) { return value, nil }

func TestBuyProductWithPointsDeductsPointsAndPersistsSource(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	now := time.Now()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id,category,name,description,price,points_price,fulfillment_type,fulfillment_value,status,0,sort_order FROM store_products WHERE id=$1 AND status='active' FOR UPDATE")).
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "category", "name", "description", "price", "points_price", "fulfillment_type", "fulfillment_value", "status", "stock", "sort_order"}).
			AddRow(int64(9), "权益", "会员权益", "", 8.0, 120.0, "entitlement", 1.0, "active", 0, 0))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET points=points-$1,updated_at=NOW() WHERE id=$2 AND points>=$1")).
		WithArgs(240.0, int64(17)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`INSERT INTO store_orders\(order_no,user_id,product_id,quantity,unit_price,total_amount,payment_source,status,paid_at\)VALUES`).
		WithArgs(sqlmock.AnyArg(), int64(17), int64(9), 2, 120.0, 240.0, "points").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(31)))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET concurrency=concurrency+$1,updated_at=NOW() WHERE id=$2")).
		WithArgs(2, int64(17)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE store_orders SET status='fulfilled',delivery=$2,fulfilled_at=NOW() WHERE id=$1")).
		WithArgs(int64(31), jsonBytesMatcher{}).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`INSERT INTO support_tickets\(user_id,subject,category,priority,status,user_unread,admin_unread\)`).
		WithArgs(int64(17), "商城订单已交付", "store").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(81)))
	mock.ExpectExec(`INSERT INTO support_ticket_messages\(ticket_id,author_user_id,author_role,content\)`).
		WithArgs(int64(81), int64(17), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(`SELECT o\.id,o\.order_no.*o\.payment_source,o\.status,o\.delivery,o\.created_at`).
		WithArgs(int64(31), int64(17)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "order_no", "user_id", "product_id", "name", "quantity", "unit_price", "total_amount", "payment_source", "status", "delivery", "created_at"}).
			AddRow(int64(31), "SC1", int64(17), int64(9), "会员权益", 2, 120.0, 240.0, "points", "fulfilled", []byte(`["负载额度到账 +2"]`), now))

	service := NewCommunityService(db, communityTestEncryptor{})
	order, err := service.BuyProduct(context.Background(), 17, 9, 2, "points")

	require.NoError(t, err)
	require.Equal(t, "points", order.PaymentSource)
	require.Equal(t, 240.0, order.TotalAmount)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFulfillPlatformStoreEntitlementCreditsConcurrency(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT o\.user_id,o\.product_id,o\.quantity,o\.status,p\.fulfillment_type,p\.fulfillment_value`).
		WithArgs(int64(31), int64(91)).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "product_id", "quantity", "status", "fulfillment_type", "fulfillment_value"}).
			AddRow(int64(17), int64(9), 2, "pending", "entitlement", 1.0))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET concurrency=concurrency+$1,updated_at=NOW() WHERE id=$2")).
		WithArgs(2, int64(17)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE store_orders SET status='fulfilled'`).
		WithArgs(int64(31), jsonBytesMatcher{}).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`INSERT INTO support_tickets\(user_id,subject,category,priority,status,user_unread,admin_unread\)`).
		WithArgs(int64(17), "商城订单已交付", "store").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(81)))
	mock.ExpectExec(`INSERT INTO support_ticket_messages\(ticket_id,author_user_id,author_role,content\)`).
		WithArgs(int64(81), int64(17), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	service := NewCommunityService(db, communityTestEncryptor{})
	require.NoError(t, service.FulfillPlatformStoreOrder(context.Background(), 31, 91))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBuyProductRejectsPointsWhenProductHasNoPointsPrice(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id,category,name,description,price,points_price,fulfillment_type,fulfillment_value,status,0,sort_order FROM store_products WHERE id=$1 AND status='active' FOR UPDATE")).
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "category", "name", "description", "price", "points_price", "fulfillment_type", "fulfillment_value", "status", "stock", "sort_order"}).
			AddRow(int64(9), "卡密", "余额专享", "", 8.0, nil, "card", 0.0, "active", 1, 0))
	mock.ExpectRollback()

	service := NewCommunityService(db, communityTestEncryptor{})
	_, err = service.BuyProduct(context.Background(), 17, 9, 1, "points")

	require.ErrorIs(t, err, ErrCommunityInvalid)
	require.NoError(t, mock.ExpectationsWereMet())
}

type jsonBytesMatcher struct{}

func (jsonBytesMatcher) Match(value driver.Value) bool {
	_, ok := value.([]byte)
	return ok
}
