# Catalogo

Status: Fase 4 IMPLEMENTADA; variantes e operacao comercial PLANEJADAS.

O catalogo apresenta produtos ativos da PrintLab com renderizacao server-side, mantendo o backend como autoridade sobre dados e preco-base.

## Implementado

- Categorias em `public.categories`.
- Produtos basicos em `public.products`.
- Listagem publica em `GET /produtos`.
- Filtro opcional por categoria via `GET /produtos?categoria=<slug>`.
- Pagina publica de produto em `GET /produtos/{slug}`.
- Slug como identificador publico do produto.
- Preco-base em `products.price_cents`, armazenado como inteiro em centavos.
- Formatacao BRL feita no backend antes do template.
- Exibicao publica somente de produtos ativos.
- Suporte a produto destacado por `is_featured`.
- Categoria opcional por `category_id`.
- Empty state honesto quando nao ha produtos publicados.
- Placeholder visual de marca enquanto imagens reais nao existem.

## Regras publicas

- `products.is_active = true` e obrigatorio para aparecer no catalogo.
- Produto inativo deve responder publicamente como inexistente.
- `categories.is_active = true` e obrigatorio para aparecer nos filtros.
- Produto ativo sem categoria continua aparecendo na listagem geral.
- Produto associado a categoria inativa continua podendo aparecer, mas sem exibir a categoria.
- O filtro de categoria usa slug, nunca UUID.
- Slug invalido em rota publica retorna 404.
- Erros de banco retornam mensagem generica e nao expoem detalhes internos.

## Preco-base

`products.price_cents` representa o preco-base comercial do produto nesta fase.

```text
R$ 39,90 -> 3990
```

Variantes poderao futuramente ter preco proprio ou override na Fase 5. A Fase 4 nao modela essa regra.

## Planejado

- Variantes.
- Cores.
- Materiais.
- Tamanhos.
- Imagens reais de produto.
- `product_images`.
- Supabase Storage.
- Estoque.
- Estrategia de produto sob demanda.
- Admin para cadastro.
- Busca.
- Avaliacoes.
- Paginacao complexa.

## Limites

- Nao ha seed ficticio.
- Nao ha produto demonstrativo.
- Nao ha carrinho ou checkout.
- Nao ha selecao de quantidade, cor, material ou tamanho.
- Nao ha upload de imagem.
- Nao ha estoque ou metricas de producao.
