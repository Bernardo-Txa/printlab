# Acompanhamento seguro do pedido

Status: IMPLEMENTADO NA FASE 12; atualizacao administrativa de status operacional implementada na Fase 13.2.

O acompanhamento publico permite que o cliente consulte o estado basico do pedido por um link de capacidade:

```text
GET /acompanhar/{public_tracking_id}
```

`public_tracking_id` e um UUID aleatorio persistido em `orders.public_tracking_id`. Ele nao e o `orders.id`, nao e `order_number` e nao e identificador de pagamento.

## Informacoes exibidas

A pagina SSR mostra somente:

- numero humano do pedido, como `#1004`;
- data do pedido;
- status de pagamento como `Aguardando pagamento` ou `Pagamento confirmado`;
- status publico de producao;
- status publico de envio;
- transportadora/servico comercial quando disponivel, por exemplo `Correios - PAC`.

A pagina nao mostra itens, produtos, valores, frete pago, CPF, e-mail, telefone, endereco, UUID interno do pedido, `source_cart_id`, `transaction_nsu`, `invoice_slug`, checkout URL, peso, dimensoes, filamento, material ou cor.

## Regras de status

Pagamento vem de `orders.status`:

- `pending_payment` -> `Aguardando pagamento`;
- `paid` -> `Pagamento confirmado`.

Producao vem de `order_fulfillment.production_status`:

- `waiting`;
- `in_production`;
- `completed`.

Envio vem de `order_fulfillment.shipping_status`:

- `waiting`;
- `preparing`;
- `shipped`;
- `delivered`.

Mapeamento publico:

- pedido pendente, producao/envio `waiting`: producao `Sera iniciada apos a confirmacao do pagamento`; envio `Sera preparado apos a producao`;
- pedido pago e producao `waiting`: producao `Aguardando producao`; envio `Aguardando producao`;
- pedido pago e producao `in_production`: producao `Em producao`; envio `Aguardando conclusao da producao`;
- pedido pago, producao `completed` e envio `waiting`: producao `Producao concluida`; envio `Aguardando preparacao do envio`;
- envio `preparing`: `Preparando envio`;
- envio `shipped`: `Enviado`;
- envio `delivered`: `Entregue`.

## Seguranca

`public_tracking_id` deve ser tratado como capability URL. Quem possui o link pode ver o resumo minimo do pedido.

Por isso a implementacao:

- valida UUID antes de consultar o banco;
- retorna 404 tanto para UUID invalido quanto para UUID desconhecido;
- consulta por `orders.public_tracking_id`;
- evita `SELECT *`;
- usa view model sem PII, valores ou identificadores internos;
- nao registra `public_tracking_id` em logs;
- usa `Cache-Control: private, no-store`;
- usa `X-Robots-Tag: noindex, nofollow, noarchive`;
- renderiza `<meta name="robots" content="noindex, nofollow, noarchive">`;
- usa `Referrer-Policy: no-referrer`.

## Relacao com `/pedido/{id}`

`GET /pedido/{id}` continua existindo para a experiencia imediata apos criar o pedido e para iniciar pagamento.

Quando o pedido possui `public_tracking_id`, `/pedido/{id}` mostra o link `Acompanhar pedido` apontando para `/acompanhar/{public_tracking_id}`. Esse link nao usa `orders.id` nem `order_number`.

## Limites

- Nao ha login de cliente.
- Nao ha endpoint publico para alterar producao ou envio.
- Nao ha etiqueta SuperFrete.
- Nao ha codigo de rastreio de transportadora.
- Atualizacao de status operacional pertence ao Admin autenticado, nao ao link publico de acompanhamento.
