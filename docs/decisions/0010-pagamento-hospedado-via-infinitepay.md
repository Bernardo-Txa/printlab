# ADR-0010 — Pagamento hospedado via InfinitePay

Status: Aprovado

Data: 2026-09-11

## Contexto

A PrintLab cria pedidos como snapshots imutaveis antes do pagamento. A Fase 10 precisava iniciar pagamento real sem expor valores financeiros ao navegador e sem tratar redirect como confirmacao.

A documentacao oficial da InfinitePay oferece checkout hospedado por `POST /links` e confirmacao por `POST /payment_check`.

## Decisao

Usaremos checkout hospedado InfinitePay iniciado pelo backend Go.

O pedido continua sendo criado primeiro com status `pending_payment`. A rota `POST /pedido/{id}/pagar` monta o payload a partir do snapshot do pedido, confere o total em centavos contra `orders.total_cents`, chama `POST /links` e redireciona o comprador para `https://checkout.infinitepay.com.br/...`.

O retorno `GET /pagamento/retorno` usa somente `order_nsu`, `transaction_nsu` e `slug` para chamar `payment_check` server-side. O pedido muda para `paid` somente quando `success=true`, `paid=true` e `amount` iguala o total congelado do pedido.

Pagamentos ficam em `public.order_payments`, 1:1 com `orders`, com RLS habilitado e sem policies publicas.

## Alternativas consideradas

- Confirmar pagamento por redirect: rejeitado por nao ser fonte confiavel.
- Criar pedido somente apos pagamento: rejeitado porque a arquitetura atual ja usa pedido como snapshot historico e fonte de payload.
- Implementar webhook na mesma fase: rejeitado para manter escopo controlado; sera Fase 11.
- Usar SDK externo: rejeitado porque `net/http` e suficiente para o contrato atual.

## Consequencias

- A aplicacao nao precisa expor credenciais ou valores financeiros ao frontend.
- `INFINITEPAY_HANDLE` e configuracao de runtime; valor real nao entra no Git.
- Sem webhook, se o comprador pagar e nao retornar ao site, o pedido pode permanecer temporariamente `pending_payment`.
- A validacao real de link e pagamento precisa de execucao controlada posterior.

## Referencias

- [InfinitePay](../integrations/infinitepay.md)
- [Pedidos](../product/orders.md)
- [Seguranca](../architecture/security.md)
