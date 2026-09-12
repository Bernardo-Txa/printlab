# Fase 12 — Acompanhamento Seguro do Pedido

Status: CONCLUIDA.

## Objetivo

Permitir que o cliente acompanhe o estado basico do pedido por link publico seguro, sem login e sem expor dados pessoais, financeiros ou operacionais sensiveis.

## Entregas

- Migration `add_order_tracking_and_fulfillment`.
- `orders.public_tracking_id` UUID aleatorio, unico, obrigatorio e com backfill para pedidos existentes.
- Tabela `public.order_fulfillment` 1:1 com `orders`.
- Status de producao: `waiting`, `in_production`, `completed`.
- Status de envio: `waiting`, `preparing`, `shipped`, `delivered`.
- Constraint impedindo envio diferente de `waiting` antes de producao `completed`.
- Insercao transacional de `order_fulfillment` na criacao de novos pedidos.
- Rota `GET /acompanhar/{public_tracking_id}`.
- Link `Acompanhar pedido` em `/pedido/{id}` usando `public_tracking_id`.
- Headers `Cache-Control: private, no-store`, `X-Robots-Tag: noindex, nofollow, noarchive` e `Referrer-Policy: no-referrer`.
- Meta robots `noindex, nofollow, noarchive`.
- Tests de dominio, handler, privacidade, migration e schema opcional via `TEST_DATABASE_URL`.

## Fora de escopo

- Login de cliente.
- Painel administrativo.
- Endpoints publicos de mutacao de status.
- Etiqueta, postagem ou rastreio SuperFrete.
- Codigo de rastreio de transportadora.
- E-mail ou WhatsApp.
- Fase 13.

## Definition of Done

- Acesso usa `public_tracking_id`, nao `orders.id` nem `order_number`: concluido ✅
- UUID invalido e UUID desconhecido retornam 404: concluido ✅
- Pagina publica nao renderiza PII, valores, produtos, IDs internos ou identificadores de pagamento: concluido ✅
- Status de pagamento, producao e envio refletem fonte server-side: concluido ✅
- Documentacao atualizada: concluido ✅
- Validacoes obrigatorias executadas.
