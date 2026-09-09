# Catalogo

Status: Catalogo e perfis logisticos IMPLEMENTADOS; operacao comercial PLANEJADA.

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

## Receita de producao

Uma variante pode possuir varios componentes em `variant_filaments`, cada um com material, cor, peso estimado em miligramas, rotulo opcional e ordenacao.

Receitas existentes sao lidas por referencia. `materials.is_active = false` e `colors.is_active = false` nao removem componentes de `variant_filaments` nem escondem os nomes de material/cor na pagina publica do produto.

O status ativo de material e cor deve ser usado para novas escolhas operacionais futuras. Como ainda nao ha admin ou formulario de configuracao nesta fase, nao existe listagem publica de novas opcoes de material/cor.

Essa modelagem suporta:

- impressao multicolorida;
- impressao multimaterial;
- AMS Lite com multiplos componentes;
- calculos futuros de custo sem persistir valores derivados.

Peso e armazenado como inteiro em `estimated_weight_mg`, evitando `float`.

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

- Admin para cadastro.
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
- Nao ha pedido ou pagamento.
