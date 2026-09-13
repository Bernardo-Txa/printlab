package admin

import (
	"context"
	"errors"
	"strconv"

	"github.com/Bernardo-Txa/printlab/internal/products"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func (r *PostgresRepository) ListAdminProducts(ctx context.Context, filter AdminProductListFilter) (AdminProductListPage, error) {
	if r == nil || r.pool == nil {
		return AdminProductListPage{}, ErrUnavailable
	}

	filter = NormalizeAdminProductListFilter(filter)
	rows, err := r.pool.Query(ctx, `
		select
			p.id::text,
			p.name,
			p.slug,
			coalesce(c.name, ''),
			coalesce(c.is_active, false),
			p.price_cents,
			p.is_active,
			p.is_featured,
			count(v.id)::integer,
			p.shipping_weight_g is not null
		from public.products p
		left join public.categories c
			on c.id = p.category_id
		left join public.product_variants v
			on v.product_id = p.id
		where
			($1::text <> 'active' or p.is_active = true)
			and ($1::text <> 'inactive' or p.is_active = false)
			and (
				$2::text = ''
				or p.name ilike '%' || $2 || '%'
				or p.slug ilike '%' || $2 || '%'
			)
		group by
			p.id,
			p.name,
			p.slug,
			c.name,
			c.is_active,
			p.price_cents,
			p.is_active,
			p.is_featured,
			p.shipping_weight_g
		order by p.is_active desc, p.name asc, p.id asc
	`, filter.Status, filter.Query)
	if err != nil {
		return AdminProductListPage{}, ErrUnavailable
	}
	defer rows.Close()

	var items []AdminProductListItem
	for rows.Next() {
		var item AdminProductListItem
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Slug,
			&item.CategoryName,
			&item.CategoryActive,
			&item.PriceCents,
			&item.IsActive,
			&item.IsFeatured,
			&item.VariantCount,
			&item.HasShippingProfile,
		); err != nil {
			return AdminProductListPage{}, ErrUnavailable
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return AdminProductListPage{}, ErrUnavailable
	}

	return PrepareAdminProductListPage(filter, items), nil
}

func (r *PostgresRepository) GetAdminProductForm(ctx context.Context, productID string) (AdminProductFormPage, error) {
	if r == nil || r.pool == nil {
		return AdminProductFormPage{}, ErrUnavailable
	}

	page := AdminProductFormPage{
		BackURL: "/admin/produtos",
		Errors:  AdminFieldErrors{},
	}
	if productID == "" {
		options, err := r.listCategoryOptions(ctx, "")
		if err != nil {
			return AdminProductFormPage{}, err
		}
		page.Categories = options
		return page, nil
	}

	var categoryID pgtype.Text
	var shortDescription pgtype.Text
	var description pgtype.Text
	var shippingWeight pgtype.Int8
	var shippingHeight pgtype.Int4
	var shippingWidth pgtype.Int4
	var shippingLength pgtype.Int4
	var priceCents int64
	err := r.pool.QueryRow(ctx, `
		select
			id::text,
			coalesce(category_id::text, ''),
			name,
			slug,
			short_description,
			description,
			price_cents,
			is_active,
			is_featured,
			shipping_weight_g,
			shipping_height_mm,
			shipping_width_mm,
			shipping_length_mm
		from public.products
		where id = $1::uuid
	`, productID).Scan(
		&productID,
		&categoryID,
		&page.Form.Name,
		&page.Form.Slug,
		&shortDescription,
		&description,
		&priceCents,
		&page.Form.IsActive,
		&page.Form.IsFeatured,
		&shippingWeight,
		&shippingHeight,
		&shippingWidth,
		&shippingLength,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AdminProductFormPage{}, ErrCatalogNotFound
		}
		return AdminProductFormPage{}, ErrUnavailable
	}
	if categoryID.Valid {
		page.Form.CategoryID = categoryID.String
	}
	if shortDescription.Valid {
		page.Form.ShortDescription = shortDescription.String
	}
	if description.Valid {
		page.Form.Description = description.String
	}
	page.Form.PriceBRL = FormatAdminCentsInput(priceCents)
	fillShippingForm(&page.Form.UseShipping, &page.Form.ShippingWeightG, &page.Form.ShippingHeightMM, &page.Form.ShippingWidthMM, &page.Form.ShippingLengthMM, shippingWeight, shippingHeight, shippingWidth, shippingLength)

	options, err := r.listCategoryOptions(ctx, page.Form.CategoryID)
	if err != nil {
		return AdminProductFormPage{}, err
	}
	page.Categories = options

	variants, err := r.listAdminVariantItems(ctx, productID)
	if err != nil {
		return AdminProductFormPage{}, err
	}
	page.Variants = variants
	page.Title = "Editar produto"
	page.Action = "/admin/produtos/" + productID
	page.SubmitLabel = "Salvar produto"

	return page, nil
}

