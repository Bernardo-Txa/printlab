# Fase 18.4 — Pagamentos pendentes e retomada

Status: Concluida; validada manualmente em producao.

## Escopo implementado

- `/conta` mostra pedidos do cliente autenticado com numero humano, data, status, total e link de acompanhamento.
- Pedido `pending_payment` exibe `Retomar pagamento` quando o servico InfinitePay esta disponivel.
- A retomada usa `POST /conta/pedidos/{id}/pagar`; nao ha GET que inicie pagamento.
- O handler exige sessao Supabase Auth, valida Origin/Referer e nunca aceita `customer_auth_user_id` do navegador.
- `internal/payments` ganhou `StartCheckoutForCustomer`, reutilizando o mesmo fluxo de criacao/reuso de checkout hospedado.
- A autorizacao ocorre no repository, dentro da transacao, carregando o pedido por `orders.id` e `orders.customer_auth_user_id`.
- Checkout pendente valido e reutilizado. Pedido pago, pagamento local `paid`, status nao pagavel, total divergente ou checkout URL invalida bloqueiam retomada.

## Regras preservadas

- Checkout convidado continua sem login obrigatorio.
- `/pedido/{id}` e `POST /pedido/{id}/pagar` continuam publicos por capability para pagamento imediato e pedidos convidados.
- Pedido guest com `customer_auth_user_id IS NULL` nao e retomado pela conta, mesmo com e-mail igual ao usuario autenticado.
- Confirmacao de pagamento continua exclusiva de `payment_check` via retorno InfinitePay ou webhook.
- Tracking publico nao recebeu CTA financeiro.
- Admin, frete, pickup, webhook e retorno mantem os contratos existentes.

## Pendencias abandonadas

Nao foi definido TTL arbitrario, cancelamento automatico, cron de exclusao ou novo status comercial. Pedidos pendentes antigos permanecem persistidos como evidencia historica/financeira e nao entram em producao ate pagamento confirmado.

Qualquer expiracao, cancelamento ou limpeza futura exige politica comercial explicita e, se necessario, modelagem propria.

## Migration

Nenhuma migration foi criada. A fase usa `orders.customer_auth_user_id` e `order_payments` existentes.

## Validacao automatizada

- Handler de `/conta`: CTA para pedido pendente, ausencia para pedido pago, mensagens PRG e ausencia de checkout URL no HTML.
- Handler de retomada: autenticacao, origem invalida, ownership invalido como not found, redirect InfinitePay, pedido pago, URL invalida e logs sem PII.
- Service de pagamentos: `StartCheckoutForCustomer` cria checkout, reutiliza checkout pendente e nao chama provider quando ownership falha.
- Repository/source: ownership por `o.customer_auth_user_id = $2::uuid`, transacao/locks preservados e sem fallback por e-mail.

Executar antes de publicar: `templ generate`, `gofmt` e `go test ./...`.

## Validacao manual em producao

Validado pelo responsável:

- pedido `pending_payment` aparece na conta com Retomar pagamento;
- retomada redireciona corretamente para InfinitePay;
- abandono do checkout mantém pedido pendente e retomável;
- pedido pago não exibe CTA de retomada;
- `/conta` continua protegida por autenticação.
