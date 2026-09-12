# ADR-0011 — Confirmacao redundante de pagamentos InfinitePay

Status: Aprovado

Data: 2026-09-11

## Contexto

A Fase 10 confirmou pagamentos InfinitePay pelo retorno do navegador usando `payment_check` server-side. Esse fluxo e seguro, mas depende do comprador voltar para a PrintLab apos pagar.

A documentacao publica da InfinitePay permite enviar `webhook_url` na criacao do link e descreve eventos com `invoice_slug`, `transaction_nsu` e `order_nsu`. O contrato publico consultado nao documenta assinatura HMAC ou allowlist de IP.

## Decisao

A PrintLab enviara `webhook_url` em `POST /links`, apontando para `POST /webhooks/infinitepay`.

O webhook sera tratado apenas como gatilho. O backend valida identificadores, localiza o pedido por `order_nsu` e chama `payment_check` server-side com `handle`, `order_nsu`, `transaction_nsu` e `invoice_slug`. O pedido muda para `paid` somente quando a InfinitePay confirma `success=true`, `paid=true` e `amount` igual a `orders.total_cents`.

O endpoint aceita JSON com limite de 64 KiB, nao aplica Origin/Referer e responde rapidamente com JSON. Sucesso confirmado ou pagamento ja confirmado retornam HTTP 200. Payload invalido, pedido ausente, pagamento pendente, falha temporaria do provider ou divergencia de valor retornam HTTP 400 para permitir retry do provider.

## Alternativas consideradas

- Marcar pago diretamente pelo webhook: rejeitado porque o contrato publico consultado nao documenta assinatura.
- Exigir Origin/Referer no webhook: rejeitado porque chamadas server-to-server de provider normalmente nao carregam esses headers.
- Regenerar checkouts pendentes antigos para adicionar webhook: rejeitado para preservar idempotencia e nao substituir checkout URL ja persistida.
- Criar migration ou tabela de eventos: rejeitado nesta fase; a entrega usa `order_payments` existente e confirmacao transacional ja implementada.

## Consequencias

- Pagamentos podem ser confirmados mesmo se o comprador nao clicar em continuar/retornar.
- Checkouts criados antes da Fase 11 nao recebem `webhook_url` retroativamente.
- A seguranca depende de `payment_check` server-side, comparacao de valor e idempotencia transacional.
- Recebimento real do webhook e confirmacao sem redirect ainda precisam ser validados em producao.

## Referencias

- [InfinitePay](../integrations/infinitepay.md)
- [Pedidos](../product/orders.md)
- [Seguranca](../architecture/security.md)