func (r *PostgresRepository) CreateAdminProduct(ctx context.Context, input AdminProductSaveInput) (string, error) {
	if r == nil || r.pool == nil {
		return "", ErrUnavailable
	}
	if err := r.ensureProductCategoryAllowed(ctx, input.CategoryID, ""); err != nil {
		return "", err
	}

	var id string
	err := r.pool.QueryRow(ctx, `
		insert into public.products (
			category_id,
			name,
			slug,
			short_description,
			description,
			price_cents,
			shipping_weight_g,
			shipping_height_mm,
			shipping_width_mm,
			shipping_length_mm,
			is_active,
			is_featured
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
		returning id::text
	`, nullableUUID(input.CategoryID), input.Name, input.Slug, nullableText(input.ShortDescription), nullableText(input.Description), input.PriceCents, shippingWeightValue(input.ShippingProfile), shippingHeightValue(input.ShippingProfile), shippingWidthValue(input.ShippingProfile), shippingLengthValue(input.ShippingProfile), input.IsActive, input.IsFeatured).Scan(&id)
	if err != nil {
		return "", mapCatalogError(err)
	}

	return id, nil
}

func (r *PostgresRepository) UpdateAdminProduct(ctx context.Context, input AdminProductSaveInput) error {
	if r == nil || r.pool == nil {
		return ErrUnavailable
	}
	if err := r.ensureProductCategoryAllowed(ctx, input.CategoryID, input.ID); err != nil {
		return err
	}

	tag, err := r.pool.Exec(ctx, `
		update public.products
		set
			category_id = $2::uuid,
			name = $3,
			slug = $4,
			short_description = $5,
			description = $6,
			price_cents = $7,
			shipping_weight_g = $8,
			shipping_height_mm = $9,
			shipping_width_mm = $10,
			shipping_length_mm = $11,
			is_active = $12,
			is_featured = $13,
			updated_at = now()
		where id = $1::uuid
	`, input.ID, nullableUUID(input.CategoryID), input.Name, input.Slug, nullableText(input.ShortDescription), nullableText(input.Description), input.PriceCents, shippingWeightValue(input.ShippingProfile), shippingHeightValue(input.ShippingProfile), shippingWidthValue(input.ShippingProfile), shippingLengthValue(input.ShippingProfile), input.IsActive, input.IsFeatured)
	if err != nil {
		return mapCatalogError(err)
	}
	if tag.RowsAffected() != 1 {
		return ErrCatalogNotFound
	}

	return nil
}

func (r *PostgresRepository) ListAdminCategories(ctx context.Context) (AdminCategoryListPage, error) {
	if r == nil || r.pool == nil {
		return AdminCategoryListPage{}, ErrUnavailable
	}

	rows, err := r.pool.Query(ctx, `
		select
			id::text,
			name,
			slug,
			coalesce(description, ''),
			is_active
		from public.categories
		order by is_active desc, name asc, id asc
	`)
	if err != nil {
		return AdminCategoryListPage{}, ErrUnavailable
	}
	defer rows.Close()

	var items []AdminCategoryListItem
	for rows.Next() {
		var item AdminCategoryListItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Slug, &item.Description, &item.IsActive); err != nil {
			return AdminCategoryListPage{}, ErrUnavailable
		}
		item.StatusLabel = ActiveStatusLabel(item.IsActive)
		item.DetailURL = "/admin/categorias/" + item.ID
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return AdminCategoryListPage{}, ErrUnavailable
	}

	return AdminCategoryListPage{Categories: items}, nil
}

func (r *PostgresRepository) GetAdminCategoryForm(ctx context.Context, categoryID string) (AdminCategoryFormPage, error) {
	if r == nil || r.pool == nil {
		return AdminCategoryFormPage{}, ErrUnavailable
	}
	page := AdminCategoryFormPage{
		Title:       "Editar categoria",
		Action:      "/admin/categorias/" + categoryID,
		SubmitLabel: "Salvar categoria",
		BackURL:     "/admin/categorias",
		Errors:      AdminFieldErrors{},
	}
	var description pgtype.Text
	err := r.pool.QueryRow(ctx, `
		select name, slug, description, is_active
		from public.categories
		where id = $1::uuid
	`, categoryID).Scan(&page.Form.Name, &page.Form.Slug, &description, &page.Form.IsActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AdminCategoryFormPage{}, ErrCatalogNotFound
		}
		return AdminCategoryFormPage{}, ErrUnavailable
	}
	if description.Valid {
		page.Form.Description = description.String
	}

	return page, nil
}

