# Fase 4 — Catalogo

Status: ATIVA.

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

## Definition of Done

- `templ generate` executado.
- `npm run css:build` executado.
- `gofmt -w .` executado.
- `go mod tidy` executado.
- `go test ./...` passa.
- `go vet ./...` passa.
- `go build ./...` passa.
- `npx supabase --version` passa.
- Migration validada localmente quando ambiente permitir.
- Migration aplicada ao Supabase DEV pelo workflow apos push.
- Vercel valida `/`, `/health`, `/ready`, `/produtos` e `/produtos/nao-existe`.
- Nenhum secret versionado.
- Nenhum seed ou produto ficticio criado.
- Fase 5 permanece planejada.
