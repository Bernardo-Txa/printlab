# Migrations

Status: fundacao IMPLEMENTADA; migrations de catalogo, variantes/imagens, carrinho, dados de checkout, frete, pedidos, pagamentos, acompanhamento, admin e limpeza transiente IMPLEMENTADAS.

A migration `20260917100000_create_product_colors.sql` adiciona somente a relacao `public.product_colors` entre produtos e cores comerciais. Ela preserva `colors`, `product_variants` e `variant_filaments`, nao carrega dados automaticamente e possui rollback por remocao da nova tabela.

A primeira migration funcional do projeto cria o catalogo basico:

- `supabase/migrations/20260909153625_create_catalog.sql`

Ela cria `public.categories`, `public.products`, constraints, foreign key, indice `products_category_id_idx` e habilita RLS sem policies publicas. Nao insere dados.

A segunda migration funcional evolui produtos, variantes e producao 3D:

- `supabase/migrations/20260909162227_create_product_variants.sql`

Ela cria `public.materials`, `public.colors`, `public.product_variants`, `public.variant_filaments`, `public.product_images`, constraints, foreign keys, indices parciais, RLS sem policies publicas e configura o bucket `product-images` em `storage.buckets`. Nao insere produtos, materiais, cores, variantes ou imagens ficticias.

A terceira migration funcional cria o carrinho anonimo persistido:

- `supabase/migrations/20260909194855_create_carts.sql`

Ela cria `public.carts` e `public.cart_items`, constraints, foreign keys, indices unique parciais para evitar linhas duplicadas, limite de quantidade `1..99` e RLS sem policies publicas. Nao insere carrinhos, itens ou dados ficticios.

A quarta migration funcional cria os dados temporarios de checkout vinculados ao carrinho:

- `supabase/migrations/20260909203845_create_cart_customer_details.sql`

Ela cria `public.cart_customer_details` e `public.cart_shipping_addresses`, constraints estruturais para CPF, telefone, CEP, UF e pais `BR`, foreign keys 1:1 para `public.carts(id)` com `on delete cascade`, e RLS sem policies publicas. Nao insere contato, endereco, CPF, PII ou dados ficticios.

A quinta migration funcional adiciona a base de frete real:

- `supabase/migrations/20260909220454_add_shipping_profiles_and_selections.sql`

Ela adiciona perfis logisticos opcionais em `public.products` e `public.product_variants`, cria `public.shipping_boxes` e `public.cart_shipping_selections`, constraints all-or-none/positivas, foreign keys, indices uteis e RLS sem policies publicas. Nao insere caixas, produtos, cotacoes, selecoes ou dados ficticios.

A sexta migration funcional cria a base de pedidos:

- `supabase/migrations/20260909233711_create_orders.sql`

Ela adiciona `converted_at` em `public.carts`, cria `public.orders`, `public.order_customer_details`, `public.order_shipping_addresses`, `public.order_shipping_details`, `public.order_items` e `public.order_item_filaments`, com constraints financeiras, snapshots historicos, idempotencia por `source_cart_id`, indices uteis e RLS sem policies publicas. Nao insere pedidos, clientes, enderecos, itens ou dados ficticios.

A setima migration funcional cria a base de pagamentos InfinitePay:

- `supabase/migrations/20260911193009_add_order_payments.sql`

Ela permite `orders.status = 'paid'`, cria `public.order_payments`, constraints financeiras e de integridade, indice unique parcial para `transaction_nsu` e RLS sem policies publicas.

A oitava migration funcional corrige a constraint de host de checkout InfinitePay:

- `supabase/migrations/20260911220406_allow_current_infinitepay_checkout_host.sql`

Ela recria `order_payments_checkout_url_host` para aceitar por allowlist explicita os hosts `checkout.infinitepay.io` e `checkout.infinitepay.com.br`, alinhando o banco com `ValidateCheckoutURL`. Nao altera dados, endpoints, regras financeiras ou status de pagamento.

A nona migration funcional cria sessoes administrativas:

- `supabase/migrations/20260912110000_create_admin_sessions.sql`

