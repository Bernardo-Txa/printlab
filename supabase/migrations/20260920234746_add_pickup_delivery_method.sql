-- Explicit checkout delivery method. Existing rows remain shipping.
alter table public.cart_shipping_selections
  add column delivery_method text not null default 'shipping';
alter table public.cart_shipping_selections
  drop constraint cart_shipping_selections_provider_not_blank,
  drop constraint cart_shipping_selections_service_code_not_blank,
  drop constraint cart_shipping_selections_service_name_not_blank,
  drop constraint cart_shipping_selections_package_weight_positive,
  drop constraint cart_shipping_selections_package_dimensions_positive,
  add constraint cart_shipping_selections_delivery_method_allowed check (delivery_method in ('shipping', 'pickup')),
  add constraint cart_shipping_selections_pickup_fields check (
    (delivery_method = 'pickup' and shipping_box_id is null and provider = '' and service_code = '' and service_name = '' and carrier_name is null and price_cents = 0 and delivery_time_days is null and package_weight_g = 0 and package_height_mm = 0 and package_width_mm = 0 and package_length_mm = 0)
    or (delivery_method = 'shipping' and shipping_box_id is not null and btrim(provider) <> '' and btrim(service_code) <> '' and btrim(service_name) <> '' and price_cents >= 0 and package_weight_g > 0 and package_height_mm > 0 and package_width_mm > 0 and package_length_mm > 0)
  );
alter table public.cart_shipping_selections alter column shipping_box_id drop not null;

alter table public.order_shipping_details
  add column delivery_method text not null default 'shipping';
alter table public.order_shipping_details
  drop constraint order_shipping_details_provider_not_blank,
  drop constraint order_shipping_details_service_code_not_blank,
  drop constraint order_shipping_details_service_name_not_blank,
  drop constraint order_shipping_details_shipping_box_name_not_blank,
  drop constraint order_shipping_details_package_weight_positive,
  drop constraint order_shipping_details_package_dimensions_positive,
  add constraint order_shipping_details_delivery_method_allowed check (delivery_method in ('shipping', 'pickup')),
  add constraint order_shipping_details_pickup_fields check (
    (delivery_method = 'pickup' and provider = '' and service_code = '' and service_name = '' and carrier_name is null and delivery_time_days is null and shipping_box_name = '' and package_weight_g = 0 and package_height_mm = 0 and package_width_mm = 0 and package_length_mm = 0)
    or (delivery_method = 'shipping' and btrim(provider) <> '' and btrim(service_code) <> '' and btrim(service_name) <> '' and btrim(shipping_box_name) <> '' and package_weight_g > 0 and package_height_mm > 0 and package_width_mm > 0 and package_length_mm > 0)
  );
