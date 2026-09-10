package cart

import (
	"context"
	"errors"
	"time"

	"github.com/Bernardo-Txa/printlab/internal/products"
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

func (r *PostgresRepository) FindActiveCart(ctx context.Context, tokenHash []byte, now time.Time) (Cart, error) {
	if r == nil || r.pool == nil {
		return Cart{}, ErrUnavailable
	}

	var activeCart Cart
	err := r.pool.QueryRow(ctx, `
		select
			id::text,
			expires_at
		from public.carts
		where token_hash = $1
			and expires_at > $2
			and converted_at is null
	`, tokenHash, now).Scan(&activeCart.ID, &activeCart.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Cart{}, ErrNotFound
		}

		return Cart{}, err
	}

	return activeCart, nil
}

func (r *PostgresRepository) CreateCart(ctx context.Context, tokenHash []byte, expiresAt time.Time) (Cart, error) {
	if r == nil || r.pool == nil {
		return Cart{}, ErrUnavailable
	}

	var activeCart Cart
	err := r.pool.QueryRow(ctx, `
		with upserted_cart as (
			insert into public.carts (
				token_hash,
				expires_at
			) values (
				$1,
				$2
			)
			on conflict (token_hash) do update set
				updated_at = now(),
				expires_at = excluded.expires_at
			where public.carts.converted_at is null
			returning id, expires_at
		),
		cleared_items as (
			delete from public.cart_items
			where cart_id = (select id from upserted_cart)
		)
		select
			id::text,
			expires_at
		from upserted_cart
	`, tokenHash, expiresAt).Scan(&activeCart.ID, &activeCart.ExpiresAt)
	if err != nil {
		return Cart{}, err
	}

	return activeCart, nil
}

func (r *PostgresRepository) RenewCart(ctx context.Context, cartID string, expiresAt time.Time) (Cart, error) {
	if r == nil || r.pool == nil {
		return Cart{}, ErrUnavailable
	}

	var activeCart Cart
	err := r.pool.QueryRow(ctx, `
		update public.carts
		set
			updated_at = now(),
			expires_at = $2
		where id = $1::uuid
			and converted_at is null
		returning
			id::text,
			expires_at
	`, cartID, expiresAt).Scan(&activeCart.ID, &activeCart.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Cart{}, ErrNotFound
		}

		return Cart{}, err
	}

	return activeCart, nil
}

func (r *PostgresRepository) ProductForAddBySlug(ctx context.Context, slug string) (ProductForAdd, error) {
	if r == nil || r.pool == nil {
		return ProductForAdd{}, ErrUnavailable
	}

	var product ProductForAdd
	err := r.pool.QueryRow(ctx, `
		select
			id::text,
			name,
			slug,
			price_cents,
			is_active
		from public.products
		where slug = $1
	`, slug).Scan(
		&product.ID,
		&product.Name,
		&product.Slug,
		&product.PriceCents,
		&product.IsActive,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ProductForAdd{}, ErrNotFound
		}

		return ProductForAdd{}, err
	}

	variants, err := r.listProductVariantsForCart(ctx, product.ID)
	if err != nil {
		return ProductForAdd{}, err
	}
	product.Variants = variants

	return product, nil
}

