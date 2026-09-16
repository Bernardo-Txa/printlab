# Fase 14 — Seguranca

Status: em andamento.

## Fase 14.1 — Hardening base de seguranca

Status: concluida; validada em producao pelo responsavel em 2026-09-16.

Validacao real confirmou headers, CSP, canonical host, redirect 303 para InfinitePay, limites de body, Origin/Referer Admin, cron, timeouts e SITE_URL.

## Objetivo

Reduzir superficie de ataque da aplicacao antes de ampliar uso real, sem alterar fluxos comerciais, financeiros, Admin, Storage ou providers ja validados.

## Escopo implementado

- Headers globais de seguranca para todas as rotas HTTP:
  - `X-Content-Type-Options: nosniff`;
  - `X-Frame-Options: DENY`;
  - `Permissions-Policy: camera=(), microphone=(), geolocation=()`;
  - `Referrer-Policy: strict-origin-when-cross-origin`;
  - `Content-Security-Policy` restritiva.
- Preservacao de headers especificos de paginas sensiveis:
  - Admin continua com `Cache-Control: private, no-store`, `X-Robots-Tag: noindex, nofollow, noarchive` e `Referrer-Policy: same-origin`;
  - Acompanhamento publico continua com `Cache-Control: private, no-store`, `X-Robots-Tag: noindex, nofollow, noarchive` e `Referrer-Policy: no-referrer`.
- CSP sem `unsafe-eval`, com `style-src 'self' 'unsafe-inline'`, origem exata do Supabase derivada somente de `SUPABASE_URL` e `form-action` limitado a `'self'` mais origins exatos de checkout InfinitePay validados pelo backend.
- Teto global de corpo de request em 1 MiB, com `413 Request Entity Too Large` para `Content-Length` conhecido acima do limite e `http.MaxBytesReader` aplicado antes do roteador.
- Preservacao dos limites menores existentes:
  - Admin forms: 256 KiB;
  - Admin image JSON: 64 KiB;
  - InfinitePay webhook JSON: 64 KiB.
- Servidor HTTP inicializado com `http.Server` e timeouts explicitos:
  - `ReadHeaderTimeout`: 5s;
  - `ReadTimeout`: 15s;
  - `WriteTimeout`: 30s;
  - `IdleTimeout`: 60s.
- Validacao central de `SITE_URL`:
  - opcional;
  - URL absoluta;
  - esquemas aceitos: `http` e `https`;
  - host obrigatorio;
  - userinfo e fragment rejeitados;
  - em `APP_ENV=production`/`prod` ou `VERCEL_ENV=production`/`prod`, exige `https`.
- Canonicalizacao de `GET` e `HEAD` para a origem configurada em `SITE_URL`, preservando path/query e sem usar `Host` recebido para construir destino.
- Comparacao de origem configurada por `SITE_URL` passa a considerar `scheme://host`.
- Migration Supabase Cron para limpar diariamente dados transientes expirados:
  - `public.admin_sessions where expires_at <= now()`;
  - `public.carts where expires_at <= now()`.

## Banco e limpeza

A limpeza de carrinhos expirados depende dos FKs ja existentes:

- `cart_items.cart_id`: `ON DELETE CASCADE`;
- `cart_customer_details.cart_id`: `ON DELETE CASCADE`;
- `cart_shipping_addresses.cart_id`: `ON DELETE CASCADE`;
- `cart_shipping_selections.cart_id`: `ON DELETE CASCADE`;
- `orders.source_cart_id`: `ON DELETE SET NULL`.

Pedidos, snapshots, pagamentos, fulfillment e eventos administrativos nao sao removidos pelo job da 14.1.

## Auditoria de endpoints sensiveis

Endpoints sensiveis revisados nesta fase:

- `POST /admin/login`;
- `GET /api/cep/{cep}`;
- `GET /checkout/frete`;
- `POST /checkout/frete`;
- `POST /webhooks/infinitepay`;
- operacoes administrativas ou externas caras, como Storage, SuperFrete, ViaCEP, InfinitePay e Supabase Auth.

A 14.1 nao implementa rate limiter em Go. A estrategia operacional recomendada e Vercel Firewall/WAF com rollout gradual:

1. criar regras por path/metodo em modo observacao/log;
2. medir volume real, falsos positivos e padroes de erro;
3. definir limites por endpoint conforme custo e criticidade;
4. ativar acao de rate limit/challenge/deny quando houver base operacional;
5. validar em producao sem bloquear checkout legitimo, webhook financeiro ou login administrativo real.

