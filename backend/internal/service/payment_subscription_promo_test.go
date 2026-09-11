package service

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/ent/paymentauditlog"
	"github.com/Wei-Shaw/sub2api/ent/promocode"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestReserveSubscriptionPromoCodeClaimsOneSlotAndRejectsTheNextOrder(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	promo, err := client.PromoCode.Create().
		SetCode("SUB-ONE-USE").
		SetMaxUses(1).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.ExecContext(ctx, "UPDATE promo_codes SET applies_to_subscriptions=$1 WHERE id=$2", true, promo.ID)
	require.NoError(t, err)

	svc := &PaymentService{entClient: client}
	tx, err := client.Tx(ctx)
	require.NoError(t, err)
	require.NoError(t, svc.reserveSubscriptionPromoCode(ctx, tx, 1001, promo.Code))
	require.NoError(t, tx.Commit())

	promo, err = client.PromoCode.Get(ctx, promo.ID)
	require.NoError(t, err)
	require.Equal(t, 1, promo.UsedCount)

	tx, err = client.Tx(ctx)
	require.NoError(t, err)
	err = svc.reserveSubscriptionPromoCode(ctx, tx, 1002, promo.Code)
	require.Error(t, err)
	require.Equal(t, "PROMO_CODE_INVALID", infraerrors.Reason(err))
	require.NoError(t, tx.Rollback())

	count, err := client.PaymentAuditLog.Query().
		Where(paymentauditlog.OrderIDEQ("1001"), paymentauditlog.ActionEQ(subscriptionPromoReserved)).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestReleaseSubscriptionPromoReservationIsIdempotent(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	promo, err := client.PromoCode.Create().
		SetCode("SUB-RELEASE").
		SetMaxUses(1).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.ExecContext(ctx, "UPDATE promo_codes SET applies_to_subscriptions=$1 WHERE id=$2", true, promo.ID)
	require.NoError(t, err)

	user, err := client.User.Create().
		SetEmail("promo-release@example.com").
		SetPasswordHash("hash").
		SetUsername("promo-release").
		Save(ctx)
	require.NoError(t, err)
	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(10).
		SetPayAmount(9).
		SetRechargeCode("PAY-PROMO-RELEASE").
		SetOutTradeNo("promo_release_order").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("").
		SetOrderType(payment.OrderTypeSubscription).
		SetStatus(OrderStatusPending).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetClientIP("127.0.0.1").
		SetSrcHost("example.com").
		Save(ctx)
	require.NoError(t, err)
	_, err = client.ExecContext(ctx, "UPDATE payment_orders SET promo_code=$1 WHERE id=$2", promo.Code, order.ID)
	require.NoError(t, err)
	order, err = client.PaymentOrder.Get(ctx, order.ID)
	require.NoError(t, err)

	svc := &PaymentService{entClient: client}
	tx, err := client.Tx(ctx)
	require.NoError(t, err)
	require.NoError(t, svc.reserveSubscriptionPromoCode(ctx, tx, order.ID, promo.Code))
	require.NoError(t, tx.Commit())

	require.NoError(t, svc.releaseSubscriptionPromoReservation(ctx, order))
	require.NoError(t, svc.releaseSubscriptionPromoReservation(ctx, order))

	promo, err = client.PromoCode.Query().Where(promocode.IDEQ(promo.ID)).Only(ctx)
	require.NoError(t, err)
	require.Zero(t, promo.UsedCount)
	releasedCount, err := client.PaymentAuditLog.Query().
		Where(paymentauditlog.OrderIDEQ(strconv.FormatInt(order.ID, 10)), paymentauditlog.ActionEQ(subscriptionPromoReleased)).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, releasedCount)
}

func TestConsumeSubscriptionPromoCodeUsesReservationWithoutIncrementingAgain(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	promo, err := client.PromoCode.Create().
		SetCode("SUB-FULFILL").
		SetMaxUses(1).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.ExecContext(ctx, "UPDATE promo_codes SET applies_to_subscriptions=$1 WHERE id=$2", true, promo.ID)
	require.NoError(t, err)

	user, err := client.User.Create().
		SetEmail("promo-fulfill@example.com").
		SetPasswordHash("hash").
		SetUsername("promo-fulfill").
		Save(ctx)
	require.NoError(t, err)
	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(20).
		SetPayAmount(17).
		SetRechargeCode("PAY-PROMO-FULFILL").
		SetOutTradeNo("promo_fulfill_order").
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("").
		SetOrderType(payment.OrderTypeSubscription).
		SetStatus(OrderStatusPaid).
		SetExpiresAt(time.Now().Add(time.Hour)).
		SetClientIP("127.0.0.1").
		SetSrcHost("example.com").
		Save(ctx)
	require.NoError(t, err)
	_, err = client.ExecContext(ctx, "UPDATE payment_orders SET promo_code=$1 WHERE id=$2", promo.Code, order.ID)
	require.NoError(t, err)
	order, err = client.PaymentOrder.Get(ctx, order.ID)
	require.NoError(t, err)

	svc := &PaymentService{entClient: client}
	tx, err := client.Tx(ctx)
	require.NoError(t, err)
	require.NoError(t, svc.reserveSubscriptionPromoCode(ctx, tx, order.ID, promo.Code))
	require.NoError(t, tx.Commit())

	require.NoError(t, svc.consumeSubscriptionPromoCode(ctx, order))
	promo, err = client.PromoCode.Get(ctx, promo.ID)
	require.NoError(t, err)
	require.Equal(t, 1, promo.UsedCount)
	appliedCount, err := client.PaymentAuditLog.Query().
		Where(paymentauditlog.OrderIDEQ(strconv.FormatInt(order.ID, 10)), paymentauditlog.ActionEQ(subscriptionPromoApplied)).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, appliedCount)
}
