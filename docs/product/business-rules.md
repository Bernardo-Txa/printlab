# Regras de negocio

Status: catalogo, variantes, receita de producao, carrinho, dados de checkout, frete, pedidos, pagamentos InfinitePay, acompanhamento e operacao administrativa basica IMPLEMENTADOS; demais regras comerciais PLANEJADAS.

Este documento registra regras de negocio previstas para a PrintLab. Ele nao representa funcionalidades prontas.

## Regras iniciais planejadas

- Produtos sao itens fisicos.
- Muitos produtos poderao ser produzidos sob demanda.
- Checkout podera funcionar sem conta obrigatoria.
- Backend e autoridade sobre precos.
- Backend e autoridade sobre pedidos.
- Backend e autoridade sobre status de pagamento.
- Frete e validado server-side.
- Pagamento nunca e confirmado somente por redirect do navegador.

## Catalogo implementado

- Somente produtos ativos aparecem publicamente.
- Produto inativo responde como inexistente em rotas publicas.
- Categoria de produto e opcional.
- Categorias inativas nao aparecem como filtro publico.
- Produto ativo sem categoria continua podendo aparecer no catalogo.
- O preco-base vem do backend e e armazenado como inteiro em centavos.
- O frontend nunca e autoridade sobre preco.
- Dinheiro nao usa `float32` ou `float64`.
- Produtos ficticios ou seeds demonstrativos nao devem ser inseridos apenas para testar catalogo.
- Slugs sao os identificadores publicos de categorias e produtos.
- Produtos podem possuir varias variantes ativas.
- Uma variante inativa responde publicamente como inexistente.
- O slug de variante e unico dentro do produto e pode ser usado em `?variante=<slug>`.
- A ausencia de variante em um produto continua valida.
- Variante default ativa e escolhida automaticamente quando existir.
- Sem variante default, a primeira variante ativa pela ordenacao publica e escolhida.
- `materials.is_active` nao controla exibicao publica de receitas existentes.
- `colors.is_active` nao controla exibicao publica de receitas existentes.

## Autoridade do backend

O navegador podera enviar IDs, quantidades, CEP e escolhas de interface. Esses dados devem ser tratados como entrada nao confiavel.

Antes de finalizar uma compra, o backend deve:

1. receber IDs e quantidades;
2. buscar produtos e precos no banco;
3. validar disponibilidade;
4. recalcular subtotal;
5. validar frete;
6. calcular total;
7. criar o pedido.

## Carrinho implementado

- Carrinho anonimo nao exige login.
- Carrinho e persistido no PostgreSQL, nao como JSON no navegador.
- Cookie armazena somente token opaco do carrinho.
- Banco armazena somente `SHA-256(token)`.
- Carrinho expira apos 30 dias.
- Mutacoes bem-sucedidas renovam a expiracao para `agora + 30 dias`.
- Quantidade permitida por item: `1..99`.
- Adicionar o mesmo produto/variante incrementa a quantidade existente.
- Produto com variantes ativas exige variante valida para adicionar.
- Produto sem variantes ativas pode ser adicionado sem variante.
- Navegador nunca determina preco, subtotal ou total.
- Carrinho nao congela preco; a leitura usa preco atual do catalogo.
- Carrinho convertido possui `carts.converted_at` preenchido e nao deve ser reutilizado como carrinho ativo.
- Produto e variante sao revalidados ao adicionar e ao renderizar.
- Item indisponivel nao some silenciosamente.
- Item indisponivel nao entra no subtotal.
- Depois que um pedido e criado, dados temporarios do carrinho sao removidos e uma nova compra deve usar novo carrinho.
- Pagamento permanece planejado.

## Dados de checkout implementados

- Checkout continua sem conta obrigatoria.
- Dados de contato e endereco pertencem ao carrinho anonimo atual.
- Nao ha entidade permanente de cliente nesta fase.
- Cada carrinho pode possuir um conjunto de contato e um endereco de entrega atual.
- A coleta de PII so ocorre quando ha carrinho existente, nao vazio e sem itens indisponiveis.
- Contato e endereco sao persistidos juntos em transacao.
- O backend valida e normaliza nome, e-mail, telefone brasileiro, CPF, CEP, UF e pais.
- CPF e necessario para documentacao futura de envio/DC-e e nao e identificador publico.
- Nao ha coleta de senha, conta, newsletter, marketing consent, data de nascimento, genero ou dados nao necessarios a compra.
- O pedido copia contato e endereco para snapshots definitivos antes de pagamento/envio.
- Limpeza programada de carrinhos expirados e PII associada e requisito antes do go-live comercial.

