# ADR-0013 — Autenticacao administrativa com Supabase Auth e sessao propria

Status: Aceita
Data: 2026-09-12

## Contexto

A PrintLab precisa iniciar o painel administrativo interno com um unico administrador autorizado. O projeto ja usa Supabase para PostgreSQL e Storage, o backend e Go SSR com `net/http`, e nao queremos armazenar senha administrativa na aplicacao.

Tambem nao queremos carregar JWT ou refresh token do Supabase em todo fluxo administrativo da Fase 13.1. O objetivo e autenticar a credencial no provider e depois operar o Admin com sessao propria simples, revogavel e server-side.

## Decisao

Usar Supabase Auth para autenticar e-mail e senha:

```text
Supabase Auth valida credencial
  -> PrintLab compara user.id com ADMIN_SUPABASE_USER_ID
  -> PrintLab gera sessao opaca propria
  -> SHA-256(token) em public.admin_sessions
  -> cookie HttpOnly em /admin
```

A aplicacao usa `SUPABASE_PUBLISHABLE_KEY`, nao usa `SUPABASE_SECRET_KEY` nem service role nesta subfase.

`ADMIN_SUPABASE_USER_ID` e o unico criterio de autorizacao administrativa. E-mail nao autoriza acesso.

## Alternativas consideradas

- Senha em environment variable: rejeitada porque colocaria a PrintLab como armazenadora/verificadora direta da senha e dificultaria ciclo de vida da conta.
- HTTP Basic Auth: rejeitada por experiencia limitada, rotacao ruim e acoplamento a credencial estatica.
- JWT Supabase em todas as requests: rejeitada nesta fase para evitar carregar access/refresh token nos fluxos internos quando uma sessao server-side simples atende melhor.
- Login proprio com Argon2: rejeitado porque adicionaria gestao de senha sem necessidade atual.
- OAuth social: rejeitado por escopo maior e dependencia de provider adicional.
- Cloudflare Access: rejeitado por adicionar infraestrutura externa antes de necessidade concreta.

## Consequencias

- Senha fica sob responsabilidade do Supabase Auth.
- A PrintLab armazena somente hash de token de sessao administrativa.
- Sessao pode ser revogada removendo linha de `admin_sessions`.
- Login depende do Supabase Auth estar disponivel.
- Auth nao e chamado em toda request administrativa.
- `admin_sessions` precisa de limpeza operacional futura para sessoes expiradas.
- Fase 14 deve revisar brute force, CAPTCHA/WAF/rate limiting adicional e MFA.

## Referencias

- https://supabase.com/docs/guides/auth
- https://supabase.com/docs/guides/auth/passwords
- https://supabase.com/docs/guides/getting-started/api-keys
