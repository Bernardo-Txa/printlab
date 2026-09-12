# ADR-0012 — Acompanhamento publico por identificador aleatorio

Status: Aprovado

Data: 2026-09-12

## Contexto

A PrintLab precisa permitir acompanhamento basico do pedido sem login. `orders.id` ja aparece em `/pedido/{id}` para a etapa de pagamento, mas nao deve se tornar o identificador publico permanente de acompanhamento. `order_number` e sequencial e serve apenas como referencia humana.

O acompanhamento tambem precisa separar estados de pagamento, producao e envio. Pagamento e confirmado pela integracao InfinitePay; producao e envio pertencem ao fluxo operacional da PrintLab.

## Decisao

Cada pedido recebe `orders.public_tracking_id`, um UUID aleatorio persistido, unico e obrigatorio. A rota publica de acompanhamento usa:

```text
GET /acompanhar/{public_tracking_id}
```

O link e tratado como capability URL. A pagina renderiza somente uma view model minimizada com `order_number`, data do pedido, status de pagamento, status de producao, status de envio e transportadora/servico comercial quando disponivel.

Os estados operacionais ficam em `public.order_fulfillment`, relacao 1:1 com `orders`:

- `production_status`: `waiting`, `in_production`, `completed`;
- `shipping_status`: `waiting`, `preparing`, `shipped`, `delivered`.

O banco impede envio diferente de `waiting` enquanto `production_status` nao estiver `completed`.

## Alternativas consideradas

- Usar `order_number` como rota publica: rejeitado por ser sequencial e enumeravel.
- Usar `orders.id` como link de acompanhamento permanente: rejeitado para separar a experiencia de pagamento da URL de acompanhamento.
- Guardar apenas hash do tracking ID: rejeitado nesta fase porque fases futuras podem precisar enviar o link por e-mail, WhatsApp ou suporte.
- Exibir itens/valores no acompanhamento: rejeitado por minimizacao de dados, ja que o link funciona como capacidade.
- Criar endpoints publicos de mutacao de fulfillment: rejeitado; atualizacao operacional pertence ao painel administrativo futuro.

## Consequencias

- O acompanhamento publico nao exige login, mas deve minimizar informacoes.
- O link precisa ser protegido contra cache, indexacao e vazamento por referrer.
- Logs nao devem registrar `public_tracking_id`.
- Admin, etiqueta SuperFrete, codigo de rastreio e atualizacao operacional ficam fora da Fase 12.

## Referencias

- [Acompanhamento seguro do pedido](../product/order-tracking.md)
- [Pedidos](../product/orders.md)
- [Schema de banco](../database/schema.md)
- [Seguranca](../architecture/security.md)