## Dinheiro

Valores monetarios nunca devem usar `float32` ou `float64` como representacao canonica.

`products.price_cents` e o preco-base comercial do produto e usa inteiro em centavos:

```text
R$ 39,90 -> 3990
```

`product_variants.price_cents` pode sobrescrever o preco-base. Quando estiver `null`, o preco efetivo da variante usa `products.price_cents`.

```text
Produto base: 3990
Variante sem preco proprio: 3990
Variante com price_cents = 5990: 5990
Variante com price_cents = 0: 0
```

Carrinho recalcula precos e subtotais no backend. Frete e calculado e selecionado no backend. Pedido recalcula subtotal, frete e total no POST de revisao antes de congelar valores historicos. Pagamento InfinitePay usa esses valores congelados; descontos continuam planejados.

## Frete implementado

- Frete e sempre calculado no backend.
- O navegador nunca determina preco de frete, prazo, transportadora, peso ou dimensoes.
- `POST /checkout/frete` recebe somente `service_code` como escolha do cliente e revalida a cotacao atual antes de persistir.
- Produtos e variantes possuem perfil logistico em gramas e milimetros, separado da receita de producao 3D.
- Produto cru, perfil logistico protegido e caixa fisica sao conceitos diferentes.
- Se a variante possui perfil logistico completo, ela substitui o perfil do produto.
- Se a variante nao possui perfil logistico completo, o frete usa o perfil completo do produto.
- Campos parciais nao sao misturados entre produto e variante.
- Produto sem perfil logistico efetivo nao recebe estimativa ficticia de peso ou dimensoes.
- A PrintLab so deve cotar com caixas fisicas reais cadastradas em `shipping_boxes`.
- A caixa menor compativel e escolhida por dimensoes internas considerando rotacao, nunca somente por volume.
- Medidas internas da caixa sao usadas para encaixe; medidas externas sao enviadas a transportadora.
- `packaging_weight_g` representa caixa/protecao/enchimento padrao e e somado ao peso dos produtos.
- A cotacao SuperFrete acontece em duas etapas: `products` para obter pacote ideal e `package` com caixa real para obter preco final.
- Somente a cotacao final com a caixa fisica real e apresentada ao cliente.
- Se nenhuma caixa real comporta o pacote ideal, o sistema mostra indisponibilidade e nao divide automaticamente em varios volumes.
- A selecao de frete expira em 30 minutos.
- A selecao e invalidada por `input_hash` quando carrinho, quantidade, variante, perfil logistico, CEP, servicos ou caixa mudam.
- Multi-volume, etiqueta/postagem e rastreio permanecem planejados.

## Pedidos implementados

- Pedido e criado somente a partir de carrinho valido, nao convertido, nao vazio, sem itens indisponiveis, com dados completos e frete selecionado valido.
- `GET /checkout/revisao` nao recota SuperFrete; apenas valida expiracao e `input_hash` da selecao persistida.
- `POST /checkout/revisao` recalcula precos, subtotal, frete e total no servidor.
- O browser envia `review_fingerprint` apenas para detectar revisao antiga; ele nao define valores financeiros.
- Se o fingerprint divergir, nenhum pedido e criado.
- Pedido e criado em transacao PostgreSQL unica.
- A conversao do carrinho usa lock por `SELECT ... FOR UPDATE`.
- `orders.source_cart_id` permite idempotencia: um carrinho gera no maximo um pedido.
- `order_number` e sequencial e serve somente como referencia humana.
- A rota publica do pedido usa UUID, nao `order_number`.
- O unico status criado pelo checkout nesta fase e `pending_payment`.
- Pedido preserva snapshots de produto, variante, preco, quantidade, subtotal, frete, cliente, endereco e receita de producao.
- Snapshots operacionais de producao e embalagem sao preservados no pedido, mas nao sao apresentados na experiencia publica do comprador.
- O snapshot de producao guarda tempo e peso por unidade, sem multiplicar pela quantidade.
- `order_item_filaments` nao possui FK para `materials`, `colors` ou `variant_filaments`.
- Depois do commit, dados temporarios de carrinho, cliente, endereco e frete sao removidos.
- A pagina `/pedido/{id}` nao deve exibir CPF completo, endereco completo, telefone, e-mail completo, SKU interno, tempo de impressao, consumo de filamento, receita operacional, caixa fisica, peso ou dimensoes do pacote.

