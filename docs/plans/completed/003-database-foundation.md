# Fase 3 — Database Foundation

Status: CONCLUIDA.

## Objetivo

Concluir a fundacao PostgreSQL/Supabase da PrintLab sem criar tabelas de negocio.

## Contexto

A aplicacao roda em Go com `net/http`, deploy Vercel e PostgreSQL hospedado no Supabase. O Supabase da PrintLab fica em South America (Sao Paulo), entao o runtime Vercel foi configurado para `gru1`.

O banco ainda nao e obrigatorio para a homepage. Temporariamente, sem `DATABASE_URL`, a aplicacao inicia e `GET /ready` retorna HTTP 503.

## Escopo

- Adicionar `pgx/v5` e `pgxpool`.
- Criar `internal/config`.
- Criar `internal/database`.
- Usar `DATABASE_URL` como unica fonte de verdade da conexao.
- Usar `DB_MAX_CONNS` com default `4`.
- Configurar `pgx.QueryExecModeExec` para Supabase Transaction Pooler.
- Adicionar `GET /ready`.
- Adicionar Supabase CLI local via npm.
- Criar `supabase/config.toml` e `supabase/migrations/`.
- Preservar e revisar workflow de migrations.
- Criar `vercel.json` minimo com `gru1`.
- Atualizar documentacao e ADR.

## Fora de escopo

- Criar schema de negocio.
- Criar migrations funcionais.
- Criar tabelas de produtos, variantes, carrinho, pedidos, clientes, pagamentos, frete ou admin.
- Implementar catalogo, carrinho, checkout, SuperFrete, InfinitePay, autenticacao ou admin.
- Executar `supabase login`, `supabase link`, `supabase db push` ou workflow remoto.
- Configurar ambiente de producao.

## Decisoes

- Runtime PostgreSQL via `pgxpool` contra Supabase Transaction Pooler.
- `DefaultQueryExecMode = pgx.QueryExecModeExec` para nao depender de prepared statement cache.
- `MaxConns = 4` por default e `MinConns = 0`.
- Migrations ficam em `supabase/migrations/` e nao rodam no startup da aplicacao Go.
- `GET /health` permanece liveness e nao consulta banco.
- `GET /ready` e readiness e consulta banco apenas quando configurado.
- `SUPABASE_SERVICE_ROLE_KEY` nao e usada para conexao PostgreSQL.

## Implementacao

- `internal/config` valida `DATABASE_URL` e `DB_MAX_CONNS`.
- `internal/database` cria e fecha `pgxpool.Pool` e expoe `Ping`.
- `cmd/server` segue o fluxo `config.Load() -> database.New(...) -> newHandler(...)`.
- Workflow `.github/workflows/supabase-migrations.yml` usa Supabase CLI `2.117.0`, roda dry-run antes de aplicar migrations e nao usa seed/reset.
- `supabase/config.toml` nao contem secrets e mantem Auth, Storage, Realtime, Edge Runtime, Analytics e seed desabilitados.
- `AGENTS.md` e `docs/development/coding-standards.md` registram a politica permanente de Git: implementar, validar, commit e push automaticos.

## Testes

- Config: `DB_MAX_CONNS` ausente, valido e invalido; `DATABASE_URL` ausente e invalida segura.
- Database: banco ausente como nao configurado; `pgxpool` com `QueryExecModeExec`; teste opcional com `TEST_DATABASE_URL`.
- HTTP: `GET /`, `GET /health`, `GET /ready` sem `DATABASE_URL`, static CSS e assets inexistentes.

## Definition of Done

- `templ generate` executado.
- `npm run css:build` executado.
- `gofmt -w .` executado.
- `go mod tidy` executado.
- `go test ./...` passa.
- `go vet ./...` passa.
- `go build ./...` passa.
- `npx supabase --version` passa.
- `GET /`, `GET /health`, `GET /ready` e `/static/css/app.css` validados localmente.
- Nenhuma credencial real adicionada.
- Nenhuma tabela de negocio criada.
- Fonte oficial de migrations e `supabase/migrations/`.
- Politica permanente de Git atualizada antes do commit final da fase.
