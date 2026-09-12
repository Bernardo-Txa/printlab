# Pedidos

Status: REVISAO, CRIACAO DE PEDIDOS, PAGAMENTO INFINITEPAY, WEBHOOK E ACOMPANHAMENTO SEGURO CONCLUIDOS.

Pedidos representam compras confirmadas a partir de um carrinho anonimo validado. O pedido e criado antes do pagamento, nasce com status `pending_payment` e pode mudar para `paid` somente apos validacao server-side com a InfinitePay.

## Fluxo implementado

```text
Carrinho
  -> Dados
  -> Frete
  -> Revisao
  -> Confirmar pedido
  -> Pedido criado
  -> Pagar agora
  -> InfinitePay
  -> Retorno ou webhook validado server-side
  -> Acompanhamento seguro
```

Rotas:

- `GET /checkout/revisao`: revisao SSR de produtos, dados, entrega, frete e total.
- `POST /checkout/revisao`: confirma o pedido de forma transacional.
- `GET /pedido/{id}`: exibicao SSR do pedido criado por UUID.
- `POST /pedido/{id}/pagar`: cria ou reutiliza checkout InfinitePay e redireciona para o ambiente hospedado.
- `GET /pagamento/retorno`: valida retorno com `payment_check` server-side.
- `POST /webhooks/infinitepay`: valida webhook com `payment_check` server-side.
- `GET /acompanhar/{public_tracking_id}`: acompanhamento seguro e minimizado do pedido.

`/pedido/{id}` aceita somente UUID. `order_number` e sequencial e apropriado para referencia humana, como `#1001`, mas nao e mecanismo de autorizacao.

`/acompanhar/{public_tracking_id}` aceita somente UUID aleatorio persistido em `orders.public_tracking_id`. Identificador invalido ou desconhecido retorna 404.

## Snapshot historico

Carrinho representa estado mutavel. Pedido representa estado historico.

Depois que um pedido e criado, ele nao depende do estado atual de:

- `products`;
- `product_variants`;
- `materials`;
- `colors`;
- `cart_customer_details`;
- `cart_shipping_addresses`;
- `cart_shipping_selections`;
- `shipping_boxes`.

O pedido copia valores autoritativos no momento da confirmacao:

- nome, slug, SKU, quantidade, preco unitario e subtotal de cada item;
- subtotal de produtos, frete e total em centavos;
- dados de cliente e endereco como snapshot privado;
- transportadora, servico, prazo, caixa, peso e dimensoes externas do frete;
- tempo estimado de impressao e peso estimado de filamento por unidade quando houver variante com receita;
- componentes de receita por material, cor, peso e label.

Componentes de receita continuam entrando no snapshot mesmo quando material ou cor estiverem inativos, preservando a semantica da Fase 5.1.

Snapshots operacionais de produção e embalagem são preservados no pedido, mas não são apresentados na experiência pública do comprador.

## Revisao

`GET /checkout/revisao` exige:

- carrinho valido e nao convertido;
- carrinho nao vazio;
- itens disponiveis;
- dados de contato e endereco completos;
- selecao de frete existente;
- selecao de frete nao expirada;
- `input_hash` de frete ainda compativel com carrinho, CEP, servicos, perfis logisticos e caixa.

Sem dados de checkout, redireciona para `/checkout/dados`. Sem frete valido, redireciona para `/checkout/frete`. Carrinho ausente, vazio ou com item indisponivel redireciona para `/carrinho`.

A revisao nao recota a SuperFrete. Ela apenas valida a selecao persistida e o `input_hash` atual.

Respostas da revisao usam `Cache-Control: private, no-store`.

## Fingerprint de revisao

A pagina de revisao inclui `review_fingerprint` em campo oculto.

Esse fingerprint e um SHA-256 deterministico do estado revisado e serve apenas para detectar tela antiga entre GET e POST. Ele nao e secret e nao determina preco.

O POST recalcula tudo no backend. Se o fingerprint enviado divergir, nenhum pedido e criado e a revisao e reexibida com a mensagem:

```text
Algumas informações da sua compra foram atualizadas. Revise os dados antes de confirmar.
```

## Criacao transacional

`POST /checkout/revisao`:

- valida `Origin`/`Referer`;
- resolve o carrinho por cookie;
- faz lock do carrinho com `SELECT ... FOR UPDATE`;
- revalida disponibilidade, dados, frete, `input_hash` e fingerprint;
- insere `orders`, `order_customer_details`, `order_shipping_addresses`, `order_shipping_details`, `order_fulfillment`, `order_items` e `order_item_filaments`;
- marca `carts.converted_at`;
- remove `cart_items`, `cart_customer_details`, `cart_shipping_addresses` e `cart_shipping_selections`;
- faz `COMMIT`;
- expira o cookie `printlab_cart`;
- redireciona 303 para `/pedido/{uuid}`.

Se qualquer insert ou validacao falhar, a transacao faz rollback e nao persiste pedido parcial.

`order_fulfillment` nasce junto do pedido com `production_status = 'waiting'` e `shipping_status = 'waiting'`.

## Idempotencia

`orders.source_cart_id` possui unique parcial quando nao nulo. Um carrinho gera no maximo um pedido.

Se a mesma confirmacao for enviada duas vezes, a segunda tentativa deve redirecionar para o pedido ja criado para aquele carrinho quando possivel.

