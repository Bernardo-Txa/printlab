# Checkout

Status: PLANEJADO.

O checkout devera transformar uma intencao de compra em pedido, com validacao server-side de produtos, endereco, frete e pagamento.

## Comportamento planejado

- Permitir checkout sem conta obrigatoria.
- Coletar dados minimos de cliente e endereco.
- Validar itens e disponibilidade.
- Recalcular subtotal e total no backend.
- Validar frete antes da criacao do pedido.
- Criar pedido antes ou durante o inicio do pagamento, conforme decisao futura.

## Regras obrigatorias

- O frontend nao determina preco final.
- O frontend nao confirma pagamento.
- Redirect de pagamento nao confirma pedido pago.
- Webhook validado sera necessario para confirmacao server-side.

## Limites

- Nao ha checkout implementado nesta fase.
- Nao ha contrato aprovado com SuperFrete ou InfinitePay.
- Nao ha schema aprovado para pedidos, pagamentos ou envios.