func (r *PostgresRepository) CreateAdminCategory(ctx context.Context, input AdminCategorySaveInput) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		insert into public.categories (name, slug, description, is_active)
		values ($1, $2, $3, $4)
		returning id::text
	`, input.Name, input.Slug, nullableText(input.Description), input.IsActive).Scan(&id)
	if err != nil {
		return "", mapCatalogError(err)
	}

	return id, nil
}

func (r *PostgresRepository) UpdateAdminCategory(ctx context.Context, input AdminCategorySaveInput) error {
	tag, err := r.pool.Exec(ctx, `
		update public.categories
		set
			name = $2,
			slug = $3,
			description = $4,
			is_active = $5,
			updated_at = now()
		where id = $1::uuid
	`, input.ID, input.Name, input.Slug, nullableText(input.Description), input.IsActive)
	if err != nil {
		return mapCatalogError(err)
	}
	if tag.RowsAffected() != 1 {
		return ErrCatalogNotFound
	}

	return nil
}

func (r *PostgresRepository) NewAdminVariantForm(ctx context.Context, productID string) (AdminVariantFormPage, error) {
	product, err := r.adminProductHeader(ctx, productID)
	if err != nil {
		return AdminVariantFormPage{}, err
	}
	materials, colors, err := r.recipeOptions(ctx)
	if err != nil {
		return AdminVariantFormPage{}, err
	}

	return AdminVariantFormPage{
		Title:           "Nova variante",
		Action:          "/admin/produtos/" + productID + "/variantes",
		SubmitLabel:     "Criar variante",
		BackURL:         product.DetailURL,
		IsNew:           true,
		Product:         product,
		Errors:          AdminFieldErrors{},
		MaterialOptions: materials,
		ColorOptions:    colors,
	}, nil
}

func (r *PostgresRepository) GetAdminVariantForm(ctx context.Context, productID string, variantID string) (AdminVariantFormPage, error) {
	product, err := r.adminProductHeader(ctx, productID)
	if err != nil {
		return AdminVariantFormPage{}, err
	}
	materials, colors, err := r.recipeOptions(ctx)
	if err != nil {
		return AdminVariantFormPage{}, err
	}

	page := AdminVariantFormPage{
		Title:           "Editar variante",
		Action:          "/admin/produtos/" + productID + "/variantes/" + variantID,
		SubmitLabel:     "Salvar variante",
		BackURL:         product.DetailURL,
		Product:         product,
		VariantID:       variantID,
		Errors:          AdminFieldErrors{},
		MaterialOptions: materials,
		ColorOptions:    colors,
	}
	var sku pgtype.Text
	var price pgtype.Int8
	var printTime pgtype.Int4
	var shippingWeight pgtype.Int8
	var shippingHeight pgtype.Int4
	var shippingWidth pgtype.Int4
	var shippingLength pgtype.Int4
	err = r.pool.QueryRow(ctx, `
		select
			name,
			slug,
			sku,
			price_cents,
			is_active,
			is_default,
			sort_order,
			print_time_minutes,
			shipping_weight_g,
			shipping_height_mm,
			shipping_width_mm,
			shipping_length_mm
		from public.product_variants
		where id = $1::uuid
			and product_id = $2::uuid
	`, variantID, productID).Scan(
		&page.Form.Name,
		&page.Form.Slug,
		&sku,
		&price,
		&page.Form.IsActive,
		&page.Form.IsDefault,
		&page.Form.SortOrder,
		&printTime,
		&shippingWeight,
		&shippingHeight,
		&shippingWidth,
		&shippingLength,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AdminVariantFormPage{}, ErrCatalogNotFound
		}
		return AdminVariantFormPage{}, ErrUnavailable
	}
	if sku.Valid {
		page.Form.SKU = sku.String
	}
	if price.Valid {
		page.Form.PriceBRL = FormatAdminCentsInput(price.Int64)
	}
	if printTime.Valid {
		page.Form.PrintTimeMinutes = strconv.FormatInt(int64(printTime.Int32), 10)
	}
	fillShippingForm(&page.Form.UseShipping, &page.Form.ShippingWeightG, &page.Form.ShippingHeightMM, &page.Form.ShippingWidthMM, &page.Form.ShippingLengthMM, shippingWeight, shippingHeight, shippingWidth, shippingLength)
	page.Recipe, err = r.adminRecipeComponents(ctx, productID, variantID)
	if err != nil {
		return AdminVariantFormPage{}, err
	}

	return page, nil
}

func (r *PostgresRepository) CreateAdminVariant(ctx context.Context, input AdminVariantSaveInput) (string, error) {
	if r == nil || r.pool == nil {
		return "", ErrUnavailable
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", ErrUnavailable
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if input.IsDefault {
		if _, err := tx.Exec(ctx, `
			update public.product_variants
			set is_default = false, updated_at = now()
			where product_id = $1::uuid
				and is_default = true
		`, input.ProductID); err != nil {
			return "", ErrUnavailable
		}
	}

	var id string
	err = tx.QueryRow(ctx, `
		insert into public.product_variants (
			product_id,
			name,
			slug,
			sku,
			price_cents,
			shipping_weight_g,
			shipping_height_mm,
			shipping_width_mm,
			shipping_length_mm,
			is_active,
			is_default,
			sort_order,
			print_time_minutes
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
			$12,
			$13
		)
		returning id::text
	`, input.ProductID, input.Name, input.Slug, nullableText(input.SKU), nullableInt64Ptr(input.PriceCents), shippingWeightValue(input.ShippingProfile), shippingHeightValue(input.ShippingProfile), shippingWidthValue(input.ShippingProfile), shippingLengthValue(input.ShippingProfile), input.IsActive, input.IsDefault, input.SortOrder, nullableIntPtr(input.PrintTimeMinutes)).Scan(&id)
	if err != nil {
		return "", mapCatalogError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return "", ErrUnavailable
	}

	return id, nil
}

func (r *PostgresRepository) UpdateAdminVariant(ctx context.Context, input AdminVariantSaveInput) error {
	if r == nil || r.pool == nil {
		return ErrUnavailable
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ErrUnavailable
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if input.IsDefault {
		if _, err := tx.Exec(ctx, `
			update public.product_variants
			set is_default = false, updated_at = now()
			where product_id = $1::uuid
				and id <> $2::uuid
				and is_default = true
		`, input.ProductID, input.ID); err != nil {
			return ErrUnavailable
		}
	}

	tag, err := tx.Exec(ctx, `
		update public.product_variants
		set
			name = $3,
			slug = $4,
			sku = $5,
			price_cents = $6,
			shipping_weight_g = $7,
			shipping_height_mm = $8,
			shipping_width_mm = $9,
			shipping_length_mm = $10,
			is_active = $11,
			is_default = $12,
			sort_order = $13,
			print_time_minutes = $14,
			updated_at = now()
		where id = $1::uuid
			and product_id = $2::uuid
	`, input.ID, input.ProductID, input.Name, input.Slug, nullableText(input.SKU), nullableInt64Ptr(input.PriceCents), shippingWeightValue(input.ShippingProfile), shippingHeightValue(input.ShippingProfile), shippingWidthValue(input.ShippingProfile), shippingLengthValue(input.ShippingProfile), input.IsActive, input.IsDefault, input.SortOrder, nullableIntPtr(input.PrintTimeMinutes))
	if err != nil {
		return mapCatalogError(err)
	}
	if tag.RowsAffected() != 1 {
		return ErrCatalogNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return ErrUnavailable
	}

	return nil
}

func (r *PostgresRepository) AddAdminRecipeComponent(ctx context.Context, input AdminRecipeSaveInput) error {
	if err := r.ensureRecipeReferencesAllowed(ctx, input, true); err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `
		insert into public.variant_filaments (
			variant_id,
			material_id,
			color_id,
			estimated_weight_mg,
			label,
			sort_order
		)
		select
			v.id,
			$3::uuid,
			$4::uuid,
			$5,
			$6,
			$7
		from public.product_variants v
		where v.id = $2::uuid
			and v.product_id = $1::uuid
	`, input.ProductID, input.VariantID, input.MaterialID, input.ColorID, input.EstimatedWeightMg, nullableText(input.Label), input.SortOrder)
	if err != nil {
		return mapCatalogError(err)
	}
	if tag.RowsAffected() != 1 {
		return ErrCatalogNotFound
	}

	return nil
}

func (r *PostgresRepository) UpdateAdminRecipeComponent(ctx context.Context, input AdminRecipeSaveInput) error {
	if err := r.ensureRecipeReferencesAllowed(ctx, input, false); err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `
		update public.variant_filaments component
		set
			material_id = $4::uuid,
			color_id = $5::uuid,
			estimated_weight_mg = $6,
			label = $7,
			sort_order = $8
		from public.product_variants v
		where component.id = $3::uuid
			and component.variant_id = v.id
			and v.id = $2::uuid
			and v.product_id = $1::uuid
	`, input.ProductID, input.VariantID, input.ID, input.MaterialID, input.ColorID, input.EstimatedWeightMg, nullableText(input.Label), input.SortOrder)
	if err != nil {
		return mapCatalogError(err)
	}
	if tag.RowsAffected() != 1 {
		return ErrCatalogNotFound
	}

	return nil
}

func (r *PostgresRepository) RemoveAdminRecipeComponent(ctx context.Context, productID string, variantID string, componentID string) error {
	tag, err := r.pool.Exec(ctx, `
		delete from public.variant_filaments component
		using public.product_variants v
		where component.id = $3::uuid
			and component.variant_id = v.id
			and v.id = $2::uuid
			and v.product_id = $1::uuid
	`, productID, variantID, componentID)
	if err != nil {
		return ErrUnavailable
	}
	if tag.RowsAffected() != 1 {
		return ErrCatalogNotFound
	}

	return nil
}

func (r *PostgresRepository) ListAdminMaterials(ctx context.Context) (AdminMaterialListPage, error) {
	rows, err := r.pool.Query(ctx, `
		select id::text, name, slug, coalesce(description, ''), is_active
		from public.materials
		order by is_active desc, name asc, id asc
	`)
	if err != nil {
		return AdminMaterialListPage{}, ErrUnavailable
	}
	defer rows.Close()

	var items []AdminMaterialListItem
	for rows.Next() {
		var item AdminMaterialListItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Slug, &item.Description, &item.IsActive); err != nil {
			return AdminMaterialListPage{}, ErrUnavailable
		}
		item.StatusLabel = ActiveStatusLabel(item.IsActive)
		item.DetailURL = "/admin/materiais/" + item.ID
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return AdminMaterialListPage{}, ErrUnavailable
	}

	return AdminMaterialListPage{Materials: items}, nil
}

func (r *PostgresRepository) GetAdminMaterialForm(ctx context.Context, materialID string) (AdminMaterialFormPage, error) {
	page := AdminMaterialFormPage{
		Title:       "Editar material",
		Action:      "/admin/materiais/" + materialID,
		SubmitLabel: "Salvar material",
		BackURL:     "/admin/materiais",
		Errors:      AdminFieldErrors{},
	}
	var description pgtype.Text
	err := r.pool.QueryRow(ctx, `
		select name, slug, description, is_active
		from public.materials
		where id = $1::uuid
	`, materialID).Scan(&page.Form.Name, &page.Form.Slug, &description, &page.Form.IsActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AdminMaterialFormPage{}, ErrCatalogNotFound
		}
		return AdminMaterialFormPage{}, ErrUnavailable
	}
	if description.Valid {
		page.Form.Description = description.String
	}

	return page, nil
}

func (r *PostgresRepository) CreateAdminMaterial(ctx context.Context, input AdminMaterialSaveInput) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		insert into public.materials (name, slug, description, is_active)
		values ($1, $2, $3, $4)
		returning id::text
	`, input.Name, input.Slug, nullableText(input.Description), input.IsActive).Scan(&id)
	if err != nil {
		return "", mapCatalogError(err)
	}

	return id, nil
}

