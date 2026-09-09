package customers

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

func (r *PostgresRepository) Get(ctx context.Context, cartID string) (CheckoutDetails, bool, error) {
	if r == nil || r.pool == nil {
		return CheckoutDetails{}, false, ErrUnavailable
	}

	var details CheckoutDetails
	var complement pgtype.Text
	err := r.pool.QueryRow(ctx, `
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
		from public.cart_customer_details as customer
		join public.cart_shipping_addresses as address
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
			return CheckoutDetails{}, false, nil
		}

		return CheckoutDetails{}, false, err
	}
	if complement.Valid {
		value := complement.String
		details.Address.Complement = &value
	}

	return details, true, nil
}

func (r *PostgresRepository) Save(ctx context.Context, cartID string, details CheckoutDetails) error {
	if r == nil || r.pool == nil {
		return ErrUnavailable
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, `
		insert into public.cart_customer_details (
			cart_id,
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
		on conflict (cart_id) do update set
			full_name = excluded.full_name,
			email = excluded.email,
			phone = excluded.phone,
			cpf = excluded.cpf,
			updated_at = now()
	`, cartID, details.Customer.FullName, details.Customer.Email, details.Customer.Phone, details.Customer.CPF); err != nil {
		return err
	}

	complement := any(nil)
	if details.Address.Complement != nil {
		complement = *details.Address.Complement
	}

	if _, err := tx.Exec(ctx, `
		insert into public.cart_shipping_addresses (
			cart_id,
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
		on conflict (cart_id) do update set
			postal_code = excluded.postal_code,
			street = excluded.street,
			number = excluded.number,
			complement = excluded.complement,
			district = excluded.district,
			city = excluded.city,
			state = excluded.state,
			country_code = excluded.country_code,
			updated_at = now()
	`, cartID, details.Address.PostalCode, details.Address.Street, details.Address.Number, complement, details.Address.District, details.Address.City, details.Address.State, details.Address.CountryCode); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
