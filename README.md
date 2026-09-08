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
- Tailwind CSS via CLI npm, sem CDN e sem bundler JavaScript.
- Assets estaticos servidos em `/static/` via `embed.FS`, a partir de `web/static/`.
- Logo oficial inicial integrada ao header e ao hero da homepage.
- Design tokens refinados com base na identidade visual da marca.
- Fase 2.1 de Brand Experience aplicada na homepage.

PLANEJADO:

- HTMX quando houver interacao real que justifique sua presenca.
- Catalogo de produtos.
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

Banco planejado:

- PostgreSQL hospedado no Supabase.
- Acesso server-side pelo backend Go usando `pgx`.

Infraestrutura planejada:

- Vercel durante desenvolvimento.
- Vercel Pro antes da operacao comercial.
- Supabase para PostgreSQL.
- Supabase Storage podera ser avaliado futuramente para imagens.

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
- Nenhuma conta externa e necessaria nesta fase.

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

## Como executar testes

```sh
go test ./...
go vet ./...
```

## Estrutura geral

```text
cmd/server/              entrada HTTP da aplicacao
internal/                pacotes internos futuros por area de dominio
web/templates/           templates server-side em templ
web/components/          componentes visuais reutilizaveis em templ
web/assets/              fontes de assets, incluindo CSS fonte
web/static/              assets compilados, embutidos no binario e servidos em /static/
migrations/              migrations futuras de banco
tests/                   suporte futuro para testes de maior escopo
docs/                    documentacao do projeto
```

## Documentacao

- [ARCHITECTURE.md](ARCHITECTURE.md): arquitetura principal.
- [AGENTS.md](AGENTS.md): regras para agentes futuros.
- [docs/README.md](docs/README.md): indice da documentacao.
- [docs/plans/roadmap.md](docs/plans/roadmap.md): roadmap por fases.

## Seguranca e credenciais

Nao utilize credenciais reais no repositorio. Arquivos `.env` sao ignorados pelo Git, e `.env.example` existe apenas como referencia sem valores reais.

Nunca confie em dados financeiros recebidos do navegador. Precos, descontos, subtotais, totais, frete, status de pagamento e status de pedido devem ser calculados ou validados no backend.