func (r *PostgresRepository) UpdateAdminMaterial(ctx context.Context, input AdminMaterialSaveInput) error {
	tag, err := r.pool.Exec(ctx, `
		update public.materials
		set name = $2, slug = $3, description = $4, is_active = $5, updated_at = now()
		where id = $1::uuid
	`, input.ID, input.Name, input.Slug, nullableText(input.Description), input.IsActive)
	if err != nil {
		return mapCatalogError(err)
	}
	if tag.RowsAffected() != 1 {
		return ErrCatalogNotFound
	}

	return nil
}

func (r *PostgresRepository) ListAdminColors(ctx context.Context) (AdminColorListPage, error) {
	rows, err := r.pool.Query(ctx, `
		select id::text, name, slug, coalesce(hex_color, ''), is_active
		from public.colors
		order by is_active desc, name asc, id asc
	`)
	if err != nil {
		return AdminColorListPage{}, ErrUnavailable
	}
	defer rows.Close()

	var items []AdminColorListItem
	for rows.Next() {
		var item AdminColorListItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Slug, &item.HexColor, &item.IsActive); err != nil {
			return AdminColorListPage{}, ErrUnavailable
		}
		item.StatusLabel = ActiveStatusLabel(item.IsActive)
		item.DetailURL = "/admin/cores/" + item.ID
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return AdminColorListPage{}, ErrUnavailable
	}

	return AdminColorListPage{Colors: items}, nil
}