func (r *PostgresRepository) listProductVariantsForCart(ctx context.Context, productID string) ([]VariantForAdd, error) {
	rows, err := r.pool.Query(ctx, `
		select
			id::text,
			product_id::text,
			name,
			slug,
			price_cents,
			is_active
		from public.product_variants
		where product_id = $1::uuid
		order by is_default desc, sort_order asc, name asc
	`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var variants []VariantForAdd
	for rows.Next() {
		var variant VariantForAdd
		var priceCents pgtype.Int8
		if err := rows.Scan(
			&variant.ID,
			&variant.ProductID,
			&variant.Name,
			&variant.Slug,
			&priceCents,
			&variant.IsActive,
		); err != nil {
			return nil, err
		}

		if priceCents.Valid {
			value := priceCents.Int64
			variant.PriceCents = &value
		}

		variants = append(variants, variant)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return variants, nil
}

func (r *PostgresRepository) ListItems(ctx context.Context, cartID string) ([]StoredItem, error) {
	if r == nil || r.pool == nil {
		return nil, ErrUnavailable
	}

	rows, err := r.pool.Query(ctx, `
		select
			ci.id::text,
			ci.cart_id::text,
			ci.quantity,
			p.id::text,
			p.name,
			p.slug,
			p.price_cents,
			p.is_active,
			coalesce(p.short_description, ''),
			coalesce(v.id::text, ''),
			coalesce(v.product_id::text, ''),
			coalesce(v.name, ''),
			coalesce(v.slug, ''),
			v.price_cents,
			coalesce(v.is_active, false),
			exists (
				select 1
				from public.product_variants active_variants
				where active_variants.product_id = p.id
					and active_variants.is_active = true
			),
			coalesce(product_image.id::text, ''),
			coalesce(product_image.storage_path, ''),
			coalesce(product_image.alt_text, ''),
			coalesce(product_image.sort_order, 0),
			coalesce(product_image.is_primary, false),
			coalesce(variant_image.id::text, ''),
			coalesce(variant_image.storage_path, ''),
			coalesce(variant_image.alt_text, ''),
			coalesce(variant_image.sort_order, 0),
			coalesce(variant_image.is_primary, false)
		from public.cart_items ci
		join public.products p
			on p.id = ci.product_id
		left join public.product_variants v
			on v.id = ci.variant_id
			and v.product_id = ci.product_id
		left join lateral (
			select
				pi.id,
				pi.storage_path,
				pi.alt_text,
				pi.sort_order,
				pi.is_primary
			from public.product_images pi
			where pi.product_id = p.id
				and pi.variant_id is null
			order by
				case when pi.is_primary then 0 else 1 end,
				pi.sort_order asc,
				pi.created_at asc,
				pi.id asc
			limit 1
		) product_image on true
		left join lateral (
			select
				pi.id,
				pi.storage_path,
				pi.alt_text,
				pi.sort_order,
				pi.is_primary
			from public.product_images pi
			where pi.variant_id = ci.variant_id
			order by
				case when pi.is_primary then 0 else 1 end,
				pi.sort_order asc,
				pi.created_at asc,
				pi.id asc
			limit 1
		) variant_image on ci.variant_id is not null
		where ci.cart_id = $1::uuid
		order by ci.created_at asc, ci.id asc
	`, cartID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []StoredItem
	for rows.Next() {
		item, err := scanStoredItem(rows)
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

type itemScanner interface {
	Scan(dest ...any) error
}

func scanStoredItem(scanner itemScanner) (StoredItem, error) {
	var item StoredItem
	var variantID string
	var variantProductID string
	var variantName string
	var variantSlug string
	var variantPriceCents pgtype.Int8
	var variantIsActive bool
	var productImageID string
	var productImageStoragePath string
	var productImageAltText string
	var productImageSortOrder int
	var productImageIsPrimary bool
	var variantImageID string
	var variantImageStoragePath string
	var variantImageAltText string
	var variantImageSortOrder int
	var variantImageIsPrimary bool

	if err := scanner.Scan(
		&item.ID,
		&item.CartID,
		&item.Quantity,
		&item.Product.ID,
		&item.Product.Name,
		&item.Product.Slug,
		&item.Product.PriceCents,
		&item.Product.IsActive,
		&item.Product.Description,
		&variantID,
		&variantProductID,
		&variantName,
		&variantSlug,
		&variantPriceCents,
		&variantIsActive,
		&item.ProductHasActiveVariants,
		&productImageID,
		&productImageStoragePath,
		&productImageAltText,
		&productImageSortOrder,
		&productImageIsPrimary,
		&variantImageID,
		&variantImageStoragePath,
		&variantImageAltText,
		&variantImageSortOrder,
		&variantImageIsPrimary,
	); err != nil {
		return StoredItem{}, err
	}

	if variantID != "" {
		item.Variant = &VariantSnapshot{
			ID:        variantID,
			ProductID: variantProductID,
			Name:      variantName,
			Slug:      variantSlug,
			IsActive:  variantIsActive,
		}
		if variantPriceCents.Valid {
			value := variantPriceCents.Int64
			item.Variant.PriceCents = &value
		}
	}

	if productImageID != "" {
		item.ProductImage = &products.ProductImage{
			ID:          productImageID,
			ProductID:   item.Product.ID,
			StoragePath: productImageStoragePath,
			AltText:     productImageAltText,
			SortOrder:   productImageSortOrder,
			IsPrimary:   productImageIsPrimary,
		}
	}

	if variantImageID != "" {
		item.VariantImage = &products.ProductImage{
			ID:          variantImageID,
			ProductID:   item.Product.ID,
			VariantID:   variantID,
			StoragePath: variantImageStoragePath,
			AltText:     variantImageAltText,
			SortOrder:   variantImageSortOrder,
			IsPrimary:   variantImageIsPrimary,
		}
	}

	return item, nil
}

func (r *PostgresRepository) AddItem(ctx context.Context, cartID string, productID string, variantID *string, quantity int) error {
	if r == nil || r.pool == nil {
		return ErrUnavailable
	}

	if variantID == nil {
		return r.addItemWithoutVariant(ctx, cartID, productID, quantity)
	}

	return r.addItemWithVariant(ctx, cartID, productID, *variantID, quantity)
}

func (r *PostgresRepository) addItemWithoutVariant(ctx context.Context, cartID string, productID string, quantity int) error {
	var storedQuantity int
	err := r.pool.QueryRow(ctx, `
		insert into public.cart_items (
			cart_id,
			product_id,
			variant_id,
			quantity
		)
		select
			active_cart.id,
			$2::uuid,
			null,
			$3
		from public.carts active_cart
		where active_cart.id = $1::uuid
			and active_cart.converted_at is null
		on conflict (cart_id, product_id) where variant_id is null
		do update set
			quantity = public.cart_items.quantity + excluded.quantity,
			updated_at = now()
		where public.cart_items.quantity <= 99 - excluded.quantity
		returning quantity
	`, cartID, productID, quantity).Scan(&storedQuantity)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			active, activeErr := r.activeCartExists(ctx, cartID)
			if activeErr != nil {
				return activeErr
			}
			if !active {
				return ErrNotFound
			}

			return ErrQuantityLimit
		}

		return err
	}

	return nil
}

func (r *PostgresRepository) addItemWithVariant(ctx context.Context, cartID string, productID string, variantID string, quantity int) error {
	var storedQuantity int
	err := r.pool.QueryRow(ctx, `
		insert into public.cart_items (
			cart_id,
			product_id,
			variant_id,
			quantity
		)
		select
			active_cart.id,
			$2::uuid,
			$3::uuid,
			$4
		from public.carts active_cart
		where active_cart.id = $1::uuid
			and active_cart.converted_at is null
		on conflict (cart_id, product_id, variant_id) where variant_id is not null
		do update set
			quantity = public.cart_items.quantity + excluded.quantity,
			updated_at = now()
		where public.cart_items.quantity <= 99 - excluded.quantity
		returning quantity
	`, cartID, productID, variantID, quantity).Scan(&storedQuantity)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			active, activeErr := r.activeCartExists(ctx, cartID)
			if activeErr != nil {
				return activeErr
			}
			if !active {
				return ErrNotFound
			}

			return ErrQuantityLimit
		}

		return err
	}

	return nil
}

func (r *PostgresRepository) UpdateItemQuantity(ctx context.Context, cartID string, itemID string, quantity int) (bool, error) {
	if r == nil || r.pool == nil {
		return false, ErrUnavailable
	}

	commandTag, err := r.pool.Exec(ctx, `
		update public.cart_items
		set
			quantity = $3,
			updated_at = now()
		where cart_id = $1::uuid
			and id = $2::uuid
			and exists (
				select 1
				from public.carts active_cart
				where active_cart.id = public.cart_items.cart_id
					and active_cart.converted_at is null
			)
	`, cartID, itemID, quantity)
	if err != nil {
		return false, err
	}

	return commandTag.RowsAffected() > 0, nil
}

func (r *PostgresRepository) RemoveItem(ctx context.Context, cartID string, itemID string) (bool, error) {
	if r == nil || r.pool == nil {
		return false, ErrUnavailable
	}

	commandTag, err := r.pool.Exec(ctx, `
		delete from public.cart_items
		where cart_id = $1::uuid
			and id = $2::uuid
			and exists (
				select 1
				from public.carts active_cart
				where active_cart.id = public.cart_items.cart_id
					and active_cart.converted_at is null
			)
	`, cartID, itemID)
	if err != nil {
		return false, err
	}

	return commandTag.RowsAffected() > 0, nil
}

func (r *PostgresRepository) activeCartExists(ctx context.Context, cartID string) (bool, error) {
	var active bool
	err := r.pool.QueryRow(ctx, `
		select exists (
			select 1
			from public.carts
			where id = $1::uuid
				and converted_at is null
		)
	`, cartID).Scan(&active)
	if err != nil {
		return false, err
	}

	return active, nil
}
