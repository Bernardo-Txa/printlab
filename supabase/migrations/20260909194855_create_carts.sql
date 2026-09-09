create table public.carts (
  id uuid primary key default gen_random_uuid(),
  token_hash bytea not null unique,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  expires_at timestamptz not null,

  constraint carts_token_hash_length check (octet_length(token_hash) = 32),
  constraint carts_expires_after_created check (expires_at > created_at)
);

create table public.cart_items (
  id uuid primary key default gen_random_uuid(),
  cart_id uuid not null,
  product_id uuid not null,
  variant_id uuid null,
  quantity integer not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),

  constraint cart_items_cart_id_fkey foreign key (cart_id)
    references public.carts (id)
    on delete cascade,
  constraint cart_items_product_id_fkey foreign key (product_id)
    references public.products (id)
    on delete cascade,
  constraint cart_items_variant_product_fkey foreign key (variant_id, product_id)
    references public.product_variants (id, product_id),
  constraint cart_items_quantity_range check (quantity between 1 and 99)
);

create index cart_items_cart_id_idx on public.cart_items (cart_id);
create index cart_items_product_id_idx on public.cart_items (product_id);
create index cart_items_variant_id_idx on public.cart_items (variant_id) where variant_id is not null;

create unique index cart_items_cart_product_no_variant_unique_idx
  on public.cart_items (cart_id, product_id)
  where variant_id is null;

create unique index cart_items_cart_product_variant_unique_idx
  on public.cart_items (cart_id, product_id, variant_id)
  where variant_id is not null;

alter table public.carts enable row level security;
alter table public.cart_items enable row level security;
