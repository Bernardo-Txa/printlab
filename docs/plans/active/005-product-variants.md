# Fase 5 — Produtos, Variantes e Producao

Status: ATIVA.

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
