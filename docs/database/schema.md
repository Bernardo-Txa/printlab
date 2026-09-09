# Schema de banco

Status: catalogo com variantes e receita de producao IMPLEMENTADO; demais entidades de negocio PLANEJADAS.

A Fase 4 criou o catalogo basico com categorias e produtos. A Fase 5 adiciona variantes, materiais, cores, receita estimada de producao 3D e imagens publicas de catalogo.

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

## Tabela `public.materials`

Materiais logicos usados em receitas de producao de variantes. Eles nao representam marca, carretel, lote ou estoque fisico de filamento.

Campos:

| Coluna | Tipo | Nulo | Default | Observacao |
| --- | --- | --- | --- | --- |
| `id` | `uuid` | nao | `gen_random_uuid()` | Chave primaria. |
| `name` | `text` | nao | - | Nome do material logico. |
| `slug` | `text` | nao | - | Identificador canonico unico. |
| `description` | `text` | sim | - | Descricao opcional. |
| `is_active` | `boolean` | nao | `true` | Controla uso publico/operacional. |
| `created_at` | `timestamptz` | nao | `now()` | Criacao do registro. |
| `updated_at` | `timestamptz` | nao | `now()` | Deve ser atualizado explicitamente em updates futuros. |

Constraints:

- `materials_pkey`: chave primaria em `id`.
- `materials_slug_unique`: `slug` unico.
- `materials_name_not_blank`: `btrim(name) <> ''`.
- `materials_slug_format`: `slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'`.
- `materials_description_not_blank`: `description is null or btrim(description) <> ''`.

RLS:

- RLS habilitado.
- Nenhuma policy publica criada.

## Tabela `public.colors`

Cores logicas usadas em receitas de producao de variantes.

Campos:

| Coluna | Tipo | Nulo | Default | Observacao |
| --- | --- | --- | --- | --- |
| `id` | `uuid` | nao | `gen_random_uuid()` | Chave primaria. |
| `name` | `text` | nao | - | Nome da cor. |
| `slug` | `text` | nao | - | Identificador canonico unico. |
| `hex_color` | `text` | sim | - | Cor em hexadecimal canonico `#RRGGBB`. |
| `is_active` | `boolean` | nao | `true` | Controla uso publico/operacional. |
| `created_at` | `timestamptz` | nao | `now()` | Criacao do registro. |
| `updated_at` | `timestamptz` | nao | `now()` | Deve ser atualizado explicitamente em updates futuros. |

Constraints:

- `colors_pkey`: chave primaria em `id`.
- `colors_slug_unique`: `slug` unico.
- `colors_name_not_blank`: `btrim(name) <> ''`.
- `colors_slug_format`: `slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'`.
- `colors_hex_color_format`: `hex_color is null or hex_color ~ '^#[0-9A-F]{6}$'`.

Semantica:

- `hex_color`, quando preenchido, deve ser armazenado em formato canonico com `#`, seis caracteres e letras maiusculas.

RLS:

- RLS habilitado.
- Nenhuma policy publica criada.

## Tabela `public.product_variants`

Variantes publicas e operacionais de um produto.

Campos:

| Coluna | Tipo | Nulo | Default | Observacao |
| --- | --- | --- | --- | --- |
| `id` | `uuid` | nao | `gen_random_uuid()` | Chave primaria. |
| `product_id` | `uuid` | nao | - | Produto dono da variante. |
| `name` | `text` | nao | - | Nome publico da variante. |
| `slug` | `text` | nao | - | Slug unico dentro do produto. |
| `sku` | `text` | sim | - | Codigo interno opcional. |
| `price_cents` | `bigint` | sim | - | Override opcional de preco em centavos. |
| `is_active` | `boolean` | nao | `false` | Controla exibicao publica da variante. |
| `is_default` | `boolean` | nao | `false` | Variante inicial preferencial. |
| `sort_order` | `integer` | nao | `0` | Ordenacao publica/operacional. |
| `print_time_minutes` | `integer` | sim | - | Tempo estimado de maquina, nao prazo de entrega. |
| `created_at` | `timestamptz` | nao | `now()` | Criacao do registro. |
| `updated_at` | `timestamptz` | nao | `now()` | Deve ser atualizado explicitamente em updates futuros. |

