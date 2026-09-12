# Fase 10 — Pagamentos InfinitePay

Status: CONCLUIDA; link real e pagamento real validados antes da Fase 11.

## Objetivo

Implementar inicio de pagamento hospedado InfinitePay a partir de pedido ja criado, com confirmacao server-side por `payment_check`, sem webhook nesta fase.

## Entregas

- Configuracao opcional `INFINITEPAY_HANDLE`, sem valor real no repositorio.
- Migration `add_order_payments`, criando `public.order_payments` e permitindo `orders.status = 'paid'`.
- Pacote `internal/payments` com tipos de dominio, service, client HTTP InfinitePay e repository PostgreSQL.
- `POST /pedido/{id}/pagar` para criar ou reutilizar checkout hospedado.
- `GET /pagamento/retorno` para validar retorno com `payment_check`.
- Pagina publica de pedido com etapa 4 de pagamento SSR.
- Pedido `paid` exibindo `Pagamento confirmado` sem botao de pagamento.
- Testes de client HTTP, regras de total, idempotencia, retorno, handlers, migration e repository.
- Documentacao de produto, arquitetura, seguranca, schema, setup, deployment, integracao InfinitePay e ADR.

## Fora do escopo

- Webhook InfinitePay.
- Envio de pedido para producao.
- Etiqueta, postagem, rastreio ou SuperFrete de envio.
- Painel administrativo.
- Webhook ou confirmacao sem retorno do comprador.

## Validacao

- Implementacao automatizada: concluida por testes locais.
- Checkout/link real InfinitePay: validado em producao.
- Pagamento real confirmado via `payment_check`: validado em producao.

Confirmacao sem retorno do comprador passa a ser tratada pela Fase 11.