## Pagamento implementado

- Pedido nasce com `orders.status = 'pending_payment'`.
- `POST /pedido/{id}/pagar` inicia checkout hospedado InfinitePay somente a partir de pedido pendente.
- O navegador nunca envia preco, total, frete, `redirect_url` ou status de pagamento.
- O backend monta o payload InfinitePay a partir de `orders`, `order_items`, `order_customer_details`, `order_shipping_addresses` e `order_shipping_details`.
- O item de produto usa preco unitario em centavos e quantidade do snapshot, nunca total unitario multiplicado como preco.
- Frete entra como item separado somente quando seu valor e maior que zero.
- Antes de chamar a InfinitePay, o backend exige que a soma dos itens do payload seja exatamente `orders.total_cents`.
- `order_payments.status` usa somente `pending` ou `paid`.
- `orders.status` pode mudar para `paid` somente apos `payment_check` server-side com `success=true`, `paid=true` e `amount` igual ao total do pedido.
- O webhook InfinitePay nao confirma pagamento diretamente; ele apenas aciona `payment_check` server-side com `order_nsu`, `transaction_nsu` e `invoice_slug`.
- `paid_amount` pode divergir de `amount` e e persistido sem ser usado para validar o total do pedido.
- Redirect, query string, `receipt_url` e `capture_method` do navegador nao confirmam pagamento.
- Checkout abandonado, retorno/webhook com `paid=false` ou falha de API mantem o pedido pendente.
- Recebimento real de webhook InfinitePay em producao foi validado na Fase 11.
- Checkouts pendentes criados antes da Fase 11 nao recebem `webhook_url` retroativamente.

## Acompanhamento de pedido implementado

- Todo pedido possui `orders.public_tracking_id` UUID aleatorio, unico e obrigatorio.
- `/acompanhar/{public_tracking_id}` usa o identificador publico aleatorio, nao `orders.id` nem `order_number`.
- `order_number` continua sendo apenas referencia humana.
- UUID invalido e UUID desconhecido retornam 404.
- A pagina de acompanhamento nao mostra itens, produtos, valores, CPF, e-mail, telefone, endereco, UUID interno do pedido, `source_cart_id`, `transaction_nsu`, `invoice_slug`, checkout URL, peso, dimensoes, filamento, material ou cor.
- Acompanhamento mostra somente numero humano do pedido, data, status de pagamento, status de producao, status de envio e transportadora/servico comercial quando houver.
- Respostas de acompanhamento usam `Cache-Control: private, no-store`, `X-Robots-Tag: noindex, nofollow, noarchive` e `Referrer-Policy: no-referrer`.
- O HTML de acompanhamento usa meta robots `noindex, nofollow, noarchive`.
- Logs nao devem registrar `public_tracking_id`.
- Nao ha endpoint publico para alterar status de producao ou envio.

## Admin implementado

- Admin exige login por Supabase Auth com e-mail e senha.
- A PrintLab autoriza acesso administrativo somente por `ADMIN_SUPABASE_USER_ID`.
- E-mail do usuario nao e regra de autorizacao.
- Nao ha signup administrativo pela aplicacao.
- Senha administrativa nunca e persistida pela PrintLab.
- Access token e refresh token do Supabase nao sao persistidos.
- Sessao administrativa usa token opaco em cookie HttpOnly e `SHA-256(token)` em `admin_sessions`.
- Sessao expira em 8 horas e nao possui renovacao automatica nesta fase.
- Dashboard mostra apenas contagens agregadas de pedidos.
- Dashboard e listagem administrativa de pedidos nao devem carregar CPF, endereco, telefone, e-mail de cliente, `transaction_nsu`, `invoice_slug`, checkout URL ou detalhes operacionais de item/producao.
- Detalhe administrativo autenticado pode exibir PII, endereco, frete completo, itens e snapshots operacionais necessarios para producao/envio.
- Alteracoes administrativas de producao e envio exigem POST, sessao valida e validacao de `Origin`/`Referer`.
- `Origin: null` em POST administrativo e rejeitado independentemente de `Referer`.
- O ator da mutacao operacional e sempre o `Session.AuthUserID` resolvido da sessao administrativa.
- Mutacoes de producao/envio devem atualizar `order_fulfillment` e inserir `admin_order_events` na mesma transacao PostgreSQL.
- Auditoria operacional registra somente pedido, UUID do usuario Auth, tipo de evento, status anterior, status novo e horario; nao registra PII de cliente.
- CRUD de produtos, alteracao de valores/dados de pedido, papeis multiplos, upload de imagens e integracao de etiqueta/postagem permanecem planejados para subfases futuras.

