# Banco de dados

Status: fundacao PostgreSQL/Supabase, catalogo, variantes, producao, carrinho, dados de checkout, frete, pedidos e pagamentos InfinitePay IMPLEMENTADOS; demais schemas de negocio PLANEJADOS.

## Responsabilidade

O banco armazenara dados persistentes de produtos, clientes, enderecos, carrinhos, pedidos, pagamentos, envios e informacoes operacionais aprovadas.

## Limites

- Existem as tabelas `public.categories`, `public.products`, `public.materials`, `public.colors`, `public.product_variants`, `public.variant_filaments`, `public.product_images`, `public.carts`, `public.cart_items`, `public.cart_customer_details`, `public.cart_shipping_addresses`, `public.shipping_boxes`, `public.cart_shipping_selections`, `public.orders`, `public.order_customer_details`, `public.order_shipping_addresses`, `public.order_shipping_details`, `public.order_items`, `public.order_item_filaments` e `public.order_payments`.
- As migrations funcionais criam o catalogo basico, a modelagem de variantes/producao, o carrinho anonimo, os dados temporarios de checkout, a base de frete, os snapshots de pedido e o registro 1:1 de pagamento.
- Ha workflow GitHub Actions para aplicar futuras migrations versionadas ao Supabase de desenvolvimento.
- Ha acesso PostgreSQL server-side com `pgx/v5` e `pgxpool`.
- A conexao depende de `DATABASE_URL` em runtime.
- Sem `DATABASE_URL`, a aplicacao inicia, mas `/ready` retorna HTTP 503.

## Decisoes

- Usar PostgreSQL.
- Hospedar o PostgreSQL no Supabase.
- Acessar o banco pelo backend Go usando `pgx/v5` e `pgxpool`.
- Nao usar Supabase Data API como interface primaria da aplicacao.
- Manter frontend sem acesso direto a tabelas sensiveis.
- Usar `supabase/migrations/` para migrations versionadas quando o schema for aprovado.
- Aplicar migrations remotas pelo GitHub Actions com `supabase db push --dry-run` antes de `supabase db push`.
- Usar `DATABASE_URL` como unica fonte de verdade da conexao PostgreSQL em runtime.
- Nao montar connection string manualmente no codigo.
- Configurar `pgx.QueryExecModeExec` como `DefaultQueryExecMode` para compatibilidade com Supabase Transaction Pooler.
- Manter migrations separadas do startup da aplicacao.
- Criar `categories` e `products` com UUID, slug unico, RLS habilitado e sem policies publicas nesta fase.
- Usar `products.price_cents` como preco-base em centavos.
- Criar `materials`, `colors`, `product_variants`, `variant_filaments` e `product_images` com UUID, constraints declarativas, FKs explicitas, RLS habilitado e sem policies publicas do Data API.
- Usar `product_variants.price_cents` como override opcional de preco em centavos.
- Usar `variant_filaments.estimated_weight_mg` como peso estimado em miligramas.
- Usar `product_variants.print_time_minutes` como tempo estimado de maquina em minutos, sem representar prazo de entrega.
- Usar Supabase Storage apenas para imagens publicas de catalogo no bucket `product-images`.
- Criar `carts` e `cart_items` para carrinho anonimo persistido server-side.
- Persistir somente `SHA-256` do token de carrinho no banco.
- Nao persistir preco em `cart_items`.
- Criar `cart_customer_details` e `cart_shipping_addresses` como dados temporarios 1:1 do carrinho, sem entidade permanente de cliente nesta fase.
- Salvar contato e endereco em transacao e remover por `ON DELETE CASCADE` quando o carrinho for removido.
- Adicionar perfil logistico opcional em `products` e `product_variants`, com peso em gramas e dimensoes em milimetros.
- Criar `shipping_boxes` vazia para caixas fisicas reais da PrintLab, sem seed ficticio.
- Criar `cart_shipping_selections` 1:1 com `carts` para armazenar a selecao de frete com snapshot do pacote real, preco em centavos, validade e `input_hash`.
- Adicionar `carts.converted_at` para impedir reutilizacao de carrinho convertido.
- Criar pedidos como snapshots historicos, com `orders.source_cart_id` unique para idempotencia.
- Usar `order_number` apenas como referencia humana e UUID como identificador de rota.
- Criar `order_payments` 1:1 com `orders`, RLS habilitado, provider fixo `infinitepay`, status `pending`/`paid`, `order_nsu` derivado do UUID do pedido e unique parcial para `transaction_nsu`.

