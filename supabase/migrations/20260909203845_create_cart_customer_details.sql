create table public.cart_customer_details (
  cart_id uuid primary key,
  full_name text not null,
  email text not null,
  phone text not null,
  cpf text not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),

  constraint cart_customer_details_cart_id_fkey foreign key (cart_id)
    references public.carts (id)
    on delete cascade,
  constraint cart_customer_details_full_name_length check (char_length(full_name) between 1 and 120),
  constraint cart_customer_details_full_name_trimmed check (full_name = btrim(full_name)),
  constraint cart_customer_details_email_length check (char_length(email) between 3 and 254),
  constraint cart_customer_details_email_lower_trimmed check (email = lower(btrim(email))),
  constraint cart_customer_details_phone_format check (phone ~ '^\+55[0-9]{10,11}$'),
  constraint cart_customer_details_cpf_format check (cpf ~ '^[0-9]{11}$')
);

create table public.cart_shipping_addresses (
  cart_id uuid primary key,
  postal_code text not null,
  street text not null,
  number text not null,
  complement text null,
  district text not null,
  city text not null,
  state text not null,
  country_code text not null default 'BR',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),

  constraint cart_shipping_addresses_cart_id_fkey foreign key (cart_id)
    references public.carts (id)
    on delete cascade,
  constraint cart_shipping_addresses_postal_code_format check (postal_code ~ '^[0-9]{8}$'),
  constraint cart_shipping_addresses_street_length check (char_length(street) between 1 and 160),
  constraint cart_shipping_addresses_street_trimmed check (street = btrim(street)),
  constraint cart_shipping_addresses_number_length check (char_length(number) between 1 and 30),
  constraint cart_shipping_addresses_number_trimmed check (number = btrim(number)),
  constraint cart_shipping_addresses_complement_length check (complement is null or char_length(complement) between 1 and 120),
  constraint cart_shipping_addresses_complement_trimmed check (complement is null or complement = btrim(complement)),
  constraint cart_shipping_addresses_district_length check (char_length(district) between 1 and 100),
  constraint cart_shipping_addresses_district_trimmed check (district = btrim(district)),
  constraint cart_shipping_addresses_city_length check (char_length(city) between 1 and 100),
  constraint cart_shipping_addresses_city_trimmed check (city = btrim(city)),
  constraint cart_shipping_addresses_state_allowed check (state in (
    'AC', 'AL', 'AP', 'AM', 'BA', 'CE', 'DF', 'ES', 'GO',
    'MA', 'MT', 'MS', 'MG', 'PA', 'PB', 'PR', 'PE', 'PI',
    'RJ', 'RN', 'RS', 'RO', 'RR', 'SC', 'SP', 'SE', 'TO'
  )),
  constraint cart_shipping_addresses_country_code_br check (country_code = 'BR')
);

alter table public.cart_customer_details enable row level security;
alter table public.cart_shipping_addresses enable row level security;
