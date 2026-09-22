package customerprofile

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct{ pool *pgxpool.Pool }

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}
func (r *PostgresRepository) Get(ctx context.Context, id string) (Profile, bool, error) {
	if r == nil || r.pool == nil {
		return Profile{}, false, errors.New("customer profile unavailable")
	}
	var p Profile
	err := r.pool.QueryRow(ctx, `select auth_user_id::text,full_name,phone,cpf,postal_code,street,number,coalesce(complement,''),district,city,state,country_code,created_at,updated_at from public.customer_profiles where auth_user_id=$1::uuid`, id).Scan(&p.AuthUserID, &p.FullName, &p.Phone, &p.CPF, &p.PostalCode, &p.Street, &p.Number, &p.Complement, &p.District, &p.City, &p.State, &p.CountryCode, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, false, nil
	}
	if err != nil {
		return Profile{}, false, err
	}
	return p, true, nil
}
func (r *PostgresRepository) Upsert(ctx context.Context, p Profile) error {
	if r == nil || r.pool == nil {
		return errors.New("customer profile unavailable")
	}
	_, err := r.pool.Exec(ctx, `insert into public.customer_profiles(auth_user_id,full_name,phone,cpf,postal_code,street,number,complement,district,city,state,country_code) values($1::uuid,$2,$3,$4,$5,$6,$7,nullif($8,''),$9,$10,$11,$12) on conflict(auth_user_id) do update set full_name=excluded.full_name,phone=excluded.phone,cpf=excluded.cpf,postal_code=excluded.postal_code,street=excluded.street,number=excluded.number,complement=excluded.complement,district=excluded.district,city=excluded.city,state=excluded.state,country_code=excluded.country_code,updated_at=now()`, p.AuthUserID, p.FullName, p.Phone, p.CPF, p.PostalCode, p.Street, p.Number, p.Complement, p.District, p.City, p.State, p.CountryCode)
	return err
}
