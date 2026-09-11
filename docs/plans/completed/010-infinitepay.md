# Fase 10 — Pagamentos InfinitePay

Status: IMPLEMENTACAO CONCLUIDA; validacao real de link e pagamento pendente.

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
- Pagamento real automatico ou validacao manual nao controlada.

## Validacao

- Implementacao automatizada: concluida por testes locais.
- Checkout/link real InfinitePay: pendente.
- Pagamento real confirmado via `payment_check`: pendente.

Enquanto a validacao real nao ocorrer, a fase deve ser comunicada como implementacao concluida com validacao real pendente.
