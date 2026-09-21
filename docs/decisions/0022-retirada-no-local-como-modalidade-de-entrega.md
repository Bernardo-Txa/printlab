# ADR 0022 — Retirada no local como modalidade explicita de entrega

Status: aceita

## Contexto

O checkout ja possuia selecao de frete via SuperFrete, persistida em `cart_shipping_selections` e congelada em `order_shipping_details`. A nova modalidade de retirada no local precisa coexistir com envio sem depender de caixa, perfil logistico ou cotacao externa.

Inferir retirada por `shipping_price_cents = 0` seria ambiguo: envios promocionais ou descontos futuros tambem poderiam resultar em frete zero.

## Decisao

Persistir a modalidade em campo explicito `delivery_method`, com valores:

- `shipping`;
- `pickup`.

Para `shipping`, as regras existentes de SuperFrete continuam autoritativas: caixa real, provider, servico, pacote, `input_hash` e validade.

Para `pickup`, o backend grava preco zero, provider/servico vazios, caixa nula e pacote zerado. A revisao e a criacao do pedido validam essa combinacao explicitamente.

## Consequencias

- Pedidos historicos indicam claramente se foram enviados ou retirados.
- O Admin pode exibir retirada sem transportadora, servico ou caixa.
- O frontend nao controla preco de retirada; o servidor determina sempre zero.
- O modelo permite adicionar novas modalidades futuras sem sobrecarregar campos da SuperFrete.
- Foi necessaria migration condicional nas tabelas de selecao de carrinho e snapshot de pedido.
