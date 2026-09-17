# Regras de negocio

Status: catalogo, variantes, receita de producao, carrinho, dados de checkout, frete, pedidos, pagamentos InfinitePay, acompanhamento, operacao administrativa com imagens e hardening base IMPLEMENTADOS; demais regras comerciais PLANEJADAS.

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
- Produtos podem possuir configuracoes internas em `product_variants`.
- Uma configuracao inativa responde publicamente como inexistente quando solicitada explicitamente.
- O slug de configuracao e unico dentro do produto e pode ser usado em `?variante=<slug>`.
- A ausencia de configuracao ativa em um produto continua valida.
- Uma unica configuracao ativa e selecionada automaticamente sem expor escolha publica.
- Com duas ou mais configuracoes ativas, a configuracao default ativa e escolhida inicialmente quando existir.
- Sem configuracao default, a primeira configuracao ativa pela ordenacao publica e escolhida.
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
- Adicionar o mesmo produto/configuracao incrementa a quantidade existente.
- Produto sem configuracao ativa pode ser adicionado sem `variant_id`.
- Produto com exatamente uma configuracao ativa resolve essa configuracao no backend.
- Produto com duas ou mais configuracoes ativas exige configuracao valida para adicionar.
- Navegador nunca determina preco, subtotal ou total.
- Carrinho nao congela preco; a leitura usa preco atual do catalogo.
- Carrinho convertido possui `carts.converted_at` preenchido e nao deve ser reutilizado como carrinho ativo.
- Produto e configuracao sao revalidados ao adicionar e ao renderizar.
- Item indisponivel nao some silenciosamente.
- Item indisponivel nao entra no subtotal.
- Depois que um pedido e criado, dados temporarios do carrinho sao removidos e uma nova compra deve usar novo carrinho.
- Pagamento ocorre depois da criacao do pedido, via checkout hospedado InfinitePay.

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
- Carrinhos expirados e PII temporaria associada sao removidos pelo job diario de limpeza transiente.

## Dinheiro

Valores monetarios nunca devem usar `float32` ou `float64` como representacao canonica.

`products.price_cents` e o preco-base comercial do produto e usa inteiro em centavos:

```text
R$ 39,90 -> 3990
```

`product_variants.price_cents` pode sobrescrever o preco-base. Quando estiver `null`, o preco efetivo da configuracao usa `products.price_cents`.

```text
Produto base: 3990
Configuracao sem preco proprio: 3990
Configuracao com price_cents = 5990: 5990
Configuracao com price_cents = 0: 0
```

Carrinho recalcula precos e subtotais no backend. Frete e calculado e selecionado no backend. Pedido recalcula subtotal, frete e total no POST de revisao antes de congelar valores historicos. Pagamento InfinitePay usa esses valores congelados; descontos continuam planejados.

## Frete implementado

- Frete e sempre calculado no backend.
- O navegador nunca determina preco de frete, prazo, transportadora, peso ou dimensoes.
- `POST /checkout/frete` recebe somente `service_code` como escolha do cliente e revalida a cotacao atual antes de persistir.
- Produtos e configuracoes possuem perfil logistico em gramas e milimetros, separado da receita de producao 3D.
- Produto cru, perfil logistico protegido e caixa fisica sao conceitos diferentes.
- Se a configuracao possui perfil logistico completo, ela substitui o perfil do produto.
- Se a configuracao nao possui perfil logistico completo, o frete usa o perfil completo do produto.
- Campos parciais nao sao misturados entre produto e configuracao.
- Produto sem perfil logistico efetivo nao recebe estimativa ficticia de peso ou dimensoes.
- A PrintLab so deve cotar com caixas fisicas reais cadastradas em `shipping_boxes`.
- A caixa menor compativel e escolhida por dimensoes internas considerando rotacao, nunca somente por volume.
- Medidas internas da caixa sao usadas para encaixe; medidas externas sao enviadas a transportadora.
- `packaging_weight_g` representa caixa/protecao/enchimento padrao e e somado ao peso dos produtos.
- A cotacao SuperFrete acontece em duas etapas: `products` para obter pacote ideal e `package` com caixa real para obter preco final.
- Somente a cotacao final com a caixa fisica real e apresentada ao cliente.
- Se nenhuma caixa real comporta o pacote ideal, o sistema mostra indisponibilidade e nao divide automaticamente em varios volumes.
- A selecao de frete expira em 30 minutos.
- A selecao e invalidada por `input_hash` quando carrinho, quantidade, configuracao, perfil logistico, CEP, servicos ou caixa mudam.
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
- Pedido preserva snapshots de produto, configuracao, preco, quantidade, subtotal, frete, cliente, endereco e receita de producao.
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
- Respostas de acompanhamento tambem recebem os headers globais de seguranca da Fase 14.1.
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
- Alteracoes administrativas de producao e envio exigem POST, sessao valida e validacao administrativa estrita de `Origin`/`Referer`.
- `Origin: null`, origem cross-site e requests sem `Origin` e sem `Referer` em POST administrativo sao rejeitados.
- O ator da mutacao operacional e sempre o `Session.AuthUserID` resolvido da sessao administrativa.
- Mutacoes de producao/envio devem atualizar `order_fulfillment` e inserir `admin_order_events` na mesma transacao PostgreSQL.
- Auditoria operacional registra somente pedido, UUID do usuario Auth, tipo de evento, status anterior, status novo e horario; nao registra PII de cliente.
- Catalogo Admin gerencia categorias, produtos, configuracoes do produto, receita estimada, materiais, cores e caixas por SSR protegido.
- Mutacoes de catalogo Admin exigem POST, sessao valida, `Origin` same-origin valido ou `Referer` same-origin como fallback quando `Origin` estiver ausente, rejeicao de `Origin: null`, rejeicao de requests sem `Origin` e sem `Referer` e limite conservador de body.
- Imagens Admin usam upload direto ao Supabase Storage com signed upload URL; o backend valida Admin, produto, configuracao, MIME, tamanho e path antes de finalizar `product_images`.
- Remocao fisica de imagem ocorre somente para paths gerenciados no bucket `product-images`; associacoes legadas/manuais nao disparam DELETE arbitrario.
- Entidades principais de catalogo e logistica usam ativacao/inativacao por `is_active`; nao ha hard delete de categorias, produtos, configuracoes, materiais, cores ou caixas.
- `admin_order_events` e exclusivo de pedidos; catalogo nao reutiliza essa auditoria e nao cria tabela de eventos antecipada.
- Alteracao de valores/dados de pedido, papeis multiplos e integracao de etiqueta/postagem permanecem planejados para subfases futuras.

