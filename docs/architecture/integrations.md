# Integracoes

Status: PLANEJADO.

## Responsabilidade

Integracoes externas devem permitir calculo de frete, pagamentos e outros servicos necessarios sem expor credenciais ou regras sensiveis ao navegador.

## Limites

- Nenhuma integracao real foi implementada.
- Nenhuma chamada HTTP externa foi criada.
- Nenhum endpoint, payload ou contrato de API foi assumido.

## Decisoes

- SuperFrete sera avaliado para frete.
- InfinitePay sera avaliado para checkout/pagamentos.
- Supabase hospedara PostgreSQL.
- O backend Go fara chamadas para servicos externos quando necessario.

## Praticas recomendadas

- Confirmar contratos na documentacao oficial durante a implementacao.
- Guardar credenciais em environment variables.
- Validar respostas externas antes de persistir estado.
- Implementar timeouts e tratamento explicito de erro.
- Planejar idempotencia para webhooks.

## Praticas proibidas

- Inventar endpoints ou payloads.
- Expor tokens no frontend.
- Confiar em redirect de pagamento como confirmacao.
- Persistir dados externos sem validacao.

## Documentos especificos

- [Supabase](../integrations/supabase.md)
- [SuperFrete](../integrations/superfrete.md)
- [InfinitePay](../integrations/infinitepay.md)
