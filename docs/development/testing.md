# Testes

Status: estrategia PLANEJADA; testes de fundacao, banco, catalogo e variantes IMPLEMENTADOS.

## Estrategia futura

- Unit tests para regras de dominio.
- Handler tests para endpoints HTTP.
- Integration tests para fluxos entre pacotes.
- Database tests para consultas e migrations.
- Testes opcionais de integracao PostgreSQL usando `TEST_DATABASE_URL`.
- Handler tests de catalogo e detalhe de produto sem rede.
- Unit tests de service, slug, formatacao de dinheiro, preco efetivo, peso, tempo e URLs de imagem.
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
- `GET /produtos` com catalogo vazio retorna HTTP 200 em teste com service fake.
- `GET /produtos` com produtos retorna HTTP 200.
- Filtro `categoria=<slug>` e encaminhado ao service.
- `GET /produtos/{slug}` retorna HTTP 200 para produto encontrado.
- `GET /produtos/{slug}?variante=<slug>` encaminha a variante ao service.
- Slug invalido de variante retorna HTTP 404 antes de consultar o service.
- Produto com variante selecionada renderiza preco efetivo e canonical do produto sem query string.
- Produto inexistente retorna HTTP 404.
- Slug invalido nao consulta service/repository.
- Erro de repository/database retorna HTTP 503 sem detalhes internos.
- Formatacao BRL cobre centavos, centenas e milhares sem `float`.
- Preco efetivo cobre fallback para preco-base, override de variante e override zero.
- Peso de filamento soma componentes em miligramas.
- Formatacao de peso cobre `42000 -> 42 g`, `3250 -> 3,25 g` e `125500 -> 125,5 g`.
- Formatacao de tempo cobre `60 -> 1h`, `275 -> 4h 35min` e `45 -> 45min`.
- Selecao de variante cobre default ativa, primeira ativa sem default, explicita valida, inexistente, inativa, de outro produto e produto sem variantes.
- Fallback de imagem cobre imagem de variante, imagem geral de produto e placeholder quando `SUPABASE_URL` nao esta disponivel.
- URL publica de Storage e testada sem baixar arquivos do Supabase.

## Teste de integracao PostgreSQL opcional

O teste opcional de ping usa exclusivamente `TEST_DATABASE_URL`. Se a variavel nao existir, o teste e ignorado.

Nunca use `DATABASE_URL` de producao automaticamente em testes.

O teste opcional faz apenas `Ping` com timeout curto e nao altera dados.

Repository tests que consultem PostgreSQL real devem usar somente ambiente explicito de teste, como `TEST_DATABASE_URL`, e nunca a `DATABASE_URL` de producao automaticamente.

## Praticas recomendadas

- Testes deterministico.
- Tabelas de teste para variacoes de regras.
- Fixtures pequenas e explicitas.
- Testes de erro tao importantes quanto testes de sucesso.
- Webhooks devem ter testes de idempotencia antes de producao.
