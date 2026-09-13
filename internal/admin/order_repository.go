package admin

import (
	"context"
	"errors"
	"strings"

	"github.com/Bernardo-Txa/printlab/internal/products"
	"github.com/Bernardo-Txa/printlab/internal/shipping"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type orderQueryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (r *PostgresRepository) ListOrders(ctx context.Context, filter OrderListFilter) (OrderListPage, error) {
	if r == nil || r.pool == nil {
		return OrderListPage{}, ErrUnavailable
	}

	filter = NormalizeOrderListFilter(filter)
	rows, err := r.pool.Query(ctx, `
		select
			o.id::text,
			o.order_number,
			o.status,
			o.total_cents,
			o.created_at,
			fulfillment.production_status,
			fulfillment.shipping_status,
			count(items.id)::integer,
			coalesce(sum(items.quantity), 0)::integer
		from public.orders o
		join public.order_fulfillment fulfillment
			on fulfillment.order_id = o.id
		left join public.order_items items
			on items.order_id = o.id
		`+orderListWhereSQL(filter.Status)+`
		group by
			o.id,
			o.order_number,
			o.status,
			o.total_cents,
			o.created_at,
			fulfillment.production_status,
			fulfillment.shipping_status
		`+orderListSortSQL(filter.Status)+`
		limit $1
		offset $2
	`, OrderPageSize+1, (filter.Page-1)*OrderPageSize)
	if err != nil {
		return OrderListPage{}, ErrUnavailable
	}
	defer rows.Close()

	var items []OrderListItem
	for rows.Next() {
		var item OrderListItem
		if err := rows.Scan(
			&item.ID,
			&item.OrderNumber,
			&item.OrderStatus,
			&item.TotalCents,
			&item.CreatedAt,
			&item.ProductionStatus,
			&item.ShippingStatus,
			&item.ItemCount,
			&item.UnitCount,
		); err != nil {
			return OrderListPage{}, ErrUnavailable
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return OrderListPage{}, ErrUnavailable
	}

	return PrepareOrderListPage(filter, items), nil
}

func (r *PostgresRepository) GetOrder(ctx context.Context, orderID string) (OrderDetail, error) {
	if r == nil || r.pool == nil {
		return OrderDetail{}, ErrUnavailable
	}

	var detail OrderDetail
	err := r.pool.QueryRow(ctx, `
		select
			o.id::text,
			o.public_tracking_id::text,
			o.order_number,
			o.status,
			o.products_subtotal_cents,
			o.shipping_price_cents,
			o.total_cents,
			o.created_at,
			fulfillment.production_status,
			fulfillment.shipping_status
		from public.orders o
		join public.order_fulfillment fulfillment
			on fulfillment.order_id = o.id
		where o.id = $1::uuid
	`, orderID).Scan(
		&detail.ID,
		&detail.PublicTrackingID,
		&detail.OrderNumber,
		&detail.OrderStatus,
		&detail.ProductsSubtotalCents,
		&detail.ShippingPriceCents,
		&detail.TotalCents,
		&detail.CreatedAt,
		&detail.ProductionStatus,
		&detail.ShippingStatus,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return OrderDetail{}, ErrOrderNotFound
		}
		return OrderDetail{}, ErrUnavailable
	}

	customer, err := r.orderCustomer(ctx, r.pool, orderID)
	if err != nil {
		return OrderDetail{}, err
	}
	detail.Customer = customer

	address, err := r.orderAddress(ctx, r.pool, orderID)
	if err != nil {
		return OrderDetail{}, err
	}
	detail.Address = address

	shippingDetails, err := r.orderShipping(ctx, r.pool, orderID, detail.ShippingPriceCents)
	if err != nil {
		return OrderDetail{}, err
	}
	detail.Shipping = shippingDetails

	payment, err := r.orderPayment(ctx, r.pool, orderID)
	if err != nil {
		return OrderDetail{}, err
	}
	detail.Payment = payment

	items, err := r.orderItems(ctx, r.pool, orderID)
	if err != nil {
		return OrderDetail{}, err
	}
	detail.Items = items

	events, err := r.orderEvents(ctx, r.pool, orderID)
	if err != nil {
		return OrderDetail{}, err
	}
	detail.Events = events

	PrepareOrderDetail(&detail)
	return detail, nil
}

func (r *PostgresRepository) ChangeProductionStatus(ctx context.Context, orderID string, targetStatus string, actorAuthUserID string) error {
	if r == nil || r.pool == nil {
		return ErrUnavailable
	}
	targetStatus = strings.TrimSpace(targetStatus)

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ErrUnavailable
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	snapshot, err := r.lockOrderStatuses(ctx, tx, orderID)
	if err != nil {
		return err
	}
	if err := ValidateProductionTransition(snapshot, targetStatus); err != nil {
		return err
	}

	tag, err := tx.Exec(ctx, `
		update public.order_fulfillment
		set
			production_status = $2,
			updated_at = now()
		where order_id = $1::uuid
			and production_status = $3
			and shipping_status = $4
	`, orderID, targetStatus, snapshot.ProductionStatus, snapshot.ShippingStatus)
	if err != nil {
		return ErrUnavailable
	}
	if tag.RowsAffected() != 1 {
		return ErrTransitionConflict
	}

	if err := r.insertAdminOrderEvent(ctx, tx, orderID, actorAuthUserID, EventTypeProductionStatusChanged, snapshot.ProductionStatus, targetStatus); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return ErrUnavailable
	}

	return nil
}

