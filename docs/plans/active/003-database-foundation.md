# Fase 3 — Database Foundation

Status: EM ANDAMENTO.

## Objetivo

Criar a fundacao segura para evoluir o banco PostgreSQL no Supabase, com migrations versionadas e automacao controlada por GitHub Actions.

## Entregue nesta etapa

- Workflow `.github/workflows/supabase-migrations.yml`.
- Execucao automatica em push para `main` somente quando `supabase/migrations/**` ou `supabase/config.toml` mudarem.
- Execucao manual emergencial via `workflow_dispatch`.
- Supabase CLI fixado em `2.20.3` via `supabase/setup-cli@v1`.
- `supabase link --project-ref "$SUPABASE_PROJECT_ID"` antes de qualquer push de migration.
- `supabase db push --dry-run` antes de `supabase db push`.
- Documentacao da politica de migrations e dos GitHub Actions Secrets exigidos.

## Fora de escopo

- Criar schema de banco.
- Criar migrations funcionais.
- Conectar a aplicacao Go ao Supabase.
- Adicionar `pgx`.
- Implementar catalogo, carrinho, checkout, pedidos, pagamentos, frete, autenticacao ou painel admin.
- Configurar ambiente de producao.

## Secrets exigidos

Os valores devem ser configurados pelo responsavel do projeto diretamente em GitHub Actions Secrets:

- `SUPABASE_ACCESS_TOKEN`
- `SUPABASE_DB_PASSWORD`
- `SUPABASE_PROJECT_ID`

Nao incluir valores no repositorio.

## Politica de migrations

Mudancas de schema devem seguir:

```text
Codex/desenvolvedor
   |
   v
migration SQL
   |
   v
Git
   |
   v
main
   |
   v
GitHub Actions
   |
   v
Supabase de desenvolvimento
```

Table Editor e SQL Editor remoto nao devem ser usados como workflow normal para mudancas de schema.

## Seguranca

O workflow nao deve imprimir secrets, nao usa `--include-seed`, nao executa reset remoto e nao aplica migrations se o dry-run falhar.

O workflow atual aponta para desenvolvimento. Antes da operacao comercial, production devera ter separacao explicita de ambiente e politica de aprovacao propria.
