# Supabase

Status: PLANEJADO. Nenhuma conexao foi configurada nesta fase.

## Arquitetura planejada

```text
Go Backend
   |
   v
pgx
   |
   v
Supabase PostgreSQL
```

## Decisoes

- Supabase hospedara o PostgreSQL.
- A aplicacao Go usara `pgx`.
- O acesso principal ao banco sera server-side.
- O banco sera acessado via conexao PostgreSQL apropriada para ambiente hospedado.
- Credenciais virao de environment variables.
- Nenhuma credencial sera colocada em Git.
- Supabase Data API nao sera a interface primaria da aplicacao.
- Supabase Storage podera ser avaliado futuramente para imagens.

## Variaveis previstas

As variaveis abaixo existem como placeholders em `.env.example` e devem ser revisadas quando a configuracao oficial for feita:

- `DATABASE_URL`
- `SUPABASE_URL`
- `SUPABASE_SERVICE_ROLE_KEY`

## Praticas proibidas

- Conectar ao Supabase sem plano aprovado.
- Versionar senha do banco ou service role key.
- Dar ao frontend acesso direto a tabelas sensiveis.
- Criar schema manualmente sem migration registrada.