func (r *PostgresRepository) ChangeShippingStatus(ctx context.Context, orderID string, targetStatus string, actorAuthUserID string) error {
	if r == nil || r.pool == nil {
		return ErrUnavailable
	}
	targetStatus = strings.TrimSpace(targetStatus)

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ErrUnavailable
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	snapshot, err := r.lockOrderStatuses(ctx, tx, orderID)
	if err != nil {
		return err
	}
	if err := ValidateShippingTransition(snapshot, targetStatus); err != nil {
		return err
	}

	tag, err := tx.Exec(ctx, `
		update public.order_fulfillment
		set
			shipping_status = $2,
			updated_at = now()
		where order_id = $1::uuid
			and production_status = $3
			and shipping_status = $4
	`, orderID, targetStatus, snapshot.ProductionStatus, snapshot.ShippingStatus)
	if err != nil {
		return ErrUnavailable
	}
	if tag.RowsAffected() != 1 {
		return ErrTransitionConflict
	}

	if err := r.insertAdminOrderEvent(ctx, tx, orderID, actorAuthUserID, EventTypeShippingStatusChanged, snapshot.ShippingStatus, targetStatus); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return ErrUnavailable
	}

	return nil
}

func (r *PostgresRepository) lockOrderStatuses(ctx context.Context, tx pgx.Tx, orderID string) (OrderStatusSnapshot, error) {
	var snapshot OrderStatusSnapshot
	err := tx.QueryRow(ctx, `
		select
			o.status,
			fulfillment.production_status,
			fulfillment.shipping_status
		from public.orders o
		join public.order_fulfillment fulfillment
			on fulfillment.order_id = o.id
		where o.id = $1::uuid
		for update of o, fulfillment
	`, orderID).Scan(
		&snapshot.OrderStatus,
		&snapshot.ProductionStatus,
		&snapshot.ShippingStatus,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return OrderStatusSnapshot{}, ErrOrderNotFound
		}
		return OrderStatusSnapshot{}, ErrUnavailable
	}

	return snapshot, nil
}

func (r *PostgresRepository) insertAdminOrderEvent(ctx context.Context, tx pgx.Tx, orderID string, actorAuthUserID string, eventType string, fromStatus string, toStatus string) error {
	if _, err := tx.Exec(ctx, `
		insert into public.admin_order_events (
			order_id,
			actor_auth_user_id,
			event_type,
			from_status,
			to_status
		) values (
			$1::uuid,
			$2::uuid,
			$3,
			$4,
			$5
		)
	`, orderID, actorAuthUserID, eventType, fromStatus, toStatus); err != nil {
		return ErrUnavailable
	}

	return nil
}

func (r *PostgresRepository) orderCustomer(ctx context.Context, q orderQueryer, orderID string) (OrderCustomer, error) {
	var customer OrderCustomer
	err := q.QueryRow(ctx, `
		select
			full_name,
			email,
			phone,
			cpf
		from public.order_customer_details
		where order_id = $1::uuid
	`, orderID).Scan(
		&customer.FullName,
		&customer.Email,
		&customer.Phone,
		&customer.CPF,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return OrderCustomer{}, ErrOrderNotFound
		}
		return OrderCustomer{}, ErrUnavailable
	}

	return customer, nil
}

func (r *PostgresRepository) orderAddress(ctx context.Context, q orderQueryer, orderID string) (OrderAddress, error) {
	var address OrderAddress
	var complement pgtype.Text
	err := q.QueryRow(ctx, `
		select
			postal_code,
			street,
			number,
			complement,
			district,
			city,
			state,
			country_code
		from public.order_shipping_addresses
		where order_id = $1::uuid
	`, orderID).Scan(
		&address.PostalCode,
		&address.Street,
		&address.Number,
		&complement,
		&address.District,
		&address.City,
		&address.State,
		&address.CountryCode,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return OrderAddress{}, ErrOrderNotFound
		}
		return OrderAddress{}, ErrUnavailable
	}
	if complement.Valid {
		address.Complement = complement.String
	}
	address.LineOne = address.Street + ", " + address.Number
	if address.Complement != "" {
		address.LineOne += " - " + address.Complement
	}
	address.LineTwo = address.District + " - " + address.City + "/" + address.State + " - CEP " + FormatPostalCode(address.PostalCode)

	return address, nil
}

