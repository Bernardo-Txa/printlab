package shipping

import (
	"context"
	"errors"

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

func (r *PostgresRepository) ListCartItems(ctx context.Context, cartID string) ([]CartItem, error) {
	if r == nil || r.pool == nil {
		return nil, ErrUnavailable
	}

	rows, err := r.pool.Query(ctx, `
		select
			ci.id::text,
			ci.product_id::text,
			coalesce(ci.variant_id::text, ''),
			ci.quantity,
			p.name,
			coalesce(v.name, ''),
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
		return nil, err
	}
	defer rows.Close()

	var items []CartItem
	for rows.Next() {
		item, err := scanCartItem(rows)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *PostgresRepository) ListActiveBoxes(ctx context.Context) ([]ShippingBox, error) {
	if r == nil || r.pool == nil {
		return nil, ErrUnavailable
	}

	rows, err := r.pool.Query(ctx, `
		select
			id::text,
			name,
			slug,
			internal_height_mm,
			internal_width_mm,
			internal_length_mm,
			external_height_mm,
			external_width_mm,
			external_length_mm,
			packaging_weight_g,
			sort_order
		from public.shipping_boxes
		where is_active = true
		order by sort_order asc, name asc, id asc
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var boxes []ShippingBox
	for rows.Next() {
		var box ShippingBox
		if err := rows.Scan(
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
		); err != nil {
			return nil, err
		}

		boxes = append(boxes, box)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return boxes, nil
}

func (r *PostgresRepository) GetSelection(ctx context.Context, cartID string) (ShippingSelection, bool, error) {
	if r == nil || r.pool == nil {
		return ShippingSelection{}, false, ErrUnavailable
	}

	var selection ShippingSelection
	var deliveryTime pgtype.Int4
	var carrierName pgtype.Text
	err := r.pool.QueryRow(ctx, `
		select
			selection.cart_id::text,
			selection.shipping_box_id::text,
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
			selection.expires_at
		from public.cart_shipping_selections selection
		join public.shipping_boxes box
			on box.id = selection.shipping_box_id
			and box.is_active = true
		where selection.cart_id = $1::uuid
	`, cartID).Scan(
		&selection.CartID,
		&selection.ShippingBoxID,
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
		&selection.InputHash,
		&selection.QuotedAt,
		&selection.ExpiresAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ShippingSelection{}, false, nil
		}

		return ShippingSelection{}, false, err
	}

	if carrierName.Valid {
		selection.CarrierName = carrierName.String
	}
	if deliveryTime.Valid {
		value := int(deliveryTime.Int32)
		selection.DeliveryTimeDays = &value
	}

	return selection, true, nil
}

func (r *PostgresRepository) SaveSelection(ctx context.Context, cartID string, selection ShippingSelection) error {
	if r == nil || r.pool == nil {
		return ErrUnavailable
	}

	var carrierName any
	if selection.CarrierName != "" {
		carrierName = selection.CarrierName
	}

	var deliveryTime any
	if selection.DeliveryTimeDays != nil {
		deliveryTime = *selection.DeliveryTimeDays
	}

	_, err := r.pool.Exec(ctx, `
		insert into public.cart_shipping_selections (
			cart_id,
			shipping_box_id,
			provider,
			service_code,
			service_name,
			carrier_name,
			price_cents,
			delivery_time_days,
			package_weight_g,
			package_height_mm,
			package_width_mm,
			package_length_mm,
			input_hash,
			quoted_at,
			expires_at
		) values (
			$1::uuid,
			$2::uuid,
			$3,
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
			$14,
			$15
		)
		on conflict (cart_id) do update set
			shipping_box_id = excluded.shipping_box_id,
			provider = excluded.provider,
			service_code = excluded.service_code,
			service_name = excluded.service_name,
			carrier_name = excluded.carrier_name,
			price_cents = excluded.price_cents,
			delivery_time_days = excluded.delivery_time_days,
			package_weight_g = excluded.package_weight_g,
			package_height_mm = excluded.package_height_mm,
			package_width_mm = excluded.package_width_mm,
			package_length_mm = excluded.package_length_mm,
			input_hash = excluded.input_hash,
			quoted_at = excluded.quoted_at,
			expires_at = excluded.expires_at,
			updated_at = now()
	`, cartID, selection.ShippingBoxID, selection.Provider, selection.ServiceCode, selection.ServiceName, carrierName, selection.PriceCents, deliveryTime, selection.PackageWeightG, selection.PackageHeightMM, selection.PackageWidthMM, selection.PackageLengthMM, selection.InputHash, selection.QuotedAt, selection.ExpiresAt)

	return err
}

type cartItemScanner interface {
	Scan(dest ...any) error
}

func scanCartItem(scanner cartItemScanner) (CartItem, error) {
	var item CartItem
	var productWeight pgtype.Int8
	var productHeight pgtype.Int4
	var productWidth pgtype.Int4
	var productLength pgtype.Int4
	var variantWeight pgtype.Int8
	var variantHeight pgtype.Int4
	var variantWidth pgtype.Int4
	var variantLength pgtype.Int4

	if err := scanner.Scan(
		&item.ID,
		&item.ProductID,
		&item.VariantID,
		&item.Quantity,
		&item.ProductName,
		&item.VariantName,
		&productWeight,
		&productHeight,
		&productWidth,
		&productLength,
		&variantWeight,
		&variantHeight,
		&variantWidth,
		&variantLength,
	); err != nil {
		return CartItem{}, err
	}

	item.ProductProfile = profileFromColumns(productWeight, productHeight, productWidth, productLength)
	item.VariantProfile = profileFromColumns(variantWeight, variantHeight, variantWidth, variantLength)

	return item, nil
}

func profileFromColumns(weight pgtype.Int8, height pgtype.Int4, width pgtype.Int4, length pgtype.Int4) *ShippingProfile {
	if !weight.Valid && !height.Valid && !width.Valid && !length.Valid {
		return nil
	}
	if !weight.Valid || !height.Valid || !width.Valid || !length.Valid {
		return &ShippingProfile{}
	}

	return &ShippingProfile{
		WeightG: weight.Int64,
		Dimensions: DimensionsMM{
			Height: int(height.Int32),
			Width:  int(width.Int32),
			Length: int(length.Int32),
		},
	}
}
