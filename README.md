# PrintLab

A PrintLab e uma empresa brasileira de impressao 3D. Este repositorio sera a base do sistema proprio da empresa, inicialmente voltado para e-commerce e, futuramente, para funcionalidades internas de gestao da operacao.

## Objetivo do projeto

Construir uma aplicacao web simples, segura e manutenivel para venda de produtos impressos em 3D, com evolucao planejada para catalogo, variantes, carrinho, checkout, frete, pagamentos, pedidos e painel administrativo.

## Status atual

IMPLEMENTADO:

- Fundacao inicial do repositorio.
- Documentacao de arquitetura, produto, banco, desenvolvimento, integracoes e roadmap.
- Aplicacao Go minima em `cmd/web`.
- Rota `GET /health` retornando HTTP 200.

PLANEJADO:

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

- Renderizacao server-side.
- `templ`.
- HTMX.
- Tailwind CSS.
- Minimo possivel de JavaScript.

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

- Go 1.22.2 ou versao compativel disponivel no ambiente.
- Nenhuma conta externa e necessaria nesta fase.

## Como executar

```sh
go run ./cmd/web
```

Por padrao, o servidor usa a porta `8080`. Para mudar:

```sh
PORT=3000 go run ./cmd/web
```

Health check:

```sh
curl http://localhost:8080/health
```

## Como executar testes

```sh
go test ./...
go vet ./...
```

## Estrutura geral

```text
cmd/web/                 entrada HTTP da aplicacao
internal/                pacotes internos futuros por area de dominio
web/                     templates, componentes, assets e arquivos estaticos futuros
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
