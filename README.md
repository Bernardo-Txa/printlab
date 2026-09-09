# PrintLab

A PrintLab e uma empresa brasileira de impressao 3D. Este repositorio sera a base do sistema proprio da empresa, inicialmente voltado para e-commerce e, futuramente, para funcionalidades internas de gestao da operacao.

## Objetivo do projeto

Construir uma aplicacao web simples, segura e manutenivel para venda de produtos impressos em 3D, com evolucao planejada para catalogo, variantes, carrinho, checkout, frete, pagamentos, pedidos e painel administrativo.

## Status atual

IMPLEMENTADO:

- Fundacao inicial do repositorio.
- Documentacao de arquitetura, produto, banco, desenvolvimento, integracoes e roadmap.
- Aplicacao Go em `cmd/server` usando `net/http`.
- Rota `GET /health` retornando HTTP 200.
- Homepage server-side em `GET /` renderizada com `templ`.
- Catalogo SSR em `GET /produtos`.
- Pagina publica de produto em `GET /produtos/{slug}`.
- Tailwind CSS via CLI npm, sem CDN e sem bundler JavaScript.
- Assets estaticos servidos em `/static/` via `embed.FS`, a partir de `web/static/`.
- Logo oficial inicial integrada ao header e ao hero da homepage.
- Design tokens refinados com base na identidade visual da marca.
- Fase 2.1 de Brand Experience aplicada na homepage.
- Workflow de GitHub Actions para aplicar migrations Supabase de desenvolvimento com dry-run previo.
- Fundacao PostgreSQL/Supabase da Fase 3.
- Configuracao centralizada em `internal/config`.
- Pool PostgreSQL em `internal/database` com `pgx/v5` e `pgxpool`.
- Rota `GET /ready` para readiness do banco.
- Supabase CLI local via npm e estrutura `supabase/`.
- Vercel configurada para a regiao `gru1`.
- Primeiro schema de negocio com `categories` e `products`.

PLANEJADO:

- HTMX quando houver interacao real que justifique sua presenca.
- Produtos com variantes, cores e materiais.
- Carrinho e checkout sem obrigatoriedade de conta.
- Enderecos, frete e integracao com SuperFrete.
- Pedidos, pagamentos e integracao com InfinitePay.
- Webhooks, acompanhamento de pedido e painel administrativo.
- Informacoes de producao 3D, custos estimados, peso de filamento e tempo de impressao.

Este projeto ainda esta em desenvolvimento e nao deve ser usado em operacao comercial.

## Stack

Backend:

- Go.
- `net/http` da biblioteca padrao.
- `pgx/v5` com `pgxpool` para PostgreSQL.
- Dependencias externas somente quando houver justificativa real.

Frontend planejado:

- HTMX.
- Minimo possivel de JavaScript.

Frontend implementado:

- Renderizacao server-side.
- `templ` v0.3.1020.
- Tailwind CSS v4.3.3 via Tailwind CLI.
- Design tokens iniciais em `web/assets/css/app.css`.
- CSS compilado em `web/static/css/app.css`.
- Assets estaticos embutidos no binario Go para compatibilidade com deploy na Vercel.
- Logo de marca em `web/static/images/branding/logo-printlab-primary.png`.
- Linguagem visual com blocos coloridos, grid tecnico, camadas de impressao e elementos inspirados em laboratorio.
- Catalogo publico e detalhe de produto renderizados no servidor, sem JavaScript obrigatorio.
- Card de produto com placeholder visual de marca enquanto imagens reais nao existem.

Banco planejado:

- PostgreSQL hospedado no Supabase.
- Schema de negocio de produtos, carrinho, pedidos, pagamentos e entregas.

Banco implementado:

- Acesso server-side pelo backend Go usando `pgx/v5`.
- Pool de conexoes com `pgxpool`.
- `DATABASE_URL` como unica fonte de verdade da conexao PostgreSQL em runtime.
- `DB_MAX_CONNS` com default `4`.
- `DefaultQueryExecMode` configurado como `pgx.QueryExecModeExec` para compatibilidade com Supabase Transaction Pooler.
- `GET /ready` retorna 503 enquanto `DATABASE_URL` estiver ausente ou o banco estiver indisponivel.
- A homepage e `GET /health` continuam funcionando sem `DATABASE_URL` nesta fase.
- `categories` e `products` implementam o catalogo basico.
- Slugs sao unicos e usados em URLs publicas.
- `products.price_cents` armazena o preco-base em centavos.
- `products.is_active` controla exibicao publica.
- `products.is_featured` participa da ordenacao inicial.

Infraestrutura planejada:

- Vercel durante desenvolvimento.
- Vercel Pro antes da operacao comercial.
- Supabase para PostgreSQL.
- Supabase Storage podera ser avaliado futuramente para imagens.

Infraestrutura implementada para desenvolvimento:

- GitHub Actions em `.github/workflows/supabase-migrations.yml` para migrations Supabase.
- Execucao automatica apenas em mudancas de `supabase/migrations/**` ou `supabase/config.toml` na branch `main`.
- Supabase CLI fixado em `2.117.0`, com `supabase db push --dry-run` antes de `supabase db push`.
- `vercel.json` minimo com `regions: ["gru1"]`.

## Arquitetura resumida

```text
Browser
   |
   v
Go Backend
   |
   v
pgx
   |
   v
Supabase PostgreSQL
```

O frontend nao deve acessar diretamente tabelas sensiveis. O backend sera a autoridade sobre precos, frete, totais, pedidos e pagamentos.

O module path Go esta definido como `github.com/Bernardo-Txa/printlab`.

## Requisitos locais

- Go 1.26.0 ou versao compativel.
- Node.js e npm para tooling frontend.
- CLI do `templ` v0.3.1020.
- Nenhuma conta externa e necessaria para executar a aplicacao local atual.

Para o workflow remoto de migrations Supabase, o responsavel pelo projeto deve configurar estes GitHub Actions Secrets, sem incluir valores no repositorio:

- `SUPABASE_ACCESS_TOKEN`
- `SUPABASE_DB_PASSWORD`
- `SUPABASE_PROJECT_ID`

Configuracao local ou de hosting para runtime:

- `DATABASE_URL`: secret PostgreSQL. Deve apontar para o Supabase Transaction Pooler.
- `DB_MAX_CONNS`: opcional, default `4`.

`SUPABASE_SERVICE_ROLE_KEY` nao e usada para conexao PostgreSQL da aplicacao.

Instalacao local do tooling:

```sh
npm install
go install github.com/a-h/templ/cmd/templ@v0.3.1020
```

Garanta que o diretorio de binarios do Go, normalmente `$(go env GOPATH)/bin`, esteja no `PATH`.

## Como gerar frontend

Gerar templates Go a partir dos arquivos `.templ`:

```sh
templ generate
```

Compilar CSS de producao:

```sh
npm run css:build
```

Modo watch do CSS:

```sh
npm run css:watch
```

## Como executar

Depois de gerar templates e CSS:

```sh
go run ./cmd/server
```

Por padrao, o servidor usa a porta `8080`. Para mudar:

```sh
PORT=3000 go run ./cmd/server
```

Health check:

```sh
curl http://localhost:8080/health
```

Validar CSS servido pela aplicacao:

```sh
curl -I http://localhost:8080/static/css/app.css
```

Validar logo servida pela aplicacao:

```sh
curl -I http://localhost:8080/static/images/branding/logo-printlab-primary.png
```

Readiness do banco:

```sh
curl -i http://localhost:8080/ready
```

Sem `DATABASE_URL`, a resposta esperada nesta fase e HTTP 503. Com `DATABASE_URL` configurada e banco acessivel, a resposta esperada e HTTP 200 com body `ok`.

Catalogo:

```sh
curl -i http://localhost:8080/produtos
```

Com banco configurado e migrations aplicadas, a resposta esperada e HTTP 200. Com catalogo vazio, a pagina mostra um empty state honesto. Sem banco ou sem schema aplicado, a rota retorna indisponibilidade generica.

## Supabase local

A CLI do Supabase esta instalada como devDependency:

```sh
npx supabase --version
```

Scripts disponiveis:

```sh
npm run db:start
npm run db:stop
npm run db:status
npm run db:reset
npm run db:push:dry-run
npm run db:push
```

`npm run db:push` e manual e nao faz parte do build da aplicacao.

## Como executar testes

```sh
go test ./...
go vet ./...
```

## Estrutura geral

```text
cmd/server/              entrada HTTP da aplicacao
.github/workflows/       automacoes de CI/CD
internal/config/         leitura e validacao de configuracao
internal/database/       pool PostgreSQL via pgxpool
internal/products/       catalogo, service e repository PostgreSQL
internal/                demais pacotes internos futuros por area de dominio
web/templates/           templates server-side em templ
web/components/          componentes visuais reutilizaveis em templ
web/assets/              fontes de assets, incluindo CSS fonte
web/static/              assets compilados, embutidos no binario e servidos em /static/
supabase/migrations/     migrations futuras do Supabase
tests/                   suporte futuro para testes de maior escopo
docs/                    documentacao do projeto
vercel.json              regiao Vercel gru1
```

## Documentacao

- [ARCHITECTURE.md](ARCHITECTURE.md): arquitetura principal.
- [AGENTS.md](AGENTS.md): regras para agentes futuros.
- [docs/README.md](docs/README.md): indice da documentacao.
- [docs/plans/roadmap.md](docs/plans/roadmap.md): roadmap por fases.

## Seguranca e credenciais

Nao utilize credenciais reais no repositorio. Arquivos `.env` sao ignorados pelo Git, e `.env.example` existe apenas como referencia sem valores reais.

Nunca confie em dados financeiros recebidos do navegador. Precos, descontos, subtotais, totais, frete, status de pagamento e status de pedido devem ser calculados ou validados no backend.