## Runtime de conexao

```text
Vercel Go
   |
   v
pgxpool
   |
   v
Supabase Transaction Pooler
   |
   v
PostgreSQL
```

Pool padrao por instancia:

- `MaxConns = 4`
- `MinConns = 0`

`DB_MAX_CONNS` permite ajuste explicito, mas valores invalidos ou menores que 1 sao erro de configuracao.

## Catalogo implementado

- `public.categories` organiza filtros publicos por slug.
- `public.products` guarda produtos basicos do catalogo.
- `products.category_id` e opcional e usa `on delete set null`.
- Produtos publicos exigem `products.is_active = true`.
- Categorias publicas exigem `categories.is_active = true`.
- Produtos inativos se comportam como inexistentes nas rotas publicas.
- `products.is_featured` participa da ordenacao inicial.
- `products.price_cents` e `bigint` com constraint `>= 0`.
- `product_variants` guarda variantes ativas/inativas por produto.
- `product_variants.price_cents` pode sobrescrever o preco-base.
- `materials` e `colors` sao catalogo logico de producao, nao estoque fisico.
- `variant_filaments` permite multicolor e multimaterial por variante.
- `product_images` guarda metadados e caminhos relativos no bucket `product-images`.
- O bucket `product-images` e publico para leitura de imagens de catalogo e nao possui policy publica de upload.
- `carts` guarda `token_hash`, timestamps e `expires_at`.
- `cart_items` guarda produto, variante opcional e quantidade `1..99`.
- Indices unique parciais impedem linhas duplicadas por produto sem variante e produto com variante.
- `cart_customer_details` guarda nome, e-mail, telefone e CPF normalizados para a etapa de dados.
- `cart_shipping_addresses` guarda endereco brasileiro normalizado para frete futuro.
- `products` e `product_variants` possuem perfil logistico all-or-none para frete.
- `shipping_boxes` guarda caixas reais ativas/inativas, medidas internas para encaixe, medidas externas para transportadora e peso de embalagem/protecao.
- `cart_shipping_selections` guarda a escolha atual de frete do carrinho, com snapshot logistico, `quoted_at`, `expires_at` e `input_hash`.
- `carts.converted_at` marca carrinhos convertidos e fora do fluxo ativo de compra.
- `orders` guarda pedido pendente de pagamento ou pago, total em centavos, UUID e `order_number`.
- `order_customer_details` e `order_shipping_addresses` guardam snapshots privados.
- `order_shipping_details` guarda o snapshot logistico do frete selecionado.
- `order_items` e `order_item_filaments` guardam snapshots de itens e receita de producao.
- `order_payments` guarda checkout InfinitePay, status de pagamento, `order_nsu`, retorno confirmado e valores validados.
- RLS esta habilitado em `carts`, `cart_items`, `cart_customer_details`, `cart_shipping_addresses`, `shipping_boxes`, `cart_shipping_selections` e tabelas de pedido sem policies publicas.

## Convencoes de schema futuras