## Hardening base implementado

- Todas as rotas recebem headers globais de seguranca e CSP.
- A CSP nao usa `unsafe-eval` e so inclui a origem Supabase quando derivada de `SUPABASE_URL`.
- O teto global de body e 1 MiB, preservando limites menores de Admin, imagens Admin e webhook InfinitePay.
- `SITE_URL`, quando preenchida, deve ser URL absoluta `http` ou `https`, com host, sem userinfo e sem fragment; em producao deve usar `https`.
- Comparacoes com `SITE_URL` usam `scheme://host`.
- Rate limiting, MFA, RBAC e CAPTCHA nao foram implementados na 14.1.

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

`product_variants` permanece sendo a entidade interna para SKU, preco especifico, tempo estimado, receita, perfil logistico, status ativo e snapshots. Na interface administrativa, o conceito deve ser apresentado como configuracao do produto.

Na loja publica:

- produto sem configuracao ativa continua funcionando como produto simples;
- produto com exatamente uma configuracao ativa seleciona essa configuracao automaticamente e nao mostra seletor ao cliente;
- produto com duas ou mais configuracoes ativas mostra escolha publica;
- `is_default` define a opcao inicialmente selecionada quando houver escolha;
- se dados antigos nao tiverem default, o fallback deterministico usa a primeira configuracao ativa pela ordenacao publica;
- a URL `?variante=<slug>` continua suportada por compatibilidade.

O catalogo representa receita estimada de producao por configuracao interna:

```text
Produto
  -> Configuracao interna
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

No Admin, novos componentes de receita usam somente material e cor ativos. Componentes existentes que apontam para material ou cor inativos continuam carregaveis, podem preservar a referencia atual e podem ter peso, rotulo e ordenacao editados.

O peso total estimado de uma configuracao deve somar todos os componentes carregados de `variant_filaments`, incluindo componentes que referenciem material ou cor inativos.

Custos derivados como `production_cost`, `material_cost`, `machine_cost`, `profit` e `margin` nao sao persistidos nesta fase. Futuramente eles poderao ser calculados a partir de peso estimado, tempo de maquina, filamento fisico, preco por kg e outros custos aprovados.

Remover componente de `variant_filaments` no Admin afeta somente configuracoes futuras. Pedidos ja criados preservam snapshot proprio em `order_item_filaments`.

## Estoque e filamento fisico

Nao ha controle de estoque unitario de produtos nesta fase. A disponibilidade publica depende de `products.is_active` e `product_variants.is_active`.

`materials` e `colors` sao conceitos logicos de catalogo/producao. Eles nao representam marca de filamento, carretel fisico, lote, preco de compra ou peso disponivel.

`materials.is_active = false` e `colors.is_active = false` devem ser tratados como indisponibilidade para novas escolhas futuras. A pagina publica de produto nao deve ocultar, renomear ou marcar como inativo um componente ja usado por uma receita existente.

Filamento fisico, inventario, lotes, custo por kg e reserva de material permanecem planejados para modulo operacional futuro.

## Evolucao planejada pre-go-live

- A Fase 17.1 está concluída e validada em produção: slugs administrativos são derivados no servidor, recebem sufixo determinístico em colisões e permanecem estáveis quando o nome muda. O operador não precisa preencher slug, o browser não consegue alterá-lo por POST comum, nenhuma migration foi necessária e nenhum dado existente é recalculado em lote.
- A Fase 17 planeja um unico perfil logistico por produto e uma interface simplificada de caixas; a regra atual de override por configuracao continua implementada ate essa mudanca futura.
- Materiais e cores permanecem dados de producao. Cor comercial escolhida pelo cliente sera conceito separado e planejado para a Fase 17.3.
- A Fase 18 planeja conta opcional de cliente com checkout convidado preservado. Autenticacao de cliente nao concede autorizacao administrativa.
- A retomada de pagamento para cliente autenticado e uma funcionalidade futura; `pending_payment` continua como estado interno necessario.

## Imagens

Imagens publicas de catalogo usam caminhos relativos em `product_images.storage_path` e arquivos no bucket `product-images` do Supabase Storage.

Imagens podem ser gerais do produto ou especificas de uma variante. A pagina de produto prioriza imagens da variante selecionada; se nao existirem, usa imagens gerais do produto; se nenhuma imagem publica estiver disponivel, usa placeholder visual da PrintLab.

Upload e gestao administrativa de imagens existem somente no painel Admin, por signed upload URL gerada pelo backend apos sessao administrativa e validacao de origem. A finalizacao confirma metadata por `GET /storage/v1/object/info/{bucket}/{path}` e usa `size` e `content_type` do JSON retornado. Nao ha policy publica de escrita em Storage.
