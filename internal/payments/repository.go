package payments

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateOrReuseCheckout(ctx context.Context, orderID string, create CheckoutCreator) (CheckoutStartResult, error) {
	if r == nil || r.pool == nil || create == nil {
		return CheckoutStartResult{}, ErrUnavailable
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return CheckoutStartResult{}, ErrUnavailable
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	order, err := r.checkoutOrderForUpdate(ctx, tx, orderID)
	if err != nil {
		return CheckoutStartResult{}, err
	}

	payment, found, err := r.paymentForOrderForUpdate(ctx, tx, order.ID)
	if err != nil {
		return CheckoutStartResult{}, err
	}
	result := CheckoutStartResult{OrderID: order.ID}
	if order.Status == OrderStatusPaid || (found && payment.Status == PaymentStatusPaid) {
		return result, ErrOrderAlreadyPaid
	}
	if order.Status != OrderStatusPendingPayment {
		return result, ErrOrderNotPayable
	}
	if found && payment.CheckoutURL.Valid && ValidateCheckoutURL(payment.CheckoutURL.String) == nil {
		result.CheckoutURL = payment.CheckoutURL.String
		result.Reused = true
		if err := tx.Commit(ctx); err != nil {
			return CheckoutStartResult{}, ErrUnavailable
		}
		return result, nil
	}

	created, err := create(ctx, order)
	if err != nil {
		return result, err
	}
	if err := ValidateCheckoutURL(created.URL); err != nil {
		return result, err
	}

	if err := r.upsertPendingPayment(ctx, tx, order.ID, order.OrderNSU, created.URL); err != nil {
		return CheckoutStartResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return CheckoutStartResult{}, ErrUnavailable
	}

	log.Printf("payment checkout created order_id=%s reused=false", order.ID)
	result.CheckoutURL = created.URL
	return result, nil
}

func (r *PostgresRepository) PaymentTargetByOrderNSU(ctx context.Context, orderNSU string) (PaymentVerificationTarget, error) {
	if r == nil || r.pool == nil {
		return PaymentVerificationTarget{}, ErrUnavailable
	}

	var target PaymentVerificationTarget
	err := r.pool.QueryRow(ctx, `
		select
			o.id::text,
			o.order_number,
			o.status,
			p.status,
			o.total_cents
		from public.order_payments p
		join public.orders o
			on o.id = p.order_id
		where p.order_nsu = $1
	`, orderNSU).Scan(
		&target.OrderID,
		&target.OrderNumber,
		&target.OrderStatus,
		&target.PaymentStatus,
		&target.TotalCents,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PaymentVerificationTarget{}, ErrPaymentNotFound
		}
		return PaymentVerificationTarget{}, ErrUnavailable
	}

	return target, nil
}

func (r *PostgresRepository) MarkPaid(ctx context.Context, orderNSU string, payment VerifiedPayment, paidAt time.Time) (ReturnResult, error) {
	if r == nil || r.pool == nil {
		return ReturnResult{}, ErrUnavailable
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ReturnResult{}, ErrUnavailable
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	target, err := r.lockPaymentTargetByOrderNSU(ctx, tx, orderNSU)
	if err != nil {
		return ReturnResult{}, err
	}
	if target.OrderStatus == OrderStatusPaid || target.PaymentStatus == PaymentStatusPaid {
		if err := tx.Commit(ctx); err != nil {
			return ReturnResult{}, ErrUnavailable
		}
		return ReturnResult{Status: ReturnStatusConfirmed, OrderID: target.OrderID}, nil
	}
	if target.OrderStatus != OrderStatusPendingPayment {
		return ReturnResult{Status: ReturnStatusUnavailable, OrderID: target.OrderID}, ErrOrderNotPayable
	}
	if payment.AmountCents != target.TotalCents {
		return ReturnResult{Status: ReturnStatusUnavailable, OrderID: target.OrderID}, ErrAmountMismatch
	}

	if _, err := tx.Exec(ctx, `
		update public.order_payments
		set
			status = 'paid',
			invoice_slug = $2,
			transaction_nsu = $3,
			amount_cents = $4,
			paid_amount_cents = $5,
			installments = $6,
			capture_method = $7,
			paid_at = $8,
			updated_at = now()
		where order_nsu = $1
	`, orderNSU, payment.InvoiceSlug, payment.TransactionNSU, payment.AmountCents, payment.PaidAmountCents, intOrNil(payment.Installments), textOrNil(payment.CaptureMethod), paidAt); err != nil {
		return ReturnResult{}, ErrUnavailable
	}

	if _, err := tx.Exec(ctx, `
		update public.orders
		set
			status = 'paid',
			updated_at = now()
		where id = $1::uuid
	`, target.OrderID); err != nil {
		return ReturnResult{}, ErrUnavailable
	}

	if err := tx.Commit(ctx); err != nil {
		return ReturnResult{}, ErrUnavailable
	}

	log.Printf("payment verified order_id=%s order_number=%d", target.OrderID, target.OrderNumber)
	return ReturnResult{Status: ReturnStatusConfirmed, OrderID: target.OrderID}, nil
}

func (r *PostgresRepository) checkoutOrderForUpdate(ctx context.Context, tx pgx.Tx, orderID string) (CheckoutOrder, error) {
	var order CheckoutOrder
	var complement pgtype.Text
	err := tx.QueryRow(ctx, `
		select
			o.id::text,
			o.order_number,
			o.status,
			o.total_cents,
			o.shipping_price_cents,
			shipping.service_name,
			customer.full_name,
			customer.email,
			customer.phone,
			address.postal_code,
			address.street,
			address.number,
			address.complement,
			address.district
		from public.orders o
		join public.order_customer_details customer
			on customer.order_id = o.id
		join public.order_shipping_addresses address
			on address.order_id = o.id
		join public.order_shipping_details shipping
			on shipping.order_id = o.id
		where o.id = $1::uuid
		for update of o
	`, orderID).Scan(
		&order.ID,
		&order.OrderNumber,
		&order.Status,
		&order.TotalCents,
		&order.Shipping.PriceCents,
		&order.Shipping.ServiceName,
		&order.Customer.Name,
		&order.Customer.Email,
		&order.Customer.Phone,
		&order.Address.PostalCode,
		&order.Address.Street,
		&order.Address.Number,
		&complement,
		&order.Address.Neighborhood,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CheckoutOrder{}, ErrOrderNotFound
		}
		return CheckoutOrder{}, ErrUnavailable
	}
	if complement.Valid {
		order.Address.Complement = complement.String
	}
	orderNSU, ok := CanonicalOrderNSU(order.ID)
	if !ok {
		return CheckoutOrder{}, ErrInvalidOrderID
	}
	order.OrderNSU = orderNSU

	items, err := r.checkoutOrderItems(ctx, tx, order.ID)
	if err != nil {
		return CheckoutOrder{}, err
	}
	order.Items = items

	return order, nil
}