func (r *PostgresRepository) orderShipping(ctx context.Context, q orderQueryer, orderID string, shippingPriceCents int64) (OrderShipping, error) {
	var details OrderShipping
	var carrierName pgtype.Text
	var deliveryTimeDays pgtype.Int4
	var quotedAt pgtype.Timestamptz
	err := q.QueryRow(ctx, `
		select
			provider,
			service_code,
			service_name,
			carrier_name,
			delivery_time_days,
			shipping_box_name,
			package_weight_g,
			package_height_mm,
			package_width_mm,
			package_length_mm,
			quoted_at
		from public.order_shipping_details
		where order_id = $1::uuid
	`, orderID).Scan(
		&details.Provider,
		&details.ServiceCode,
		&details.ServiceName,
		&carrierName,
		&deliveryTimeDays,
		&details.ShippingBoxName,
		&details.PackageWeightG,
		&details.PackageHeightMM,
		&details.PackageWidthMM,
		&details.PackageLengthMM,
		&quotedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return OrderShipping{}, ErrOrderNotFound
		}
		return OrderShipping{}, ErrUnavailable
	}
	if carrierName.Valid {
		details.CarrierName = carrierName.String
	}
	if deliveryTimeDays.Valid {
		value := int(deliveryTimeDays.Int32)
		details.DeliveryTime = shipping.DeliveryTimeLabel(&value)
	} else {
		details.DeliveryTime = shipping.DeliveryTimeLabel(nil)
	}
	if quotedAt.Valid {
		value := quotedAt.Time
		details.QuotedAt = &value
		details.QuotedAtLabel = FormatOrderDate(value)
	}
	details.PriceBRL = products.FormatBRL(shippingPriceCents)
	details.PackageWeightLabel = FormatWeightG(details.PackageWeightG)
	details.DimensionsLabel = FormatDimensionsMM(details.PackageHeightMM, details.PackageWidthMM, details.PackageLengthMM)

	return details, nil
}

func (r *PostgresRepository) orderPayment(ctx context.Context, q orderQueryer, orderID string) (OrderPayment, error) {
	var payment OrderPayment
	var captureMethod pgtype.Text
	var paidAt pgtype.Timestamptz
	var amountCents pgtype.Int8
	var paidAmountCents pgtype.Int8
	var installments pgtype.Int4
	err := q.QueryRow(ctx, `
		select
			status,
			capture_method,
			paid_at,
			amount_cents,
			paid_amount_cents,
			installments
		from public.order_payments
		where order_id = $1::uuid
	`, orderID).Scan(
		&payment.Status,
		&captureMethod,
		&paidAt,
		&amountCents,
		&paidAmountCents,
		&installments,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return OrderPayment{StatusLabel: PaymentStatusLabel("")}, nil
		}
		return OrderPayment{}, ErrUnavailable
	}
	payment.Available = true
	payment.StatusLabel = PaymentStatusLabel(payment.Status)
	if captureMethod.Valid {
		payment.CaptureMethod = captureMethod.String
	}
	if paidAt.Valid {
		value := paidAt.Time
		payment.PaidAt = &value
		payment.PaidAtLabel = FormatOrderDate(value)
	}
	if amountCents.Valid {
		payment.AmountBRL = products.FormatBRL(amountCents.Int64)
	}
	if paidAmountCents.Valid {
		payment.PaidAmountBRL = products.FormatBRL(paidAmountCents.Int64)
	}
	if installments.Valid {
		value := int(installments.Int32)
		payment.Installments = &value
		payment.InstallmentsLabel = FormatInstallments(&value)
	}

	return payment, nil
}

