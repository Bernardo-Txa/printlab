-- Fase 17.3.1: associa cores comerciais selecionaveis ao produto.
-- A relacao e nova e nao altera colors, product_variants ou variant_filaments.
-- Rollback: drop table public.product_colors;

create table public.product_colors (
  id uuid primary key default gen_random_uuid(),
  product_id uuid not null references public.products (id) on delete cascade,
  color_id uuid not null references public.colors (id),
  sort_order integer not null default 0,
  created_at timestamptz not null default now(),

  constraint product_colors_product_color_unique unique (product_id, color_id),
  constraint product_colors_sort_order_non_negative check (sort_order >= 0)
);

create index product_colors_product_order_idx
  on public.product_colors (product_id, sort_order asc, id asc);

alter table public.product_colors enable row level security;
