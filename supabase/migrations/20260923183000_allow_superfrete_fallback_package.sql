alter table public.cart_shipping_selections
  drop constraint cart_shipping_selections_pickup_fields,
  add constraint cart_shipping_selections_pickup_fields check (
    (delivery_method = 'pickup' and shipping_box_id is null and provider = '' and service_code = '' and service_name = '' and carrier_name is null and price_cents = 0 and delivery_time_days is null and package_weight_g = 0 and package_height_mm = 0 and package_width_mm = 0 and package_length_mm = 0)
    or (delivery_method = 'shipping' and btrim(provider) <> '' and btrim(service_code) <> '' and btrim(service_name) <> '' and price_cents >= 0 and package_weight_g > 0 and package_height_mm > 0 and package_width_mm > 0 and package_length_mm > 0)
  );