func (r *PostgresRepository) orderItems(ctx context.Context, q orderQueryer, orderID string) ([]OrderDetailItem, error) {
	rows, err := q.Query(ctx, `
		select
			id::text,
			product_name,
			coalesce(variant_name, ''),
			coalesce(sku, ''),
			quantity,
			unit_price_cents,
			line_total_cents,
			unit_print_time_minutes,
			unit_estimated_filament_weight_mg
		from public.order_items
		where order_id = $1::uuid
		order by sort_order asc, created_at asc, id asc
	`, orderID)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()

	var items []OrderDetailItem
	var itemIDs []string
	for rows.Next() {
		var item OrderDetailItem
		var itemID string
		var unitPriceCents int64
		var lineTotalCents int64
		var printTime pgtype.Int4
		var filamentWeight pgtype.Int8
		if err := rows.Scan(
			&itemID,
			&item.ProductName,
			&item.VariantName,
			&item.SKU,
			&item.Quantity,
			&unitPriceCents,
			&lineTotalCents,
			&printTime,
			&filamentWeight,
		); err != nil {
			return nil, ErrUnavailable
		}
		if item.VariantName != "" {
			item.HasVariant = true
		}
		if printTime.Valid {
			value := int(printTime.Int32)
			item.UnitPrintTimeMinutes = &value
		}
		if filamentWeight.Valid {
			value := filamentWeight.Int64
			item.UnitEstimatedFilamentWeightMg = &value
			item.UnitEstimatedFilamentWeightLabel = FormatWeightMg(value)
		}
		item.UnitPriceBRL = products.FormatBRL(unitPriceCents)
		item.LineTotalBRL = products.FormatBRL(lineTotalCents)
		itemIDs = append(itemIDs, itemID)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, ErrUnavailable
	}

	filaments, err := r.orderItemFilaments(ctx, q, orderID)
	if err != nil {
		return nil, err
	}
	for i, itemID := range itemIDs {
		items[i].Filaments = filaments[itemID]
	}

	return items, nil
}

func (r *PostgresRepository) orderItemFilaments(ctx context.Context, q orderQueryer, orderID string) (map[string][]OrderItemFilament, error) {
	rows, err := q.Query(ctx, `
		select
			item.id::text,
			filament.material_name,
			filament.color_name,
			coalesce(filament.hex_color, ''),
			filament.estimated_weight_mg_per_unit,
			coalesce(filament.label, '')
		from public.order_item_filaments filament
		join public.order_items item
			on item.id = filament.order_item_id
		where item.order_id = $1::uuid
		order by item.sort_order asc, item.created_at asc, item.id asc, filament.sort_order asc, filament.created_at asc, filament.id asc
	`, orderID)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()

	filaments := map[string][]OrderItemFilament{}
	for rows.Next() {
		var itemID string
		var filament OrderItemFilament
		if err := rows.Scan(
			&itemID,
			&filament.MaterialName,
			&filament.ColorName,
			&filament.HexColor,
			&filament.EstimatedWeightMgPerUnit,
			&filament.Label,
		); err != nil {
			return nil, ErrUnavailable
		}
		filament.EstimatedWeightLabel = FormatWeightMg(filament.EstimatedWeightMgPerUnit)
		filaments[itemID] = append(filaments[itemID], filament)
	}
	if err := rows.Err(); err != nil {
		return nil, ErrUnavailable
	}

	return filaments, nil
}

func (r *PostgresRepository) orderEvents(ctx context.Context, q orderQueryer, orderID string) ([]OrderEvent, error) {
	rows, err := q.Query(ctx, `
		select
			event_type,
			from_status,
			to_status,
			created_at
		from public.admin_order_events
		where order_id = $1::uuid
		order by created_at desc, id desc
		limit 50
	`, orderID)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()

	var events []OrderEvent
	for rows.Next() {
		var event OrderEvent
		if err := rows.Scan(
			&event.EventType,
			&event.FromStatus,
			&event.ToStatus,
			&event.CreatedAt,
		); err != nil {
			return nil, ErrUnavailable
		}
		event.EventTypeLabel = EventTypeLabel(event.EventType)
		event.FromStatusLabel = EventStatusLabel(event.EventType, event.FromStatus)
		event.ToStatusLabel = EventStatusLabel(event.EventType, event.ToStatus)
		event.CreatedAtLabel = FormatOrderDate(event.CreatedAt)
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, ErrUnavailable
	}

	return events, nil
}

func orderListWhereSQL(status string) string {
	switch status {
	case OrderListStatusPendingPayment:
		return " where o.status = 'pending_payment'"
	case OrderListStatusWaitingProduction:
		return " where o.status = 'paid' and fulfillment.production_status = 'waiting'"
	case OrderListStatusInProduction:
		return " where o.status = 'paid' and fulfillment.production_status = 'in_production'"
	case OrderListStatusWaitingShipment:
		return " where o.status = 'paid' and fulfillment.production_status = 'completed' and fulfillment.shipping_status = 'waiting'"
	case OrderListStatusPreparingShipment:
		return " where o.status = 'paid' and fulfillment.shipping_status = 'preparing'"
	case OrderListStatusShipped:
		return " where o.status = 'paid' and fulfillment.shipping_status = 'shipped'"
	case OrderListStatusDelivered:
		return " where o.status = 'paid' and fulfillment.shipping_status = 'delivered'"
	default:
		return ""
	}
}

func orderListSortSQL(status string) string {
	switch status {
	case OrderListStatusWaitingProduction, OrderListStatusInProduction, OrderListStatusWaitingShipment, OrderListStatusPreparingShipment:
		return " order by o.created_at asc, o.order_number asc"
	default:
		return " order by o.created_at desc, o.order_number desc"
	}
}
