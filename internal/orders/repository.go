package orders

import (
	"bytes"
	"context"
	"errors"
	"log"
	"time"

	"github.com/Bernardo-Txa/printlab/internal/products"
	"github.com/Bernardo-Txa/printlab/internal/shipping"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

type queryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Review(ctx context.Context, tokenHash []byte, now time.Time, params ReviewParams) (ReviewPage, error) {
	if r == nil || r.pool == nil {
		return ReviewPage{}, ErrUnavailable
	}

	cartID, err := r.activeCartIDByToken(ctx, r.pool, tokenHash, now)
	if err != nil {
		return ReviewPage{}, err
	}

	return r.reviewForCart(ctx, r.pool, cartID, now, params)
}

func (r *PostgresRepository) Confirm(ctx context.Context, tokenHash []byte, expectedFingerprint string, now time.Time, params ReviewParams) (ConfirmResult, error) {
	if r == nil || r.pool == nil {
		return ConfirmResult{}, ErrUnavailable
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ConfirmResult{}, ErrUnavailable
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	cart, err := r.lockCartByToken(ctx, tx, tokenHash, now)
	if err != nil {
		return ConfirmResult{}, err
	}

	if existing, found, err := r.existingOrderForCart(ctx, tx, cart.ID); err != nil {
		return ConfirmResult{}, ErrUnavailable
	} else if found {
		return ConfirmResult{
			OrderID:      existing.OrderID,
			OrderNumber:  existing.OrderNumber,
			Status:       existing.Status,
			ExpireCookie: true,
		}, nil
	}

	if cart.ConvertedAt.Valid {
		return ConfirmResult{}, ErrCartRequired
	}

	page, err := r.reviewForCart(ctx, tx, cart.ID, now, params)
	if err != nil {
		return ConfirmResult{}, err
	}

	if expectedFingerprint == "" || expectedFingerprint != page.Fingerprint {
		page.Message = StaleReviewMessage
		return ConfirmResult{Page: page}, ErrStaleReview
	}

	orderID, orderNumber, status, err := r.insertOrder(ctx, tx, cart.ID, page)
	if err != nil {
		if existing, found, lookupErr := r.existingOrderForCart(ctx, tx, cart.ID); lookupErr == nil && found {
			return ConfirmResult{
				OrderID:      existing.OrderID,
				OrderNumber:  existing.OrderNumber,
				Status:       existing.Status,
				ExpireCookie: true,
			}, nil
		}
		return ConfirmResult{}, ErrUnavailable
	}

	if err := r.insertOrderCustomer(ctx, tx, orderID, page.Customer); err != nil {
		return ConfirmResult{}, ErrUnavailable
	}
	if err := r.insertOrderAddress(ctx, tx, orderID, page.Address); err != nil {
		return ConfirmResult{}, ErrUnavailable
	}
	if err := r.insertOrderShipping(ctx, tx, orderID, page.Shipping); err != nil {
		return ConfirmResult{}, ErrUnavailable
	}
	if err := r.insertOrderItems(ctx, tx, orderID, page.Items); err != nil {
		return ConfirmResult{}, err
	}
	if err := r.convertCart(ctx, tx, cart.ID, now); err != nil {
		return ConfirmResult{}, ErrUnavailable
	}
	if err := r.clearTemporaryCartData(ctx, tx, cart.ID); err != nil {
		return ConfirmResult{}, ErrUnavailable
	}

	if err := tx.Commit(ctx); err != nil {
		return ConfirmResult{}, ErrUnavailable
	}

	log.Printf("order created order_id=%s order_number=%d status=%s cart_converted=true", orderID, orderNumber, status)

	return ConfirmResult{
		OrderID:      orderID,
		OrderNumber:  orderNumber,
		Status:       status,
		ExpireCookie: true,
	}, nil
}

func (r *PostgresRepository) Get(ctx context.Context, orderID string) (OrderPage, error) {
	if r == nil || r.pool == nil {
		return OrderPage{}, ErrUnavailable
	}

	var page OrderPage
	var productsSubtotalCents int64
	var shippingPriceCents int64
	var totalCents int64
	err := r.pool.QueryRow(ctx, `
		select
			id::text,
			order_number,
			status,
			products_subtotal_cents,
			shipping_price_cents,
			total_cents,
			created_at
		from public.orders
		where id = $1::uuid
	`, orderID).Scan(
		&page.ID,
		&page.OrderNumber,
		&page.Status,
		&productsSubtotalCents,
		&shippingPriceCents,
		&totalCents,
		&page.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return OrderPage{}, ErrNotFound
		}

		return OrderPage{}, ErrUnavailable
	}

	page.OrderNumberLabel = OrderNumberLabel(page.OrderNumber)
	page.StatusLabel = StatusLabel(page.Status)
	page.ProductsSubtotalBRL = products.FormatBRL(productsSubtotalCents)
	page.ShippingPriceBRL = products.FormatBRL(shippingPriceCents)
	page.TotalBRL = products.FormatBRL(totalCents)

	items, err := r.orderItems(ctx, r.pool, orderID)
	if err != nil {
		return OrderPage{}, ErrUnavailable
	}
	page.Items = items

	shippingDetails, err := r.orderShipping(ctx, r.pool, orderID)
	if err != nil {
		return OrderPage{}, err
	}
	page.Shipping = shippingDetails

	return page, nil
}

type cartLock struct {
	ID          string
	ConvertedAt pgtype.Timestamptz
}

func (r *PostgresRepository) activeCartIDByToken(ctx context.Context, q queryer, tokenHash []byte, now time.Time) (string, error) {
	var cartID string
	err := q.QueryRow(ctx, `
		select id::text
		from public.carts
		where token_hash = $1
			and expires_at > $2
			and converted_at is null
	`, tokenHash, now).Scan(&cartID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrCartRequired
		}

		return "", ErrUnavailable
	}

	return cartID, nil
}

