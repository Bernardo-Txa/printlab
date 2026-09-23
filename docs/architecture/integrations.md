# Integracoes

Status: SuperFrete IMPLEMENTADO e VALIDADO em Sandbox para cotacao de frete; InfinitePay IMPLEMENTADO com link, pagamento e webhook reais validados; Supabase Auth IMPLEMENTADO para Admin; Supabase Storage para imagens Admin VALIDADO em producao; Supabase Cron IMPLEMENTADO para limpeza transiente; demais integracoes comerciais PLANEJADAS.

## Responsabilidade

Integracoes externas devem permitir calculo de frete, pagamentos e outros servicos necessarios sem expor credenciais ou regras sensiveis ao navegador.

## Limites

- SuperFrete foi implementado apenas para cotacao server-side de frete.
- Sandbox SuperFrete real foi validado manualmente para cotacao, pacote planejado, caixa real compativel, modalidades e selecao persistida.
- Nenhuma etiqueta, postagem ou rastreio foi implementado.
- Pagamento InfinitePay foi implementado como checkout hospedado server-side com confirmacao por `payment_check`; link real, pagamento real e webhook real foram validados.
- Supabase Auth foi implementado somente para login administrativo por e-mail/senha; signup, MFA, service role e RBAC permanecem fora do escopo.
- Supabase Storage foi implementado para imagens Admin com signed upload URL e finalizacao server-side, validado em producao.
- Supabase Cron foi implementado somente para limpeza diaria de `admin_sessions` e `carts` expirados.

## Decisoes

- SuperFrete e usado para cotacao de frete via backend Go.
- A cotacao SuperFrete usa `POST /api/v0/calculator`, Bearer token e `User-Agent` operacional conforme documentacao oficial.
- O backend seleciona a caixa fisica real e envia `package` diretamente para a SuperFrete cotar preco e prazo.
- InfinitePay e usado para checkout hospedado, retorno e webhook, sempre com validacao server-side via `payment_check`.
- Supabase hospedara PostgreSQL.
- Supabase Auth autentica credenciais administrativas; a autorizacao real da PrintLab usa `ADMIN_SUPABASE_USER_ID`.
- Supabase Cron usa `pg_cron` via migration versionada e nao faz chamadas HTTP nem usa secrets.
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
- [Supabase Auth](../integrations/supabase-auth.md)
- [SuperFrete](../integrations/superfrete.md)
- [InfinitePay](../integrations/infinitepay.md)