## Interface publica

`GET /checkout/revisao` e `GET /pedido/{id}` exibem somente dados comercialmente relevantes para o comprador:

- produto;
- variante;
- quantidade;
- precos;
- servico de frete;
- transportadora;
- prazo;
- subtotal, frete e total.

Dados operacionais como SKU interno, tempo de impressao, consumo de filamento, componentes da receita, materiais, cores, caixa fisica, peso e dimensoes do pacote permanecem no snapshot para operacao futura, mas nao aparecem na interface publica do comprador.

Quando o pedido esta `pending_payment` e `INFINITEPAY_HANDLE` esta configurado, `/pedido/{id}` mostra o CTA real `Pagar agora`. O POST valida origem, cria ou reutiliza um checkout pendente e redireciona somente para checkout URL `https` em host autorizado pela allowlist explicita:

- `checkout.infinitepay.io`;
- `checkout.infinitepay.com.br`.

Quando o pedido esta `paid`, `/pedido/{id}` mostra `Pagamento confirmado` e nao mostra botao de pagamento.

Checkout URL, `transaction_nsu`, `invoice_slug` e detalhes tecnicos do provedor nao sao renderizados na pagina publica.

## Acompanhamento seguro

`GET /acompanhar/{public_tracking_id}` e uma pagina SSR sem JavaScript obrigatorio. Ela usa `public_tracking_id`, nao `orders.id` nem `order_number`, e trata o link como capability URL.

A view model publica contem somente:

- `order_number`;
- data do pedido;
- status de pagamento;
- status de producao;
- status de envio;
- transportadora/servico comercial quando houver.

A pagina nao exibe itens, produtos, valores, frete pago, CPF, e-mail, telefone, endereco, UUID interno do pedido, `source_cart_id`, `transaction_nsu`, `invoice_slug`, checkout URL, peso, dimensoes, filamento, material ou cor.

Headers da resposta:

- `Cache-Control: private, no-store`;
- `X-Robots-Tag: noindex, nofollow, noarchive`;
- `Referrer-Policy: no-referrer`.

O HTML tambem inclui meta robots `noindex, nofollow, noarchive`.

Status operacionais:

- producao: `waiting`, `in_production`, `completed`;
- envio: `waiting`, `preparing`, `shipped`, `delivered`.

Enquanto producao nao estiver `completed`, o banco impede envio diferente de `waiting`.

## Pagamento

`order_payments.order_nsu` e derivado do UUID canonico do pedido. Ele nao contem PII e nao deve ser tratado como mecanismo de autorizacao.

O payload enviado a InfinitePay e montado apenas a partir do snapshot do pedido:

- item de produto com descricao publica, quantidade e preco unitario em centavos;
- item de frete apenas quando o frete for maior que zero;
- nome, e-mail e telefone do cliente;
- CEP, logradouro, bairro, numero e complemento do endereco.

Antes de chamar a InfinitePay, o backend soma `quantity * unit_price_cents` e o frete e exige igualdade exata com `orders.total_cents`.

O retorno do navegador nunca confirma pagamento. `GET /pagamento/retorno` usa somente `order_nsu`, `transaction_nsu` e `slug` como entrada para `payment_check`. `POST /webhooks/infinitepay` usa `invoice_slug`, `transaction_nsu` e `order_nsu` para acionar a mesma confirmacao server-side. A aplicacao marca `orders.status = 'paid'` somente se a InfinitePay responder `success=true`, `paid=true` e `amount` igual a `orders.total_cents`.

`paid_amount` e persistido separadamente e pode divergir de `amount`.

O webhook reduz dependencia do comprador clicar em continuar/retornar depois do pagamento. Checkouts pendentes criados antes da Fase 11 nao recebem `webhook_url` retroativamente.

Validacao real em producao confirmou pagamento Pix com webhook: novo checkout continha `webhook_url`, a InfinitePay enviou `POST /webhooks/infinitepay`, o endpoint respondeu HTTP 200 e o pedido foi atualizado para `paid` sem redirect do comprador. `order_payments.status` tambem foi atualizado para `paid`, `amount_cents` correspondeu ao total esperado, `capture_method = pix` e `paid_at` foi preenchido.

## Privacidade

O pedido preserva PII necessaria para operacao futura, mas a rota `/pedido/{id}` nao exibe CPF completo, endereco completo, telefone ou e-mail completo. Como UUID de pedido nao e autenticacao forte, a pagina publica mostra somente resumo do pedido, itens, frete, valores e status humano.

O acompanhamento por `/acompanhar/{public_tracking_id}` e ainda mais restrito: nao mostra itens nem valores, apenas status basico.

Erros publicos nao retornam detalhes PostgreSQL, connection strings ou dados pessoais. Logs podem registrar pedido criado, UUID, `order_number`, status e conversao de carrinho; nao devem registrar CPF, e-mail, telefone, endereco ou token do carrinho.

## Limites

- Link real InfinitePay e pagamento real foram validados antes da Fase 11.
- Recebimento real de webhook InfinitePay e confirmacao sem redirect foram validados em producao.
- Nao ha etiqueta, postagem ou rastreio.
- Nao ha painel administrativo.
- Nao ha endpoint publico para alterar producao ou envio.
- O checkout cria pedidos com status inicial `pending_payment`; a confirmacao InfinitePay pode alterar para `paid`.
