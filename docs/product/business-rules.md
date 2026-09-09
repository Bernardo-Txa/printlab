# Regras de negocio

Status: catalogo basico IMPLEMENTADO; demais regras comerciais PLANEJADAS.

Este documento registra regras de negocio previstas para a PrintLab. Ele nao representa funcionalidades prontas.

## Regras iniciais planejadas

- Produtos sao itens fisicos.
- Muitos produtos poderao ser produzidos sob demanda.
- Variantes poderao representar cor, material, tamanho ou outras opcoes aprovadas.
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

## Dinheiro

Valores monetarios nunca devem usar `float32` ou `float64` como representacao canonica.

Na Fase 4, `products.price_cents` e o preco-base comercial do produto e usa inteiro em centavos:

```text
R$ 39,90 -> 3990
```

Carrinho, checkout, descontos, frete, total e pedidos continuam planejados e deverao recalcular valores no backend.

## Producao 3D

O sistema podera armazenar informacoes de producao, como material, cor, peso estimado de filamento, tempo estimado de impressao e custos estimados. Essas informacoes ainda nao possuem schema aprovado.
