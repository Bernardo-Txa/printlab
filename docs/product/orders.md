# Pedidos

Status: REVISAO E CRIACAO DE PEDIDOS IMPLEMENTADAS; pagamento PLANEJADO.

Pedidos representam compras confirmadas a partir de um carrinho anonimo validado. Nesta fase, o pedido e criado antes do pagamento e fica com status `pending_payment`.

## Fluxo implementado

```text
Carrinho
  -> Dados
  -> Frete
  -> Revisao
  -> Confirmar pedido
  -> Pedido criado
  -> Pagamento futuro
```

Rotas:

- `GET /checkout/revisao`: revisao SSR de produtos, dados, entrega, frete e total.
- `POST /checkout/revisao`: confirma o pedido de forma transacional.
- `GET /pedido/{id}`: exibicao SSR do pedido criado por UUID.

`/pedido/{id}` aceita somente UUID. `order_number` e sequencial e apropriado para referencia humana, como `#1001`, mas nao e mecanismo de autorizacao.

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
- insere `orders`, `order_customer_details`, `order_shipping_addresses`, `order_shipping_details`, `order_items` e `order_item_filaments`;
- marca `carts.converted_at`;
- remove `cart_items`, `cart_customer_details`, `cart_shipping_addresses` e `cart_shipping_selections`;
- faz `COMMIT`;
- expira o cookie `printlab_cart`;
- redireciona 303 para `/pedido/{uuid}`.

Se qualquer insert ou validacao falhar, a transacao faz rollback e nao persiste pedido parcial.

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

## Privacidade

O pedido preserva PII necessaria para operacao futura, mas a rota `/pedido/{id}` nao exibe CPF completo, endereco completo, telefone ou e-mail completo. Como UUID de pedido nao e autenticacao forte, a pagina publica mostra somente resumo do pedido, itens, frete, valores e status humano.

Erros publicos nao retornam detalhes PostgreSQL, connection strings ou dados pessoais. Logs podem registrar pedido criado, UUID, `order_number`, status e conversao de carrinho; nao devem registrar CPF, e-mail, telefone, endereco ou token do carrinho.

## Limites

- Nao ha InfinitePay implementado.
- Nao ha processamento de pagamento.
- Nao ha webhook de pagamento.
- Nao ha etiqueta, postagem ou rastreio.
- Nao ha painel administrativo.
- O unico status criado pelo checkout nesta fase e `pending_payment`.
