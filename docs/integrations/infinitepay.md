# InfinitePay

Status: IMPLEMENTACAO SERVER-SIDE CONCLUIDA; diagnostico seguro, checkout real, pagamento real e webhook real validados.

## Escopo implementado

A PrintLab inicia pagamento hospedado pela InfinitePay a partir de um pedido ja criado e congelado.

Fluxo:

```text
GET /pedido/{uuid}
  -> POST /pedido/{uuid}/pagar
  -> PrintLab cria ou reutiliza checkout server-side
  -> Redirect 303 para checkout hospedado InfinitePay validado por allowlist
  -> InfinitePay redireciona para GET /pagamento/retorno ou envia POST /webhooks/infinitepay
  -> PrintLab chama payment_check server-side
  -> Pedido muda para paid somente se success=true, paid=true e amount=orders.total_cents
```

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
- `webhook_url`;
- `order_nsu`;
- `items`;
- `customer`;
- `address`.

`redirect_url` e `webhook_url` sao derivados de `SITE_URL` no backend. O navegador nunca controla esses valores.

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

`POST /webhooks/infinitepay` e idempotente: webhooks repetidos para pagamento ja confirmado retornam sucesso e nao duplicam atualizacao.

Checkouts pendentes criados antes da Fase 11 nao recebem `webhook_url` retroativamente. A aplicacao reutiliza checkout pendente valido e nao regenera link apenas para adicionar webhook.

## Hosts de checkout

A documentacao publica possui exemplos historicos ou alternativos usando `checkout.infinitepay.com.br`.

A validacao real em producao, em 2026, confirmou que `POST https://api.checkout.infinitepay.io/links` retorna atualmente checkout URL com host `checkout.infinitepay.io`.

A PrintLab aceita ambos os hosts por allowlist explicita:

- `checkout.infinitepay.io`;
- `checkout.infinitepay.com.br`.

Essa allowlist e aplicada no dominio Go por `ValidateCheckoutURL` e tambem na constraint `order_payments_checkout_url_host` do PostgreSQL. Qualquer outro host continua rejeitado, incluindo subdominios ou sufixos parecidos como `evil.infinitepay.io`, `checkout.infinitepay.io.evil.com`, `infinitepay.io` e `api.checkout.infinitepay.io`. A URL tambem deve usar `https`.

## Seguranca

- `POST /pedido/{id}/pagar` valida `Origin`/`Referer`.
- O backend compara o total do payload com `orders.total_cents` antes de chamar a InfinitePay.
- Checkout URL so e aceita com scheme `https` e host autorizado pela allowlist explicita.
- `GET /pagamento/retorno` usa `Cache-Control: private, no-store`.
- `POST /webhooks/infinitepay` nao valida `Origin`/`Referer`, porque a chamada vem do provider, mas aceita somente JSON com limite de 64 KiB e responde com `Cache-Control: no-store`.
- A aplicacao ignora `receipt_url` e `capture_method` vindos do navegador.
- O webhook e apenas gatilho; a confirmacao depende apenas de `payment_check` server-side.
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

## Webhook InfinitePay

Endpoint publico:

```text
POST /webhooks/infinitepay
```

Payload aceito inicialmente:

- `invoice_slug`;
- `transaction_nsu`;
- `order_nsu`;
- `amount`;
- `paid_amount`;
- `installments`;
- `capture_method`.

`invoice_slug`, `transaction_nsu` e `order_nsu` sao obrigatorios e normalizados. Campos como `receipt_url` e `items` podem existir no payload oficial, mas nao sao armazenados nem usados como fonte de autoridade.

Nao ha assinatura/HMAC documentada no contrato publico consultado. Por isso o endpoint nao trata o webhook como autoridade: ele usa os identificadores recebidos para chamar `POST /payment_check` server-side. Pedido e pagamento so mudam para `paid` quando o provider confirma `success=true`, `paid=true` e `amount` igual ao total congelado do pedido.

Respostas:

- sucesso confirmado ou duplicado ja pago: HTTP 200 com `{"success":true,"message":null}`;
- payload invalido, pedido inexistente, pagamento ainda pendente, falha temporaria do provider ou divergencia de valor: HTTP 400 com mensagem generica para permitir retry do provider.

Logs seguros esperados:

```text
payment webhook received
payment webhook verified order_id=<uuid>
payment webhook already_paid order_id=<uuid>
payment webhook pending order_id=<uuid>
payment webhook unavailable provider=infinitepay operation=payment_check status=503 category=http_5xx
payment webhook amount mismatch order_id=<uuid> order_number=<numero>
```

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
curl -i -X POST http://localhost:8080/webhooks/infinitepay \
  -H 'Content-Type: application/json' \
  -d '{"invoice_slug":"slug","transaction_nsu":"txn","order_nsu":"00000000-0000-0000-0000-000000000000"}'
```

Para validar link real ou webhook real, configure `DATABASE_URL`, `SITE_URL` HTTPS e `INFINITEPAY_HANDLE` em ambiente controlado. Nao realizar pagamento real sem roteiro aprovado.

## Estado de validacao real

- Checkout/link real InfinitePay: validado.
- Pagamento real confirmado via `payment_check`: validado.
- Checkout novo contendo `webhook_url`: validado em producao.
- Webhook real recebido em producao: validado.
- Confirmacao de pagamento sem redirect do comprador: validada.

## Validacao real do webhook

Validacao manual real confirmada em 2026-09-12:

- novo checkout criado apos deploy da Fase 11 continha `webhook_url`;
- pagamento Pix real foi concluido;
- comprador nao clicou em continuar na InfinitePay;
- InfinitePay enviou `POST /webhooks/infinitepay`;
- User-Agent observado: `InfinitePay/EcommerceWebhooks`;
- endpoint respondeu HTTP 200;
- pedido foi atualizado sem redirect do comprador;
- `orders.status = paid`;
- `order_payments.status = paid`;
- `amount_cents` correspondeu ao total esperado;
- `capture_method = pix`;
- `paid_at` foi preenchido.

Nao registrar em documentacao, logs ou exemplos `transaction_nsu` real, `invoice_slug`, CPF, e-mail, telefone, endereco, checkout URL ou qualquer PII.
