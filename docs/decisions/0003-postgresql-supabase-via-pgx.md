# ADR-0003 — PostgreSQL/Supabase com pgx e Transaction Pooler

Status: Aprovado
Data: 2026-09-08

## Contexto

A PrintLab precisara persistir dados relacionais, pedidos, pagamentos, enderecos e informacoes de producao. A aplicacao deve manter o backend Go como autoridade sobre dados sensiveis.

O runtime planejado combina Go na Vercel, compute escalavel/serverless e PostgreSQL hospedado no Supabase em South America (Sao Paulo). Esse desenho precisa limitar conexoes por instancia e evitar dependencias incompativeis com transaction pooling.

## Decisao

Usar PostgreSQL hospedado no Supabase, acessado pelo backend Go via `pgx/v5` e `pgxpool`.

O runtime de banco segue:

```text
Vercel Go
   |
   v
pgxpool
   |
   v
Supabase Transaction Pooler
   |
   v
PostgreSQL
```

`DATABASE_URL` e a unica fonte de verdade da conexao PostgreSQL em runtime. A aplicacao nao monta connection string manualmente.

`pgxpool` deve ser configurado com:

- `MaxConns = 4` por default, ajustavel via `DB_MAX_CONNS`;
- `MinConns = 0`;
- `DefaultQueryExecMode = pgx.QueryExecModeExec`.

Migrations sao gerenciadas pela Supabase CLI em `supabase/migrations/` e aplicadas por GitHub Actions. A aplicacao Go nao executa migrations no startup.

## Alternativas consideradas

- Supabase Data API como interface primaria.
- ORM.
- `database/sql`.
- Conexao direta ao PostgreSQL sem pooler.
- Supabase Session Pooler.
- Banco nao relacional.
- Frontend acessando tabelas diretamente.

## Consequencias

- O backend controla acesso e regras de negocio.
- A aplicacao usa capacidades nativas do PostgreSQL.
- Credenciais de banco devem ficar em environment variables.
- Queries permanecem explicitas e revisaveis.
- A abstracao de banco e pequena; nao ha Repository generico nesta fase.
- O uso de `pgx.QueryExecModeExec` evita prepared statement cache nesse modo para compatibilidade com Transaction Pooler.
- Migrations sao independentes do runtime web.
- Producao exigira separacao de ambiente e politica propria de aprovacao antes de receber migrations automaticas.

## Referencias

- [ARCHITECTURE.md](../../ARCHITECTURE.md)
- [docs/architecture/database.md](../architecture/database.md)
- [docs/integrations/supabase.md](../integrations/supabase.md)
