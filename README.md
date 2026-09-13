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
- Frete em `GET /checkout/frete` e `POST /checkout/frete`, com cotacao server-side via SuperFrete quando configurada.
- Revisao de checkout em `GET /checkout/revisao`.
- Criacao de pedido pendente de pagamento em `POST /checkout/revisao`.
- Exibicao de pedido por UUID em `GET /pedido/{id}`.
- Inicio de pagamento InfinitePay em `POST /pedido/{id}/pagar`.
- Retorno de pagamento em `GET /pagamento/retorno`, validado por `payment_check` server-side.
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
- Fase 8 — Embalagem Real e Integracao de Frete SuperFrete concluida, incluindo validacao Sandbox real confirmada manualmente.
- Fase 8.1 — UX do Checkout, Consulta de CEP e Diagnostico Seguro de Frete concluida.
- Fase 9 — Revisao e Criacao de Pedidos, com snapshots imutaveis e status `pending_payment`.
- Fase 9.1 — Interface publica de pedidos separada de dados operacionais preservados internamente.
- Fase 10 — Pagamentos InfinitePay, com checkout hospedado server-side e validacao real concluida.
- Fase 11 — Webhooks InfinitePay concluida, com validacao real em producao e pagamento confirmado sem redirect do comprador.
- Fase 12 — Acompanhamento Seguro do Pedido, com `public_tracking_id`, rota `/acompanhar/{uuid}` e pagina SSR minimizada.
- Fase 13.1 — Fundacao de Autenticacao Administrativa, com Supabase Auth, autorizacao por UUID, sessao propria, cookie HttpOnly e dashboard inicial protegido em `/admin`.
- Fase 13.2 — Pedidos, Producao, Envio e Auditoria, com validacao real em producao concluida.
- Fase 13.3 — Catalogo, Variantes, Materiais, Cores e Caixas, com implementacao concluida no painel administrativo e validacao real pendente.

PLANEJADO:

- HTMX quando houver interacao real que justifique sua presenca.
- Fase 13.4 — imagens e Supabase Storage no painel administrativo.
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
- Etapa de dados do checkout renderizada no servidor, com forms HTML, autocomplete nativo, mascaras progressivas e consulta de CEP via backend sem JavaScript obrigatorio.
- Etapa de frete renderizada no servidor, com radios HTML e selecao por POST, sem JavaScript obrigatorio.
- Etapa de revisao, pagina de pedido e acompanhamento seguro renderizados no servidor, sem JavaScript obrigatorio e sem expor dados operacionais de producao ou embalagem ao comprador.
- Login administrativo, dashboard, lista de pedidos e detalhe operacional renderizados no servidor, sem JavaScript obrigatorio.

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
- `products` e `product_variants` possuem perfil logistico opcional em gramas e milimetros, com constraint all-or-none.
- `shipping_boxes` guarda caixas fisicas reais com medidas internas, externas, peso de embalagem, status ativo e ordenacao.
- `cart_shipping_selections` guarda a escolha de frete por carrinho com snapshot do pacote real, preco em centavos, prazo, validade de 30 minutos e `input_hash`.
- `carts.converted_at` marca carrinhos convertidos em pedido.
- `orders`, `order_customer_details`, `order_shipping_addresses`, `order_shipping_details`, `order_items` e `order_item_filaments` guardam snapshots historicos de pedido.
- Pedidos criados pelo checkout nascem com status `pending_payment` e moeda `BRL`.
- `order_payments` guarda pagamento InfinitePay 1:1 por pedido, com status `pending` ou `paid`.
- `order_fulfillment` guarda status operacional 1:1 de producao e envio do pedido.
- `orders.public_tracking_id` e UUID aleatorio unico para `/acompanhar/{uuid}`.
- `admin_sessions` guarda sessoes administrativas transitorias com `SHA-256` do token, TTL de 8 horas e RLS habilitado.
- `admin_order_events` guarda auditoria operacional de mutacoes administrativas de producao/envio.
- `order_number` e sequencial para referencia humana; `/pedido/{id}` usa UUID interno e acompanhamento usa `public_tracking_id`.
- RLS esta habilitado nas tabelas de catalogo, variantes, carrinho, dados temporarios de checkout, frete, pedidos, pagamentos e admin sem policies publicas do Data API.

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
- Nenhuma conta externa e necessaria para executar a aplicacao local basica. Cotacao real de frete exige configuracao SuperFrete de desenvolvimento.

Para o workflow remoto de migrations Supabase, o responsavel pelo projeto deve configurar estes GitHub Actions Secrets, sem incluir valores no repositorio:

- `SUPABASE_ACCESS_TOKEN`
- `SUPABASE_DB_PASSWORD`
- `SUPABASE_PROJECT_ID`

Configuracao local ou de hosting para runtime:

- `SITE_URL`: URL publica da aplicacao. Opcional, usada como origem permitida em mutacoes de carrinho.
- `DATABASE_URL`: secret PostgreSQL. Deve apontar para o Supabase Transaction Pooler.
- `DB_MAX_CONNS`: opcional, default `4`.
- `SUPABASE_URL`: opcional e nao secret, usada para montar URLs publicas de imagens do bucket `product-images`.
- `SUPABASE_PUBLISHABLE_KEY`: opcional e nao administrativa; usada somente para autenticar credenciais do Admin no Supabase Auth.
- `ADMIN_SUPABASE_USER_ID`: opcional; UUID do unico usuario Supabase Auth autorizado a acessar `/admin`.
- `SUPERFRETE_ENV`: `sandbox` ou `production`, obrigatoria somente quando a cotacao real estiver habilitada.
- `SUPERFRETE_API_TOKEN`: secret da SuperFrete, nunca versionado.
- `SUPERFRETE_ORIGIN_POSTAL_CODE`: CEP operacional de origem da PrintLab, normalizado pelo backend.
- `SUPERFRETE_CONTACT_EMAIL`: e-mail operacional usado no `User-Agent` exigido pela SuperFrete.
- `SUPERFRETE_SERVICES`: lista de codigos de servico solicitados, por exemplo `1,2,17`.
- `INFINITEPAY_HANDLE`: InfiniteTag/handle sem `$`, obrigatorio somente para exibir e iniciar pagamento real.

`SUPABASE_SERVICE_ROLE_KEY` nao e usada pela aplicacao nesta fase. O Admin das Fases 13.1 a 13.3 nao usa secret key nem service role.

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

Sem carrinho valido com itens disponiveis, a resposta redireciona para `/carrinho`. Com carrinho valido, a rota renderiza formulario SSR de contato e endereco. O POST salva dados normalizados do carrinho atual e redireciona para `/checkout/frete`.

O JavaScript progressivo da etapa de dados aplica mascaras visuais de CPF, telefone brasileiro e CEP, sem substituir a validacao server-side. A busca de CEP usa somente o endpoint interno da aplicacao e preserva preenchimento manual como fallback:

```sh
curl -i http://localhost:8080/api/cep/01001000
```

Resposta esperada para CEP valido: HTTP 200 com `street`, `district`, `city` e `state`. CEP invalido retorna 400, CEP nao encontrado retorna 404 e falhas externas retornam indisponibilidade sem expor dados sensiveis.

Frete:

```sh
curl -i http://localhost:8080/checkout/frete
```

Sem carrinho valido, a resposta redireciona para `/carrinho`. Sem dados de checkout, redireciona para `/checkout/dados`. Com carrinho, dados, perfis logisticos, caixas reais e SuperFrete configurados, a rota calcula cotacoes atuais e apresenta somente o preco da cotacao final usando a caixa fisica real. A selecao por POST redireciona para `/checkout/revisao`.

Revisao e pedido:

```sh
curl -i http://localhost:8080/checkout/revisao
curl -i http://localhost:8080/pedido/<uuid-do-pedido>
curl -i http://localhost:8080/acompanhar/<public_tracking_id>
```

Sem carrinho valido, a revisao redireciona para `/carrinho`. Sem dados, redireciona para `/checkout/dados`. Sem frete valido, expirado ou com `input_hash` divergente, redireciona para `/checkout/frete`. Pedido criado usa UUID na URL, status `pending_payment` e pagina sem CPF completo, endereco completo, telefone ou e-mail completo.

`/acompanhar/<public_tracking_id>` usa o UUID publico aleatorio do pedido, retorna 404 para identificador invalido/desconhecido, envia headers `private, no-store`, `noindex` e `no-referrer`, e mostra apenas numero humano, data, pagamento, producao, envio e transportadora/servico comercial quando disponivel.

Pagamento:

```sh
curl -i http://localhost:8080/pagamento/retorno
curl -i -X POST http://localhost:8080/webhooks/infinitepay \
  -H 'Content-Type: application/json' \
  -d '{"invoice_slug":"slug","transaction_nsu":"txn","order_nsu":"00000000-0000-0000-0000-000000000000"}'
```

`POST /pedido/<uuid>/pagar` exige `INFINITEPAY_HANDLE` e `SITE_URL` HTTPS para criar ou reutilizar checkout InfinitePay. O payload de criacao do link envia `redirect_url` e `webhook_url` gerados no servidor. O redirect do navegador nao confirma pagamento; `/pagamento/retorno` e `/webhooks/infinitepay` chamam `payment_check` server-side e so marcam o pedido como `paid` quando a InfinitePay confirma `success=true`, `paid=true` e valor igual ao total congelado do pedido. Checkouts pendentes criados antes da Fase 11 nao recebem `webhook_url` retroativamente.

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
internal/shipping/       embalagem real, cotacao SuperFrete e selecao de frete
internal/orders/         revisao, criacao transacional e snapshots de pedido
internal/payments/       checkout InfinitePay, payment_check e persistencia de pagamento
internal/admin/          Supabase Auth, sessao administrativa e dashboard inicial
internal/                demais pacotes internos futuros por area de dominio
web/templates/           templates server-side em templ
web/components/          componentes visuais reutilizaveis em templ
web/assets/              fontes de assets, incluindo CSS fonte
web/static/              assets compilados, JS progressivo, imagens, embutidos no binario e servidos em /static/
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
