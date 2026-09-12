alter table public.order_payments
  drop constraint order_payments_checkout_url_host;

alter table public.order_payments
  add constraint order_payments_checkout_url_host check (
    checkout_url is null
    or checkout_url ~ '^https://checkout[.]infinitepay[.]com[.]br/.+'
    or checkout_url ~ '^https://checkout[.]infinitepay[.]io/.+'
  );
