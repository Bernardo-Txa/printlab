# Schema de banco

Status: catalogo, variantes, receita de producao, carrinho e dados de checkout IMPLEMENTADOS; demais entidades de negocio PLANEJADAS.

A Fase 4 criou o catalogo basico com categorias e produtos. A Fase 5 adiciona variantes, materiais, cores, receita estimada de producao 3D e imagens publicas de catalogo.

A Fase 5.1 nao alterou schema. Ela corrigiu a leitura de receitas para preservar referencias a materiais e cores inativos em `variant_filaments`.

A Fase 6 adiciona carrinho anonimo persistido server-side em `public.carts` e `public.cart_items`.

A Fase 7 adiciona dados temporarios de contato e endereco vinculados ao carrinho em `public.cart_customer_details` e `public.cart_shipping_addresses`.

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
| `is_active` | `boolean` | nao | `true` | Controla oferta para novas escolhas operacionais futuras. |
| `created_at` | `timestamptz` | nao | `now()` | Criacao do registro. |
| `updated_at` | `timestamptz` | nao | `now()` | Deve ser atualizado explicitamente em updates futuros. |

Constraints:

- `materials_pkey`: chave primaria em `id`.
- `materials_slug_unique`: `slug` unico.
- `materials_name_not_blank`: `btrim(name) <> ''`.
- `materials_slug_format`: `slug ~ '^[a-z0-9]+(?:-[a-z0-9]+)*$'`.
- `materials_description_not_blank`: `description is null or btrim(description) <> ''`.

Semantica:

- `materials.is_active = false` nao remove nem oculta receitas existentes que referenciem o material.
- O material inativo deve deixar de ser oferecido para novas configuracoes futuras.

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
| `is_active` | `boolean` | nao | `true` | Controla oferta para novas escolhas operacionais futuras. |
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
- `colors.is_active = false` nao remove nem oculta receitas existentes que referenciem a cor.
- A cor inativa deve deixar de ser oferecida para novas configuracoes futuras.

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
- Componentes devem ser carregados mesmo quando `material_id` ou `color_id` referenciam registros inativos.
- O peso total estimado da variante soma todos os componentes carregados.
- Essa tabela nao representa estoque fisico nem consumo real executado.

RLS:

- RLS habilitado.
- Nenhuma policy publica criada.

## Tabela `public.carts`

Carrinhos anonimos persistidos no PostgreSQL. O token bruto fica somente no navegador e no processamento da request; o banco persiste apenas o hash.

Campos:

| Coluna | Tipo | Nulo | Default | Observacao |
| --- | --- | --- | --- | --- |
| `id` | `uuid` | nao | `gen_random_uuid()` | Chave primaria. |
| `token_hash` | `bytea` | nao | - | `SHA-256` do token bruto do cookie. |
| `created_at` | `timestamptz` | nao | `now()` | Criacao do carrinho. |
| `updated_at` | `timestamptz` | nao | `now()` | Atualizado explicitamente nas mutacoes. |
| `expires_at` | `timestamptz` | nao | - | Expiracao do carrinho anonimo. |

Constraints:

- `carts_pkey`: chave primaria em `id`.
- `carts_token_hash_key`: `token_hash` unico.
- `carts_token_hash_length`: `octet_length(token_hash) = 32`.
- `carts_expires_after_created`: `expires_at > created_at`.

Semantica:

- `token_hash` nunca deve conter o token bruto.
- Carrinho expirado e tratado como inexistente pela aplicacao.
- Mutacoes bem-sucedidas renovam `expires_at` para `agora + 30 dias`.
- Nao ha job de limpeza nesta fase.

RLS:

- RLS habilitado.
- Nenhuma policy publica criada.

## Tabela `public.cart_items`

Itens de carrinho com produto, variante opcional e quantidade. Preco nao e persistido nesta tabela.

Campos:

