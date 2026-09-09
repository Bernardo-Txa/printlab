package products

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

func (r *PostgresRepository) ListActiveCategories(ctx context.Context) ([]Category, error) {
	if r == nil || r.pool == nil {
		return nil, ErrUnavailable
	}

	rows, err := r.pool.Query(ctx, `
		select
			id::text,
			name,
			slug,
			coalesce(description, '')
		from public.categories
		where is_active = true
		order by name asc
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var category Category
		if err := rows.Scan(
			&category.ID,
			&category.Name,
			&category.Slug,
			&category.Description,
		); err != nil {
			return nil, err
		}

		categories = append(categories, category)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return categories, nil
}

func (r *PostgresRepository) ListActiveProducts(ctx context.Context, filter ListFilter) ([]Product, error) {
	if r == nil || r.pool == nil {
		return nil, ErrUnavailable
	}

	rows, err := r.pool.Query(ctx, `
		select
			p.id::text,
			coalesce(p.category_id::text, ''),
			p.name,
			p.slug,
			coalesce(p.short_description, ''),
			coalesce(p.description, ''),
			p.price_cents,
			coalesce(variant_prices.min_price_cents, p.price_cents),
			coalesce(variant_prices.variant_count, 0) > 0
				and coalesce(variant_prices.min_price_cents, p.price_cents) <> coalesce(variant_prices.max_price_cents, p.price_cents),
			p.is_featured,
			coalesce(c.id::text, ''),
			coalesce(c.name, ''),
			coalesce(c.slug, ''),
			coalesce(c.description, ''),
			coalesce(primary_image.id::text, ''),
			coalesce(primary_image.storage_path, ''),
			coalesce(primary_image.alt_text, ''),
			coalesce(primary_image.sort_order, 0),
			coalesce(primary_image.is_primary, false)
		from public.products p
		left join public.categories c
			on c.id = p.category_id
			and c.is_active = true
		left join lateral (
			select
				min(coalesce(v.price_cents, p.price_cents)) as min_price_cents,
				max(coalesce(v.price_cents, p.price_cents)) as max_price_cents,
				count(*) as variant_count
			from public.product_variants v
			where v.product_id = p.id
				and v.is_active = true
		) variant_prices on true
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
				and pi.is_primary = true
			order by pi.sort_order asc, pi.created_at asc, pi.id asc
			limit 1
		) primary_image on true
		where
			p.is_active = true
			and (
				$1::text = ''
				or exists (
					select 1
					from public.categories filter_categories
					where filter_categories.id = p.category_id
						and filter_categories.slug = $1
						and filter_categories.is_active = true
				)
			)
		order by p.is_featured desc, p.created_at desc, p.name asc
	`, filter.CategorySlug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		product, err := scanProductList(rows)
		if err != nil {
			return nil, err
		}

		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return products, nil
}

func (r *PostgresRepository) GetActiveProductDetailBySlug(ctx context.Context, slug string) (ProductDetail, error) {
	if r == nil || r.pool == nil {
		return ProductDetail{}, ErrUnavailable
	}

	row := r.pool.QueryRow(ctx, `
		select
			p.id::text,
			coalesce(p.category_id::text, ''),
			p.name,
			p.slug,
			coalesce(p.short_description, ''),
			coalesce(p.description, ''),
			p.price_cents,
			p.is_featured,
			coalesce(c.id::text, ''),
			coalesce(c.name, ''),
			coalesce(c.slug, ''),
			coalesce(c.description, '')
		from public.products p
		left join public.categories c
			on c.id = p.category_id
			and c.is_active = true
		where p.is_active = true
			and p.slug = $1
	`, slug)

	product, err := scanProduct(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ProductDetail{}, ErrNotFound
		}

		return ProductDetail{}, err
	}

	variants, err := r.listActiveVariants(ctx, product.ID)
	if err != nil {
		return ProductDetail{}, err
	}

	filamentsByVariant, err := r.listVariantFilaments(ctx, product.ID)
	if err != nil {
		return ProductDetail{}, err
	}

	imagesByProduct, imagesByVariant, err := r.listProductImages(ctx, product.ID)
	if err != nil {
		return ProductDetail{}, err
	}

	for i := range variants {
		variants[i].Filaments = filamentsByVariant[variants[i].ID]
		variants[i].Images = imagesByVariant[variants[i].ID]
	}
	product.Images = imagesByProduct

	return ProductDetail{
		Product:  product,
		Variants: variants,
	}, nil
}

type productScanner interface {
	Scan(dest ...any) error
}

func scanProduct(scanner productScanner) (Product, error) {
	var product Product
	var productCategoryID string
	var activeCategoryID string
	var categoryName string
	var categorySlug string
	var categoryDescription string

	if err := scanner.Scan(
		&product.ID,
		&productCategoryID,
		&product.Name,
		&product.Slug,
		&product.ShortDescription,
		&product.Description,
		&product.PriceCents,
		&product.IsFeatured,
		&activeCategoryID,
		&categoryName,
		&categorySlug,
		&categoryDescription,
	); err != nil {
		return Product{}, err
	}

	if activeCategoryID != "" {
		product.Category = &Category{
			ID:          activeCategoryID,
			Name:        categoryName,
			Slug:        categorySlug,
			Description: categoryDescription,
		}
	}

	return product, nil
}

func scanProductList(scanner productScanner) (Product, error) {
	var product Product
	var productCategoryID string
	var activeCategoryID string
	var categoryName string
	var categorySlug string
	var categoryDescription string
	var imageID string
	var imageStoragePath string
	var imageAltText string
	var imageSortOrder int
	var imageIsPrimary bool

	if err := scanner.Scan(
		&product.ID,
		&productCategoryID,
		&product.Name,
		&product.Slug,
		&product.ShortDescription,
		&product.Description,
		&product.PriceCents,
		&product.DisplayPriceCents,
		&product.PriceFrom,
		&product.IsFeatured,
		&activeCategoryID,
		&categoryName,
		&categorySlug,
		&categoryDescription,
		&imageID,
		&imageStoragePath,
		&imageAltText,
		&imageSortOrder,
		&imageIsPrimary,
	); err != nil {
		return Product{}, err
	}

	product.HasDisplayPrice = true

	if activeCategoryID != "" {
		product.Category = &Category{
			ID:          activeCategoryID,
			Name:        categoryName,
			Slug:        categorySlug,
			Description: categoryDescription,
		}
	}

	if imageID != "" {
		product.PrimaryImage = &ProductImage{
			ID:          imageID,
			ProductID:   product.ID,
			StoragePath: imageStoragePath,
			AltText:     imageAltText,
			SortOrder:   imageSortOrder,
			IsPrimary:   imageIsPrimary,
		}
	}

	return product, nil
}

func (r *PostgresRepository) listActiveVariants(ctx context.Context, productID string) ([]ProductVariant, error) {
	rows, err := r.pool.Query(ctx, `
		select
			id::text,
			product_id::text,
			name,
			slug,
			coalesce(sku, ''),
			price_cents,
			is_active,
			is_default,
			sort_order,
			print_time_minutes
		from public.product_variants
		where product_id = $1::uuid
			and is_active = true
		order by is_default desc, sort_order asc, name asc
	`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var variants []ProductVariant
	for rows.Next() {
		var variant ProductVariant
		var priceCents pgtype.Int8
		var printTimeMinutes pgtype.Int4
		if err := rows.Scan(
			&variant.ID,
			&variant.ProductID,
			&variant.Name,
			&variant.Slug,
			&variant.SKU,
			&priceCents,
			&variant.IsActive,
			&variant.IsDefault,
			&variant.SortOrder,
			&printTimeMinutes,
		); err != nil {
			return nil, err
		}

		if priceCents.Valid {
			value := priceCents.Int64
			variant.PriceCents = &value
		}

		if printTimeMinutes.Valid {
			value := int(printTimeMinutes.Int32)
			variant.PrintTimeMinutes = &value
		}

		variants = append(variants, variant)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return variants, nil
}

func (r *PostgresRepository) listVariantFilaments(ctx context.Context, productID string) (map[string][]VariantFilament, error) {
	rows, err := r.pool.Query(ctx, `
		select
			vf.id::text,
			vf.variant_id::text,
			m.id::text,
			m.name,
			m.slug,
			coalesce(m.description, ''),
			c.id::text,
			c.name,
			c.slug,
			coalesce(c.hex_color, ''),
			vf.estimated_weight_mg,
			coalesce(vf.label, ''),
			vf.sort_order
		from public.variant_filaments vf
		join public.product_variants v
			on v.id = vf.variant_id
		join public.materials m
			on m.id = vf.material_id
		join public.colors c
			on c.id = vf.color_id
		where v.product_id = $1::uuid
			and v.is_active = true
		order by v.is_default desc, v.sort_order asc, v.name asc, vf.sort_order asc, vf.created_at asc, vf.id asc
	`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	filamentsByVariant := map[string][]VariantFilament{}
	for rows.Next() {
		var filament VariantFilament
		if err := rows.Scan(
			&filament.ID,
			&filament.VariantID,
			&filament.Material.ID,
			&filament.Material.Name,
			&filament.Material.Slug,
			&filament.Material.Description,
			&filament.Color.ID,
			&filament.Color.Name,
			&filament.Color.Slug,
			&filament.Color.HexColor,
			&filament.EstimatedWeightMg,
			&filament.Label,
			&filament.SortOrder,
		); err != nil {
			return nil, err
		}

		filamentsByVariant[filament.VariantID] = append(filamentsByVariant[filament.VariantID], filament)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return filamentsByVariant, nil
}

func (r *PostgresRepository) listProductImages(ctx context.Context, productID string) ([]ProductImage, map[string][]ProductImage, error) {
	rows, err := r.pool.Query(ctx, `
		select
			id::text,
			product_id::text,
			coalesce(variant_id::text, ''),
			storage_path,
			coalesce(alt_text, ''),
			sort_order,
			is_primary
		from public.product_images
		where product_id = $1::uuid
		order by
			case when is_primary then 0 else 1 end,
			sort_order asc,
			created_at asc,
			id asc
	`, productID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var productImages []ProductImage
	imagesByVariant := map[string][]ProductImage{}
	for rows.Next() {
		var image ProductImage
		if err := rows.Scan(
			&image.ID,
			&image.ProductID,
			&image.VariantID,
			&image.StoragePath,
			&image.AltText,
			&image.SortOrder,
			&image.IsPrimary,
		); err != nil {
			return nil, nil, err
		}

		if image.VariantID == "" {
			productImages = append(productImages, image)
			continue
		}

		imagesByVariant[image.VariantID] = append(imagesByVariant[image.VariantID], image)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return productImages, imagesByVariant, nil
}