func (r *PostgresRepository) lockCartByToken(ctx context.Context, q queryer, tokenHash []byte, now time.Time) (cartLock, error) {
	var cart cartLock
	err := q.QueryRow(ctx, `
		select
			id::text,
			converted_at
		from public.carts
		where token_hash = $1
			and expires_at > $2
		for update
	`, tokenHash, now).Scan(&cart.ID, &cart.ConvertedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return cartLock{}, ErrCartRequired
		}

		return cartLock{}, ErrUnavailable
	}

	return cart, nil
}

type existingOrder struct {
	OrderID     string
	OrderNumber int64
	Status      string
}

func (r *PostgresRepository) existingOrderForCart(ctx context.Context, q queryer, cartID string) (existingOrder, bool, error) {
	var order existingOrder
	err := q.QueryRow(ctx, `
		select
			id::text,
			order_number,
			status
		from public.orders
		where source_cart_id = $1::uuid
	`, cartID).Scan(&order.OrderID, &order.OrderNumber, &order.Status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return existingOrder{}, false, nil
		}

		return existingOrder{}, false, err
	}

	return order, true, nil
}

func (r *PostgresRepository) reviewForCart(ctx context.Context, q queryer, cartID string, now time.Time, params ReviewParams) (ReviewPage, error) {
	items, shippingItems, err := r.reviewItems(ctx, q, cartID)
	if err != nil {
		return ReviewPage{}, err
	}
	if len(items) == 0 {
		return ReviewPage{}, ErrEmptyCart
	}

	details, err := r.checkoutDetails(ctx, q, cartID)
	if err != nil {
		return ReviewPage{}, err
	}

	shippingDetails, box, selectionHash, err := r.shippingSelection(ctx, q, cartID, now)
	if err != nil {
		return ReviewPage{}, err
	}

	currentHash, err := shipping.BuildCartInputHash(params.OriginPostalCode, details.Address.PostalCode, params.ServiceCodes, shippingItems, box)
	if err != nil {
		if errors.Is(err, shipping.ErrAmountOverflow) {
			return ReviewPage{}, ErrAmountOverflow
		}
		return ReviewPage{}, ErrShippingChanged
	}
	if !bytes.Equal(selectionHash, currentHash) {
		return ReviewPage{}, ErrShippingChanged
	}

	page := ReviewPage{
		Items:                 items,
		Customer:              details.Customer,
		Address:               details.Address,
		Shipping:              shippingDetails,
		ShippingPriceCents:    shippingDetails.PriceCents,
		ProductsSubtotalCents: 0,
	}

	for _, item := range items {
		subtotal, err := addCents(page.ProductsSubtotalCents, item.LineTotalCents)
		if err != nil {
			return ReviewPage{}, err
		}
		page.ProductsSubtotalCents = subtotal
	}

	total, err := addCents(page.ProductsSubtotalCents, page.ShippingPriceCents)
	if err != nil {
		return ReviewPage{}, err
	}
	page.TotalCents = total

	return finalizeReviewPage(page, selectionHash)
}

