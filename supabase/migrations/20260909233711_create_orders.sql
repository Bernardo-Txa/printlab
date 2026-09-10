alter table public.carts
  add column converted_at timestamptz null;

create index carts_converted_at_idx
  on public.carts (converted_at)
  where converted_at is not null;

create table public.orders (
  id uuid primary key default gen_random_uuid(),
  order_number bigint generated always as identity (start with 1001),
  source_cart_id uuid null,
  status text not null default 'pending_payment',
  currency text not null default 'BRL',
  products_subtotal_cents bigint not null,
  shipping_price_cents bigint not null,
  total_cents bigint not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),

  constraint orders_source_cart_id_fkey foreign key (source_cart_id)
    references public.carts (id)
    on delete set null,
  constraint orders_order_number_unique unique (order_number),
  constraint orders_status_allowed check (status in ('pending_payment')),
  constraint orders_currency_brl check (currency = 'BRL'),
  constraint orders_products_subtotal_non_negative check (products_subtotal_cents >= 0),
  constraint orders_shipping_price_non_negative check (shipping_price_cents >= 0),
  constraint orders_total_non_negative check (total_cents >= 0),
  constraint orders_total_matches_components check (total_cents = products_subtotal_cents + shipping_price_cents)
);

create unique index orders_source_cart_id_unique_idx
  on public.orders (source_cart_id)
  where source_cart_id is not null;

create index orders_created_at_idx on public.orders (created_at desc);

create table public.order_customer_details (
  order_id uuid primary key,
  full_name text not null,
  email text not null,
  phone text not null,
  cpf text not null,
  created_at timestamptz not null default now(),

  constraint order_customer_details_order_id_fkey foreign key (order_id)
    references public.orders (id)
    on delete cascade,
  constraint order_customer_details_full_name_length check (char_length(full_name) between 1 and 120),
  constraint order_customer_details_full_name_trimmed check (full_name = btrim(full_name)),
  constraint order_customer_details_email_length check (char_length(email) between 3 and 254),
  constraint order_customer_details_email_lower_trimmed check (email = lower(btrim(email))),
  constraint order_customer_details_phone_format check (phone ~ '^\+55[0-9]{10,11}$'),
  constraint order_customer_details_cpf_format check (cpf ~ '^[0-9]{11}$')
);

create table public.order_shipping_addresses (
  order_id uuid primary key,
  postal_code text not null,
  street text not null,
  number text not null,
  complement text null,
  district text not null,
  city text not null,
  state text not null,
  country_code text not null default 'BR',
  created_at timestamptz not null default now(),

  constraint order_shipping_addresses_order_id_fkey foreign key (order_id)
    references public.orders (id)
    on delete cascade,
  constraint order_shipping_addresses_postal_code_format check (postal_code ~ '^[0-9]{8}$'),
  constraint order_shipping_addresses_street_length check (char_length(street) between 1 and 160),
  constraint order_shipping_addresses_street_trimmed check (street = btrim(street)),
  constraint order_shipping_addresses_number_length check (char_length(number) between 1 and 30),
  constraint order_shipping_addresses_number_trimmed check (number = btrim(number)),
  constraint order_shipping_addresses_complement_length check (complement is null or char_length(complement) between 1 and 120),
  constraint order_shipping_addresses_complement_trimmed check (complement is null or complement = btrim(complement)),
  constraint order_shipping_addresses_district_length check (char_length(district) between 1 and 100),
  constraint order_shipping_addresses_district_trimmed check (district = btrim(district)),
  constraint order_shipping_addresses_city_length check (char_length(city) between 1 and 100),
  constraint order_shipping_addresses_city_trimmed check (city = btrim(city)),
  constraint order_shipping_addresses_state_allowed check (state in (
    'AC', 'AL', 'AP', 'AM', 'BA', 'CE', 'DF', 'ES', 'GO',
    'MA', 'MT', 'MS', 'MG', 'PA', 'PB', 'PR', 'PE', 'PI',
    'RJ', 'RN', 'RS', 'RO', 'RR', 'SC', 'SP', 'SE', 'TO'
  )),
  constraint order_shipping_addresses_country_code_br check (country_code = 'BR')
);

