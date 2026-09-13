# Catalogo

Status: Catalogo publico, perfis logisticos e gestao administrativa basica IMPLEMENTADOS; validacao real da Fase 13.3 pendente.

O catalogo apresenta produtos ativos da PrintLab com renderizacao server-side, mantendo o backend como autoridade sobre dados, preco-base e preco efetivo de variantes.

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
- Variantes ativas por produto em `public.product_variants`.
- Materiais logicos em `public.materials`.
- Cores logicas em `public.colors`.
- Receita estimada de producao em `public.variant_filaments`.
- Imagens gerais de produto e especificas de variante em `public.product_images`.
- Bucket publico `product-images` no Supabase Storage para imagens de catalogo.
- Placeholder visual de marca quando nao existe imagem renderizavel.
- Selecao publica de variante por `GET /produtos/{slug}?variante=<variant-slug>`, sem JavaScript obrigatorio.
- Preservacao de componentes de receita que referenciem material ou cor inativos.
- Perfil logistico opcional em produtos e variantes para cotacao de frete.
- Gestao administrativa SSR de categorias, produtos, variantes, receita, materiais, cores e caixas em `/admin`.

## Regras publicas

- `products.is_active = true` e obrigatorio para aparecer no catalogo.
- Produto inativo deve responder publicamente como inexistente.
- `categories.is_active = true` e obrigatorio para aparecer nos filtros.
- Produto ativo sem categoria continua aparecendo na listagem geral.
- Produto associado a categoria inativa continua podendo aparecer, mas sem exibir a categoria.
- O filtro de categoria usa slug, nunca UUID.
- Slug invalido em rota publica retorna 404.
- Variante inativa, inexistente, invalida ou pertencente a outro produto retorna 404 quando solicitada explicitamente.
- Produto sem variantes continua valido e usa o preco-base.
- Material ou cor inativos nao tornam uma receita existente invisivel.
- Erros de banco retornam mensagem generica e nao expoem detalhes internos.

## Preco-base

`products.price_cents` representa o preco-base comercial do produto.

```text
R$ 39,90 -> 3990
```

`product_variants.price_cents` e opcional. Quando preenchido, ele sobrescreve o preco-base para aquela variante. Quando `null`, o preco efetivo da variante usa `products.price_cents`.

No catalogo, quando um produto possui variantes ativas, o card usa o menor preco efetivo. Se houver variacao de preco entre variantes ativas, o texto publico usa "A partir de". Se todas as variantes efetivas tiverem o mesmo preco, o card mostra apenas o valor.

O frontend nunca envia preco autoritativo e nao calcula o preco efetivo.

No Admin, formularios recebem valores em BRL amigavel, como `39,90`, `39.90` ou `0,00`. O backend converte para centavos inteiros com parser decimal exato, rejeitando negativos, notacao cientifica, texto, mais de duas casas decimais e overflow. Dinheiro nao usa `float32` nem `float64`.

## Variantes

Ordenacao publica de variantes:

```text
is_default desc
sort_order asc
name asc
```

Quando `?variante=` nao e fornecido:

1. seleciona a variante ativa marcada como default;
2. se nao houver default, seleciona a primeira variante ativa pela ordenacao publica;
3. se nao houver variantes ativas, o produto continua sem variante selecionada.

O slug da variante e unico dentro do produto e nao substitui o slug do produto como URL canonica. A canonical da pagina continua sendo `/produtos/{slug}`.

No Admin, slugs vazios podem ser gerados deterministicamente a partir do nome somente na criacao. Em edicao, mudar slug e sempre uma decisao explicita. Conflitos de slug retornam erro amigavel, sem expor detalhe SQL.

## Receita de producao

Uma variante pode possuir varios componentes em `variant_filaments`, cada um com material, cor, peso estimado em miligramas, rotulo opcional e ordenacao.

Receitas existentes sao lidas por referencia. `materials.is_active = false` e `colors.is_active = false` nao removem componentes de `variant_filaments` nem escondem os nomes de material/cor na pagina publica do produto.

