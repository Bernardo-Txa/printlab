alter table public.orders
  drop constraint orders_status_allowed;

alter table public.orders
  add constraint orders_status_allowed check (status in ('pending_payment', 'paid'));

create table public.order_payments (
  order_id uuid primary key,
  provider text not null default 'infinitepay',
  status text not null default 'pending',
  order_nsu text not null unique,
  checkout_url text null,
  invoice_slug text null,
  transaction_nsu text null,
  amount_cents bigint null,
  paid_amount_cents bigint null,
  installments integer null,
  capture_method text null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  paid_at timestamptz null,

  constraint order_payments_order_id_fkey foreign key (order_id)
    references public.orders (id)
    on delete cascade,
  constraint order_payments_provider_infinitepay check (provider = 'infinitepay'),
  constraint order_payments_status_allowed check (status in ('pending', 'paid')),
  constraint order_payments_order_nsu_uuid check (
    order_nsu = lower(btrim(order_nsu))
    and order_nsu ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
  ),
  constraint order_payments_checkout_url_length check (
    checkout_url is null
    or char_length(checkout_url) between 1 and 2048
  ),
  constraint order_payments_checkout_url_host check (
    checkout_url is null
    or checkout_url like 'https://checkout.infinitepay.com.br/%'
  ),
  constraint order_payments_invoice_slug_format check (
    invoice_slug is null
    or (
      char_length(invoice_slug) between 1 and 200
      and invoice_slug ~ '^[A-Za-z0-9_-]+$'
    )
  ),
  constraint order_payments_transaction_nsu_format check (
    transaction_nsu is null
    or (
      char_length(transaction_nsu) between 1 and 200
      and transaction_nsu ~ '^[A-Za-z0-9_-]+$'
    )
  ),
  constraint order_payments_amount_non_negative check (
    amount_cents is null
    or amount_cents >= 0
  ),
  constraint order_payments_paid_amount_non_negative check (
    paid_amount_cents is null
    or paid_amount_cents >= 0
  ),
  constraint order_payments_installments_positive check (
    installments is null
    or installments > 0
  ),
  constraint order_payments_capture_method_format check (
    capture_method is null
    or (
      char_length(capture_method) between 1 and 60
      and capture_method = btrim(capture_method)
    )
  ),
  constraint order_payments_paid_fields_present check (
    status <> 'paid'
    or (
      transaction_nsu is not null
      and invoice_slug is not null
      and amount_cents is not null
      and paid_amount_cents is not null
      and paid_at is not null
    )
  )
);

create unique index order_payments_transaction_nsu_unique_idx
  on public.order_payments (transaction_nsu)
  where transaction_nsu is not null;

create index order_payments_status_idx on public.order_payments (status);

alter table public.order_payments enable row level security;
