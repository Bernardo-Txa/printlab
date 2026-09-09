# Schema de banco

Status: catalogo basico IMPLEMENTADO; demais entidades de negocio PLANEJADAS.

A Fase 4 cria o primeiro schema de negocio da PrintLab: categorias e produtos basicos para catalogo publico.

## Convencoes futuras

- Usar `snake_case` para tabelas, colunas, constraints e indices.
- Usar `timestamptz` para datas e horas.
- Tratar horarios em UTC no banco.
- Usar `NOT NULL` quando a coluna for obrigatoria.
- Declarar foreign keys explicitas para relacionamentos.
- Colocar constraints no banco para invariantes importantes.
- Criar indices a partir de queries reais ou necessidades claras.
- Evitar `SELECT *` em codigo de producao.
- Alteracoes de schema devem usar migrations versionadas em `supabase/migrations/`.
- Migrations aplicadas nao devem ser alteradas silenciosamente.

## Tabela `public.categories`

Campos:

| Coluna | Tipo | Nulo | Default | Observacao |
| --- | --- | --- | --- | --- |
| `id` | `uuid` | nao | `gen_random_uuid()` | Chave primaria. |
| `name` | `text` | nao | - | Nome publico da categoria. |
| `slug` | `text` | nao | - | Identificador publico unico. |
| `description` | `text` | sim | - | Descricao opcional. |
| `is_active` | `boolean` | nao | `true` | Controla exibicao publica nos filtros. |
| `created_at` | `timestamptz` | nao | `now()` | Criacao do registro. |
| `updated_at` | `timestamptz` | nao | `now()` | Deve ser atualizado explicitamente em operacoes futuras de update. |

Constraints:

- `categories_pkey`: chave primaria em `id`.
- `categories_slug_unique`: `slug` unico.
- `categories_name_not_blank`: `btrim(name) <> ''`.
- `categories_slug_format`: `slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'`.
- `categories_description_not_blank`: `description is null or btrim(description) <> ''`.

RLS:

- RLS habilitado.
- Nenhuma policy publica criada nesta fase.

## Tabela `public.products`

Campos:

| Coluna | Tipo | Nulo | Default | Observacao |
| --- | --- | --- | --- | --- |
| `id` | `uuid` | nao | `gen_random_uuid()` | Chave primaria. |
| `category_id` | `uuid` | sim | - | Categoria opcional. |
| `name` | `text` | nao | - | Nome publico do produto. |
| `slug` | `text` | nao | - | Identificador publico unico. |
| `short_description` | `text` | sim | - | Resumo opcional para listagem e SEO. |
| `description` | `text` | sim | - | Descricao opcional para detalhe. |
| `price_cents` | `bigint` | nao | - | Preco-base comercial em centavos. |
| `is_active` | `boolean` | nao | `false` | Controla exibicao publica. |
| `is_featured` | `boolean` | nao | `false` | Permite destaque e ordenacao inicial. |
| `created_at` | `timestamptz` | nao | `now()` | Criacao do registro. |
| `updated_at` | `timestamptz` | nao | `now()` | Deve ser atualizado explicitamente em operacoes futuras de update. |

Foreign keys:

- `products_category_id_fkey`: `category_id` referencia `public.categories(id)` com `on delete set null`.

Constraints:

- `products_pkey`: chave primaria em `id`.
- `products_slug_unique`: `slug` unico.
- `products_name_not_blank`: `btrim(name) <> ''`.
- `products_slug_format`: `slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'`.
- `products_price_cents_non_negative`: `price_cents >= 0`.
- `products_short_description_not_blank`: `short_description is null or btrim(short_description) <> ''`.
- `products_description_not_blank`: `description is null or btrim(description) <> ''`.

Indices:

- `products_category_id_idx` em `products(category_id)`.

RLS:

- RLS habilitado.
- Nenhuma policy publica criada nesta fase.

## Semantica de `price_cents`

`products.price_cents` representa o preco-base comercial do produto na Fase 4.

```text
R$ 39,90 -> 3990
```

Dinheiro nao usa `float32` ou `float64`. Quando variantes forem implementadas na Fase 5, elas poderao ter preco proprio ou override sem quebrar o contrato atual do preco-base.

## Categoria opcional

`products.category_id` pode ser `null`. Um produto ativo sem categoria continua aparecendo no catalogo publico.

Se uma categoria for removida, `on delete set null` preserva o produto. Se uma categoria estiver inativa, ela nao aparece nos filtros publicos; o produto ativo associado pode continuar aparecendo na listagem geral sem depender da categoria para existir publicamente.

## Entidades candidatas

- `product_variants`: variacoes de produto, como cor, material ou tamanho.
- `product_images`: imagens associadas a produtos.
- `customers`: dados minimos de clientes.
- `addresses`: enderecos de entrega ou cobranca quando necessario.
- `carts`: carrinhos de visitantes ou clientes.
- `cart_items`: itens dentro de carrinhos.
- `orders`: pedidos criados pelo backend.
- `order_items`: itens persistidos de pedido com valores calculados pelo backend.
- `payments`: registros de pagamento, tentativas e status validados.
- `shipments`: dados de frete e envio.

## Regras iniciais

- Valores financeiros devem ter representacao segura e deterministica.
- `float32` e `float64` nao devem ser usados como representacao canonica de dinheiro.
- Pedidos devem preservar os valores calculados no momento da compra.
- Pagamentos e webhooks exigem desenho de idempotencia antes da implementacao.
- Mudancas de schema devem usar migrations versionadas.
- Regras financeiras nunca devem depender somente de frontend ou RLS.

## Dinheiro

Valores financeiros futuros nao devem usar `float32` ou `float64` como representacao canonica.

A preferencia atual e armazenar dinheiro como inteiro em centavos:

```text
R$ 39,90 -> 3990
```

O preco-base de produto foi implementado em `products.price_cents`. Totais de carrinho, frete, pedidos, descontos e pagamentos continuam planejados.

## IDs

`categories` e `products` usam UUID. Nao ha estrategia universal aprovada para as demais entidades; `uuid` e `bigint identity` serao avaliados conforme cada caso.

Nenhuma extensao PostgreSQL deve ser habilitada sem necessidade atual.

## RLS e Data API

Supabase Data API nao e a interface principal da PrintLab. O browser nao acessa tabelas sensiveis diretamente; o backend Go controla regras criticas.

RLS continua util como camada complementar futura, mas nao substitui validacao server-side para precos, frete, pagamentos, pedidos ou permissoes sensiveis.

## Pendencias

- Definir status de pedido.
- Definir status de pagamento.
- Definir modelo de variantes.
- Definir estrategia para produtos sob demanda.
- Definir dados minimos de cliente e endereco.
- Definir imagens de produto e storage.
- Definir estoque.
- Definir metricas de producao, como custo de filamento e tempo de impressao.
