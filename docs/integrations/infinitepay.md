# InfinitePay

Status: IMPLEMENTACAO SERVER-SIDE CONCLUIDA; validacao real de link e pagamento pendente.

## Escopo implementado

A PrintLab inicia pagamento hospedado pela InfinitePay a partir de um pedido ja criado e congelado.

Fluxo:

```text
GET /pedido/{uuid}
  -> POST /pedido/{uuid}/pagar
  -> PrintLab cria ou reutiliza checkout server-side
  -> Redirect 303 para https://checkout.infinitepay.com.br/...
  -> InfinitePay redireciona para GET /pagamento/retorno
  -> PrintLab chama payment_check server-side
  -> Pedido muda para paid somente se success=true, paid=true e amount=orders.total_cents
```

Webhooks nao foram implementados nesta fase.

## Configuracao

Variavel de ambiente:

- `INFINITEPAY_HANDLE`: InfiniteTag/handle da conta, sem `$`.

O valor real nao deve ser versionado. Sem `INFINITEPAY_HANDLE`, a aplicacao continua operando catalogo, carrinho, checkout, pedido e frete; apenas o CTA de pagamento fica indisponivel de forma segura.

`SITE_URL` deve apontar para a URL publica HTTPS da aplicacao para montar `redirect_url` internamente. O navegador nunca envia `redirect_url`.

## Contratos usados

Base URL interna:

```text
https://api.checkout.infinitepay.io
```

Criacao de link:

```text
POST /links
```

Payload enviado:

- `handle`;
- `redirect_url`;
- `order_nsu`;
- `items`;
- `customer`;
- `address`.

Nao enviamos `webhook_url` nesta fase.

Confirmacao no retorno:

```text
POST /payment_check
```

Payload enviado:

- `handle`;
- `order_nsu`;
- `transaction_nsu`;
- `slug`.

O retorno do navegador fornece apenas identificadores externos. Ele nunca confirma pagamento por si so.

## Dados enviados

`order_nsu` e derivado do UUID canonico do pedido. Ele nao contem PII e nao substitui autenticacao.

Itens sao montados somente a partir do snapshot imutavel do pedido:

- descricao publica do produto e variante;
- quantidade;
- preco unitario em centavos.

Frete entra como item separado somente quando `shipping_price_cents > 0`, por exemplo `Frete - SEDEX`.

Nao sao enviados SKU, tempo de impressao, filamentos, IDs internos, caixa fisica, dimensoes, CPF, cidade ou UF.

Cliente:

- nome;
- e-mail;
- telefone.

Endereco:

- `cep`;
- `street`;
- `neighborhood`;
- `number`;
- `complement`.

## Persistencia

Pagamentos sao registrados em `public.order_payments`, 1:1 com `public.orders`.

Status internos de pagamento:

- `pending`;
- `paid`.

Status de pedido relacionados:

- `pending_payment`;
- `paid`.

Falhas de API, abandono do checkout, retorno sem pagamento confirmado ou `paid=false` nao marcam pagamento como falho e nao mudam o pedido para pago.

## Idempotencia

`POST /pedido/{id}/pagar` reutiliza um checkout pendente existente quando houver `checkout_url` valida.

Se o pedido ja estiver `paid`, a rota redireciona de volta para `/pedido/{id}` e nao cria novo checkout.

`GET /pagamento/retorno` e idempotente: retornos repetidos para pagamento ja confirmado mantem o pedido pago sem duplicar registros.

## Seguranca

- `POST /pedido/{id}/pagar` valida `Origin`/`Referer`.
- O backend compara o total do payload com `orders.total_cents` antes de chamar a InfinitePay.
- Checkout URL so e aceita com scheme `https` e host `checkout.infinitepay.com.br`.
- `GET /pagamento/retorno` usa `Cache-Control: private, no-store`.
- A aplicacao ignora `receipt_url` e `capture_method` vindos do navegador.
- A confirmacao depende apenas de `payment_check` server-side.
- A comparacao financeira usa `amount` contra `orders.total_cents`; `paid_amount` e persistido, mas pode divergir por juros/tarifas.
- Logs nao devem conter checkout URL, e-mail, telefone, endereco, query params completos, transaction NSU ou secrets.

## Limitacao sem webhook

Se o comprador pagar e fechar a InfinitePay antes de clicar em continuar/retornar, a PrintLab pode permanecer temporariamente em `pending_payment`. A Fase 11 devera resolver isso com webhooks validados e idempotentes.

## Validacao local

Validacoes automatizadas:

```sh
go test ./...
go vet ./...
go build ./...
```

Validacao manual local de rotas, sem transacao real:

```sh
curl -i http://localhost:8080/pedido/00000000-0000-0000-0000-000000000000
curl -i http://localhost:8080/pagamento/retorno
```

Para validar link real, configure `DATABASE_URL`, `SITE_URL` HTTPS e `INFINITEPAY_HANDLE` em ambiente controlado. Nao realizar pagamento real sem roteiro aprovado.

## Estado de validacao real

- Checkout/link real InfinitePay: pendente.
- Pagamento real confirmado via `payment_check`: pendente.
- Webhook: fora do escopo desta fase.
