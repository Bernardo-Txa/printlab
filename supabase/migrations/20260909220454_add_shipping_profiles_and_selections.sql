alter table public.products
  add column shipping_weight_g bigint null,
  add column shipping_height_mm integer null,
  add column shipping_width_mm integer null,
  add column shipping_length_mm integer null,
  add constraint products_shipping_profile_all_or_none check (
    (
      shipping_weight_g is null
      and shipping_height_mm is null
      and shipping_width_mm is null
      and shipping_length_mm is null
    )
    or (
      shipping_weight_g is not null
      and shipping_height_mm is not null
      and shipping_width_mm is not null
      and shipping_length_mm is not null
    )
  ),
  add constraint products_shipping_profile_positive check (
    shipping_weight_g is null
    or (
      shipping_weight_g > 0
      and shipping_height_mm > 0
      and shipping_width_mm > 0
      and shipping_length_mm > 0
    )
  );

alter table public.product_variants
  add column shipping_weight_g bigint null,
  add column shipping_height_mm integer null,
  add column shipping_width_mm integer null,
  add column shipping_length_mm integer null,
  add constraint product_variants_shipping_profile_all_or_none check (
    (
      shipping_weight_g is null
      and shipping_height_mm is null
      and shipping_width_mm is null
      and shipping_length_mm is null
    )
    or (
      shipping_weight_g is not null
      and shipping_height_mm is not null
      and shipping_width_mm is not null
      and shipping_length_mm is not null
    )
  ),
  add constraint product_variants_shipping_profile_positive check (
    shipping_weight_g is null
    or (
      shipping_weight_g > 0
      and shipping_height_mm > 0
      and shipping_width_mm > 0
      and shipping_length_mm > 0
    )
  );

create table public.shipping_boxes (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  slug text not null unique,
  internal_height_mm integer not null,
  internal_width_mm integer not null,
  internal_length_mm integer not null,
  external_height_mm integer not null,
  external_width_mm integer not null,
  external_length_mm integer not null,
  packaging_weight_g integer not null,
  is_active boolean not null default true,
  sort_order integer not null default 0,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),

  constraint shipping_boxes_name_not_blank check (btrim(name) <> ''),
  constraint shipping_boxes_slug_format check (slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
  constraint shipping_boxes_internal_dimensions_positive check (
    internal_height_mm > 0
    and internal_width_mm > 0
    and internal_length_mm > 0
  ),
  constraint shipping_boxes_external_dimensions_positive check (
    external_height_mm > 0
    and external_width_mm > 0
    and external_length_mm > 0
  ),
  constraint shipping_boxes_external_dimensions_fit_internal check (
    external_height_mm >= internal_height_mm
    and external_width_mm >= internal_width_mm
    and external_length_mm >= internal_length_mm
  ),
  constraint shipping_boxes_packaging_weight_positive check (packaging_weight_g > 0),
  constraint shipping_boxes_sort_order_non_negative check (sort_order >= 0)
);

create table public.cart_shipping_selections (
  cart_id uuid primary key,
  shipping_box_id uuid not null,
  provider text not null,
  service_code text not null,
  service_name text not null,
  carrier_name text null,
  price_cents bigint not null,
  delivery_time_days integer null,
  package_weight_g bigint not null,
  package_height_mm integer not null,
  package_width_mm integer not null,
  package_length_mm integer not null,
  input_hash bytea not null,
  quoted_at timestamptz not null,
  expires_at timestamptz not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),

  constraint cart_shipping_selections_cart_id_fkey foreign key (cart_id)
    references public.carts (id)
    on delete cascade,
  constraint cart_shipping_selections_shipping_box_id_fkey foreign key (shipping_box_id)
    references public.shipping_boxes (id),
  constraint cart_shipping_selections_provider_not_blank check (btrim(provider) <> ''),
  constraint cart_shipping_selections_service_code_not_blank check (btrim(service_code) <> ''),
  constraint cart_shipping_selections_service_name_not_blank check (btrim(service_name) <> ''),
  constraint cart_shipping_selections_carrier_name_not_blank check (carrier_name is null or btrim(carrier_name) <> ''),
  constraint cart_shipping_selections_price_cents_non_negative check (price_cents >= 0),
  constraint cart_shipping_selections_delivery_time_days_non_negative check (delivery_time_days is null or delivery_time_days >= 0),
  constraint cart_shipping_selections_package_weight_positive check (package_weight_g > 0),
  constraint cart_shipping_selections_package_dimensions_positive check (
    package_height_mm > 0
    and package_width_mm > 0
    and package_length_mm > 0
  ),
  constraint cart_shipping_selections_input_hash_length check (octet_length(input_hash) = 32),
  constraint cart_shipping_selections_expires_after_quoted check (expires_at > quoted_at)
);

create index shipping_boxes_active_selection_idx
  on public.shipping_boxes (is_active, sort_order asc, name asc, id asc)
  where is_active = true;

create index cart_shipping_selections_shipping_box_id_idx
  on public.cart_shipping_selections (shipping_box_id);

create index cart_shipping_selections_expires_at_idx
  on public.cart_shipping_selections (expires_at);

alter table public.shipping_boxes enable row level security;
alter table public.cart_shipping_selections enable row level security;
