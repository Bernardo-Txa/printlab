# Runbook inicial de incidentes

Preserve apenas horario UTC, evento, status HTTP e `request_id`; nunca copie PII, tokens, payloads ou URLs de checkout completas.

| Severidade | Detectar e primeira verificacao | Acao e limite |
| --- | --- | --- |
| SEV-1 | Pagamento confirmado sem pedido pago, perda/corrupcao de dados ou exposicao de secret/PII. Pesquisar os eventos de pagamento e confirmar `payment_check` na InfinitePay. | Interromper vendas se a integridade financeira ou privacidade estiver em risco; preservar evidencias seguras, revogar secret exposto e acionar responsavel. Nao alterar status manualmente sem `payment_check`. Rollback do ultimo deploy somente se for causa confirmada. |
| SEV-2 | Checkout, frete, Admin ou webhook indisponivel; buscar 5xx e `shipping_*`/`payment_*` na ultima hora. | Confirmar dependencia e ultimo deploy, registrar `request_id` e monitorar retries de webhook. Suspender vendas se checkout/pagamento continuar indisponivel. Nao desabilitar webhook, WAF ou MFA como atalho. |
| SEV-3 | Erro isolado, falha visual ou degradacao parcial. | Registrar evento seguro, reproduzir sem dados reais e abrir correcao. Nao executar migration, limpar dados ou publicar rollback sem evidencia. |

Para pagamentos, a fonte de verdade e sempre `payment_check` server-side: checkout indisponivel, retorno sem confirmacao, divergencia, webhook invalido e repeticoes anormais devem ser investigados pelos eventos respectivos. O webhook pode repetir; idempotencia e esperada.
