create table public.materials (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  slug text not null,
  description text null,
  is_active boolean not null default true,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),

  constraint materials_name_not_blank check (btrim(name) <> ''),
  constraint materials_slug_unique unique (slug),
  constraint materials_slug_format check (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
  constraint materials_description_not_blank check (description is null or btrim(description) <> '')
);

create table public.colors (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  slug text not null,
  hex_color text null,
  is_active boolean not null default true,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),

  constraint colors_name_not_blank check (btrim(name) <> ''),
  constraint colors_slug_unique unique (slug),
  constraint colors_slug_format check (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
  constraint colors_hex_color_format check (hex_color is null or hex_color ~ '^#[0-9A-F]{6}$')
);

create table public.product_variants (
  id uuid primary key default gen_random_uuid(),
  product_id uuid not null,
  name text not null,
  slug text not null,
  sku text null,
  price_cents bigint null,
  is_active boolean not null default false,
  is_default boolean not null default false,
  sort_order integer not null default 0,
  print_time_minutes integer null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),

  constraint product_variants_product_id_fkey foreign key (product_id)
    references public.products (id)
    on delete cascade,
  constraint product_variants_id_product_id_unique unique (id, product_id),
  constraint product_variants_product_slug_unique unique (product_id, slug),
  constraint product_variants_sku_unique unique (sku),
  constraint product_variants_name_not_blank check (btrim(name) <> ''),
  constraint product_variants_slug_format check (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
  constraint product_variants_sku_not_blank check (sku is null or btrim(sku) <> ''),
  constraint product_variants_price_cents_non_negative check (price_cents is null or price_cents >= 0),
  constraint product_variants_sort_order_non_negative check (sort_order >= 0),
  constraint product_variants_print_time_minutes_positive check (print_time_minutes is null or print_time_minutes > 0),
  constraint product_variants_default_is_active check (not is_default or is_active)
);

create table public.variant_filaments (
  id uuid primary key default gen_random_uuid(),
  variant_id uuid not null,
  material_id uuid not null,
  color_id uuid not null,
  estimated_weight_mg bigint not null,
  label text null,
  sort_order integer not null default 0,
  created_at timestamptz not null default now(),

  constraint variant_filaments_variant_id_fkey foreign key (variant_id)
    references public.product_variants (id)
    on delete cascade,
  constraint variant_filaments_material_id_fkey foreign key (material_id)
    references public.materials (id),
  constraint variant_filaments_color_id_fkey foreign key (color_id)
    references public.colors (id),
  constraint variant_filaments_estimated_weight_mg_positive check (estimated_weight_mg > 0),
  constraint variant_filaments_sort_order_non_negative check (sort_order >= 0),
  constraint variant_filaments_label_not_blank check (label is null or btrim(label) <> '')
);

create table public.product_images (
  id uuid primary key default gen_random_uuid(),
  product_id uuid not null,
  variant_id uuid null,
  storage_path text not null,
  alt_text text null,
  sort_order integer not null default 0,
  is_primary boolean not null default false,
  created_at timestamptz not null default now(),

  constraint product_images_product_id_fkey foreign key (product_id)
    references public.products (id)
    on delete cascade,
  constraint product_images_variant_product_fkey foreign key (variant_id, product_id)
    references public.product_variants (id, product_id)
    on delete cascade,
  constraint product_images_storage_path_not_blank check (btrim(storage_path) <> ''),
  constraint product_images_storage_path_relative check (
    storage_path = btrim(storage_path)
    and storage_path !~* '^[a-z][a-z0-9+.-]*:'
    and storage_path !~ '^//'
    and storage_path !~ '^/'
    and storage_path !~ '(^|/)\.\.(/|$)'
  ),
  constraint product_images_alt_text_not_blank check (alt_text is null or btrim(alt_text) <> ''),
  constraint product_images_sort_order_non_negative check (sort_order >= 0)
);

create index product_variants_product_id_idx on public.product_variants (product_id);
create unique index product_variants_one_default_per_product_idx
  on public.product_variants (product_id)
  where is_default = true;
create index product_variants_public_order_idx
  on public.product_variants (product_id, is_active, is_default desc, sort_order asc, name asc);

create index variant_filaments_variant_id_idx on public.variant_filaments (variant_id);
create index variant_filaments_material_id_idx on public.variant_filaments (material_id);
create index variant_filaments_color_id_idx on public.variant_filaments (color_id);

create index product_images_product_id_idx on public.product_images (product_id);
create index product_images_variant_id_idx on public.product_images (variant_id) where variant_id is not null;
create index product_images_product_general_order_idx
  on public.product_images (product_id, is_primary desc, sort_order asc, created_at asc)
  where variant_id is null;
create index product_images_variant_order_idx
  on public.product_images (variant_id, is_primary desc, sort_order asc, created_at asc)
  where variant_id is not null;
create unique index product_images_one_primary_per_product_idx
  on public.product_images (product_id)
  where variant_id is null and is_primary = true;
create unique index product_images_one_primary_per_variant_idx
  on public.product_images (variant_id)
  where variant_id is not null and is_primary = true;

alter table public.materials enable row level security;
alter table public.colors enable row level security;
alter table public.product_variants enable row level security;
alter table public.variant_filaments enable row level security;
alter table public.product_images enable row level security;

insert into storage.buckets (
  id,
  name,
  public,
  avif_autodetection,
  file_size_limit,
  allowed_mime_types
) values (
  'product-images',
  'product-images',
  true,
  true,
  5242880,
  array[
    'image/avif',
    'image/webp',
    'image/jpeg',
    'image/png'
  ]
) on conflict (id) do update set
  name = excluded.name,
  public = excluded.public,
  avif_autodetection = excluded.avif_autodetection,
  file_size_limit = excluded.file_size_limit,
  allowed_mime_types = excluded.allowed_mime_types,
  updated_at = now();
