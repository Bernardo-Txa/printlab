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
- Sem operacao comercial.
- Sem conexao da aplicacao Go com Supabase.
- Sem secrets reais.
- Workflow de CI/CD para migrations Supabase configurado em `.github/workflows/supabase-migrations.yml`.

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

O job instala `supabase/setup-cli@v1` com Supabase CLI `2.20.3`, valida que os secrets existem, executa `supabase link --project-ref "$SUPABASE_PROJECT_ID"`, roda `supabase db push --dry-run` e somente depois aplica `supabase db push`.

O workflow nao usa `--include-seed`, nao executa reset remoto e nao deve imprimir valores de secrets nos logs.

Este workflow aponta para o projeto Supabase de desenvolvimento da PrintLab. Antes da operacao comercial sera necessario separar development/staging e production, com politica de aprovacao propria para producao.

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
