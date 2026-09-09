# Fase 5 — Produtos, Variantes e Producao

Status: CONCLUIDA.

## Contexto

A Fase 4 concluiu o catalogo publico SSR com `categories` e `products`. A Fase 5 evolui esse catalogo para representar variantes, imagens e receita estimada de producao 3D sem iniciar carrinho, checkout, estoque fisico ou admin.

## Decisoes aprovadas

- Um produto pode possuir varias variantes.
- Uma variante pode usar varias cores.
- Uma variante pode usar varios materiais.
- `products.price_cents` continua sendo o preco-base.
- `product_variants.price_cents` pode sobrescrever o preco-base.
- Preco efetivo de variante usa override quando preenchido; caso contrario usa o preco-base do produto.
- Producao inicial e sob demanda.
- Nao ha controle de estoque unitario de produto nesta fase.
- Material logico e filamento fisico sao conceitos diferentes.
- Custos derivados nao sao persistidos; peso e tempo de maquina sao persistidos para calculo futuro.

## Escopo

- Migration `create_product_variants`.
- Tabelas `materials`, `colors`, `product_variants`, `variant_filaments` e `product_images`.
- Bucket publico `product-images` no Supabase Storage.
- RLS habilitado nas novas tabelas sem policies publicas do Data API.
- Repository PostgreSQL sem N+1 para catalogo e detalhe.
- Service com preco efetivo, variante default, peso, tempo e selecao de imagem.
- `GET /produtos` com menor preco efetivo entre variantes ativas.
- `GET /produtos/{slug}?variante=<variant-slug>` com selecao SSR de variante.
- Galeria SSR simples com fallback para placeholder PrintLab.
- Documentacao e ADR.

## Fora de escopo

- Carrinho.
- Checkout.
- Estoque.
- Filamento fisico, carretel, lote ou inventario.
- Custo por kg, margem, lucro ou custo persistido.
- Admin.
- Upload de imagens.
- Autenticacao.
- SuperFrete.
- InfinitePay.
- Jobs de producao.
- Fila de impressora.
- Manutencao de impressora.
- Seeds ou dados ficticios.

## Schema

- `materials`: materiais logicos ativos/inativos usados em receitas de variante.
- `colors`: cores logicas com hexadecimal canonico `#RRGGBB` quando preenchido.
- `product_variants`: variantes por produto, slug unico dentro do produto, preco opcional, default opcional e tempo estimado de impressao.
- `variant_filaments`: componentes de receita por variante, com material, cor, peso estimado em miligramas e rotulo opcional.
- `product_images`: caminhos relativos de imagens no bucket `product-images`, gerais de produto ou especificas de variante.

## Storage

O bucket `product-images` e destinado somente a imagens publicas de catalogo. Ele nao deve ser usado para documentos, dados de clientes, invoices, arquivos privados ou secrets.

O upload permanece futuro. Nenhuma policy de `INSERT`, `UPDATE` ou `DELETE` publico em `storage.objects` e criada nesta fase.

## Implementacao

- `internal/products` concentra regras de preco efetivo, formatacao de peso/tempo, URL publica de Storage e selecao de variante/imagem.
- `cmd/server` valida slug de produto e variante antes de chamar o service.
- Templates `templ` exibem variantes com links SSR e canonical do produto sem query string.
- A ausencia de `SUPABASE_URL` nao quebra rotas publicas; imagens caem no placeholder.

## Testes

- Preco efetivo com fallback, override e override zero.
- Total de filamento em mg.
- Formatacao de peso em gramas pt-BR.
- Formatacao de tempo em horas/minutos.
- URL publica de imagem e rejeicao de paths inseguros.
- Selecao de variante default, primeira ativa, explicita valida, inexistente, inativa, de outro produto e produto sem variantes.
- Fallback de imagem: variante, produto e placeholder.
- Handlers de catalogo e produto preservados.

## Validacoes executadas

- `templ generate` passou.
- `npm run css:build` passou.
- `gofmt -w .` passou.
- `go mod tidy` passou.
- `go test ./...` passou.
- `go vet ./...` passou.
- `go build ./...` passou.
- `npx supabase --version` retornou `2.117.0`.
- `npx supabase db reset` aplicou `20260909153625_create_catalog.sql` e `20260909162227_create_product_variants.sql` localmente.
- Supabase local confirmou:
  - tabelas `materials`, `colors`, `product_variants`, `variant_filaments` e `product_images`;
  - RLS habilitado nas novas tabelas;
  - constraints e indices criados;
  - bucket `product-images` publico, com limite de 5 MB e MIME types de imagem;
  - nenhuma policy publica criada para as novas tabelas ou `storage.objects`;
  - nenhuma linha persistida em produtos, variantes, materiais, cores ou imagens.
- Aplicacao local com banco vazio retornou:
  - `GET /`: HTTP 200.
  - `GET /health`: HTTP 200 com body `ok`.
  - `GET /ready`: HTTP 200 com body `ok`.
  - `GET /produtos`: HTTP 200 com empty state.
  - `GET /produtos/nao-existe`: HTTP 404.
  - `GET /static/css/app.css`: HTTP 200.
- GitHub Actions `Supabase Migrations` run `34378986883` passou apos push:
  - checagem de secrets;
  - `supabase link`;
  - `supabase db push --dry-run`;
  - `supabase db push`.
- Vercel publico em `https://printlab-pied.vercel.app` retornou:
  - `GET /`: HTTP 200.
  - `GET /health`: HTTP 200 com body `ok`.
  - `GET /ready`: HTTP 200 com body `ok`.
  - `GET /produtos`: HTTP 200 com empty state.
  - `GET /static/css/app.css`: HTTP 200.
- Nenhum secret real foi identificado no diff.
- Nenhum seed ou produto ficticio foi criado.
- Nenhuma imagem real foi validada remotamente porque nao ha `product_images` cadastradas.

## Definition of Done

- `templ generate` executado.
- `npm run css:build` executado.
- `gofmt -w .` executado.
- `go mod tidy` executado.
- `go test ./...` passa.
- `go vet ./...` passa.
- `go build ./...` passa.
- `npx supabase --version` passa.
- `npx supabase db reset` aplica Fase 4 e Fase 5 localmente quando Supabase local estiver disponivel.
- Migration aplicada ao Supabase DEV pelo workflow apos push.
- Vercel validada em `/`, `/health`, `/ready` e `/produtos`.
- Nenhum secret versionado.
- Nenhum seed ou produto ficticio criado.
- Fase 6 permanece planejada.
