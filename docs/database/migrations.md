# Migrations

Status: PLANEJADO para schema; workflow de CI/CD para aplicar migrations Supabase configurado na Fase 3.

Ainda nao ha migrations funcionais nesta fase. Quando aprovadas, migrations Supabase devem ficar em `supabase/migrations/` e ser revisadas antes de chegar a `main`.

## Workflow normal

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

O workflow `.github/workflows/supabase-migrations.yml` roda automaticamente em push para `main` somente quando houver alteracao em `supabase/migrations/**` ou `supabase/config.toml`. Ele tambem aceita `workflow_dispatch` para execucao manual emergencial pelo GitHub.

A sequencia remota e:

```sh
supabase link --project-ref "$SUPABASE_PROJECT_ID"
supabase db push --dry-run
supabase db push
```

Se o dry-run falhar, o GitHub Actions interrompe o job antes de aplicar migrations.

## Regras

- Toda mudanca de schema deve possuir migration versionada.
- Migrations sao imutaveis depois de aplicadas em ambientes compartilhados.
- Rollback deve ser considerado antes da aplicacao.
- Migrations devem ser revisadas.
- Mudancas destrutivas precisam de cuidado adicional.
- Alteracoes de banco nao devem ser feitas manualmente em producao sem registro.
- Table Editor e SQL Editor remoto nao devem ser usados como workflow normal para mudancas de schema.

## Praticas recomendadas

- Uma migration deve representar uma mudanca coesa.
- O nome deve deixar clara a intencao da mudanca.
- Dados sensiveis nao devem aparecer em migrations.
- Migrations que alteram dados devem ser especialmente revisadas.
- Criacao de indices deve considerar impacto em tabelas grandes.
- Migrations destinadas ao CI devem estar em `supabase/migrations/`.
- O workflow de CI nao usa `--include-seed`.
- O workflow de CI nao executa reset remoto.

## Antes de aprovar uma migration

- O schema relacionado foi documentado.
- O impacto em codigo e dados foi entendido.
- Testes aplicaveis foram planejados ou executados.
- O caminho de rollback foi discutido quando necessario.

## Secrets do CI

O responsavel pelo projeto deve configurar no GitHub Actions Secrets, sem incluir valores no repositorio:

- `SUPABASE_ACCESS_TOKEN`
- `SUPABASE_DB_PASSWORD`
- `SUPABASE_PROJECT_ID`

O workflow atual aponta para o Supabase de desenvolvimento. Producao exigira politica separada de aprovacao antes da operacao comercial.