## Status operacionais implementados

- Pedido `pending_payment` nao pode iniciar producao.
- Producao so pode avancar quando `orders.status = 'paid'`.
- Producao avanca somente em sequencia: `waiting` -> `in_production` -> `completed`.
- Producao nao pode regredir nem pular diretamente de `waiting` para `completed`.
- Envio so pode sair de `waiting` quando producao estiver `completed`.
- Envio avanca somente em sequencia: `waiting` -> `preparing` -> `shipped` -> `delivered`.
- Envio nao pode regredir nem pular etapas.
- `delivered` e terminal para mutacoes de envio.
- Transicao repetida ou baseada em estado stale deve falhar sem criar evento de auditoria.
- Status de pagamento continua sendo autoridade exclusiva dos fluxos InfinitePay server-side.

## Producao 3D

O catalogo representa receita estimada de producao por variante:

```text
Produto
  -> Variante
      -> componente: material + cor + peso estimado
      -> componente: material + cor + peso estimado
```

Essa estrutura suporta impressao multicolorida e multimaterial sem gravar `color_id`, `material_id` ou peso diretamente em `product_variants`.

`variant_filaments.estimated_weight_mg` armazena peso em miligramas como inteiro:

```text
42 g = 42000 mg
3,25 g = 3250 mg
```

`product_variants.print_time_minutes` armazena tempo estimado de maquina em minutos. Esse tempo nao e prazo de entrega e nao deve ser exibido ao cliente como promessa de envio.

Receitas ja cadastradas preservam os nomes de material e cor referenciados em `variant_filaments`, mesmo quando o material ou a cor estiverem inativos. Inativar material ou cor significa retirar a opcao de novas configuracoes operacionais futuras, nao remover componentes de receitas historicas.

O peso total estimado de uma variante deve somar todos os componentes carregados de `variant_filaments`, incluindo componentes que referenciem material ou cor inativos.

Custos derivados como `production_cost`, `material_cost`, `machine_cost`, `profit` e `margin` nao sao persistidos nesta fase. Futuramente eles poderao ser calculados a partir de peso estimado, tempo de maquina, filamento fisico, preco por kg e outros custos aprovados.

## Estoque e filamento fisico

Nao ha controle de estoque unitario de produtos nesta fase. A disponibilidade publica depende de `products.is_active` e `product_variants.is_active`.

`materials` e `colors` sao conceitos logicos de catalogo/producao. Eles nao representam marca de filamento, carretel fisico, lote, preco de compra ou peso disponivel.

`materials.is_active = false` e `colors.is_active = false` devem ser tratados como indisponibilidade para novas escolhas futuras. A pagina publica de produto nao deve ocultar, renomear ou marcar como inativo um componente ja usado por uma receita existente.

Filamento fisico, inventario, lotes, custo por kg e reserva de material permanecem planejados para modulo operacional futuro.

## Imagens

Imagens publicas de catalogo usam caminhos relativos em `product_images.storage_path` e arquivos no bucket `product-images` do Supabase Storage.

Imagens podem ser gerais do produto ou especificas de uma variante. A pagina de produto prioriza imagens da variante selecionada; se nao existirem, usa imagens gerais do produto; se nenhuma imagem publica estiver disponivel, usa placeholder visual da PrintLab.

Upload de imagens, admin de imagens e policies de escrita permanecem fora do escopo desta fase.