Foreign keys:

- `product_variants_product_id_fkey`: `product_id` referencia `public.products(id)` com `on delete cascade`.

Constraints:

- `product_variants_pkey`: chave primaria em `id`.
- `product_variants_id_product_id_unique`: par `(id, product_id)` unico para FK composta de imagens.
- `product_variants_product_slug_unique`: par `(product_id, slug)` unico.
- `product_variants_sku_unique`: `sku` unico quando preenchido; multiplos `null` sao permitidos.
- `product_variants_name_not_blank`: `btrim(name) <> ''`.
- `product_variants_slug_format`: `slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'`.
- `product_variants_sku_not_blank`: `sku is null or btrim(sku) <> ''`.
- `product_variants_price_cents_non_negative`: `price_cents is null or price_cents >= 0`.
- `product_variants_sort_order_non_negative`: `sort_order >= 0`.
- `product_variants_print_time_minutes_positive`: `print_time_minutes is null or print_time_minutes > 0`.
- `product_variants_default_is_active`: `not is_default or is_active`.

Indices:

- `product_variants_product_id_idx` em `product_variants(product_id)`.
- `product_variants_one_default_per_product_idx`: unique parcial em `product_id` quando `is_default = true`.
- `product_variants_public_order_idx` em `(product_id, is_active, is_default desc, sort_order asc, name asc)`.

Semantica:

- Um produto pode nao ter variantes.
- Um produto pode ter varias variantes ativas.
- No maximo uma variante default pode existir por produto.
- Variante default deve estar ativa.
- `sku` e opcional e nao e identificador publico principal.
- Ordenacao publica: `is_default desc`, `sort_order asc`, `name asc`.

RLS:

- RLS habilitado.
- Nenhuma policy publica criada.

## Tabela `public.variant_filaments`

Receita estimada de producao de uma variante. Cada linha representa um componente de material + cor + quantidade estimada.

Campos:

| Coluna | Tipo | Nulo | Default | Observacao |
| --- | --- | --- | --- | --- |
| `id` | `uuid` | nao | `gen_random_uuid()` | Chave primaria. |
| `variant_id` | `uuid` | nao | - | Variante produzida. |
| `material_id` | `uuid` | nao | - | Material logico. |
| `color_id` | `uuid` | nao | - | Cor logica. |
| `estimated_weight_mg` | `bigint` | nao | - | Peso estimado em miligramas. |
| `label` | `text` | sim | - | Rotulo opcional do componente. |
| `sort_order` | `integer` | nao | `0` | Ordenacao dos componentes. |
| `created_at` | `timestamptz` | nao | `now()` | Criacao do registro. |

Foreign keys:

- `variant_filaments_variant_id_fkey`: `variant_id` referencia `public.product_variants(id)` com `on delete cascade`.
- `variant_filaments_material_id_fkey`: `material_id` referencia `public.materials(id)`.
- `variant_filaments_color_id_fkey`: `color_id` referencia `public.colors(id)`.

Constraints:

- `variant_filaments_pkey`: chave primaria em `id`.
- `variant_filaments_estimated_weight_mg_positive`: `estimated_weight_mg > 0`.
- `variant_filaments_sort_order_non_negative`: `sort_order >= 0`.
- `variant_filaments_label_not_blank`: `label is null or btrim(label) <> ''`.

Indices:

- `variant_filaments_variant_id_idx` em `variant_filaments(variant_id)`.
- `variant_filaments_material_id_idx` em `variant_filaments(material_id)`.
- `variant_filaments_color_id_idx` em `variant_filaments(color_id)`.

Semantica:

- Peso usa miligramas para evitar `float` e permitir pecas pequenas.
- Uma variante pode ter varios componentes, materiais e cores.
- Essa tabela nao representa estoque fisico nem consumo real executado.

RLS:

- RLS habilitado.
- Nenhuma policy publica criada.

## Tabela `public.product_images`

Metadados de imagens publicas de catalogo armazenadas no Supabase Storage.

Campos:

| Coluna | Tipo | Nulo | Default | Observacao |
| --- | --- | --- | --- | --- |
| `id` | `uuid` | nao | `gen_random_uuid()` | Chave primaria. |
| `product_id` | `uuid` | nao | - | Produto dono da imagem. |
| `variant_id` | `uuid` | sim | - | Variante especifica, quando houver. |
| `storage_path` | `text` | nao | - | Caminho relativo no bucket `product-images`. |
| `alt_text` | `text` | sim | - | Texto alternativo opcional. |
| `sort_order` | `integer` | nao | `0` | Ordenacao da galeria. |
| `is_primary` | `boolean` | nao | `false` | Imagem primaria dentro do escopo. |
| `created_at` | `timestamptz` | nao | `now()` | Criacao do registro. |

Foreign keys:

- `product_images_product_id_fkey`: `product_id` referencia `public.products(id)` com `on delete cascade`.
- `product_images_variant_product_fkey`: `(variant_id, product_id)` referencia `public.product_variants(id, product_id)` com `on delete cascade`.

Constraints:

- `product_images_pkey`: chave primaria em `id`.
- `product_images_storage_path_not_blank`: `btrim(storage_path) <> ''`.
- `product_images_storage_path_relative`: impede caminho absoluto, URL externa, `//` e `..`.
- `product_images_alt_text_not_blank`: `alt_text is null or btrim(alt_text) <> ''`.
- `product_images_sort_order_non_negative`: `sort_order >= 0`.

Indices:

- `product_images_product_id_idx` em `product_images(product_id)`.
- `product_images_variant_id_idx` em `product_images(variant_id)` quando `variant_id is not null`.
- `product_images_product_general_order_idx` para imagens gerais por produto.
- `product_images_variant_order_idx` para imagens por variante.
- `product_images_one_primary_per_product_idx`: no maximo uma imagem geral primaria por produto.
- `product_images_one_primary_per_variant_idx`: no maximo uma imagem primaria por variante.

Semantica:

- `variant_id = null` indica imagem geral do produto.
- `variant_id != null` indica imagem especifica de variante.
- A FK composta impede imagem com `product_id` de um produto e `variant_id` de outro.
- URLs absolutas nao sao armazenadas no banco; o backend constroi a URL publica a partir de `SUPABASE_URL`, bucket e `storage_path`.

RLS:

- RLS habilitado.
- Nenhuma policy publica criada.

## Semantica de `price_cents`

`products.price_cents` representa o preco-base comercial do produto.

```text
R$ 39,90 -> 3990
```

Dinheiro nao usa `float32` ou `float64`.

`product_variants.price_cents` pode sobrescrever o preco-base. Se for `null`, o preco efetivo da variante usa `products.price_cents`. Se for `0`, o valor zero e um override explicito e nao deve ser tratado como `null`.

## Categoria opcional

`products.category_id` pode ser `null`. Um produto ativo sem categoria continua aparecendo no catalogo publico.

Se uma categoria for removida, `on delete set null` preserva o produto. Se uma categoria estiver inativa, ela nao aparece nos filtros publicos; o produto ativo associado pode continuar aparecendo na listagem geral sem depender da categoria para existir publicamente.

## Supabase Storage

O bucket `product-images` e configurado por migration em `storage.buckets` para imagens publicas de catalogo.

- `public = true`.
- `file_size_limit = 5242880`.
- `allowed_mime_types = image/avif, image/webp, image/jpeg, image/png`.
- `avif_autodetection = true`.
- Nenhuma policy publica de upload, update ou delete em `storage.objects` e criada.

## Entidades candidatas

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
- Refinar operacao de produtos sob demanda quando houver modulo de producao.
- Definir dados minimos de cliente e endereco.
- Definir upload/admin de imagens.
- Definir estoque fisico e inventario de filamento.
- Definir calculo de custos de producao a partir de insumos e tempo.