create table public.order_shipping_details (
  order_id uuid primary key,
  provider text not null,
  service_code text not null,
  service_name text not null,
  carrier_name text null,
  delivery_time_days integer null,
  shipping_box_name text not null,
  package_weight_g bigint not null,
  package_height_mm integer not null,
  package_width_mm integer not null,
  package_length_mm integer not null,
  quoted_at timestamptz null,
  created_at timestamptz not null default now(),

  constraint order_shipping_details_order_id_fkey foreign key (order_id)
    references public.orders (id)
    on delete cascade,
  constraint order_shipping_details_provider_not_blank check (btrim(provider) <> ''),
  constraint order_shipping_details_service_code_not_blank check (btrim(service_code) <> ''),
  constraint order_shipping_details_service_name_not_blank check (btrim(service_name) <> ''),
  constraint order_shipping_details_carrier_name_not_blank check (carrier_name is null or btrim(carrier_name) <> ''),
  constraint order_shipping_details_delivery_time_days_non_negative check (delivery_time_days is null or delivery_time_days >= 0),
  constraint order_shipping_details_shipping_box_name_not_blank check (btrim(shipping_box_name) <> ''),
  constraint order_shipping_details_package_weight_positive check (package_weight_g > 0),
  constraint order_shipping_details_package_dimensions_positive check (
    package_height_mm > 0
    and package_width_mm > 0
    and package_length_mm > 0
  )
);

create table public.order_items (
  id uuid primary key default gen_random_uuid(),
  order_id uuid not null,
  product_id uuid null,
  variant_id uuid null,
  product_name text not null,
  product_slug text not null,
  variant_name text null,
  variant_slug text null,
  sku text null,
  unit_price_cents bigint not null,
  quantity integer not null,
  line_total_cents bigint not null,
  unit_print_time_minutes integer null,
  unit_estimated_filament_weight_mg bigint null,
  sort_order integer not null default 0,
  created_at timestamptz not null default now(),

  constraint order_items_order_id_fkey foreign key (order_id)
    references public.orders (id)
    on delete cascade,
  constraint order_items_product_id_fkey foreign key (product_id)
    references public.products (id)
    on delete set null,
  constraint order_items_variant_id_fkey foreign key (variant_id)
    references public.product_variants (id)
    on delete set null,
  constraint order_items_product_name_not_blank check (btrim(product_name) <> ''),
  constraint order_items_product_slug_format check (product_slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
  constraint order_items_variant_name_not_blank check (variant_name is null or btrim(variant_name) <> ''),
  constraint order_items_variant_slug_format check (variant_slug is null or variant_slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
  constraint order_items_sku_not_blank check (sku is null or btrim(sku) <> ''),
  constraint order_items_unit_price_non_negative check (unit_price_cents >= 0),
  constraint order_items_quantity_range check (quantity between 1 and 99),
  constraint order_items_line_total_non_negative check (line_total_cents >= 0),
  constraint order_items_line_total_matches_quantity check (line_total_cents = unit_price_cents * quantity),
  constraint order_items_unit_print_time_positive check (unit_print_time_minutes is null or unit_print_time_minutes > 0),
  constraint order_items_unit_filament_weight_positive check (unit_estimated_filament_weight_mg is null or unit_estimated_filament_weight_mg > 0),
  constraint order_items_sort_order_non_negative check (sort_order >= 0)
);

create index order_items_order_id_idx on public.order_items (order_id);

create table public.order_item_filaments (
  id uuid primary key default gen_random_uuid(),
  order_item_id uuid not null,
  material_name text not null,
  material_slug text not null,
  color_name text not null,
  color_slug text not null,
  hex_color text null,
  estimated_weight_mg_per_unit bigint not null,
  label text null,
  sort_order integer not null default 0,
  created_at timestamptz not null default now(),

  constraint order_item_filaments_order_item_id_fkey foreign key (order_item_id)
    references public.order_items (id)
    on delete cascade,
  constraint order_item_filaments_material_name_not_blank check (btrim(material_name) <> ''),
  constraint order_item_filaments_material_slug_format check (material_slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
  constraint order_item_filaments_color_name_not_blank check (btrim(color_name) <> ''),
  constraint order_item_filaments_color_slug_format check (color_slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'),
  constraint order_item_filaments_hex_color_format check (hex_color is null or hex_color ~ '^#[0-9A-F]{6}$'),
  constraint order_item_filaments_weight_positive check (estimated_weight_mg_per_unit > 0),
  constraint order_item_filaments_label_not_blank check (label is null or btrim(label) <> ''),
  constraint order_item_filaments_sort_order_non_negative check (sort_order >= 0)
);

create index order_item_filaments_order_item_id_idx
  on public.order_item_filaments (order_item_id);

alter table public.orders enable row level security;
alter table public.order_customer_details enable row level security;
alter table public.order_shipping_addresses enable row level security;
alter table public.order_shipping_details enable row level security;
alter table public.order_items enable row level security;
alter table public.order_item_filaments enable row level security;