| Coluna | Tipo | Nulo | Default | Observacao |
| --- | --- | --- | --- | --- |
| `id` | `uuid` | nao | `gen_random_uuid()` | Chave primaria. |
| `cart_id` | `uuid` | nao | - | Carrinho dono da linha. |
| `product_id` | `uuid` | nao | - | Produto escolhido. |
| `variant_id` | `uuid` | sim | - | Variante escolhida, quando aplicavel. |
| `quantity` | `integer` | nao | - | Quantidade entre 1 e 99. |
| `created_at` | `timestamptz` | nao | `now()` | Criacao da linha. |
| `updated_at` | `timestamptz` | nao | `now()` | Atualizado explicitamente nas mutacoes. |

Foreign keys:

- `cart_items_cart_id_fkey`: `cart_id` referencia `public.carts(id)` com `on delete cascade`.
- `cart_items_product_id_fkey`: `product_id` referencia `public.products(id)` com `on delete cascade`.
- `cart_items_variant_product_fkey`: `(variant_id, product_id)` referencia `public.product_variants(id, product_id)`.

Constraints:

- `cart_items_pkey`: chave primaria em `id`.
- `cart_items_quantity_range`: `quantity between 1 and 99`.

Indices:

- `cart_items_cart_id_idx` em `cart_items(cart_id)`.
- `cart_items_product_id_idx` em `cart_items(product_id)`.
- `cart_items_variant_id_idx` em `cart_items(variant_id)` quando `variant_id is not null`.
- `cart_items_cart_product_no_variant_unique_idx`: unique parcial em `(cart_id, product_id)` quando `variant_id is null`.
- `cart_items_cart_product_variant_unique_idx`: unique parcial em `(cart_id, product_id, variant_id)` quando `variant_id is not null`.

Semantica:

- O mesmo produto sem variante nao pode gerar multiplas linhas no mesmo carrinho.
- O mesmo produto com a mesma variante nao pode gerar multiplas linhas no mesmo carrinho.
- Adicionar novamente incrementa `quantity`, respeitando limite 99.
- Atualizacoes e remocoes devem ser limitadas por `cart_id` e `id`.
- Subtotais sao calculados em leitura usando preco atual do catalogo.
- Itens indisponiveis podem continuar visiveis e nao entram no subtotal.

RLS:

- RLS habilitado.
- Nenhuma policy publica criada.

## Tabela `public.cart_customer_details`

Dados temporarios de contato do checkout vinculados ao carrinho anonimo. Esta tabela nao representa uma identidade permanente de cliente.

Campos:

| Coluna | Tipo | Nulo | Default | Observacao |
| --- | --- | --- | --- | --- |
| `cart_id` | `uuid` | nao | - | Chave primaria e FK 1:1 para `public.carts(id)`. |
| `full_name` | `text` | nao | - | Nome completo normalizado por trim. |
| `email` | `text` | nao | - | E-mail normalizado por trim e lowercase. |
| `phone` | `text` | nao | - | Telefone brasileiro canonico, preferencialmente E.164. |
| `cpf` | `text` | nao | - | CPF normalizado com 11 digitos ASCII. |
| `created_at` | `timestamptz` | nao | `now()` | Criacao do registro. |
| `updated_at` | `timestamptz` | nao | `now()` | Atualizado explicitamente em upserts. |

Foreign keys:

- `cart_customer_details_cart_id_fkey`: `cart_id` referencia `public.carts(id)` com `on delete cascade`.

Constraints:

- `cart_customer_details_pkey`: chave primaria em `cart_id`.
- `cart_customer_details_full_name_length`: `char_length(full_name) between 1 and 120`.
- `cart_customer_details_full_name_trimmed`: `full_name = btrim(full_name)`.
- `cart_customer_details_email_length`: `char_length(email) between 3 and 254`.
- `cart_customer_details_email_lower_trimmed`: `email = lower(btrim(email))`.
- `cart_customer_details_phone_format`: `phone ~ '^\+55[0-9]{10,11}$'`.
- `cart_customer_details_cpf_format`: `cpf ~ '^[0-9]{11}$'`.

Semantica:

- O backend valida os digitos verificadores do CPF no Go.
- Nao ha indice ou unique em CPF nesta fase.
- Uma mesma pessoa pode possuir carrinhos diferentes.
- O browser nao acessa esta tabela diretamente.
- O registro e removido junto com o carrinho por cascade.

RLS:

- RLS habilitado.
- Nenhuma policy publica criada.

## Tabela `public.cart_shipping_addresses`

