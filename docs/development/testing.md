# Testes

Status: estrategia PLANEJADA; testes de fundacao, banco, catalogo, variantes, carrinho, dados de checkout e frete IMPLEMENTADOS.

## Estrategia futura

- Unit tests para regras de dominio.
- Handler tests para endpoints HTTP.
- Integration tests para fluxos entre pacotes.
- Database tests para consultas e migrations.
- Testes opcionais de integracao PostgreSQL usando `TEST_DATABASE_URL`.
- Handler tests de catalogo e detalhe de produto sem rede.
- Unit tests de service, slug, formatacao de dinheiro, preco efetivo, peso, tempo e URLs de imagem.
- Testes de cliente SuperFrete com `httptest`, sem chamada real de internet em `go test ./...`.
- Integration contract tests reais para SuperFrete e InfinitePay somente como opt-in controlado, com credenciais de sandbox.
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
- Token de carrinho cobre entropia/tamanho esperado, tokens diferentes, hash SHA-256 e token bruto diferente do hash.
- Cookie de carrinho cobre `HttpOnly`, `SameSite=Lax`, `Path=/`, `Secure`, `MaxAge` e `Expires`.
- Service de carrinho cobre carrinho inexistente, expirado, criacao no primeiro add, produto sem variante, variante obrigatoria, variante valida, variante inativa, variante de outro produto, produto inativo, produto inexistente, add/increment, limite 99 e quantidade invalida.
- Carrinho cobre disponibilidade: produto ativo + variante ativa, produto inativo, variante inativa e produto que passa a exigir variante.
- Dinheiro no carrinho cobre `price * quantity`, subtotal apenas de itens disponiveis e protecao contra overflow.
- Mutacoes de item cobrem escopo por `cart_id + item_id`.
- Handlers de carrinho cobrem `GET /carrinho` vazio, `POST /carrinho/adicionar`, quantidade invalida, produto inexistente, variante invalida, update, remove e origem cross-site invalida.
- CPF cobre valor valido formatado, valor valido sem mascara, primeiro digito incorreto, segundo digito incorreto, comprimento invalido, letras, whitespace e todas as sequencias repetidas.
- Telefone cobre formatos brasileiros com DDD, com `+55` opcional e rejeicoes conservadoras.
- CEP cobre formato com hifen, sem hifen, letras e comprimento invalido.
- UF cobre normalizacao para uppercase, UFs validas e rejeicao de valores inexistentes.
- Service de dados de checkout cobre carrinho ausente, carrinho vazio, item indisponivel, validacao completa, normalizacao, dados validos, erro de repository e ausencia de persistencia em entrada invalida.
- Repository de dados de checkout cobre uso de transacao, upserts 1:1 por `cart_id`, colunas explicitas e teste opcional de rollback com `TEST_DATABASE_URL`.
- Handlers de `/checkout/dados` cobrem redirect sem carrinho, GET com carrinho, POST valido, CPF invalido, endereco invalido, origem cross-site invalida e resposta generica sem detalhes internos.
- Packaging de frete cobre rotacao, eixo incompativel apesar de volume suficiente, menor caixa, desempates deterministicos, ausencia de caixa, fallback de perfil produto/variante, conversoes de unidade, arredondamento conservador cm -> mm, dinheiro em centavos, peso final e overflow.
- Cliente SuperFrete cobre Authorization Bearer, `User-Agent`, `Content-Type`, endpoint `/api/v0/calculator`, payload `products`, payload `package`, parsing 200, erros HTTP, timeout, JSON invalido e ausencia de token em mensagens de erro.
- Service de frete cobre pre-condicoes de carrinho/dados, ausencia de perfil logistico, ausencia de caixa, caixa sem encaixe, duas chamadas SuperFrete, uso de caixa real na cotacao final, preco revalidado, persistencia de selecao, selecao expirada, hash divergente e servico indisponivel.
- Repository de frete cobre queries explicitamente escopadas por carrinho e teste opcional com `TEST_DATABASE_URL` para perfis, caixas ativas, upsert de selecao e ignorar caixa inativa.
- Handlers de `/checkout/frete` cobrem redirect sem carrinho, redirect sem dados, estados sem perfil/caixa, cotacao valida, `Cache-Control: private, no-store`, POST cross-site rejeitado, selecao valida e POST que ignora preco malicioso do navegador.

## Teste de integracao PostgreSQL opcional

O teste opcional de ping usa exclusivamente `TEST_DATABASE_URL`. Se a variavel nao existir, o teste e ignorado.

Nunca use `DATABASE_URL` de producao automaticamente em testes.

O teste opcional faz apenas `Ping` com timeout curto e nao altera dados.

Repository tests que consultem PostgreSQL real devem usar somente ambiente explicito de teste, como `TEST_DATABASE_URL`, e nunca a `DATABASE_URL` de producao automaticamente.

O teste opcional de transacao de `internal/customers` usa `TEST_DATABASE_URL` para confirmar que falha no upsert de endereco nao deixa contato persistido parcialmente.

## Praticas recomendadas

- Testes deterministico.
- Tabelas de teste para variacoes de regras.
- Fixtures pequenas e explicitas.
- Testes de erro tao importantes quanto testes de sucesso.
- Webhooks devem ter testes de idempotencia antes de producao.
