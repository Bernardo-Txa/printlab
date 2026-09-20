-- Optional commercial selection; production recipes remain independent.
alter table public.cart_items add column color_id uuid references public.colors(id);
create index cart_items_color_id_idx on public.cart_items(color_id) where color_id is not null;

drop index public.cart_items_cart_product_no_variant_unique_idx;
drop index public.cart_items_cart_product_variant_unique_idx;
create unique index cart_items_cart_product_no_variant_unique_idx
  on public.cart_items(cart_id, product_id) where variant_id is null and color_id is null;
create unique index cart_items_cart_product_variant_unique_idx
  on public.cart_items(cart_id, product_id, variant_id) where variant_id is not null and color_id is null;
create unique index cart_items_cart_product_color_unique_idx
  on public.cart_items(cart_id, product_id, color_id) where variant_id is null and color_id is not null;
create unique index cart_items_cart_product_variant_color_unique_idx
  on public.cart_items(cart_id, product_id, variant_id, color_id) where variant_id is not null and color_id is not null;

alter table public.order_items
  add column color_id uuid references public.colors(id) on delete set null,
  add column color_name text,
  add column color_slug text,
  add constraint order_items_color_snapshot_check check (
    (color_id is null and color_name is null and color_slug is null)
    or (nullif(btrim(color_name), '') is not null and nullif(btrim(color_slug), '') is not null)
  );
create index order_items_color_id_idx on public.order_items(color_id) where color_id is not null;
