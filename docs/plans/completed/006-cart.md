# Fase 6 — Carrinho

Status: CONCLUIDA.

## Objetivo

Permitir que visitantes anonimos adicionem produtos ao carrinho, visualizem itens, alterem quantidade, removam itens e retornem posteriormente enquanto o carrinho estiver valido.

## Decisoes

- Carrinho anonimo persistido no PostgreSQL.
- Cookie `printlab_cart` contem somente token opaco gerado com `crypto/rand`.
- Banco persiste somente `SHA-256(token)` em `carts.token_hash`.
- Validade inicial de 30 dias, renovada em mutacoes bem-sucedidas.
- Precos nao sao persistidos em `cart_items`; sao recalculados server-side a partir de `products` e `product_variants`.
- Produto e variante sao revalidados ao adicionar e ao renderizar o carrinho.
- Itens indisponiveis continuam visiveis e nao entram no subtotal.
- Mutacoes usam POST e redirect 303.
- Protecao cross-site proporcional por SameSite=Lax, cookie HttpOnly e validacao centralizada de Origin/Referer.

## Schema

- `public.carts`: carrinho anonimo identificado por `token_hash`, com `created_at`, `updated_at` e `expires_at`.
- `public.cart_items`: linhas de carrinho com `cart_id`, `product_id`, `variant_id` opcional e `quantity`.
- `quantity` deve ficar entre 1 e 99.
- Indices unique parciais impedem linhas duplicadas para produto sem variante e produto com variante.
- RLS habilitado sem policies publicas.

## Seguranca

- Token bruto nao e persistido, logado, enviado em URL ou renderizado em HTML.
- Navegador nao envia preco, subtotal, total ou nome de produto como fonte de verdade.
- Alteracoes de item sempre sao limitadas por `cart_id` e `item_id`.
- Cookie e host-only, `HttpOnly`, `SameSite=Lax`, `Path=/` e `Secure` em producao.
- Requests de mutacao com Origin/Referer cross-site sao rejeitadas.

## Escopo

- `GET /carrinho`.
- `POST /carrinho/adicionar`.
- `POST /carrinho/itens/{id}/quantidade`.
- `POST /carrinho/itens/{id}/remover`.
- Formulario real de adicionar ao carrinho no detalhe de produto.
- Header com link para `/carrinho`.
- Templates SSR sem JavaScript obrigatorio.
- Migration `create_carts`.
- Testes de token, cookie, service, dinheiro, disponibilidade, handlers e escopo de seguranca.

## Fora de escopo

- Login.
- Checkout.
- Endereco.
- Frete.
- Pedido.
- Pagamento.
- Cupom.
- Estoque.
- Reserva de estoque.
- Admin.
- HTMX.
- Redis.
- Dados ficticios.

## Implementacao

- `internal/cart` concentra tipos, token/cookie, regras de service e repository PostgreSQL.
- `cmd/server` registra handlers HTTP e centraliza validacao de Origin/Referer para mutacoes.
- `web/templates/cart.templ` renderiza carrinho vazio, linhas, resumo e estados indisponiveis.
- `web/templates/catalog.templ` passa a postar produto, variante selecionada e quantidade para `/carrinho/adicionar`.

## Testes

- Token, hash e atributos de cookie.
- Carrinho inexistente e expirado.
- Criacao no primeiro add.
- Produto ativo sem variante.
- Produto com variantes exigindo variante.
- Variante valida, inativa e de outro produto.
- Produto inativo.
- Add/increment, limite 99 e quantidade invalida.
- Disponibilidade de itens e preservacao de linhas indisponiveis.
- Subtotal por linha, subtotal total e overflow.
- Escopo de update/remove por `cart_id + item_id`.
- Handlers de GET, POST add, update, remove e origem cross-site.

## Definition of Done

- `templ generate` executado.
- `npm run css:build` executado.
- `gofmt -w .` executado.
- `go mod tidy` executado.
- `go test ./...` passa.
- `go vet ./...` passa.
- `go build ./...` passa.
- `npx supabase --version` passa.
- `npx supabase db reset` passa quando Supabase local estiver disponivel.
- Migration aplicada pelo GitHub Actions apos push.
- Vercel validada quando observavel.
- Nenhum secret versionado.
- Nenhum dado ficticio criado.
- Fase 7 permanece planejada.

## Validacao local

- `templ generate`: passou.
- `npm run css:build`: passou.
- `gofmt -w .`: executado.
- `go mod tidy`: executado.
- `go test ./...`: passou.
- `go vet ./...`: passou.
- `go build ./...`: passou.
- `npx supabase --version`: passou com CLI disponivel.
- `npx supabase db reset`: passou apos iniciar o Supabase local, aplicando `20260909194855_create_carts.sql`.
