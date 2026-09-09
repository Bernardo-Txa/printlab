# Supabase

Status: PLANEJADO para conexao da aplicacao; workflow de CI/CD para migrations configurado na Fase 3.

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
- Migrations versionadas serao aplicadas ao Supabase remoto de desenvolvimento pelo GitHub Actions quando houver alteracao em `supabase/migrations/**` ou `supabase/config.toml` na branch `main`.
- O workflow usa `supabase/setup-cli@v1` com Supabase CLI `2.20.3` fixado, executa `supabase link`, roda `supabase db push --dry-run` e so depois executa `supabase db push`.

## Variaveis previstas

As variaveis abaixo existem como placeholders em `.env.example` e devem ser revisadas quando a configuracao oficial for feita:

- `DATABASE_URL`
- `SUPABASE_URL`
- `SUPABASE_SERVICE_ROLE_KEY`

## GitHub Actions Secrets

O responsavel pelo projeto deve configurar estes secrets diretamente no GitHub, sem registrar valores no repositorio:

- `SUPABASE_ACCESS_TOKEN`
- `SUPABASE_DB_PASSWORD`
- `SUPABASE_PROJECT_ID`

Esses secrets sao usados apenas pelo workflow `.github/workflows/supabase-migrations.yml`. O workflow nao deve imprimir valores de secrets nos logs.

## Ambiente remoto de migrations

O workflow aponta para o projeto Supabase de desenvolvimento da PrintLab identificado por `SUPABASE_PROJECT_ID`.

Antes da operacao comercial, deve existir separacao explicita entre ambientes de desenvolvimento/staging e producao. O banco de producao nao deve receber migrations automaticas sem uma politica de aprovacao propria.

## Praticas proibidas

- Conectar ao Supabase sem plano aprovado.
- Versionar senha do banco ou service role key.
- Dar ao frontend acesso direto a tabelas sensiveis.
- Criar schema manualmente sem migration registrada.
- Usar Table Editor ou SQL Editor remoto como workflow normal para mudancas de schema.
- Executar `supabase db reset --linked` contra ambiente remoto.
- Aplicar seed automaticamente em deploy de migrations.