O status ativo de material e cor e usado para novas escolhas administrativas. Se uma receita existente aponta para material ou cor inativo, o Admin continua carregando a referencia atual, indica o estado inativo e permite preservar essa referencia ao editar peso, rotulo ou ordenacao. Novos componentes oferecem somente materiais e cores ativos.

Essa modelagem suporta:

- impressao multicolorida;
- impressao multimaterial;
- AMS Lite com multiplos componentes;
- calculos futuros de custo sem persistir valores derivados.

Peso e armazenado como inteiro em `estimated_weight_mg`, evitando `float`.

No Admin, peso de receita e digitado em gramas com ate tres casas decimais e convertido exatamente para miligramas. Exemplos: `12` vira `12000 mg`, `12,5` vira `12500 mg` e `0,850` vira `850 mg`.

O peso total estimado soma todos os componentes carregados da receita, inclusive componentes que referenciem material ou cor inativos.

Tempo estimado de maquina fica em `product_variants.print_time_minutes`. Ele nao representa prazo de entrega e nao deve ser apresentado ao cliente como promessa de envio.

## Perfil logistico

Produtos e variantes podem possuir perfil logistico para frete:

- `shipping_weight_g`;
- `shipping_height_mm`;
- `shipping_width_mm`;
- `shipping_length_mm`.

Esse perfil representa uma unidade preparada para acondicionamento, nao necessariamente a geometria crua da peca 3D nem a receita de filamento. Exemplo: uma peca pode medir `190 x 85 x 70 mm`, mas seu perfil protegido para envio ser `210 x 105 x 90 mm`.

A variante pode possuir override completo. Se nao possuir, a cotacao usa o perfil completo do produto. Campos parciais nao sao misturados.

Produto ou variante sem perfil logistico efetivo nao recebe estimativa ficticia no checkout de frete.

No Admin, perfil logistico e atomico: todos os quatro campos precisam estar preenchidos ou todos precisam ficar vazios. A variante com perfil ausente herda o perfil completo do produto; campos isolados de produto e variante nao sao combinados.

## Gestao administrativa

A Fase 13.3 usa as tabelas existentes `categories`, `products`, `product_variants`, `materials`, `colors`, `variant_filaments` e `shipping_boxes`. Nenhuma migration foi criada.

Entidades principais nao possuem hard delete no painel. O Admin ativa ou desativa registros por `is_active`, preservando referencias, carrinhos, receitas, caixas logisticas e historico. A excecao operacional e a receita atual em `variant_filaments`: componentes podem ser removidos porque pedidos antigos ja possuem snapshots historicos.

Alteracoes de produto, variante, material, cor, receita ou caixa nao atualizam pedidos existentes. `orders`, `order_items`, `order_item_filaments` e `order_shipping_details` permanecem snapshots do momento da compra.

Produtos ativos podem mostrar avisos operacionais no Admin quando nao possuem perfil logistico ou variantes, mas esses avisos nao alteram dados silenciosamente nem bloqueiam regras publicas existentes.

## Imagens

`product_images.storage_path` guarda caminho relativo no bucket `product-images`; URLs absolutas nao sao armazenadas no banco.

Estrategia publica:

1. card de catalogo usa imagem geral primaria do produto quando existir e houver `SUPABASE_URL`;
2. detalhe de produto prioriza imagens da variante selecionada;
3. se a variante nao tiver imagem, usa imagens gerais do produto;
4. se nao houver imagem renderizavel, usa placeholder visual da PrintLab.

Formatos preferidos para operacao:

- WebP como formato inicial preferencial;
- AVIF quando apropriado;
- JPEG;
- PNG quando transparencia for necessaria.

## Planejado

- Busca.
- Avaliacoes.
- Paginacao complexa.
- Upload de imagens.
- Estoque fisico e inventario de filamento.
- Custos de producao calculados.

## Limites

- Nao ha seed ficticio.
- Nao ha produto demonstrativo.
- Nao ha upload de imagem.
- Nao ha estoque unitario de produtos.
- Nao ha filamento fisico, marca, lote, carretel, preco por kg ou peso disponivel.
- Nao ha custos derivados persistidos.
- Nao ha upload, exclusao, reordenacao ou definicao administrativa de imagem primaria; imagens ficam para a Fase 13.4.