## Supabase Auth

O fluxo atual `Browser -> Go -> Supabase Auth` pode concentrar tentativas no IP server-side da aplicacao perante os limites de Auth do Supabase. A mitigacao oficial de IP forwarding exige `Sb-Forwarded-For`, uma secret API key iniciada por `sb_secret` e habilitacao explicita do recurso no projeto. A 14.1 documenta o risco, mas nao adiciona esse header nem troca a credencial do fluxo de Auth.

## Auditorias realizadas

- Sessoes Admin mantem `crypto/rand`, 32 bytes, `SHA-256(token)`, cookie HttpOnly, `SameSite=Strict`, `Path=/admin`, Secure em producao, TTL de 8 horas, logout e expiracao.
- Acompanhamento publico continua usando UUID aleatorio em `orders.public_tracking_id`, sem `orders.id` nem `order_number`, com view minimizada sem PII.
- Webhook InfinitePay continua usando JSON limitado a 64 KiB, `Content-Type` JSON, segundo decode exigindo EOF, `payment_check` server-side, idempotencia e comparacao de valor com `orders.total_cents`.
- Validacao real confirmou que o Chrome aplica `form-action` ao redirect `303` da submissao de pagamento; a CSP passou a permitir somente os origins exatos `https://checkout.infinitepay.io` e `https://checkout.infinitepay.com.br`, sem alterar o fluxo server-side InfinitePay nem liberar `api.checkout.infinitepay.io`.
- Storage Admin continua usando `SUPABASE_SECRET_KEY` somente server-side, signed upload temporario, path gerado pelo backend, MIME/tamanho validados, metadata verificada e delete limitado a path gerenciado.
- Migrations atuais habilitam RLS nas tabelas de negocio/operacao criadas e nao criam policies publicas.
- Logs existentes usam categorias seguras e identificadores operacionais minimizados; nao devem registrar CPF, e-mail completo, telefone, endereco, tokens, secrets, signed upload URL, checkout URL completa ou `public_tracking_id`.
- `DATABASE_URL` real nao foi lida nem impressa; verificacao de role dedicada/minima fica como requisito operacional futuro para Fase 17.

## MFA

MFA nao foi implementado na 14.1. Uma Fase 14.2 pode avaliar TOTP/AAL2 se o risco operacional justificar:

- manter o primeiro fator e o desafio MFA sem persistir access/refresh tokens desnecessariamente;
- usar challenge/verify antes de criar a sessao propria da PrintLab;
- garantir recuperacao segura para evitar lockout do unico administrador;
- registrar testes de fluxo incompleto, credenciais invalidas, fator ausente e sessao expirada.

## HSTS

`Strict-Transport-Security` e especialmente `preload` nao foram habilitados nesta fase. A decisao fica para preparacao final de producao, apos dominio definitivo, HTTPS validado em todos os subdominios relevantes e confirmacao de que nao ha dependencia legitima de HTTP.

## Fora do escopo

- MFA.
- RBAC ou multiplos papeis administrativos.
- Rate limiter em memoria ou distribuido dentro do Go.
- CAPTCHA.
- Alteracoes em InfinitePay, SuperFrete, ViaCEP, Supabase Storage ou Supabase Auth alem da allowlist CSP de checkout e da documentacao de risco.
- Mudancas em regras comerciais, pedidos, checkout, catalogo ou precos.
- Refatoracao ampla de sessao Admin, acompanhamento publico, webhook ou Storage.

## Definition of Done

- Codigo de hardening implementado e testado.
- Migration Supabase Cron append-only criada e coberta por teste estrutural.
- Documentacao de seguranca, backend, database, migrations, roadmap e CHANGELOG atualizada.
- Validacoes locais executadas antes do commit.
- Validacao real de deploy e headers remotos concluida pelo responsavel.

## Validacoes executadas

- `go run github.com/a-h/templ/cmd/templ@v0.3.1020 generate`;
- `npm run css:build`;
- `gofmt -w .`;
- `go mod tidy`;
- `go test ./...`;
- `go vet ./...`;
- `go build ./...`;
- `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`;
- `npm audit`;
- `npm run db:start`;
- `npx supabase migration up --local`;
- consulta local a `cron.job` via container PostgreSQL;
- `npm run db:stop`.

O dry-run remoto `npm run db:push:dry-run` nao executou localmente porque o checkout nao esta linkado ao projeto Supabase (`LegacyProjectNotLinkedError`). O workflow remoto existente deve executar dry-run e push da migration apos o push para `main`.
