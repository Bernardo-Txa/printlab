# Regras de negocio

Status: PLANEJADO, exceto quando explicitamente indicado como implementado.

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

Valores monetarios nunca devem usar `float32` ou `float64` como representacao canonica. A estrategia final de dinheiro sera definida antes da implementacao de catalogo, carrinho e checkout.

## Producao 3D

O sistema podera armazenar informacoes de producao, como material, cor, peso estimado de filamento, tempo estimado de impressao e custos estimados. Essas informacoes ainda nao possuem schema aprovado.
