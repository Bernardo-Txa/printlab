alter table public.admin_sessions
  add column mfa_verified_at timestamptz;