type rawReviewItem struct {
	CartItemID               string
	ProductID                string
	VariantID                string
	Quantity                 int
	ProductName              string
	ProductSlug              string
	ProductPriceCents        int64
	ProductIsActive          bool
	ProductHasActiveVariants bool
	VariantProductID         string
	VariantName              string
	VariantSlug              string
	SKU                      string
	VariantPriceCents        *int64
	VariantIsActive          bool
	VariantPrintTimeMinutes  *int
	ShippingItem             shipping.CartItem
}

func (r *PostgresRepository) reviewItems(ctx context.Context, q queryer, cartID string) ([]ReviewItem, []shipping.CartItem, error) {
	rows, err := q.Query(ctx, `
		select
			ci.id::text,
			ci.product_id::text,
			coalesce(ci.variant_id::text, ''),
			ci.quantity,
			p.name,
			p.slug,
			p.price_cents,
			p.is_active,
			exists (
				select 1
				from public.product_variants active_variants
				where active_variants.product_id = p.id
					and active_variants.is_active = true
			),
			coalesce(v.product_id::text, ''),
			coalesce(v.name, ''),
			coalesce(v.slug, ''),
			coalesce(v.sku, ''),
			v.price_cents,
			coalesce(v.is_active, false),
			v.print_time_minutes,
			p.shipping_weight_g,
			p.shipping_height_mm,
			p.shipping_width_mm,
			p.shipping_length_mm,
			v.shipping_weight_g,
			v.shipping_height_mm,
			v.shipping_width_mm,
			v.shipping_length_mm
		from public.cart_items ci
		join public.products p
			on p.id = ci.product_id
		left join public.product_variants v
			on v.id = ci.variant_id
			and v.product_id = ci.product_id
		where ci.cart_id = $1::uuid
		order by ci.created_at asc, ci.id asc
	`, cartID)
	if err != nil {
		return nil, nil, ErrUnavailable
	}
	defer rows.Close()

	var rawItems []rawReviewItem
	for rows.Next() {
		item, err := scanRawReviewItem(rows)
		if err != nil {
			return nil, nil, ErrUnavailable
		}
		rawItems = append(rawItems, item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, ErrUnavailable
	}

	filaments, err := r.cartFilaments(ctx, q, cartID)
	if err != nil {
		return nil, nil, ErrUnavailable
	}

	items := make([]ReviewItem, 0, len(rawItems))
	shippingItems := make([]shipping.CartItem, 0, len(rawItems))
	for index, raw := range rawItems {
		item, err := prepareReviewItem(raw, filaments[raw.CartItemID], index)
		if err != nil {
			return nil, nil, err
		}
		items = append(items, item)
		shippingItems = append(shippingItems, raw.ShippingItem)
	}

	return items, shippingItems, nil
}

func scanRawReviewItem(scanner interface{ Scan(dest ...any) error }) (rawReviewItem, error) {
	var item rawReviewItem
	var variantPrice pgtype.Int8
	var variantPrintTime pgtype.Int4
	var productWeight pgtype.Int8
	var productHeight pgtype.Int4
	var productWidth pgtype.Int4
	var productLength pgtype.Int4
	var variantWeight pgtype.Int8
	var variantHeight pgtype.Int4
	var variantWidth pgtype.Int4
	var variantLength pgtype.Int4

	if err := scanner.Scan(
		&item.CartItemID,
		&item.ProductID,
		&item.VariantID,
		&item.Quantity,
		&item.ProductName,
		&item.ProductSlug,
		&item.ProductPriceCents,
		&item.ProductIsActive,
		&item.ProductHasActiveVariants,
		&item.VariantProductID,
		&item.VariantName,
		&item.VariantSlug,
		&item.SKU,
		&variantPrice,
		&item.VariantIsActive,
		&variantPrintTime,
		&productWeight,
		&productHeight,
		&productWidth,
		&productLength,
		&variantWeight,
		&variantHeight,
		&variantWidth,
		&variantLength,
	); err != nil {
		return rawReviewItem{}, err
	}

	if variantPrice.Valid {
		value := variantPrice.Int64
		item.VariantPriceCents = &value
	}
	if variantPrintTime.Valid {
		value := int(variantPrintTime.Int32)
		item.VariantPrintTimeMinutes = &value
	}

	item.ShippingItem = shipping.CartItem{
		ID:             item.CartItemID,
		ProductID:      item.ProductID,
		ProductName:    item.ProductName,
		VariantID:      item.VariantID,
		VariantName:    item.VariantName,
		Quantity:       item.Quantity,
		ProductProfile: profileFromColumns(productWeight, productHeight, productWidth, productLength),
		VariantProfile: profileFromColumns(variantWeight, variantHeight, variantWidth, variantLength),
	}

	return item, nil
}

func prepareReviewItem(raw rawReviewItem, filaments []ReviewItemFilament, index int) (ReviewItem, error) {
	if !raw.ProductIsActive {
		return ReviewItem{}, ErrUnavailableItems
	}
	if raw.VariantID == "" && raw.ProductHasActiveVariants {
		return ReviewItem{}, ErrUnavailableItems
	}
	if raw.VariantID != "" && (!raw.VariantIsActive || raw.VariantProductID != raw.ProductID) {
		return ReviewItem{}, ErrUnavailableItems
	}

	unitPrice := raw.ProductPriceCents
	if raw.VariantID != "" && raw.VariantPriceCents != nil {
		unitPrice = *raw.VariantPriceCents
	}

	lineTotal, err := lineTotalCents(unitPrice, raw.Quantity)
	if err != nil {
		return ReviewItem{}, err
	}

	item := ReviewItem{
		CartItemID:           raw.CartItemID,
		ProductID:            raw.ProductID,
		VariantID:            raw.VariantID,
		ProductName:          raw.ProductName,
		ProductSlug:          raw.ProductSlug,
		VariantName:          raw.VariantName,
		VariantSlug:          raw.VariantSlug,
		SKU:                  raw.SKU,
		HasVariant:           raw.VariantID != "",
		Quantity:             raw.Quantity,
		UnitPriceCents:       unitPrice,
		LineTotalCents:       lineTotal,
		UnitPrintTimeMinutes: raw.VariantPrintTimeMinutes,
		Filaments:            filaments,
		SortOrder:            index,
	}

	if len(filaments) > 0 {
		var totalWeight int64
		for _, filament := range filaments {
			totalWeight, err = addWeightMg(totalWeight, filament.EstimatedWeightMgPerUnit)
			if err != nil {
				return ReviewItem{}, err
			}
		}
		item.UnitEstimatedFilamentWeightMg = &totalWeight
	}

	return item, nil
}

func (r *PostgresRepository) cartFilaments(ctx context.Context, q queryer, cartID string) (map[string][]ReviewItemFilament, error) {
	rows, err := q.Query(ctx, `
		select
			ci.id::text,
			m.name,
			m.slug,
			c.name,
			c.slug,
			coalesce(c.hex_color, ''),
			vf.estimated_weight_mg,
			coalesce(vf.label, ''),
			vf.sort_order
		from public.cart_items ci
		join public.variant_filaments vf
			on vf.variant_id = ci.variant_id
		join public.materials m
			on m.id = vf.material_id
		join public.colors c
			on c.id = vf.color_id
		where ci.cart_id = $1::uuid
		order by ci.created_at asc, ci.id asc, vf.sort_order asc, vf.created_at asc, vf.id asc
	`, cartID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	filaments := map[string][]ReviewItemFilament{}
	for rows.Next() {
		var cartItemID string
		var filament ReviewItemFilament
		if err := rows.Scan(
			&cartItemID,
			&filament.MaterialName,
			&filament.MaterialSlug,
			&filament.ColorName,
			&filament.ColorSlug,
			&filament.HexColor,
			&filament.EstimatedWeightMgPerUnit,
			&filament.Label,
			&filament.SortOrder,
		); err != nil {
			return nil, err
		}
		filaments[cartItemID] = append(filaments[cartItemID], filament)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return filaments, nil
}

type checkoutSnapshot struct {
	Customer ReviewCustomer
	Address  ReviewAddress
}

func (r *PostgresRepository) checkoutDetails(ctx context.Context, q queryer, cartID string) (checkoutSnapshot, error) {
	var details checkoutSnapshot
	var complement pgtype.Text
	err := q.QueryRow(ctx, `
		select
			customer.full_name,
			customer.email,
			customer.phone,
			customer.cpf,
			address.postal_code,
			address.street,
			address.number,
			address.complement,
			address.district,
			address.city,
			address.state,
			address.country_code
		from public.cart_customer_details customer
		join public.cart_shipping_addresses address
			on address.cart_id = customer.cart_id
		where customer.cart_id = $1::uuid
	`, cartID).Scan(
		&details.Customer.FullName,
		&details.Customer.Email,
		&details.Customer.Phone,
		&details.Customer.CPF,
		&details.Address.PostalCode,
		&details.Address.Street,
		&details.Address.Number,
		&complement,
		&details.Address.District,
		&details.Address.City,
		&details.Address.State,
		&details.Address.CountryCode,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return checkoutSnapshot{}, ErrDetailsRequired
		}

		return checkoutSnapshot{}, ErrUnavailable
	}
	if complement.Valid {
		details.Address.Complement = complement.String
	}

	return details, nil
}

func (r *PostgresRepository) shippingSelection(ctx context.Context, q queryer, cartID string, now time.Time) (ReviewShipping, shipping.ShippingBox, []byte, error) {
	var selection ReviewShipping
	var box shipping.ShippingBox
	var carrierName pgtype.Text
	var deliveryTime pgtype.Int4
	var inputHash []byte
	var quotedAt time.Time
	var expiresAt time.Time

	err := q.QueryRow(ctx, `
		select
			selection.provider,
			selection.service_code,
			selection.service_name,
			selection.carrier_name,
			selection.price_cents,
			selection.delivery_time_days,
			selection.package_weight_g,
			selection.package_height_mm,
			selection.package_width_mm,
			selection.package_length_mm,
			selection.input_hash,
			selection.quoted_at,
			selection.expires_at,
			box.id::text,
			box.name,
			box.slug,
			box.internal_height_mm,
			box.internal_width_mm,
			box.internal_length_mm,
			box.external_height_mm,
			box.external_width_mm,
			box.external_length_mm,
			box.packaging_weight_g,
			box.sort_order
		from public.cart_shipping_selections selection
		join public.shipping_boxes box
			on box.id = selection.shipping_box_id
			and box.is_active = true
		where selection.cart_id = $1::uuid
	`, cartID).Scan(
		&selection.Provider,
		&selection.ServiceCode,
		&selection.ServiceName,
		&carrierName,
		&selection.PriceCents,
		&deliveryTime,
		&selection.PackageWeightG,
		&selection.PackageHeightMM,
		&selection.PackageWidthMM,
		&selection.PackageLengthMM,
		&inputHash,
		&quotedAt,
		&expiresAt,
		&box.ID,
		&box.Name,
		&box.Slug,
		&box.Internal.Height,
		&box.Internal.Width,
		&box.Internal.Length,
		&box.External.Height,
		&box.External.Width,
		&box.External.Length,
		&box.PackagingWeightG,
		&box.SortOrder,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ReviewShipping{}, shipping.ShippingBox{}, nil, ErrShippingRequired
		}

		return ReviewShipping{}, shipping.ShippingBox{}, nil, ErrUnavailable
	}
	if selection.Provider != shipping.ProviderSuperFrete {
		return ReviewShipping{}, shipping.ShippingBox{}, nil, ErrShippingChanged
	}
	if !expiresAt.After(now) {
		return ReviewShipping{}, shipping.ShippingBox{}, nil, ErrShippingExpired
	}

	if carrierName.Valid {
		selection.CarrierName = carrierName.String
	}
	if deliveryTime.Valid {
		value := int(deliveryTime.Int32)
		selection.DeliveryTimeDays = &value
	}
	selection.ShippingBoxName = box.Name
	selection.QuotedAt = &quotedAt

	return selection, box, inputHash, nil
}

func (r *PostgresRepository) insertOrder(ctx context.Context, q queryer, cartID string, page ReviewPage) (string, int64, string, error) {
	var orderID string
	var orderNumber int64
	var status string
	err := q.QueryRow(ctx, `
		insert into public.orders (
			source_cart_id,
			status,
			currency,
			products_subtotal_cents,
			shipping_price_cents,
			total_cents
		) values (
			$1::uuid,
			$2,
			$3,
			$4,
			$5,
			$6
		)
		returning id::text, order_number, status
	`, cartID, StatusPendingPayment, CurrencyBRL, page.ProductsSubtotalCents, page.ShippingPriceCents, page.TotalCents).Scan(&orderID, &orderNumber, &status)
	if err != nil {
		return "", 0, "", err
	}

	return orderID, orderNumber, status, nil
}

func (r *PostgresRepository) insertOrderCustomer(ctx context.Context, q queryer, orderID string, customer ReviewCustomer) error {
	_, err := q.Exec(ctx, `
		insert into public.order_customer_details (
			order_id,
			full_name,
			email,
			phone,
			cpf
		) values (
			$1::uuid,
			$2,
			$3,
			$4,
			$5
		)
	`, orderID, customer.FullName, customer.Email, customer.Phone, customer.CPF)
	return err
}

func (r *PostgresRepository) insertOrderAddress(ctx context.Context, q queryer, orderID string, address ReviewAddress) error {
	_, err := q.Exec(ctx, `
		insert into public.order_shipping_addresses (
			order_id,
			postal_code,
			street,
			number,
			complement,
			district,
			city,
			state,
			country_code
		) values (
			$1::uuid,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9
		)
	`, orderID, address.PostalCode, address.Street, address.Number, textOrNil(address.Complement), address.District, address.City, address.State, address.CountryCode)
	return err
}

func (r *PostgresRepository) insertOrderShipping(ctx context.Context, q queryer, orderID string, details ReviewShipping) error {
	_, err := q.Exec(ctx, `
		insert into public.order_shipping_details (
			order_id,
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
		) values (
			$1::uuid,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10,
			$11,
			$12
		)
	`, orderID, details.Provider, details.ServiceCode, details.ServiceName, textOrNil(details.CarrierName), intOrNil(details.DeliveryTimeDays), details.ShippingBoxName, details.PackageWeightG, details.PackageHeightMM, details.PackageWidthMM, details.PackageLengthMM, timeOrNil(details.QuotedAt))
	return err
}

func (r *PostgresRepository) insertOrderItems(ctx context.Context, q queryer, orderID string, items []ReviewItem) error {
	for _, item := range items {
		var orderItemID string
		err := q.QueryRow(ctx, `
			insert into public.order_items (
				order_id,
				product_id,
				variant_id,
				product_name,
				product_slug,
				variant_name,
				variant_slug,
				sku,
				unit_price_cents,
				quantity,
				line_total_cents,
				unit_print_time_minutes,
				unit_estimated_filament_weight_mg,
				sort_order
			) values (
				$1::uuid,
				$2::uuid,
				$3::uuid,
				$4,
				$5,
				$6,
				$7,
				$8,
				$9,
				$10,
				$11,
				$12,
				$13,
				$14
			)
			returning id::text
		`, orderID, uuidOrNil(item.ProductID), uuidOrNil(item.VariantID), item.ProductName, item.ProductSlug, textOrNil(item.VariantName), textOrNil(item.VariantSlug), textOrNil(item.SKU), item.UnitPriceCents, item.Quantity, item.LineTotalCents, intOrNil(item.UnitPrintTimeMinutes), int64OrNil(item.UnitEstimatedFilamentWeightMg), item.SortOrder).Scan(&orderItemID)
		if err != nil {
			return ErrUnavailable
		}

		for _, filament := range item.Filaments {
			if _, err := q.Exec(ctx, `
				insert into public.order_item_filaments (
					order_item_id,
					material_name,
					material_slug,
					color_name,
					color_slug,
					hex_color,
					estimated_weight_mg_per_unit,
					label,
					sort_order
				) values (
					$1::uuid,
					$2,
					$3,
					$4,
					$5,
					$6,
					$7,
					$8,
					$9
				)
			`, orderItemID, filament.MaterialName, filament.MaterialSlug, filament.ColorName, filament.ColorSlug, textOrNil(filament.HexColor), filament.EstimatedWeightMgPerUnit, textOrNil(filament.Label), filament.SortOrder); err != nil {
				return ErrUnavailable
			}
		}
	}

	return nil
}

func (r *PostgresRepository) convertCart(ctx context.Context, q queryer, cartID string, now time.Time) error {
	commandTag, err := q.Exec(ctx, `
		update public.carts
		set
			converted_at = $2,
			updated_at = now()
		where id = $1::uuid
			and converted_at is null
	`, cartID, now)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() != 1 {
		return ErrCartRequired
	}

	return nil
}

func (r *PostgresRepository) clearTemporaryCartData(ctx context.Context, q queryer, cartID string) error {
	statements := []string{
		`delete from public.cart_shipping_selections where cart_id = $1::uuid`,
		`delete from public.cart_shipping_addresses where cart_id = $1::uuid`,
		`delete from public.cart_customer_details where cart_id = $1::uuid`,
		`delete from public.cart_items where cart_id = $1::uuid`,
	}

	for _, statement := range statements {
		if _, err := q.Exec(ctx, statement, cartID); err != nil {
			return err
		}
	}

	return nil
}

func (r *PostgresRepository) orderItems(ctx context.Context, q queryer, orderID string) ([]OrderItem, error) {
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
		return nil, err
	}
	defer rows.Close()

	var items []OrderItem
	var itemIDs []string
	for rows.Next() {
		var item OrderItem
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
			return nil, err
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
			item.UnitEstimatedFilamentWeightLabel = formatWeightMg(value)
		}
		item.UnitPriceBRL = products.FormatBRL(unitPriceCents)
		item.LineTotalBRL = products.FormatBRL(lineTotalCents)
		itemIDs = append(itemIDs, itemID)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
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

func (r *PostgresRepository) orderItemFilaments(ctx context.Context, q queryer, orderID string) (map[string][]ReviewItemFilament, error) {
	rows, err := q.Query(ctx, `
		select
			item.id::text,
			filament.material_name,
			filament.material_slug,
			filament.color_name,
			filament.color_slug,
			coalesce(filament.hex_color, ''),
			filament.estimated_weight_mg_per_unit,
			coalesce(filament.label, ''),
			filament.sort_order
		from public.order_item_filaments filament
		join public.order_items item
			on item.id = filament.order_item_id
		where item.order_id = $1::uuid
		order by item.sort_order asc, item.created_at asc, item.id asc, filament.sort_order asc, filament.created_at asc, filament.id asc
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	filaments := map[string][]ReviewItemFilament{}
	for rows.Next() {
		var itemID string
		var filament ReviewItemFilament
		if err := rows.Scan(
			&itemID,
			&filament.MaterialName,
			&filament.MaterialSlug,
			&filament.ColorName,
			&filament.ColorSlug,
			&filament.HexColor,
			&filament.EstimatedWeightMgPerUnit,
			&filament.Label,
			&filament.SortOrder,
		); err != nil {
			return nil, err
		}
		filament.EstimatedWeightLabel = formatWeightMg(filament.EstimatedWeightMgPerUnit)
		filaments[itemID] = append(filaments[itemID], filament)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return filaments, nil
}

func (r *PostgresRepository) orderShipping(ctx context.Context, q queryer, orderID string) (OrderShipping, error) {
	var details OrderShipping
	var carrierName pgtype.Text
	var deliveryTime pgtype.Int4
	err := q.QueryRow(ctx, `
		select
			service_name,
			carrier_name,
			delivery_time_days,
			shipping_box_name,
			package_weight_g,
			package_height_mm,
			package_width_mm,
			package_length_mm
		from public.order_shipping_details
		where order_id = $1::uuid
	`, orderID).Scan(
		&details.ServiceName,
		&carrierName,
		&deliveryTime,
		&details.ShippingBoxName,
		&details.PackageWeightG,
		&details.PackageHeightMM,
		&details.PackageWidthMM,
		&details.PackageLengthMM,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return OrderShipping{}, ErrNotFound
		}

		return OrderShipping{}, ErrUnavailable
	}
	if carrierName.Valid {
		details.CarrierName = carrierName.String
	}
	if deliveryTime.Valid {
		value := int(deliveryTime.Int32)
		details.DeliveryTimeDays = &value
	}
	details.DeliveryTime = shipping.DeliveryTimeLabel(details.DeliveryTimeDays)

	var priceCents int64
	err = q.QueryRow(ctx, `
		select shipping_price_cents
		from public.orders
		where id = $1::uuid
	`, orderID).Scan(&priceCents)
	if err != nil {
		return OrderShipping{}, ErrUnavailable
	}
	details.PriceBRL = products.FormatBRL(priceCents)

	return details, nil
}

func profileFromColumns(weight pgtype.Int8, height pgtype.Int4, width pgtype.Int4, length pgtype.Int4) *shipping.ShippingProfile {
	if !weight.Valid && !height.Valid && !width.Valid && !length.Valid {
		return nil
	}
	if !weight.Valid || !height.Valid || !width.Valid || !length.Valid {
		return &shipping.ShippingProfile{}
	}

	return &shipping.ShippingProfile{
		WeightG: weight.Int64,
		Dimensions: shipping.DimensionsMM{
			Height: int(height.Int32),
			Width:  int(width.Int32),
			Length: int(length.Int32),
		},
	}
}

func textOrNil(value string) any {
	if value == "" {
		return nil
	}

	return value
}

func uuidOrNil(value string) any {
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

func int64OrNil(value *int64) any {
	if value == nil {
		return nil
	}

	return *value
}

func timeOrNil(value *time.Time) any {
	if value == nil {
		return nil
	}

	return *value
}
