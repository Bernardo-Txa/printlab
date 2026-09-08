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

## Situacao atual

- Desenvolvimento inicial.
- Vercel Hobby pode ser usado durante desenvolvimento.
- Entrada Go compativel com zero-config da Vercel em `cmd/server/main.go`.
- Assets estaticos servidos via `embed.FS`, reduzindo dependencia de filesystem local no runtime da Vercel.
- Sem operacao comercial.
- Sem conexao com Supabase.
- Sem secrets reais.

## Assets estaticos

O CSS compilado em `web/static/css/app.css` e embutido no binario Go e servido em `/static/css/app.css`. Essa abordagem evita falhas em deploys onde o runtime nao encontra o diretorio `web/static/` no filesystem local.

Validacao local:

```sh
curl -I http://localhost:8080/static/css/app.css
```

## Antes da loja operar comercialmente

- Revisar plano de hospedagem.
- Revisar environment variables.
- Revisar secrets.
- Revisar banco.
- Revisar migrations.
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
