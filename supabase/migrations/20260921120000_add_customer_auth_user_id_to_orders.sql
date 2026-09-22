alter table public.orders add column customer_auth_user_id uuid null;
create index orders_customer_auth_user_created_at_idx on public.orders (customer_auth_user_id, created_at desc) where customer_auth_user_id is not null;
