# Fase 13.1 — Fundacao de Autenticacao Administrativa

Status: CONCLUIDA em 2026-09-12.

## Objetivo

Criar a fundacao administrativa segura para `/admin` sem implementar CRUD, alteracao de pedidos, mutations de fulfillment, upload de imagens ou service role.

## Entregas

- Login administrativo em `GET /admin/login` e `POST /admin/login`.
- Autenticacao por Supabase Auth usando `SUPABASE_URL` e `SUPABASE_PUBLISHABLE_KEY`.
- Autorizacao por `ADMIN_SUPABASE_USER_ID`, comparando com `user.id`.
- Sessao propria da PrintLab com token opaco, hash SHA-256 no banco e TTL de 8 horas.
- Cookie `printlab_admin_session` com `HttpOnly`, `SameSite=Strict`, `Path=/admin`, host-only e `Secure` em producao.
- Logout em `POST /admin/logout`.
- Guard administrativo para `/admin` e `/admin/*`.
- Dashboard inicial somente leitura com contagens agregadas de pedidos.
- Headers privados/noindex/no-referrer em paginas administrativas.
- Protecao `Origin`/`Referer` em POSTs administrativos.
- Migration `create_admin_sessions`.
- Documentacao e ADR da decisao de autenticacao.

## Banco

Migration:

- `supabase/migrations/20260912110000_create_admin_sessions.sql`

Tabela:

- `public.admin_sessions`

Sem FK para `auth.users`, sem policies publicas e sem dados seed.

## Definition of Done

- [x] Loja publica continua iniciando sem configuracao Admin.
- [x] `/admin/login` mostra indisponibilidade segura quando config Admin ou banco esta ausente.
- [x] Senha nao e persistida nem logada.
- [x] Access token e refresh token Supabase nao sao persistidos.
- [x] Usuario Supabase valido mas nao autorizado nao cria sessao.
- [x] Sessao expirada e tratada como nao autenticada.
- [x] Dashboard nao carrega PII nem identificadores de pagamento.
- [x] Testes automatizados de auth client, token, cookie, sessao, middleware, login, logout e dashboard.
- [x] Documentacao atualizada.

## Fora do escopo

- Fase 13.2: pedidos, producao, envio e auditoria.
- Fase 13.3: catalogo, variantes, materiais, cores e caixas.
- Fase 13.4: imagens e Supabase Storage.
- Fase 14: revisao de abuso/brute force, CAPTCHA/WAF, MFA e hardening adicional.
