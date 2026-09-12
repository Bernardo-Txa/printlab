# Fase 11 — Webhook InfinitePay

Status: CONCLUIDA.

## Objetivo

Adicionar webhook InfinitePay para confirmar pagamentos sem depender do comprador clicar em continuar/retornar ao site, mantendo `payment_check` server-side como unica fonte de autoridade.

## Entregas implementadas

- `webhook_url` enviado no payload de `POST /links`.
- `PaymentWebhookURL(SITE_URL)` gerando caminho canonico `/webhooks/infinitepay` com HTTPS obrigatorio.
- Endpoint `POST /webhooks/infinitepay`.
- Aceite inicial de `invoice_slug`, `transaction_nsu`, `order_nsu`, `amount`, `paid_amount`, `installments` e `capture_method`.
- JSON limitado a 64 KiB e resposta `Cache-Control: no-store`.
- Sem validacao de `Origin`/`Referer` no webhook.
- Confirmacao por `payment_check` server-side antes de marcar pedido como pago.
- Comparacao de `amount` contra `orders.total_cents`.
- Idempotencia para webhook duplicado, retorno depois do webhook e webhook depois do retorno.
- Logs seguros sem PII, checkout URL completa, payload bruto, `transaction_nsu` ou `invoice_slug`.
- Nenhuma migration.

## Fora do escopo respeitado

- Nenhum webhook marca pedido como pago diretamente pelo payload recebido.
- Nenhum HMAC/IP allowlist foi inventado sem contrato oficial documentado.
- Nenhuma alteracao de schema.
- Nenhuma alteracao de endpoint InfinitePay.
- Nenhuma regra financeira alterada.
- Nenhum painel administrativo, producao, envio ou rastreio.

## Definition of Done

- A. Implementacao local, testes automatizados e documentacao: concluida ✅
- B. Novo checkout real contendo `webhook_url`: validado em producao ✅
- C. Webhook real recebido em producao: validado ✅
- D. Pagamento real confirmado sem redirect do comprador: validado ✅

## Validacao automatizada final

- `templ generate`;
- `npm run css:build`;
- `gofmt -w .`;
- `go test ./...`;
- `go vet ./...`;
- `go build ./...`.

## Validacao real em producao

Validacao manual real confirmada em 2026-09-12:

1. Foi criado um novo checkout apos o deploy da Fase 11.
2. O checkout continha `webhook_url`.
3. Foi realizado pagamento Pix real.
4. O comprador nao clicou em continuar na InfinitePay.
5. A InfinitePay enviou `POST /webhooks/infinitepay`.
6. User-Agent observado: `InfinitePay/EcommerceWebhooks`.
7. O endpoint respondeu HTTP 200.
8. O pedido foi atualizado sem redirect do comprador.
9. `orders.status = paid`.
10. `order_payments.status = paid`.
11. `amount_cents` correspondeu ao total esperado.
12. `capture_method = pix`.
13. `paid_at` foi preenchido.

Nao foram registrados neste documento `transaction_nsu`, `invoice_slug`, CPF, e-mail, telefone, endereco, checkout URL ou qualquer PII.
