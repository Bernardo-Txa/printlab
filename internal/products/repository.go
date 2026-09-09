package products

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
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
			p.is_featured,
			coalesce(c.id::text, ''),
			coalesce(c.name, ''),
			coalesce(c.slug, ''),
			coalesce(c.description, '')
		from public.products p
		left join public.categories c
			on c.id = p.category_id
			and c.is_active = true
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
		product, err := scanProduct(rows)
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

func (r *PostgresRepository) GetActiveProductBySlug(ctx context.Context, slug string) (Product, error) {
	if r == nil || r.pool == nil {
		return Product{}, ErrUnavailable
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
			return Product{}, ErrNotFound
		}

		return Product{}, err
	}

	return product, nil
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
