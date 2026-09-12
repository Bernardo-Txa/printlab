# Integracoes

Status: SuperFrete IMPLEMENTADO e VALIDADO em Sandbox para cotacao de frete; InfinitePay IMPLEMENTADO com link e pagamento reais validados e webhook implementado; demais integracoes comerciais PLANEJADAS.

## Responsabilidade

Integracoes externas devem permitir calculo de frete, pagamentos e outros servicos necessarios sem expor credenciais ou regras sensiveis ao navegador.

## Limites

- SuperFrete foi implementado apenas para cotacao server-side de frete.
- Sandbox SuperFrete real foi validado manualmente para cotacao, pacote planejado, caixa real compativel, modalidades e selecao persistida.
- Nenhuma etiqueta, postagem ou rastreio foi implementado.
- Pagamento InfinitePay foi implementado como checkout hospedado server-side com confirmacao por `payment_check`; link real e pagamento real foram validados, e recebimento real de webhook ainda precisa de validacao controlada.

## Decisoes

- SuperFrete e usado para cotacao de frete via backend Go.
- A cotacao SuperFrete usa `POST /api/v0/calculator`, Bearer token e `User-Agent` operacional conforme documentacao oficial.
- O backend envia `products` primeiro para obter pacote ideal e depois `package` com caixa real para cotacao final.
- InfinitePay e usado para checkout hospedado, retorno e webhook, sempre com validacao server-side via `payment_check`.
- Supabase hospedara PostgreSQL.
- O backend Go fara chamadas para servicos externos quando necessario.

## Praticas recomendadas

- Confirmar contratos na documentacao oficial durante a implementacao.
- Guardar credenciais em environment variables.
- Validar respostas externas antes de persistir estado.
- Implementar timeouts e tratamento explicito de erro.
- Preservar idempotencia para webhooks.

## Praticas proibidas

- Inventar endpoints ou payloads.
- Expor tokens no frontend.
- Confiar em redirect de pagamento como confirmacao.
- Persistir dados externos sem validacao.

## Documentos especificos

- [Supabase](../integrations/supabase.md)
- [SuperFrete](../integrations/superfrete.md)
- [InfinitePay](../integrations/infinitepay.md)