Ela cria `public.admin_sessions` com UUID primario, `auth_user_id`, `token_hash` unico de 32 bytes, `created_at`, `expires_at`, indice por expiracao e RLS habilitado sem policies publicas. Nao cria usuario Auth, nao referencia `auth.users`, nao insere dados e nao armazena token bruto, senha, access token ou refresh token.

A decima migration funcional cria auditoria operacional administrativa:

- `supabase/migrations/20260912130000_create_admin_order_events.sql`

Ela cria `public.admin_order_events` com UUID primario, referencia restritiva a `public.orders`, `actor_auth_user_id`, tipo de evento limitado a mudancas de producao/envio, status anterior/novo, timestamp, indice por pedido/data e RLS habilitado sem policies publicas. Nao referencia `auth.users`, nao insere dados e nao armazena PII de cliente, checkout URL, `transaction_nsu` ou `invoice_slug`.

A Fase 13.3 nao criou migration. A gestao administrativa de catalogo, variantes, receita, materiais, cores e caixas usa as tabelas existentes `public.categories`, `public.products`, `public.product_variants`, `public.materials`, `public.colors`, `public.variant_filaments` e `public.shipping_boxes`.

A Fase 13.3A tambem nao criou migration. O refinamento alterou semantica e UX de configuracoes sem renomear `product_variants`.

A Fase 13.4 nao criou migration. A gestao administrativa de imagens usa `public.product_images` e o bucket `product-images` criados na Fase 5.

A decima primeira migration funcional agenda limpeza de dados transientes:

- `supabase/migrations/20260913140000_schedule_transient_data_cleanup.sql`

Ela habilita `pg_cron` e agenda o job diario `printlab_transient_data_cleanup` para remover apenas `public.admin_sessions` expiradas e `public.carts` expirados. Dados temporarios de carrinho sao removidos pelos FKs `ON DELETE CASCADE`; pedidos sao preservados porque `orders.source_cart_id` usa `ON DELETE SET NULL`.

Novas migrations Supabase devem continuar em `supabase/migrations/` e ser revisadas antes de chegar a `main`.

A Fase 14.2 adiciona `20260916120000_require_admin_session_mfa.sql`, somente `ALTER TABLE public.admin_sessions ADD COLUMN mfa_verified_at timestamptz`. Sem default, backfill, indice novo ou alteracao em auth.*. Sessoes existentes ficam NULL e nao autenticam mais; o INSERT apos AAL2 grava `now()`. O indice de token hash e o cron por `expires_at` permanecem suficientes.

Aplicar a migration antes de disponibilizar o novo login. Durante eventual intervalo de deploy sem a coluna, o Admin falha fechado; a loja publica permanece independente. Rollback de codigo para antes do MFA restauraria acesso por senha e nao e um rollback seguro. Preferir corrigir o deploy mantendo a coluna e a exigencia MFA.

A pasta antiga `migrations/` na raiz foi removida para evitar duas fontes de verdade.

## Workflow normal

```text
Codex/desenvolvedor
   |
   v
migration SQL
   |
   v
Git
   |
   v
main
   |
   v
GitHub Actions
   |
   v
Supabase de desenvolvimento
```

O workflow `.github/workflows/supabase-migrations.yml` roda automaticamente em push para `main` somente quando houver alteracao em `supabase/migrations/**` ou `supabase/config.toml`. Ele tambem aceita `workflow_dispatch` para execucao manual emergencial pelo GitHub.

A sequencia remota e:

```sh
supabase link --project-ref "$SUPABASE_PROJECT_ID"
supabase db push --dry-run
supabase db push
```

Se o dry-run falhar, o GitHub Actions interrompe o job antes de aplicar migrations.

A aplicacao Go nao executa migrations no startup. Nao existe AutoMigrate, migration runner, DDL automatico ou `supabase db push` no processo web.

## Regras

