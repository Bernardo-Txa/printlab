# Testes

Status: estrategia PLANEJADA; testes de fundacao HTTP, config e database IMPLEMENTADOS.

## Estrategia futura

- Unit tests para regras de dominio.
- Handler tests para endpoints HTTP.
- Integration tests para fluxos entre pacotes.
- Database tests para consultas e migrations.
- Testes opcionais de integracao PostgreSQL usando `TEST_DATABASE_URL`.
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
npx supabase --version
```

## Testes implementados nesta fase

- `GET /health` retorna HTTP 200 e corpo `ok`.
- `GET /ready` retorna HTTP 503 quando `DATABASE_URL` nao esta configurada.
- `GET /` retorna HTTP 200 com `Content-Type: text/html; charset=utf-8`.
- A homepage contem identificacao da PrintLab e skip link.
- `/static/css/app.css` e servido.
- `/static/images/branding/logo-printlab-primary.png` e servido com `Content-Type` de PNG.
- Diretorios de `/static/` nao sao listados.
- Rotas desconhecidas retornam 404.
- `DB_MAX_CONNS` ausente usa default `4`.
- `DB_MAX_CONNS` valido e aceito.
- `DB_MAX_CONNS` invalido e erro de configuracao.
- `DATABASE_URL` ausente e permitido nesta fase.
- `DATABASE_URL` invalida gera erro seguro sem expor senha.
- `pgxpool` usa `MaxConns`, `MinConns = 0` e `pgx.QueryExecModeExec`.

## Teste de integracao PostgreSQL opcional

O teste opcional de ping usa exclusivamente `TEST_DATABASE_URL`. Se a variavel nao existir, o teste e ignorado.

Nunca use `DATABASE_URL` de producao automaticamente em testes.

O teste opcional faz apenas `Ping` com timeout curto e nao altera dados.

## Praticas recomendadas

- Testes deterministico.
- Tabelas de teste para variacoes de regras.
- Fixtures pequenas e explicitas.
- Testes de erro tao importantes quanto testes de sucesso.
- Webhooks devem ter testes de idempotencia antes de producao.
