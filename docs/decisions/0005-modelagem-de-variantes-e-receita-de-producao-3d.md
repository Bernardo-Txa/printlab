# ADR-0005 — Modelagem de variantes e receita de producao 3D

Status: aprovado
Data: 2026-09-09

## Contexto

A PrintLab precisa representar produtos impressos em 3D com variantes publicas, imagens e informacoes operacionais de producao sem iniciar carrinho, checkout, estoque fisico ou custos derivados.

O AMS Lite e produtos multicoloridos tornam insuficiente gravar apenas uma cor ou um material diretamente na variante. Uma variante pode usar varios componentes de filamento, cada um com material logico, cor logica, peso estimado e rotulo operacional.

A producao inicial e predominantemente sob demanda. A fase atual precisa persistir ingredientes para calculo futuro de custo, mas nao deve persistir custo calculado, margem ou lucro.

## Decisao

Modelar o dominio como:

```text
products
  |
  v
product_variants
  |
  v
variant_filaments
  |        |
  v        v
materials colors
```

`products.price_cents` continua sendo o preco-base comercial. `product_variants.price_cents` pode sobrescrever esse valor; quando for `null`, o preco efetivo da variante usa o preco-base do produto.

`variant_filaments.estimated_weight_mg` armazena peso estimado em miligramas como inteiro. `product_variants.print_time_minutes` armazena tempo estimado de maquina em minutos como inteiro.

`materials` e `colors` representam catalogo/producao logica, nao filamento fisico comprado, lote, carretel ou estoque.

`product_images` referencia imagens publicas no bucket `product-images` do Supabase Storage por caminho relativo. Imagens podem ser gerais do produto ou especificas de uma variante. Upload permanece futuro e nao ha policy publica de escrita.

## Alternativas consideradas

- `color_id` direto em `product_variants`: rejeitado porque nao suporta multicolor.
- `material_id` direto em `product_variants`: rejeitado porque nao suporta multimaterial.
- Coluna `filament_grams` como decimal/float: rejeitada para evitar representacao canonica com ponto flutuante.
- Estoque por variante nesta fase: rejeitado porque a operacao inicial e sob demanda e filamento fisico e outro conceito.
- Sistema generico `option/value`: adiado por ser flexivel demais para as regras ja conhecidas de producao 3D.
- Persistir `production_cost`, `material_cost`, `machine_cost`, margem ou lucro: rejeitado porque sao derivados de peso, tempo e custos futuros de insumo.

## Consequencias

- A modelagem suporta naturalmente impressao multicolorida e multimaterial.
- O backend consegue calcular preco efetivo sem confiar no navegador.
- O catalogo pode exibir produtos sem variantes, preservando compatibilidade com a Fase 4.
- A pagina de produto pode selecionar variante por slug via query string, sem JavaScript obrigatorio.
- Custos futuros poderao ser calculados a partir de peso, tempo de maquina e filamento fisico quando esse modulo existir.
- A estrutura adiciona tabelas e joins, mas evita um modelo artificialmente generico antes de haver necessidade real.
- Estoque fisico, upload/admin, carrinho e checkout continuam separados para fases futuras.

## Referencias

- [Schema](../database/schema.md)
- [Catalogo](../product/catalog.md)
- [Regras de negocio](../product/business-rules.md)
- [Supabase](../integrations/supabase.md)
