# Fase 4 — Catalogo

Status: CONCLUIDA.

## Contexto

Fases 3 e 3.1 concluiram a fundacao PostgreSQL/Supabase, o runtime Vercel em `gru1`, `pgxpool`, `/ready` e o workflow GitHub Actions -> Supabase DEV.

A Fase 4 e a primeira funcionalidade de negocio persistida no PostgreSQL.

## Objetivo

Implementar catalogo publico server-side com categorias, produtos basicos, listagem, filtro por categoria e pagina de detalhe de produto.

## Escopo

- Criar migration real `create_catalog`.
- Criar `public.categories`.
- Criar `public.products`.
- Habilitar RLS nas duas tabelas, sem policies publicas.
- Implementar repository PostgreSQL com `pgxpool`.
- Implementar service de catalogo.
- Validar slugs antes de consultar banco.
- Formatar preco-base BRL no backend.
- Criar `GET /produtos`.
- Criar `GET /produtos/{slug}`.
- Criar templates SSR com empty state, card e detalhe.
- Atualizar navegacao com link real para `/produtos`.
- Atualizar documentacao e ADR.

## Fora de escopo

- Variantes.
- Materiais.
- Cores.
- Tamanhos.
- Imagens reais de produto.
- `product_images`.
- Supabase Storage.
- Upload.
- Estoque.
- Peso de filamento.
- Tempo de impressao.
- Custo de producao.
- Carrinho.
- Checkout.
- Pedidos.
- Clientes.
- Pagamentos.
- SuperFrete.
- InfinitePay.
- Autenticacao.
- Admin.
- Busca full-text.
- Paginacao complexa.
- Seeds ou produtos ficticios.

## Schema

`categories` usa UUID, `name`, `slug`, descricao opcional, estado ativo e timestamps.

`products` usa UUID, `category_id` opcional, `name`, `slug`, descricoes opcionais, `price_cents`, estado ativo, destaque e timestamps.

`products.category_id` referencia `categories(id)` com `on delete set null`.

`products.price_cents` representa o preco-base comercial em centavos.

## Decisoes

- URLs publicas usam slug, nunca UUID.
- Produtos publicos exigem `is_active = true`.
- Categorias publicas exigem `is_active = true`.
- Produto inativo responde como inexistente.
- Produto ativo sem categoria continua aparecendo.
- Produto associado a categoria inativa pode aparecer sem categoria na listagem geral.
- Ordenacao inicial: `is_featured desc`, `created_at desc`, `name asc`.
- `updated_at` usa `default now()`; updates futuros devem alterar explicitamente.
- Dinheiro usa `int64` em Go e `bigint` no PostgreSQL.
- Templates recebem preco formatado e nao fazem calculo financeiro.
- Erros PostgreSQL nao chegam ao usuario.

## Implementacao

- `internal/products` contem tipos, slug, dinheiro, service e repository.
- `cmd/server` registra rotas de catalogo e injeta service quando o banco esta configurado.
- `web/templates/catalog.templ` renderiza catalogo, detalhe, 404 e indisponibilidade.
- `web/components/product_card.templ` renderiza card e placeholder visual de marca.
- Homepage continua independente do PostgreSQL.

## Testes

- Catalogo vazio retorna HTTP 200 em handler test com service fake.
- Catalogo com produtos retorna HTTP 200.
- Filtro por categoria e encaminhado ao service.
- Produto encontrado retorna HTTP 200.
- Produto inexistente retorna HTTP 404.
- Slug invalido nao consulta service/repository.
- Erro de catalogo retorna HTTP 503 sem vazar detalhes.
- Formatacao BRL cobre centavos e milhares sem `float`.
- Service valida slug e formata preco.

## Validacoes executadas

- `templ generate` passou.
- `npm run css:build` passou.
- `gofmt -w .` passou.
- `go mod tidy` passou.
- `go test ./...` passou.
- `go vet ./...` passou.
- `go build ./...` passou.
- `npx supabase --version` retornou `2.117.0`.
- `npx supabase start` aplicou a migration localmente.
- `npx supabase db reset` aplicou `20260909153625_create_catalog.sql` localmente.
- Aplicacao local com banco vazio retornou:
  - `GET /`: HTTP 200.
  - `GET /health`: HTTP 200 com body `ok`.
  - `GET /ready`: HTTP 200 com body `ok`.
  - `GET /produtos`: HTTP 200 com empty state.
  - `GET /produtos/nao-existe`: HTTP 404.
  - `GET /static/css/app.css`: HTTP 200.
- GitHub Actions `Supabase Migrations` run `34372918466` passou apos push:
  - checagem de secrets;
  - `supabase link`;
  - `supabase db push --dry-run`;
  - `supabase db push`.
- Vercel publico em `https://printlab-pied.vercel.app` retornou:
  - `GET /`: HTTP 200.
  - `GET /health`: HTTP 200 com body `ok`.
  - `GET /ready`: HTTP 200 com body `ok`.
  - `GET /produtos`: HTTP 200 com empty state.
  - `GET /produtos/nao-existe`: HTTP 404.

## Definition of Done

- `templ generate` executado.
- `npm run css:build` executado.
- `gofmt -w .` executado.
- `go mod tidy` executado.
- `go test ./...` passa.
- `go vet ./...` passa.
- `go build ./...` passa.
- `npx supabase --version` passa.
- Migration validada localmente com Supabase local.
- Migration aplicada ao Supabase DEV pelo workflow apos push.
- Vercel validada em `/`, `/health`, `/ready`, `/produtos` e `/produtos/nao-existe`.
- Nenhum secret versionado.
- Nenhum seed ou produto ficticio criado.
- Fase 5 permanece planejada.
