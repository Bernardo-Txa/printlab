# Fase 3.1 — Remote Environment Validation

Status: ATIVA.

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
- Vercel CLI: disponivel via `npx vercel@latest`, mas sem autenticacao no ambiente atual.
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

Resultados:

- `GET /`: HTTP 200.
- `GET /health`: HTTP 200 com body `ok`.
- `GET /static/css/app.css`: HTTP 200 com `Content-Type` de CSS.
- `GET /ready`: HTTP 503 com texto generico.

O resultado de `/ready` indica que a conexao runtime da Vercel com o PostgreSQL ainda nao esta validada neste ambiente.

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
| Supabase direto pelo ambiente local | PENDENTE | Secrets Supabase nao estao disponiveis como environment variables locais. |
| Vercel CLI | CONFIGURADO PARCIALMENTE | CLI executa via `npx`, mas esta deslogada. |
| Vercel region | VALIDADO | `vercel.json` contem somente `gru1`. |
| Vercel `DATABASE_URL` | PENDENTE | Nao foi possivel configurar ou listar env vars sem autenticacao Vercel; `/ready` remoto retorna 503. |
| Vercel `DB_MAX_CONNS` | PENDENTE | Nao foi possivel configurar ou listar env vars sem autenticacao Vercel. |
| Vercel `/ready` com banco | PENDENTE | Precisa retornar 200 depois de `DATABASE_URL` correta e redeploy. |

## Pendencias

- Autenticar a Vercel CLI ou disponibilizar autenticacao segura por ambiente, sem expor token.
- Confirmar que o projeto Vercel alvo e a PrintLab conectada ao repositorio `Bernardo-Txa/printlab`.
- Configurar `DATABASE_URL` na Vercel com a connection string real do Supabase Transaction Pooler.
- Configurar `DB_MAX_CONNS=4` nos ambientes Vercel relevantes.
- Fazer redeploy da aplicacao apos configurar env vars.
- Validar `GET /ready` remoto retornando HTTP 200.
- Se for necessario teste direto local com banco, disponibilizar `DATABASE_URL` ou `TEST_DATABASE_URL` apenas por mecanismo seguro de environment variable, sem registrar valor.

## Seguranca

- Nenhum valor secreto foi lido, impresso, documentado ou versionado.
- Nenhuma tabela foi criada.
- Nenhum `INSERT`, `UPDATE`, `DELETE`, `CREATE TABLE`, `db reset` ou seed foi executado.
- A resposta publica de `/ready` continua generica e nao expoe host, usuario, project ref, nome do banco ou erro interno do `pgx`.
