create table public.admin_order_events (
  id uuid primary key default gen_random_uuid(),
  order_id uuid not null,
  actor_auth_user_id uuid not null,
  event_type text not null,
  from_status text not null,
  to_status text not null,
  created_at timestamptz not null default now(),

  constraint admin_order_events_order_id_fkey foreign key (order_id)
    references public.orders (id)
    on delete restrict,
  constraint admin_order_events_event_type_allowed check (
    event_type in ('production_status_changed', 'shipping_status_changed')
  ),
  constraint admin_order_events_from_status_not_blank check (btrim(from_status) <> ''),
  constraint admin_order_events_to_status_not_blank check (btrim(to_status) <> ''),
  constraint admin_order_events_status_changed check (from_status <> to_status)
);

create index admin_order_events_order_created_at_idx
  on public.admin_order_events (order_id, created_at desc);

alter table public.admin_order_events enable row level security;
