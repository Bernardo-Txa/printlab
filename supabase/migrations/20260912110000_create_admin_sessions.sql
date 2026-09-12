create table public.admin_sessions (
  id uuid primary key default gen_random_uuid(),
  auth_user_id uuid not null,
  token_hash bytea not null unique,
  created_at timestamptz not null default now(),
  expires_at timestamptz not null,

  constraint admin_sessions_token_hash_length check (octet_length(token_hash) = 32),
  constraint admin_sessions_expires_after_created check (expires_at > created_at)
);

create index admin_sessions_expires_at_idx
  on public.admin_sessions (expires_at);

alter table public.admin_sessions enable row level security;
