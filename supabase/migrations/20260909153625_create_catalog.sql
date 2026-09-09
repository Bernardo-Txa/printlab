create table public.categories (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  slug text not null,
  description text null,
  is_active boolean not null default true,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),

  constraint categories_name_not_blank check (btrim(name) <> ''),
  constraint categories_slug_unique unique (slug),
  constraint categories_slug_format check (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
  constraint categories_description_not_blank check (description is null or btrim(description) <> '')
);

create table public.products (
  id uuid primary key default gen_random_uuid(),
  category_id uuid null,
  name text not null,
  slug text not null,
  short_description text null,
  description text null,
  price_cents bigint not null,
  is_active boolean not null default false,
  is_featured boolean not null default false,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),

  constraint products_category_id_fkey foreign key (category_id)
    references public.categories (id)
    on delete set null,
  constraint products_name_not_blank check (btrim(name) <> ''),
  constraint products_slug_unique unique (slug),
  constraint products_slug_format check (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
  constraint products_price_cents_non_negative check (price_cents >= 0),
  constraint products_short_description_not_blank check (short_description is null or btrim(short_description) <> ''),
  constraint products_description_not_blank check (description is null or btrim(description) <> '')
);

create index products_category_id_idx on public.products (category_id);

alter table public.categories enable row level security;
alter table public.products enable row level security;
