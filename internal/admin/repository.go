package admin

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateSession(ctx context.Context, authUserID string, tokenHash []byte, expiresAt time.Time) (Session, error) {
	if r == nil || r.pool == nil {
		return Session{}, ErrUnavailable
	}

	var session Session
	err := r.pool.QueryRow(ctx, `
		insert into public.admin_sessions (
			auth_user_id,
			token_hash,
			expires_at
		) values (
			$1::uuid,
			$2,
			$3
		)
		returning
			id::text,
			auth_user_id::text,
			token_hash,
			created_at,
			expires_at
	`, authUserID, tokenHash, expiresAt).Scan(
		&session.ID,
		&session.AuthUserID,
		&session.TokenHash,
		&session.CreatedAt,
		&session.ExpiresAt,
	)
	if err != nil {
		return Session{}, ErrUnavailable
	}

	return session, nil
}

func (r *PostgresRepository) ResolveSession(ctx context.Context, tokenHash []byte) (Session, error) {
	if r == nil || r.pool == nil {
		return Session{}, ErrUnavailable
	}

	var session Session
	err := r.pool.QueryRow(ctx, `
		select
			id::text,
			auth_user_id::text,
			token_hash,
			created_at,
			expires_at
		from public.admin_sessions
		where token_hash = $1
	`, tokenHash).Scan(
		&session.ID,
		&session.AuthUserID,
		&session.TokenHash,
		&session.CreatedAt,
		&session.ExpiresAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Session{}, ErrSessionNotFound
		}
		return Session{}, ErrUnavailable
	}

	return session, nil
}

func (r *PostgresRepository) DeleteSession(ctx context.Context, tokenHash []byte) error {
	if r == nil || r.pool == nil {
		return ErrUnavailable
	}

	if _, err := r.pool.Exec(ctx, `
		delete from public.admin_sessions
		where token_hash = $1
	`, tokenHash); err != nil {
		return ErrUnavailable
	}

	return nil
}

func (r *PostgresRepository) Dashboard(ctx context.Context) (Dashboard, error) {
	if r == nil || r.pool == nil {
		return Dashboard{}, ErrUnavailable
	}

	var dashboard Dashboard
	err := r.pool.QueryRow(ctx, `
		select
			count(*) filter (where o.status = 'pending_payment')::integer,
			count(*) filter (where o.status = 'paid' and fulfillment.production_status = 'waiting')::integer,
			count(*) filter (where o.status = 'paid' and fulfillment.production_status = 'in_production')::integer,
			count(*) filter (
				where o.status = 'paid'
					and fulfillment.production_status = 'completed'
					and fulfillment.shipping_status = 'waiting'
			)::integer
		from public.orders o
		join public.order_fulfillment fulfillment
			on fulfillment.order_id = o.id
	`).Scan(
		&dashboard.PendingPayment,
		&dashboard.PaidWaitingProduction,
		&dashboard.InProduction,
		&dashboard.WaitingShipment,
	)
	if err != nil {
		return Dashboard{}, ErrUnavailable
	}

	return dashboard, nil
}
