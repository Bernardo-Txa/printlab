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
- Carrinho anonimo SSR em `GET /carrinho`.
- Mutacoes de carrinho por POST para adicionar, atualizar quantidade e remover item.
- Dados de checkout em `GET /checkout/dados` e `POST /checkout/dados`, com contato e endereco vinculados ao carrinho.
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
- Fase 5 — Produtos, Variantes e Producao, com materiais, cores, variantes, receita estimada, imagens e bucket publico de catalogo.
- Fase 6 — Carrinho, com persistencia PostgreSQL, token opaco em cookie e subtotal recalculado no backend.
- Fase 7 — Dados do Cliente e Endereco, com PII minimizada, validacoes brasileiras e persistencia transacional por carrinho.

PLANEJADO:

- HTMX quando houver interacao real que justifique sua presenca.
- Frete e integracao com SuperFrete.
- Pedidos, pagamentos e integracao com InfinitePay.
- Webhooks, acompanhamento de pedido e painel administrativo.
- Custos estimados derivados, estoque fisico de filamento e operacao interna de producao.

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
- Selecao de variante por links SSR via `?variante=<slug>`, sem JavaScript obrigatorio.
- Galeria SSR simples com imagem publica de produto/variante quando existir.
- Card de produto com imagem geral primaria quando existir e placeholder visual de marca como fallback.
- Carrinho renderizado no servidor, com forms HTML e redirects 303, sem JavaScript obrigatorio.
- Etapa de dados do checkout renderizada no servidor, com forms HTML, autocomplete nativo e sem JavaScript obrigatorio.

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
- `materials`, `colors`, `product_variants`, `variant_filaments` e `product_images` modelam variantes, receita estimada de producao 3D e imagens.
- `product_variants.price_cents` pode sobrescrever o preco-base; quando `null`, usa `products.price_cents`.
- `variant_filaments.estimated_weight_mg` armazena peso em miligramas como inteiro.
- `product_variants.print_time_minutes` armazena tempo estimado de maquina, sem representar prazo de entrega.
- `carts` e `cart_items` persistem carrinhos anonimos sem armazenar token bruto nem precos.
- `carts.token_hash` armazena `SHA-256` do token de cookie.
- `cart_items.quantity` e limitado a `1..99`.
- Subtotais do carrinho sao recalculados a partir do preco atual de produto/variante.
- `cart_customer_details` e `cart_shipping_addresses` persistem contato e endereco do checkout vinculados ao carrinho anonimo.
- CPF e CEP sao armazenados como digitos ASCII normalizados; telefone e armazenado em formato canonico brasileiro E.164.
- Dados de contato e endereco sao salvos em transacao e removidos por `ON DELETE CASCADE` quando o carrinho for removido.
- RLS esta habilitado nas tabelas de catalogo, variantes, carrinho e dados temporarios de checkout sem policies publicas do Data API.

Infraestrutura planejada:

- Vercel durante desenvolvimento.
- Vercel Pro antes da operacao comercial.
- Supabase para PostgreSQL.
- Supabase Storage para imagens publicas de catalogo no bucket `product-images`.

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

- `SITE_URL`: URL publica da aplicacao. Opcional, usada como origem permitida em mutacoes de carrinho.
- `DATABASE_URL`: secret PostgreSQL. Deve apontar para o Supabase Transaction Pooler.
- `DB_MAX_CONNS`: opcional, default `4`.
- `SUPABASE_URL`: opcional e nao secret, usada para montar URLs publicas de imagens do bucket `product-images`.

`SUPABASE_SERVICE_ROLE_KEY` nao e usada pela aplicacao nesta fase.

O cookie anonimo do carrinho e marcado como `Secure` quando `APP_ENV=production`, `VERCEL_ENV=production` ou `SITE_URL` usa HTTPS.

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

Detalhe de produto com variante:

```sh
curl -i "http://localhost:8080/produtos/<produto>?variante=<variante>"
```

O slug de variante e opcional e unico dentro do produto. Produto sem variantes continua usando o preco-base.

Carrinho:

```sh
curl -i http://localhost:8080/carrinho
```

Sem cookie, a resposta esperada e HTTP 200 com carrinho vazio. Com banco configurado e cookie valido, a pagina lista itens persistidos. Mutacoes usam forms POST e redirecionam com HTTP 303 para `/carrinho`.

Dados de checkout:

```sh
curl -i http://localhost:8080/checkout/dados
```

Sem carrinho valido com itens disponiveis, a resposta redireciona para `/carrinho`. Com carrinho valido, a rota renderiza formulario SSR de contato e endereco. O POST salva dados normalizados do carrinho atual e redireciona para `/checkout/dados?salvo=1`; frete, pedido e pagamento continuam planejados.

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
internal/cart/           carrinho anonimo, token, service e repository PostgreSQL
internal/customers/      dados temporarios de checkout, validacao e repository PostgreSQL
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
