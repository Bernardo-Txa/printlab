# Fase 3.1 — Remote Environment Validation

Status: CONCLUIDA.

## Objetivo

Ativar e validar o ambiente remoto de desenvolvimento da PrintLab sem criar schema de negocio, sem migration ficticia e sem avancar para a Fase 4.

Fluxos avaliados:

```text
Codex -> GitHub -> GitHub Actions -> Supabase DEV
Vercel -> Go -> Supabase Transaction Pooler -> PostgreSQL
```

## Fora de escopo

- Criar tabelas de negocio.
- Criar migration de teste.
- Implementar catalogo, carrinho, checkout, pedidos, pagamentos, frete, admin ou autenticacao.
- Alterar dominio, plano ou configuracoes comerciais da Vercel.
- Versionar ou exibir secrets.

## Preflight

- Branch local: `main`.
- Estado inicial do Git: clean.
- Remote Git: `origin` apontando para `https://github.com/Bernardo-Txa/printlab.git`.
- GitHub CLI: disponivel e autenticado.
- Supabase CLI: `2.117.0`.
- `vercel.json`: preservado com configuracao minima de `gru1`.

## Validacoes executadas

### GitHub

- `gh auth status` confirmou autenticacao GitHub sem leitura de valores de token.
- `gh secret list` confirmou a presenca dos secrets por nome:
  - `SUPABASE_ACCESS_TOKEN`
  - `SUPABASE_DB_PASSWORD`
  - `SUPABASE_PROJECT_ID`
- O workflow `Supabase Migrations` esta ativo.

### GitHub Actions -> Supabase DEV

- O workflow `.github/workflows/supabase-migrations.yml` foi revisado.
- Ele usa os tres secrets exigidos, executa `supabase link`, executa `supabase db push --dry-run` antes de `supabase db push`, nao executa reset, nao executa seed e nao imprime secrets.
- Execucao manual por `workflow_dispatch`: `34302139793`.
- Resultado: sucesso.
- Etapas validadas: checkout, setup da Supabase CLI, checagem de secrets, link do projeto, dry-run e aplicacao de migrations.
- Nenhuma migration de negocio foi criada nesta tarefa.

### Vercel publico

URL validada:

```text
https://printlab-pied.vercel.app
```

Resultados finais:

- `GET /`: HTTP 200.
- `GET /health`: HTTP 200 com body `ok`.
- `GET /static/css/app.css`: HTTP 200 com `Content-Type` de CSS.
- `GET /ready`: HTTP 200 com body `ok`.

O responsavel do projeto confirmou manualmente que `DATABASE_URL` do Supabase Transaction Pooler e `DB_MAX_CONNS` estao configuradas na Vercel. O retorno HTTP 200 de `/ready` valida a conectividade Vercel -> Go -> `pgxpool` -> Supabase Transaction Pooler -> PostgreSQL.

### Local sem banco

Sem `DATABASE_URL`, o comportamento esperado permanece:

- `GET /`: HTTP 200.
- `GET /health`: HTTP 200.
- `GET /ready`: HTTP 503.

## Status por integracao

| Item | Status | Observacao |
| --- | --- | --- |
| Git remote | VALIDADO | `origin` aponta para o repositorio esperado. |
| GitHub auth | VALIDADO | Autenticacao confirmada sem expor token. |
| GitHub Secrets Supabase | VALIDADO | Presenca confirmada por nome. |
| GitHub Actions -> Supabase DEV | VALIDADO | Run `34302139793` passou. |
| Supabase CLI local | VALIDADO | Versao `2.117.0`. |
| Vercel deployment | VALIDADO | URL publica responde. |
| Vercel region | VALIDADO | `vercel.json` contem somente `gru1`. |
| Vercel `DATABASE_URL` | VALIDADO | Confirmada pelo responsavel, sem registrar valor. |
| Vercel `DB_MAX_CONNS` | VALIDADO | Confirmada pelo responsavel. |
| Vercel `/health` | VALIDADO | HTTP 200 com body `ok`. |
| Vercel `/ready` com banco | VALIDADO | HTTP 200 com body `ok`. |
| Conectividade PostgreSQL runtime | VALIDADO | Validada por `/ready` remoto. |

## Pendencias

- Nenhuma pendencia bloqueante da Fase 3.1.
- A Fase 4 continua planejada e nao foi iniciada nesta tarefa.

## Seguranca

- Nenhum valor secreto foi lido, impresso, documentado ou versionado.
- Nenhuma tabela foi criada.
- Nenhuma migration de negocio foi criada.
- Nenhum `INSERT`, `UPDATE`, `DELETE`, `CREATE TABLE`, `db reset` ou seed foi executado.
- A resposta publica de `/ready` continua generica e nao expoe host, usuario, project ref, nome do banco ou erro interno do `pgx`.
