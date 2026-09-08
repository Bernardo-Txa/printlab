# Testes

Status: estrategia PLANEJADA; testes de health check, homepage e static CSS IMPLEMENTADOS.

## Estrategia futura

- Unit tests para regras de dominio.
- Handler tests para endpoints HTTP.
- Integration tests para fluxos entre pacotes.
- Database tests para consultas e migrations.
- Integration contract tests para SuperFrete e InfinitePay quando contratos oficiais forem usados.
- Testes criticos de checkout.
- Testes de idempotencia.
- Testes de pagamento.
- Testes de calculo financeiro.

## Prioridade

Codigo relacionado a dinheiro, pedidos, frete e pagamento tem prioridade alta de testes.

Valores monetarios nunca deverao utilizar `float32` ou `float64` como representacao canonica. A estrategia segura inicial deve considerar representacao em centavos com inteiros ou tipo decimal apropriado, mas nenhuma biblioteca externa deve ser escolhida sem necessidade real.

## Comandos atuais

```sh
templ generate
npm run css:build
go test ./...
go vet ./...
go build ./...
```

## Testes implementados nesta fase

- `GET /health` retorna HTTP 200 e corpo `ok`.
- `GET /` retorna HTTP 200 com `Content-Type: text/html; charset=utf-8`.
- A homepage contem identificacao da PrintLab e skip link.
- `/static/css/app.css` e servido.
- `/static/images/branding/logo-printlab-primary.png` e servido com `Content-Type` de PNG.
- Diretorios de `/static/` nao sao listados.
- Rotas desconhecidas retornam 404.

## Praticas recomendadas

- Testes deterministico.
- Tabelas de teste para variacoes de regras.
- Fixtures pequenas e explicitas.
- Testes de erro tao importantes quanto testes de sucesso.
- Webhooks devem ter testes de idempotencia antes de producao.