- Toda mudanca de schema deve possuir migration versionada.
- Migrations sao imutaveis depois de aplicadas em ambientes compartilhados.
- Rollback deve ser considerado antes da aplicacao.
- Migrations devem ser revisadas.
- Mudancas destrutivas precisam de cuidado adicional.
- Alteracoes de banco nao devem ser feitas manualmente em producao sem registro.
- Table Editor e SQL Editor remoto nao devem ser usados como workflow normal para mudancas de schema.
- A primeira migration real foi criada junto da Fase 4 de catalogo.
- A segunda migration real foi criada junto da Fase 5 de produtos, variantes e producao.
- A terceira migration real foi criada junto da Fase 6 de carrinho.
- A quarta migration real foi criada junto da Fase 7 de dados do cliente e endereco.
- A quinta migration real foi criada junto da Fase 8 de frete SuperFrete.
- A sexta migration real foi criada junto da Fase 9 de pedidos.
- A setima migration real foi criada junto da Fase 10 de pagamentos InfinitePay.
- A oitava migration real corrige a allowlist persistida de host de checkout InfinitePay.
- A nona migration real cria sessoes administrativas para a Fase 13.1.
- A decima migration real cria auditoria operacional administrativa para a Fase 13.2.
- A Fase 13.3 nao possui migration porque o schema existente atende ao escopo.
- A Fase 13.3A nao possui migration porque `product_variants` permanece como modelo interno.
- A Fase 13.4 nao possui migration porque `product_images` e o bucket `product-images` ja existiam.
- A decima primeira migration real agenda limpeza transiente da Fase 14.1 com Supabase Cron.

## Praticas recomendadas

- Uma migration deve representar uma mudanca coesa.
- O nome deve deixar clara a intencao da mudanca.
- Dados sensiveis nao devem aparecer em migrations.
- Migrations que alteram dados devem ser especialmente revisadas.
- Criacao de indices deve considerar impacto em tabelas grandes.
- Migrations destinadas ao CI devem estar em `supabase/migrations/`.
- O workflow de CI nao usa `--include-seed`.
- O workflow de CI nao executa reset remoto.
- `supabase/config.toml` mantem seed desabilitado nesta fase.
- Configuracao de bucket em `storage.buckets` pode fazer parte de migration quando for infraestrutura aprovada e sem dados ficticios de negocio.
- Jobs de `pg_cron` devem ter nome estavel, horario UTC documentado, escopo minimo e testes estruturais quando alterarem dados.

## Antes de aprovar uma migration

- O schema relacionado foi documentado.
- O impacto em codigo e dados foi entendido.
- Testes aplicaveis foram planejados ou executados.
- O caminho de rollback foi discutido quando necessario.
- Para jobs agendados, o impacto de retencao e preservacao de historico foi documentado.

## Secrets do CI

O responsavel pelo projeto deve configurar no GitHub Actions Secrets, sem incluir valores no repositorio:

- `SUPABASE_ACCESS_TOKEN`
- `SUPABASE_DB_PASSWORD`
- `SUPABASE_PROJECT_ID`

O workflow atual aponta para o Supabase de desenvolvimento. Producao exigira politica separada de aprovacao antes da operacao comercial.

## Fase 17.3.3 — Cor comercial no carrinho e pedido

`20260920224834_persist_commercial_colors.sql` adiciona `cart_items.color_id` opcional com FK para `colors`, indices unique parciais para produto/variante/cor, e `order_items.color_id`, `color_name`, `color_slug` opcionais com snapshot historico. Nao cria tabela, nao recalcula dados existentes e preserva RLS.

Aplicar antes do backend atualizado. A versao antiga do upsert nao e compativel com os novos predicados dos indices: coordenar publicacao do schema e codigo. Corrigir adiante e o rollback preferido; remover colunas descartaria snapshots e recriar indices antigos pode conflitar com linhas de cores diferentes. Ver [plano 17.3.3](../plans/active/017-3-3-cor-no-carrinho-e-pedido.md).

## Retirada no local

`20260920234746_add_pickup_delivery_method.sql` adiciona `delivery_method` em `cart_shipping_selections` e `order_shipping_details`, permitindo `shipping` e `pickup`. A migration transforma as constraints de frete em regras condicionais: envio continua exigindo caixa, provedor, servico e pacote positivo; retirada exige caixa nula, provider/servico vazios, frete zero e pacote zerado. Dados historicos existentes recebem default `shipping`.

Aplicar antes do backend atualizado para permitir selecoes `pickup`. A versao antiga do codigo continua compatível com o default `shipping` enquanto nao houver linhas de retirada. Rollback destrutivo removeria a modalidade e pedidos de retirada perderiam a classificacao explicita; o caminho preferido e corrigir adiante mantendo a coluna.