func (r *PostgresRepository) GetAdminColorForm(ctx context.Context, colorID string) (AdminColorFormPage, error) {
	page := AdminColorFormPage{
		Title:       "Editar cor",
		Action:      "/admin/cores/" + colorID,
		SubmitLabel: "Salvar cor",
		BackURL:     "/admin/cores",
		Errors:      AdminFieldErrors{},
	}
	var hexColor pgtype.Text
	err := r.pool.QueryRow(ctx, `
		select name, slug, hex_color, is_active
		from public.colors
		where id = $1::uuid
	`, colorID).Scan(&page.Form.Name, &page.Form.Slug, &hexColor, &page.Form.IsActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AdminColorFormPage{}, ErrCatalogNotFound
		}
		return AdminColorFormPage{}, ErrUnavailable
	}
	if hexColor.Valid {
		page.Form.HexColor = hexColor.String
	}

	return page, nil
}

func (r *PostgresRepository) CreateAdminColor(ctx context.Context, input AdminColorSaveInput) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		insert into public.colors (name, slug, hex_color, is_active)
		values ($1, $2, $3, $4)
		returning id::text
	`, input.Name, input.Slug, nullableText(input.HexColor), input.IsActive).Scan(&id)
	if err != nil {
		return "", mapCatalogError(err)
	}

	return id, nil
}

func (r *PostgresRepository) UpdateAdminColor(ctx context.Context, input AdminColorSaveInput) error {
	tag, err := r.pool.Exec(ctx, `
		update public.colors
		set name = $2, slug = $3, hex_color = $4, is_active = $5, updated_at = now()
		where id = $1::uuid
	`, input.ID, input.Name, input.Slug, nullableText(input.HexColor), input.IsActive)
	if err != nil {
		return mapCatalogError(err)
	}
	if tag.RowsAffected() != 1 {
		return ErrCatalogNotFound
	}

	return nil
}

func (r *PostgresRepository) ListAdminBoxes(ctx context.Context) (AdminBoxListPage, error) {
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
			is_active,
			sort_order
		from public.shipping_boxes
		order by is_active desc, sort_order asc, name asc, id asc
	`)
	if err != nil {
		return AdminBoxListPage{}, ErrUnavailable
	}
	defer rows.Close()

	var items []AdminBoxListItem
	for rows.Next() {
		var item AdminBoxListItem
		var internalHeight int
		var internalWidth int
		var internalLength int
		var externalHeight int
		var externalWidth int
		var externalLength int
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Slug,
			&internalHeight,
			&internalWidth,
			&internalLength,
			&externalHeight,
			&externalWidth,
			&externalLength,
			&item.PackagingWeightG,
			&item.IsActive,
			&item.SortOrder,
		); err != nil {
			return AdminBoxListPage{}, ErrUnavailable
		}
		item.InternalLabel = FormatAdminDimensionsLabel(internalHeight, internalWidth, internalLength)
		item.ExternalLabel = FormatAdminDimensionsLabel(externalHeight, externalWidth, externalLength)
		item.WeightLabel = strconv.Itoa(item.PackagingWeightG) + " g"
		item.StatusLabel = ActiveStatusLabel(item.IsActive)
		item.DetailURL = "/admin/caixas/" + item.ID
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return AdminBoxListPage{}, ErrUnavailable
	}

	return AdminBoxListPage{Boxes: items}, nil
}