Endereco temporario de entrega vinculado ao carrinho anonimo atual. Nesta fase, somente Brasil esta em escopo.

Campos:

| Coluna | Tipo | Nulo | Default | Observacao |
| --- | --- | --- | --- | --- |
| `cart_id` | `uuid` | nao | - | Chave primaria e FK 1:1 para `public.carts(id)`. |
| `postal_code` | `text` | nao | - | CEP normalizado com 8 digitos ASCII. |
| `street` | `text` | nao | - | Rua/logradouro. |
| `number` | `text` | nao | - | Numero textual, aceitando valores como `12A` ou `s/n`. |
| `complement` | `text` | sim | - | Complemento opcional; vazio e persistido como `null`. |
| `district` | `text` | nao | - | Bairro. |
| `city` | `text` | nao | - | Cidade. |
| `state` | `text` | nao | - | UF brasileira em uppercase. |
| `country_code` | `text` | nao | `'BR'` | Pais fixo Brasil nesta fase. |
| `created_at` | `timestamptz` | nao | `now()` | Criacao do registro. |
| `updated_at` | `timestamptz` | nao | `now()` | Atualizado explicitamente em upserts. |

Foreign keys:

- `cart_shipping_addresses_cart_id_fkey`: `cart_id` referencia `public.carts(id)` com `on delete cascade`.

Constraints:

- `cart_shipping_addresses_pkey`: chave primaria em `cart_id`.
- `cart_shipping_addresses_postal_code_format`: `postal_code ~ '^[0-9]{8}$'`.
- `cart_shipping_addresses_street_length`: `char_length(street) between 1 and 160`.
- `cart_shipping_addresses_street_trimmed`: `street = btrim(street)`.
- `cart_shipping_addresses_number_length`: `char_length(number) between 1 and 30`.
- `cart_shipping_addresses_number_trimmed`: `number = btrim(number)`.
- `cart_shipping_addresses_complement_length`: `complement is null or char_length(complement) between 1 and 120`.
- `cart_shipping_addresses_complement_trimmed`: `complement is null or complement = btrim(complement)`.
- `cart_shipping_addresses_district_length`: `char_length(district) between 1 and 100`.
- `cart_shipping_addresses_district_trimmed`: `district = btrim(district)`.
- `cart_shipping_addresses_city_length`: `char_length(city) between 1 and 100`.
- `cart_shipping_addresses_city_trimmed`: `city = btrim(city)`.
- `cart_shipping_addresses_state_allowed`: UF deve pertencer ao conjunto oficial brasileiro, incluindo DF.
- `cart_shipping_addresses_country_code_br`: `country_code = 'BR'`.

Semantica:

- CEP e UF sao normalizados e validados no backend.
- Nao ha ViaCEP, BrasilAPI, Google Maps ou autocomplete externo nesta fase.
- O registro e removido junto com o carrinho por cascade.
- Frete sera calculado na Fase 8 a partir desses dados revalidados.

RLS:

- RLS habilitado.
- Nenhuma policy publica criada.

## Leitura operacional de dados de checkout

`cart_customer_details` e `cart_shipping_addresses` devem ser lidas como uma unidade logica. O repository Go usa uma unica consulta SQL com `JOIN` por `cart_id`, evitando que duas queries separadas observem estados diferentes em requests concorrentes.

Como a escrita e transacional, os dois registros devem existir juntos. Se houver estado parcial anomalo, como contato sem endereco ou endereco sem contato, a aplicacao trata como dados ausentes e nao retorna PII parcial para o formulario de checkout.

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

- `customers`: identidade permanente de clientes somente se houver login ou conta futura.
- `addresses`: enderecos permanentes ou de cobranca somente se houver necessidade futura.
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

O preco-base de produto foi implementado em `products.price_cents`. Subtotal de carrinho e calculado em leitura pelo backend. Frete, pedidos, descontos, pagamentos e total final de checkout continuam planejados.

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
- Definir upload/admin de imagens.
- Definir estoque fisico e inventario de filamento.
- Definir calculo de custos de producao a partir de insumos e tempo.
- Implementar limpeza programada de carrinhos expirados e PII associada antes do go-live comercial.