- Nomes de tabelas, colunas, constraints e indices devem usar `snake_case`.
- Datas e horas persistidas devem usar `timestamptz`.
- Horarios devem ser tratados em UTC no banco.
- Colunas obrigatorias devem usar `NOT NULL`.
- Relacionamentos devem usar foreign keys explicitas.
- Invariantes importantes devem ser reforcadas por constraints no banco.
- Indices devem nascer de queries reais ou requisitos claros.
- Evitar `SELECT *` em codigo de producao.
- Migrations aplicadas em ambientes compartilhados nao devem ser alteradas silenciosamente.

## Dinheiro

Valores financeiros futuros nao devem usar `float32` ou `float64` como representacao canonica. A preferencia inicial e armazenar valores inteiros em centavos, por exemplo `R$ 39,90` como `3990`.

O preco-base de produto foi implementado em `products.price_cents`. Variantes podem sobrescrever esse valor com `product_variants.price_cents`; `null` significa fallback para o preco-base, enquanto `0` e override explicito.

Carrinho calcula subtotal atual em leitura. Dados de contato/endereco pertencem ao carrinho ate a criacao do pedido. Frete selecionado usa `cart_shipping_selections.price_cents`. Pedido congela subtotal, frete e total definitivo em `orders`. Pagamento InfinitePay persiste `amount_cents` e `paid_amount_cents` em centavos; descontos continuam planejados.

## Producao 3D

Peso estimado de componente usa `variant_filaments.estimated_weight_mg` como `bigint`, por exemplo:

```text
42 g = 42000 mg
3,25 g = 3250 mg
```

Tempo estimado de maquina usa `product_variants.print_time_minutes` como `integer`. Esse tempo e dado operacional e nao representa prazo de entrega.

Custos derivados como material, maquina, lucro e margem nao sao persistidos nesta fase. Eles deverao ser calculados futuramente a partir de peso, tempo e dados de filamento fisico.

## Filamento fisico

`materials` e `colors` nao representam carretel, marca, lote, preco de compra ou peso disponivel.

Tabelas como `filament_spools`, `filament_inventory`, `filament_batches`, `purchase_price` e `remaining_weight` continuam fora do escopo e pertencem a um modulo operacional futuro.

## IDs

`categories`, `products`, `product_variants`, `carts`, `cart_items`, `shipping_boxes`, `orders`, `order_items` e `order_item_filaments` usam UUID. `cart_customer_details`, `cart_shipping_addresses`, `cart_shipping_selections`, `order_customer_details`, `order_shipping_addresses` e `order_shipping_details` usam a chave primaria da entidade pai por serem relacoes 1:1. `orders.order_number` usa `bigint identity` sequencial apenas para referencia humana. Nenhuma extensao PostgreSQL deve ser habilitada sem necessidade atual.

## RLS e Data API

Supabase Data API nao e a interface principal da PrintLab. O browser nao deve acessar tabelas sensiveis diretamente. O backend Go controla regras criticas.

RLS continua util como camada complementar futura, mas regras financeiras nunca devem depender somente de frontend ou RLS.

## Praticas recomendadas

- Toda mudanca de schema deve passar por migration versionada.
- Migrations aplicadas em ambientes compartilhados devem ser tratadas como imutaveis.
- Consultas devem ser claras, revisaveis e testaveis.
- Transacoes devem proteger criacao de pedidos e mudancas financeiras.
- Valores monetarios nao devem usar `float32` ou `float64` como representacao canonica.
- Producao deve ter separacao explicita e politica de aprovacao antes de receber migrations automaticas.

## Praticas proibidas

- Criar ou alterar tabelas manualmente em producao sem registro.
- Versionar credenciais de banco.
- Permitir que o navegador escreva diretamente em tabelas sensiveis.
- Criar indice ou unique em CPF sem necessidade aprovada.
- Gerar schema antes de aprovacao das entidades e regras.
- Executar reset remoto automatico.
- Aplicar seed automaticamente no workflow de migrations.
- Rodar `supabase db push`, DDL automatico ou migration runner no startup da aplicacao Go.
- Logar connection string, senha ou `DATABASE_URL`.
