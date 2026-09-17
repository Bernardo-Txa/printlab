# Observabilidade operacional

## Runtime Logs Vercel

No plano Hobby, Runtime Logs ficam disponiveis por uma hora. Consulte Dashboard > Logs ou CLI autenticada, sem versionar `VERCEL_TOKEN`:

```bash
vercel logs --environment production --status-code 5xx --since 30m
vercel logs --environment production --level error --since 30m
vercel logs --environment production --query "payment_webhook" --since 1h
vercel logs --environment production --query "payment_check_failed" --since 1h
```

Eventos operacionais usam `event=`, `level=`, `reason=` seguro e `request_id=` opaco. `info` registra sucesso ou evento operacional esperado; `warning`, rejeicao ou falha esperada relevante; `error`, indisponibilidade ou falha operacional. O servidor gera o ID e devolve `X-Request-ID`; nao aceita um ID enviado pelo cliente. Eventos `info` e `warning` escrevem em stdout; `error` usa stderr para preservar a classificacao do Runtime Logs. Nunca pesquisar ou registrar PII, tokens, URLs completas de checkout, bodies ou IDs de pedido/carrinho.

Taxonomia inicial: `payment_checkout_unavailable`, `payment_check_failed`, `payment_webhook_unavailable`, `payment_webhook_processing_failed` (inclusive `reason=amount_mismatch`), `admin_mfa_provider_unavailable` (inclusive `reason=configuration_invalid`) e falhas operacionais de `order_creation_failed` usam `error`; `payment_webhook_invalid`, `admin_auth_rejected`, `admin_mfa_invalid_code` e `admin_mfa_rate_limited` usam `warning`; `payment_webhook_processed` usa `info`. Frete ja registra somente estagio e categoria segura; sua padronizacao com correlacao permanece uma lacuna consciente para evitar refactor amplo nesta subfase.

## Alertas

Em 2026-09, os Alerts da Vercel estao em beta para Pro/Enterprise com Observability Plus, portanto nao estao configuraveis no Hobby. Nao houve upgrade nem ferramenta externa. Enquanto o volume e baixo, revisar Logs/Runtime Errors diariamente em dias de venda e apos cada relato de falha; investigar imediatamente 5xx, qualquer `payment_*failed` e repeticoes de webhook. Reavaliar alerta automatizado antes de aumento relevante de volume ou go-live comercial.

Referencias: [Vercel Logs](https://vercel.com/docs/cli/logs), [Runtime Logs](https://vercel.com/docs/logs/runtime), [Alerts](https://vercel.com/docs/alerts).
