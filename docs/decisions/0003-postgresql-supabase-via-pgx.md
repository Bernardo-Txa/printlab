# ADR-0003 — PostgreSQL/Supabase com acesso server-side via pgx

Status: Aprovado
Data: 2026-09-08

## Contexto

A PrintLab precisara persistir dados relacionais, pedidos, pagamentos, enderecos e informacoes de producao. A aplicacao deve manter o backend como autoridade sobre dados sensiveis.

## Decisao

Usar PostgreSQL hospedado no Supabase, acessado pelo backend Go via `pgx`.

## Alternativas consideradas

- Supabase Data API como interface primaria.
- Banco nao relacional.
- Frontend acessando tabelas diretamente.

## Consequencias

- O backend controla acesso e regras de negocio.
- A aplicacao usa capacidades nativas do PostgreSQL.
- Credenciais de banco devem ficar em environment variables.
- `pgx` sera adicionado apenas quando a fase de banco comecar.

## Referencias

- [ARCHITECTURE.md](../../ARCHITECTURE.md)
- [docs/architecture/database.md](../architecture/database.md)
- [docs/integrations/supabase.md](../integrations/supabase.md)
