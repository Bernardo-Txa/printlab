# Retirada no local

Status: Concluida; validada manualmente em producao.

## Objetivo

Adicionar uma modalidade de entrega gratuita para retirada no local, paralela ao fluxo existente de envio via SuperFrete.

## Escopo implementado

- Checkout exibe a escolha “Como você deseja receber?” com `shipping` e `pickup`.
- `shipping` preserva a cotacao SuperFrete, selecao de servico, caixa real e validacoes existentes.
- `pickup` salva selecao server-side com `delivery_method=pickup`, frete zero, caixa nula e campos de provider/servico vazios.
- Retirada nao chama SuperFrete, nao planeja embalagem, nao lista caixas e nao exige perfil logistico do produto.
- Revisao e pedido validam a modalidade explicitamente; preco zero nao e usado como inferencia.
- Pedido e Admin exibem “Retirada no local” e “Grátis”, sem transportadora, servico ou caixa.
- Endereco exato de retirada nao e exibido publicamente nesta versao.

## Banco

Migration criada:

- `supabase/migrations/20260920234746_add_pickup_delivery_method.sql`

A migration adiciona `delivery_method` em `cart_shipping_selections` e `order_shipping_details`, com default `shipping` para preservar dados existentes. Constraints condicionais separam os campos validos de envio e retirada.

## Validacao manual em producao

- Carrinho com produto sem perfil logistico consegue seguir por retirada.
- Etapa de entrega nao mostra erro de perfil/caixa para retirada.
- Revisao mostra “Retirada no local” e “Grátis”.
- Pedido e Admin mostram retirada sem transportadora/servico/caixa.
- Envio via SuperFrete continua cotando e persistindo normalmente.
- Os dois fluxos de entrega foram testados em producao.

## Fora do escopo

- Endereco publico de retirada.
- Agendamento de horario.
- Alteracao de caixa, algoritmo de embalagem ou SuperFrete.
- Etiqueta, rastreio ou multi-volume.
