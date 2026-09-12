alter table public.orders
  add column public_tracking_id uuid null default gen_random_uuid();

update public.orders
set public_tracking_id = gen_random_uuid()
where public_tracking_id is null;

alter table public.orders
  alter column public_tracking_id set not null;

alter table public.orders
  add constraint orders_public_tracking_id_unique unique (public_tracking_id);

create table public.order_fulfillment (
  order_id uuid primary key,
  production_status text not null default 'waiting',
  shipping_status text not null default 'waiting',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),

  constraint order_fulfillment_order_id_fkey foreign key (order_id)
    references public.orders (id)
    on delete cascade,
  constraint order_fulfillment_production_status_allowed check (
    production_status in ('waiting', 'in_production', 'completed')
  ),
  constraint order_fulfillment_shipping_status_allowed check (
    shipping_status in ('waiting', 'preparing', 'shipped', 'delivered')
  ),
  constraint order_fulfillment_shipping_requires_completed_production check (
    shipping_status = 'waiting'
    or production_status = 'completed'
  )
);

insert into public.order_fulfillment (order_id)
select id
from public.orders
on conflict (order_id) do nothing;

alter table public.order_fulfillment enable row level security;
