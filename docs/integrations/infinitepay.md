# InfinitePay

Status: IMPLEMENTACAO SERVER-SIDE CONCLUIDA; diagnostico seguro implementado; validacao real de link e pagamento pendente.

## Escopo implementado

A PrintLab inicia pagamento hospedado pela InfinitePay a partir de um pedido ja criado e congelado.

Fluxo:

```text
GET /pedido/{uuid}
  -> POST /pedido/{uuid}/pagar
  -> PrintLab cria ou reutiliza checkout server-side
  -> Redirect 303 para checkout hospedado InfinitePay validado por allowlist
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

Observacao da Fase 10.1: tambem foi encontrada documentacao da Central de Ajuda da InfinitePay citando `https://api.infinitepay.io/invoices/public/checkout/links`. A implementacao nao troca endpoint, nao faz fallback e nao executa duas chamadas nesta fase. A discrepancia fica como ponto de validacao controlada usando os diagnosticos seguros abaixo.

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

## Hosts de checkout

A documentacao publica possui exemplos historicos ou alternativos usando `checkout.infinitepay.com.br`.

A validacao real em producao, em 2026, confirmou que `POST https://api.checkout.infinitepay.io/links` retorna atualmente checkout URL com host `checkout.infinitepay.io`.

A PrintLab aceita ambos os hosts por allowlist explicita:

- `checkout.infinitepay.io`;
- `checkout.infinitepay.com.br`.

Qualquer outro host continua rejeitado, incluindo subdominios ou sufixos parecidos como `evil.infinitepay.io`, `checkout.infinitepay.io.evil.com`, `infinitepay.io` e `api.checkout.infinitepay.io`. A URL tambem deve usar `https`.

## Seguranca

- `POST /pedido/{id}/pagar` valida `Origin`/`Referer`.
- O backend compara o total do payload com `orders.total_cents` antes de chamar a InfinitePay.
- Checkout URL so e aceita com scheme `https` e host autorizado pela allowlist explicita.
- `GET /pagamento/retorno` usa `Cache-Control: private, no-store`.
- A aplicacao ignora `receipt_url` e `capture_method` vindos do navegador.
- A confirmacao depende apenas de `payment_check` server-side.
- A comparacao financeira usa `amount` contra `orders.total_cents`; `paid_amount` e persistido, mas pode divergir por juros/tarifas.
- Logs nao devem conter checkout URL, e-mail, telefone, endereco, query params completos, transaction NSU ou secrets.

## Diagnostico seguro

A Fase 10.1 adiciona um erro estruturado interno para falhas do provider InfinitePay. Ele preserva somente campos operacionais seguros:

- `provider=infinitepay`;
- `operation=create_checkout` ou `operation=payment_check`;
- `status=<codigo HTTP>` quando houver resposta HTTP;
- `category=<categoria segura>`;
- `host=<host>` apenas para checkout URL invalida, quando util e sem caminho/query.

Categorias seguras:

- `network_error`;
- `timeout`;
- `http_400`;
- `http_401`;
- `http_403`;
- `http_404`;
- `http_409`;
- `http_422`;
- `http_429`;
- `http_5xx`;
- `invalid_json`;
- `invalid_checkout_url`;
- `unknown`.

Quando a InfinitePay responde com status nao 2xx, o backend le apenas um body pequeno e tenta extrair apenas campos genericos sanitizados como `message`, `error` e `code`. Corpo bruto, payload completo, cliente, e-mail, telefone, endereco, checkout URL completa, `transaction_nsu`, `invoice_slug` e secrets nao podem aparecer em `Error()` nem em logs.

Exemplos de logs seguros:

```text
payment checkout unavailable provider=infinitepay operation=create_checkout status=422 category=http_422
payment confirmation unavailable provider=infinitepay operation=payment_check status=503 category=http_5xx
payment checkout unavailable provider=infinitepay operation=create_checkout category=timeout
payment checkout unavailable provider=infinitepay operation=create_checkout category=invalid_checkout_url host=example.invalid
```

`payment checkout amount mismatch` continua sendo log separado e nao e classificado como indisponibilidade do provider.

## Limitacao sem webhook

Se o comprador pagar e fechar a InfinitePay antes de clicar em continuar/retornar, a PrintLab pode permanecer temporariamente em `pending_payment`. A Fase 11 devera resolver isso com webhooks validados e idempotentes.

## Validacao local

Validacoes automatizadas:

```sh
templ generate
npm run css:build
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