func (r *PostgresRepository) GetAdminBoxForm(ctx context.Context, boxID string) (AdminBoxFormPage, error) {
	page := AdminBoxFormPage{
		Title:       "Editar caixa",
		Action:      "/admin/caixas/" + boxID,
		SubmitLabel: "Salvar caixa",
		BackURL:     "/admin/caixas",
		Errors:      AdminFieldErrors{},
	}
	var internalHeight int
	var internalWidth int
	var internalLength int
	var externalHeight int
	var externalWidth int
	var externalLength int
	var packagingWeight int
	var sortOrder int
	err := r.pool.QueryRow(ctx, `
		select
			name,
			slug,
			internal_height_mm,
			internal_width_mm,
			internal_length_mm,
			external_height_mm,
			external_width_mm,
			external_length_mm,
			packaging_weight_g,
			is_active,
			sort_order
		from public.shipping_boxes
		where id = $1::uuid
	`, boxID).Scan(
		&page.Form.Name,
		&page.Form.Slug,
		&internalHeight,
		&internalWidth,
		&internalLength,
		&externalHeight,
		&externalWidth,
		&externalLength,
		&packagingWeight,
		&page.Form.IsActive,
		&sortOrder,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AdminBoxFormPage{}, ErrCatalogNotFound
		}
		return AdminBoxFormPage{}, ErrUnavailable
	}
	page.Form.InternalHeightMM = strconv.Itoa(internalHeight)
	page.Form.InternalWidthMM = strconv.Itoa(internalWidth)
	page.Form.InternalLengthMM = strconv.Itoa(internalLength)
	page.Form.ExternalHeightMM = strconv.Itoa(externalHeight)
	page.Form.ExternalWidthMM = strconv.Itoa(externalWidth)
	page.Form.ExternalLengthMM = strconv.Itoa(externalLength)
	page.Form.PackagingWeightG = strconv.Itoa(packagingWeight)
	page.Form.SortOrder = strconv.Itoa(sortOrder)

	return page, nil
}

