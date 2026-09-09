# Supabase

Status: fundacao IMPLEMENTADA; schema de negocio PLANEJADO.

## Arquitetura planejada

```text
Go Backend
   |
   v
pgx
   |
   v
Supabase Transaction Pooler
   |
   v
Supabase PostgreSQL
```

## Decisoes

- Supabase hospedara o PostgreSQL.
- A aplicacao Go usa `pgx/v5` e `pgxpool`.
- O acesso principal ao banco sera server-side.
- O banco sera acessado por `DATABASE_URL`, que deve apontar para o Supabase Transaction Pooler.
- Credenciais virao de environment variables.
- Nenhuma credencial sera colocada em Git.
- Supabase Data API nao sera a interface primaria da aplicacao.
- Supabase Storage podera ser avaliado futuramente para imagens.
- Migrations versionadas serao aplicadas ao Supabase remoto de desenvolvimento pelo GitHub Actions quando houver alteracao em `supabase/migrations/**` ou `supabase/config.toml` na branch `main`.
- O workflow usa `supabase/setup-cli@v1` com Supabase CLI `2.117.0` fixado, executa `supabase link`, roda `supabase db push --dry-run` e so depois executa `supabase db push`.
- `pgx.QueryExecModeExec` e usado para evitar dependencia de prepared statement cache incompativel com transaction pooling.
- A aplicacao Go nao executa migrations no startup.

## Variaveis previstas

As variaveis abaixo existem como placeholders em `.env.example` e devem ser revisadas quando a configuracao oficial for feita:

- `DATABASE_URL`
- `DB_MAX_CONNS`
- `SUPABASE_URL`
- `SUPABASE_SERVICE_ROLE_KEY`

`DATABASE_URL` e a unica fonte de verdade da conexao PostgreSQL. `SUPABASE_SERVICE_ROLE_KEY` nao deve ser usada para conexao PostgreSQL.

## Supabase CLI local

A CLI esta instalada como devDependency npm:

```sh
npx supabase --version
```

Scripts locais:

```sh
npm run db:start
npm run db:stop
npm run db:status
npm run db:reset
npm run db:push:dry-run
npm run db:push
```

`db:push` e manual e nao faz parte do build da aplicacao.

## GitHub Actions Secrets

O responsavel pelo projeto deve configurar estes secrets diretamente no GitHub, sem registrar valores no repositorio:

- `SUPABASE_ACCESS_TOKEN`
- `SUPABASE_DB_PASSWORD`
- `SUPABASE_PROJECT_ID`

Esses secrets sao usados apenas pelo workflow `.github/workflows/supabase-migrations.yml`. O workflow nao deve imprimir valores de secrets nos logs.

## Ambiente remoto de migrations

O workflow aponta para o projeto Supabase de desenvolvimento da PrintLab identificado por `SUPABASE_PROJECT_ID`.

Antes da operacao comercial, deve existir separacao explicita entre ambientes de desenvolvimento/staging e producao. O banco de producao nao deve receber migrations automaticas sem uma politica de aprovacao propria.

## Estrutura local

```text
supabase/
├── .gitignore
├── config.toml
└── migrations/
```

`supabase/config.toml` nao contem secrets. Auth, Storage, Realtime, Edge Runtime, Analytics e seed ficam desabilitados nesta fase.

## Praticas proibidas

- Conectar ao Supabase sem plano aprovado.
- Versionar senha do banco ou service role key.
- Dar ao frontend acesso direto a tabelas sensiveis.
- Criar schema manualmente sem migration registrada.
- Usar Table Editor ou SQL Editor remoto como workflow normal para mudancas de schema.
- Executar `supabase db reset --linked` contra ambiente remoto.
- Aplicar seed automaticamente em deploy de migrations.
- Fazer `supabase login`, `supabase link` ou `supabase db push` manual como workflow normal de desenvolvimento.
