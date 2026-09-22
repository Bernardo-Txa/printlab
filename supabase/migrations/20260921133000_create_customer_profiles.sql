create table public.customer_profiles (
  auth_user_id uuid primary key,
  full_name text not null default '',
  phone text not null default '',
  cpf text not null default '',
  postal_code text not null default '',
  street text not null default '',
  number text not null default '',
  complement text null,
  district text not null default '',
  city text not null default '',
  state text not null default '',
  country_code text not null default 'BR',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
alter table public.customer_profiles enable row level security;
create index customer_profiles_updated_at_idx on public.customer_profiles (updated_at desc);