func (r *PostgresRepository) checkoutOrderItems(ctx context.Context, tx pgx.Tx, orderID string) ([]CheckoutOrderItem, error) {
	rows, err := tx.Query(ctx, `
		select
			product_name,
			coalesce(variant_name, ''),
			quantity,
			unit_price_cents
		from public.order_items
		where order_id = $1::uuid
		order by sort_order asc, created_at asc, id asc
	`, orderID)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()

	var items []CheckoutOrderItem
	for rows.Next() {
		var item CheckoutOrderItem
		if err := rows.Scan(
			&item.ProductName,
			&item.VariantName,
			&item.Quantity,
			&item.UnitPriceCents,
		); err != nil {
			return nil, ErrUnavailable
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, ErrUnavailable
	}

	return items, nil
}

type paymentRecord struct {
	Status      string
	OrderNSU    string
	CheckoutURL pgtype.Text
}

func (r *PostgresRepository) paymentForOrderForUpdate(ctx context.Context, tx pgx.Tx, orderID string) (paymentRecord, bool, error) {
	var payment paymentRecord
	err := tx.QueryRow(ctx, `
		select
			status,
			order_nsu,
			checkout_url
		from public.order_payments
		where order_id = $1::uuid
		for update
	`, orderID).Scan(
		&payment.Status,
		&payment.OrderNSU,
		&payment.CheckoutURL,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return paymentRecord{}, false, nil
		}
		return paymentRecord{}, false, ErrUnavailable
	}

	return payment, true, nil
}

func (r *PostgresRepository) upsertPendingPayment(ctx context.Context, tx pgx.Tx, orderID string, orderNSU string, checkoutURL string) error {
	_, err := tx.Exec(ctx, `
		insert into public.order_payments (
			order_id,
			provider,
			status,
			order_nsu,
			checkout_url
		) values (
			$1::uuid,
			'infinitepay',
			'pending',
			$2,
			$3
		)
		on conflict (order_id) do update
		set
			checkout_url = excluded.checkout_url,
			updated_at = now()
		where public.order_payments.status = 'pending'
	`, orderID, orderNSU, checkoutURL)
	if err != nil {
		return ErrUnavailable
	}

	return nil
}

func (r *PostgresRepository) lockPaymentTargetByOrderNSU(ctx context.Context, tx pgx.Tx, orderNSU string) (PaymentVerificationTarget, error) {
	var target PaymentVerificationTarget
	err := tx.QueryRow(ctx, `
		select
			o.id::text,
			o.order_number,
			o.status,
			p.status,
			o.total_cents
		from public.order_payments p
		join public.orders o
			on o.id = p.order_id
		where p.order_nsu = $1
		for update of o, p
	`, orderNSU).Scan(
		&target.OrderID,
		&target.OrderNumber,
		&target.OrderStatus,
		&target.PaymentStatus,
		&target.TotalCents,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PaymentVerificationTarget{}, ErrPaymentNotFound
		}
		return PaymentVerificationTarget{}, ErrUnavailable
	}

	return target, nil
}

func textOrNil(value string) any {
	if value == "" {
		return nil
	}

	return value
}

func intOrNil(value *int) any {
	if value == nil {
		return nil
	}

	return *value
}
