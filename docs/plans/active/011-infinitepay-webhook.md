# Fase 11 — Webhook InfinitePay

Status: IMPLEMENTACAO CONCLUIDA; validacao real de webhook pendente.

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

- A. Implementacao local, testes automatizados e documentacao: concluida.
- B. Novo checkout real contendo `webhook_url`: pendente de validacao controlada.
- C. Webhook real recebido em producao: pendente.
- D. Pagamento real confirmado sem redirect do comprador: pendente.

## Validacao automatizada prevista

- `templ generate`;
- `npm run css:build`;
- `gofmt -w .`;
- `go mod tidy`;
- `go test ./...`;
- `go vet ./...`;
- `go build ./...`.

## Validacao real pendente

1. Criar um novo pedido apos deploy desta fase.
2. Iniciar o pagamento e confirmar que o checkout criado pela InfinitePay recebeu `webhook_url`.
3. Pagar sem clicar em continuar/retornar.
4. Confirmar nos logs que `payment webhook received` ocorreu.
5. Confirmar que o pedido mudou para `paid` apos `payment_check` server-side.
