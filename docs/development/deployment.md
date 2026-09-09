# Deployment

Status: PLANEJADO.

## Ambientes

- `local`: maquina de desenvolvimento.
- `development`: ambiente remoto para validacao durante desenvolvimento.
- `production`: ambiente de operacao comercial, ainda nao ativo.

## Fluxo planejado

```text
Git
  |
  v
GitHub
  |
  v
Vercel
```

Banco planejado:

```text
Supabase PostgreSQL
```

Migrations Supabase em desenvolvimento:

```text
Git
  |
  v
GitHub Actions
  |
  v
Supabase CLI
  |
  v
Supabase de desenvolvimento
```

## Situacao atual

- Desenvolvimento inicial.
- Vercel Hobby pode ser usado durante desenvolvimento.
- Entrada Go compativel com zero-config da Vercel em `cmd/server/main.go`.
- Assets estaticos servidos via `embed.FS`, reduzindo dependencia de filesystem local no runtime da Vercel.
- Vercel configurada em `vercel.json` para executar na regiao `gru1`.
- Sem operacao comercial.
- Conexao PostgreSQL da aplicacao via `DATABASE_URL` quando configurada.
- Sem secrets reais.
- Workflow de CI/CD para migrations Supabase configurado em `.github/workflows/supabase-migrations.yml`.

`gru1` foi escolhida porque o projeto Supabase da PrintLab esta em South America (Sao Paulo). Isso reduz a latencia entre o runtime Go na Vercel e o PostgreSQL no Supabase.

## Validacao remota de desenvolvimento

Status da Fase 3.1: ATIVA.

Validacoes feitas em 2026-09-09:

- GitHub CLI autenticada sem leitura de token.
- Secrets `SUPABASE_ACCESS_TOKEN`, `SUPABASE_DB_PASSWORD` e `SUPABASE_PROJECT_ID` presentes por nome no repositorio.
- Workflow `Supabase Migrations` ativo.
- Run manual `34302139793` por `workflow_dispatch` passou, incluindo `supabase link`, `supabase db push --dry-run` e `supabase db push`.
- Nenhuma migration de negocio foi criada para essa validacao.
- Deploy publico `https://printlab-pied.vercel.app` respondeu HTTP 200 em `/`, `/health` e `/static/css/app.css`.
- `/ready` remoto respondeu HTTP 503 com texto generico.

A Vercel CLI esta disponivel via `npx vercel@latest`, mas o ambiente atual esta deslogado. Portanto, `DATABASE_URL` e `DB_MAX_CONNS` ainda nao foram configuradas ou validadas pelo agente na Vercel.

Pendencia para concluir a validacao runtime:

- autenticar Vercel de forma segura;
- confirmar o projeto PrintLab conectado a `Bernardo-Txa/printlab`;
- configurar `DATABASE_URL` como secret do Supabase Transaction Pooler;
- configurar `DB_MAX_CONNS=4`;
- redeployar;
- validar `/ready` remoto com HTTP 200.

## Assets estaticos

O CSS compilado em `web/static/css/app.css` e embutido no binario Go e servido em `/static/css/app.css`. Essa abordagem evita falhas em deploys onde o runtime nao encontra o diretorio `web/static/` no filesystem local.

A logo em `web/static/images/branding/logo-printlab-primary.png` tambem e embutida e deve ser validada no deploy pela rota `/static/images/branding/logo-printlab-primary.png`.

Validacao local:

```sh
curl -I http://localhost:8080/static/css/app.css
```

## Migrations Supabase via GitHub Actions

O workflow `.github/workflows/supabase-migrations.yml` executa em push para `main`, mas apenas quando arquivos de banco mudarem:

- `supabase/migrations/**`
- `supabase/config.toml`

Tambem existe `workflow_dispatch` para execucao manual emergencial pelo GitHub.

O responsavel pelo projeto deve configurar estes GitHub Actions Secrets:

- `SUPABASE_ACCESS_TOKEN`
- `SUPABASE_DB_PASSWORD`
- `SUPABASE_PROJECT_ID`

O job instala `supabase/setup-cli@v1` com Supabase CLI `2.117.0`, valida que os secrets existem, executa `supabase link --project-ref "$SUPABASE_PROJECT_ID"`, roda `supabase db push --dry-run` e somente depois aplica `supabase db push`.

O workflow nao usa `--include-seed`, nao executa reset remoto e nao deve imprimir valores de secrets nos logs.

Este workflow aponta para o projeto Supabase de desenvolvimento da PrintLab. Antes da operacao comercial sera necessario separar development/staging e production, com politica de aprovacao propria para producao.

## Runtime PostgreSQL

```text
Vercel Go em gru1
  |
  v
pgxpool
  |
  v
Supabase Transaction Pooler em South America (Sao Paulo)
  |
  v
PostgreSQL
```

Secrets de runtime no ambiente de hosting:

- `DATABASE_URL`
- `DB_MAX_CONNS`, opcional, default `4`

`DATABASE_URL` deve ser configurada como secret e nunca impressa em logs. Se estiver ausente, `GET /ready` retorna 503, mas `GET /` e `GET /health` continuam funcionando temporariamente nesta fase.

Na validacao remota da Fase 3.1, a URL publica retornou HTTP 503 em `/ready`; isso deve permanecer como pendencia ate a configuracao segura de `DATABASE_URL` e a verificacao de conectividade com o Supabase Transaction Pooler.

## Vercel

`vercel.json` contem apenas:

```json
{
  "$schema": "https://openapi.vercel.sh/vercel.json",
  "regions": ["gru1"]
}
```

Nao ha builds, rewrites, routes, outputDirectory, installCommand ou Docker customizado.

## Antes da loja operar comercialmente

- Revisar plano de hospedagem.
- Revisar environment variables.
- Revisar secrets.
- Revisar banco.
- Revisar migrations.
- Separar ambientes Supabase de desenvolvimento/staging e producao.
- Definir politica de aprovacao para migrations de producao.
- Revisar webhooks.
- Revisar dominio.
- Revisar observabilidade.
- Revisar backups.
- Revisar seguranca.

## Praticas proibidas

- Publicar ambiente de producao com credenciais expostas.
- Operar comercialmente sem revisar webhooks de pagamento.
- Alterar banco de producao manualmente sem registro.
- Fazer deploy de funcionalidade financeira sem testes aplicaveis.
- Executar migrations automaticas em producao sem politica de aprovacao.
