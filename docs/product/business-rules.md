# Regras de negocio

Status: catalogo, variantes, receita de producao, carrinho, dados de checkout e frete IMPLEMENTADOS; demais regras comerciais PLANEJADAS.

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

Antes de finalizar uma compra, o backend devera futuramente:

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
- Produto e variante sao revalidados ao adicionar e ao renderizar.
- Item indisponivel nao some silenciosamente.
- Item indisponivel nao entra no subtotal.
- Pedido e pagamento permanecem planejados.

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
- O pedido futuro devera copiar contato e endereco para snapshots definitivos antes de pagamento/envio.
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

Carrinho recalcula precos e subtotais no backend. Frete e calculado e selecionado no backend. Checkout final, descontos, total definitivo e pedidos continuam planejados e deverao recalcular valores no backend novamente.

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
- Multi-volume, etiqueta/postagem, rastreio e pedido permanecem planejados.

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

Upload de imagens, admin e policies de escrita permanecem fora do escopo desta fase.