func (r *PostgresRepository) CreateAdminBox(ctx context.Context, input AdminBoxSaveInput) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		insert into public.shipping_boxes (
			name,
			slug,
			internal_height_mm,
			internal_width_mm,
			internal_length_mm,
			external_height_mm,
			external_width_mm,
			external_length_mm,
			packaging_weight_g,
			is_active,
			sort_order
		) values (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10,
			$11
		)
		returning id::text
	`, input.Name, input.Slug, input.InternalHeightMM, input.InternalWidthMM, input.InternalLengthMM, input.ExternalHeightMM, input.ExternalWidthMM, input.ExternalLengthMM, input.PackagingWeightG, input.IsActive, input.SortOrder).Scan(&id)
	if err != nil {
		return "", mapCatalogError(err)
	}

	return id, nil
}

func (r *PostgresRepository) UpdateAdminBox(ctx context.Context, input AdminBoxSaveInput) error {
	tag, err := r.pool.Exec(ctx, `
		update public.shipping_boxes
		set
			name = $2,
			slug = $3,
			internal_height_mm = $4,
			internal_width_mm = $5,
			internal_length_mm = $6,
			external_height_mm = $7,
			external_width_mm = $8,
			external_length_mm = $9,
			packaging_weight_g = $10,
			is_active = $11,
			sort_order = $12,
			updated_at = now()
		where id = $1::uuid
	`, input.ID, input.Name, input.Slug, input.InternalHeightMM, input.InternalWidthMM, input.InternalLengthMM, input.ExternalHeightMM, input.ExternalWidthMM, input.ExternalLengthMM, input.PackagingWeightG, input.IsActive, input.SortOrder)
	if err != nil {
		return mapCatalogError(err)
	}
	if tag.RowsAffected() != 1 {
		return ErrCatalogNotFound
	}

	return nil
}

func (r *PostgresRepository) listCategoryOptions(ctx context.Context, currentID string) ([]AdminSelectOption, error) {
	rows, err := r.pool.Query(ctx, `
		select id::text, name, is_active
		from public.categories
		where is_active = true
			or ($1::text <> '' and id = $1::uuid)
		order by is_active desc, name asc, id asc
	`, nullableUUID(currentID))
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()

	var options []AdminSelectOption
	for rows.Next() {
		var option AdminSelectOption
		if err := rows.Scan(&option.ID, &option.Label, &option.Active); err != nil {
			return nil, ErrUnavailable
		}
		if !option.Active {
			option.Label += " — inativa"
		}
		option.Selected = currentID != "" && option.ID == currentID
		options = append(options, option)
	}
	if err := rows.Err(); err != nil {
		return nil, ErrUnavailable
	}

	return options, nil
}

func (r *PostgresRepository) listAdminVariantItems(ctx context.Context, productID string) ([]AdminVariantListItem, error) {
	rows, err := r.pool.Query(ctx, `
		select
			v.id::text,
			v.name,
			v.slug,
			coalesce(v.sku, ''),
			v.price_cents,
			v.is_active,
			v.is_default,
			v.sort_order,
			v.print_time_minutes,
			v.shipping_weight_g is not null,
			count(f.id)::integer,
			coalesce(sum(f.estimated_weight_mg), 0)::bigint
		from public.product_variants v
		left join public.variant_filaments f
			on f.variant_id = v.id
		where v.product_id = $1::uuid
		group by
			v.id,
			v.name,
			v.slug,
			v.sku,
			v.price_cents,
			v.is_active,
			v.is_default,
			v.sort_order,
			v.print_time_minutes,
			v.shipping_weight_g
		order by v.is_active desc, v.is_default desc, v.sort_order asc, v.name asc, v.id asc
	`, productID)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()

	var items []AdminVariantListItem
	for rows.Next() {
		var item AdminVariantListItem
		var price pgtype.Int8
		var printTime pgtype.Int4
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Slug,
			&item.SKU,
			&price,
			&item.IsActive,
			&item.IsDefault,
			&item.SortOrder,
			&printTime,
			&item.HasShippingProfile,
			&item.RecipeCount,
			&item.RecipeWeightMg,
		); err != nil {
			return nil, ErrUnavailable
		}
		if price.Valid {
			item.PriceLabel = products.FormatBRL(price.Int64)
		} else {
			item.PriceLabel = "Herda preco-base"
		}
		item.StatusLabel = ActiveStatusLabel(item.IsActive)
		if item.IsDefault {
			item.DefaultLabel = "Default"
		} else {
			item.DefaultLabel = "Nao default"
		}
		if printTime.Valid {
			item.PrintTimeLabel = products.FormatPrintTime(int(printTime.Int32))
		}
		if item.HasShippingProfile {
			item.ShippingLabel = "Usa perfil proprio"
		} else {
			item.ShippingLabel = "Herda perfil logistico do produto"
		}
		item.RecipeWeightLabel = products.FormatWeightGrams(item.RecipeWeightMg)
		item.DetailURL = "/admin/produtos/" + productID + "/variantes/" + item.ID
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, ErrUnavailable
	}

	return items, nil
}

func (r *PostgresRepository) adminProductHeader(ctx context.Context, productID string) (AdminProductHeader, error) {
	var product AdminProductHeader
	err := r.pool.QueryRow(ctx, `
		select id::text, name, slug
		from public.products
		where id = $1::uuid
	`, productID).Scan(&product.ID, &product.Name, &product.Slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AdminProductHeader{}, ErrCatalogNotFound
		}
		return AdminProductHeader{}, ErrUnavailable
	}
	product.DetailURL = "/admin/produtos/" + product.ID

	return product, nil
}

func (r *PostgresRepository) recipeOptions(ctx context.Context) ([]AdminSelectOption, []AdminSelectOption, error) {
	materials, err := r.activeOptions(ctx, "materials")
	if err != nil {
		return nil, nil, err
	}
	colors, err := r.activeOptions(ctx, "colors")
	if err != nil {
		return nil, nil, err
	}

	return materials, colors, nil
}

func (r *PostgresRepository) activeOptions(ctx context.Context, table string) ([]AdminSelectOption, error) {
	query := `
		select id::text, name, is_active
		from public.materials
		where is_active = true
		order by name asc, id asc
	`
	if table == "colors" {
		query = `
			select id::text, name, is_active
			from public.colors
			where is_active = true
			order by name asc, id asc
		`
	}
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()

	var options []AdminSelectOption
	for rows.Next() {
		var option AdminSelectOption
		if err := rows.Scan(&option.ID, &option.Label, &option.Active); err != nil {
			return nil, ErrUnavailable
		}
		options = append(options, option)
	}
	if err := rows.Err(); err != nil {
		return nil, ErrUnavailable
	}

	return options, nil
}

func (r *PostgresRepository) adminRecipeComponents(ctx context.Context, productID string, variantID string) ([]AdminRecipeComponent, error) {
	rows, err := r.pool.Query(ctx, `
		select
			component.id::text,
			component.material_id::text,
			m.name,
			m.is_active,
			component.color_id::text,
			c.name,
			coalesce(c.hex_color, ''),
			c.is_active,
			component.estimated_weight_mg,
			coalesce(component.label, ''),
			component.sort_order
		from public.variant_filaments component
		join public.product_variants v
			on v.id = component.variant_id
		join public.materials m
			on m.id = component.material_id
		join public.colors c
			on c.id = component.color_id
		where v.product_id = $1::uuid
			and v.id = $2::uuid
		order by component.sort_order asc, component.created_at asc, component.id asc
	`, productID, variantID)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()

	var components []AdminRecipeComponent
	for rows.Next() {
		var component AdminRecipeComponent
		if err := rows.Scan(
			&component.ID,
			&component.MaterialID,
			&component.MaterialName,
			&component.MaterialActive,
			&component.ColorID,
			&component.ColorName,
			&component.ColorHex,
			&component.ColorActive,
			&component.EstimatedWeightMg,
			&component.Label,
			&component.SortOrder,
		); err != nil {
			return nil, ErrUnavailable
		}
		component.MaterialLabel = component.MaterialName
		if !component.MaterialActive {
			component.MaterialLabel += " — inativo"
		}
		component.ColorLabel = component.ColorName
		if !component.ColorActive {
			component.ColorLabel += " — inativa"
		}
		component.EstimatedWeightInput = FormatAdminWeightInput(component.EstimatedWeightMg)
		component.EstimatedWeightLabel = products.FormatWeightGrams(component.EstimatedWeightMg)
		component.SortOrderInput = strconv.Itoa(component.SortOrder)
		component.UpdateAction = "/admin/produtos/" + productID + "/variantes/" + variantID + "/receita/" + component.ID
		component.RemoveAction = component.UpdateAction + "/remover"
		components = append(components, component)
	}
	if err := rows.Err(); err != nil {
		return nil, ErrUnavailable
	}

	return components, nil
}

func (r *PostgresRepository) ensureProductCategoryAllowed(ctx context.Context, categoryID string, productID string) error {
	if categoryID == "" {
		return nil
	}
	var allowed bool
	err := r.pool.QueryRow(ctx, `
		select exists (
			select 1
			from public.categories c
			where c.id = $1::uuid
				and (
					c.is_active = true
					or (
						$2::text <> ''
						and exists (
							select 1
							from public.products p
							where p.id = $2::uuid
								and p.category_id = c.id
						)
					)
				)
		)
	`, categoryID, nullableUUID(productID)).Scan(&allowed)
	if err != nil {
		return ErrUnavailable
	}
	if !allowed {
		return ErrValidation
	}

	return nil
}

func (r *PostgresRepository) ensureRecipeReferencesAllowed(ctx context.Context, input AdminRecipeSaveInput, create bool) error {
	var materialAllowed bool
	var colorAllowed bool
	if create {
		err := r.pool.QueryRow(ctx, `
			select
				exists (select 1 from public.materials where id = $1::uuid and is_active = true),
				exists (select 1 from public.colors where id = $2::uuid and is_active = true)
		`, input.MaterialID, input.ColorID).Scan(&materialAllowed, &colorAllowed)
		if err != nil {
			return ErrUnavailable
		}
		if !materialAllowed || !colorAllowed {
			return ErrValidation
		}
		return nil
	}

	err := r.pool.QueryRow(ctx, `
		select
			exists (
				select 1
				from public.materials m
				where m.id = $1::uuid
					and (
						m.is_active = true
						or exists (
							select 1
							from public.variant_filaments component
							join public.product_variants v
								on v.id = component.variant_id
							where component.id = $5::uuid
								and component.material_id = m.id
								and v.id = $4::uuid
								and v.product_id = $3::uuid
						)
					)
			),
			exists (
				select 1
				from public.colors c
				where c.id = $2::uuid
					and (
						c.is_active = true
						or exists (
							select 1
							from public.variant_filaments component
							join public.product_variants v
								on v.id = component.variant_id
							where component.id = $5::uuid
								and component.color_id = c.id
								and v.id = $4::uuid
								and v.product_id = $3::uuid
						)
					)
			)
	`, input.MaterialID, input.ColorID, input.ProductID, input.VariantID, input.ID).Scan(&materialAllowed, &colorAllowed)
	if err != nil {
		return ErrUnavailable
	}
	if !materialAllowed || !colorAllowed {
		return ErrValidation
	}

	return nil
}

func fillShippingForm(use *bool, weight *string, height *string, width *string, length *string, weightValue pgtype.Int8, heightValue pgtype.Int4, widthValue pgtype.Int4, lengthValue pgtype.Int4) {
	if !weightValue.Valid || !heightValue.Valid || !widthValue.Valid || !lengthValue.Valid {
		return
	}
	*use = true
	*weight = strconv.FormatInt(weightValue.Int64, 10)
	*height = strconv.Itoa(int(heightValue.Int32))
	*width = strconv.Itoa(int(widthValue.Int32))
	*length = strconv.Itoa(int(lengthValue.Int32))
}

func nullableText(value string) any {
	if value == "" {
		return nil
	}

	return value
}

func nullableUUID(value string) any {
	if value == "" {
		return nil
	}

	return value
}

func nullableIntPtr(value *int) any {
	if value == nil {
		return nil
	}

	return *value
}

func nullableInt64Ptr(value *int64) any {
	if value == nil {
		return nil
	}

	return *value
}

func shippingWeightValue(profile *AdminShippingProfile) any {
	if profile == nil {
		return nil
	}

	return profile.WeightG
}

func shippingHeightValue(profile *AdminShippingProfile) any {
	if profile == nil {
		return nil
	}

	return profile.HeightMM
}

func shippingWidthValue(profile *AdminShippingProfile) any {
	if profile == nil {
		return nil
	}

	return profile.WidthMM
}

func shippingLengthValue(profile *AdminShippingProfile) any {
	if profile == nil {
		return nil
	}

	return profile.LengthMM
}

func mapCatalogError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.ConstraintName {
		case "products_slug_unique",
			"categories_slug_unique",
			"materials_slug_unique",
			"colors_slug_unique",
			"shipping_boxes_slug_key",
			"product_variants_product_slug_unique":
			return ErrDuplicateSlug
		case "product_variants_sku_unique":
			return ErrDuplicateSKU
		default:
			return ErrValidation
		}
	}

	return ErrUnavailable
}
