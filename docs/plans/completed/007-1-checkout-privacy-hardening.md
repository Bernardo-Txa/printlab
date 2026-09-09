# Fase 7.1 - Hardening de privacidade e consistencia do checkout

Status: CONCLUIDA.

## Contexto

A Fase 7 implementou `GET /checkout/dados` e `POST /checkout/dados` com formulario SSR, dados de contato, CPF, telefone, endereco e persistencia transacional vinculada ao carrinho anonimo.

Esta fase curta endurece dois pontos:

- respostas HTML que podem conter PII nao devem ser armazenadas em cache;
- contato e endereco devem ser lidos como uma unidade consistente.

## Riscos identificados

- HTML de checkout pode conter nome, e-mail, telefone, CPF e endereco pre-preenchidos.
- Re-renderizacao apos erro de validacao pode devolver os valores submetidos ao browser.
- Duas queries separadas para contato e endereco poderiam observar estados diferentes em requests concorrentes.
- Um estado parcial anomalo no banco nao deve ser apresentado como checkout valido.

## Decisoes

- Usar somente `Cache-Control: private, no-store` nas respostas HTML de checkout que podem conter PII.
- Nao aplicar `no-store` globalmente a homepage, catalogo, carrinho ou assets.
- Consolidar a leitura de `cart_customer_details` e `cart_shipping_addresses` em uma unica query com `JOIN`.
- Tratar estado parcial anomalo como dados ausentes, sem preencher formulario com PII incompleta.
- Preservar a escrita transacional existente: `BEGIN`, upsert de contato, upsert de endereco e `COMMIT`.
- Nao criar migration e nao alterar schema.

## Alteracoes

- Header de cache privado para `GET /checkout/dados` quando ha carrinho valido.
- Header de cache privado para re-renderizacao de formulario apos erro de validacao no `POST /checkout/dados`.
- Repository de dados de checkout passa a ler contato e endereco em uma unica statement SQL.
- Documentacao de checkout, seguranca e schema atualizada com a semantica operacional.

## Testes

- Handler de `GET /checkout/dados` com carrinho valido deve retornar `Cache-Control: private, no-store`.
- Handler de `POST /checkout/dados` invalido deve re-renderizar com `Cache-Control: private, no-store`.
- Repository deve usar uma unica query com `JOIN` para leitura.
- Repository deve ler dados completos corretamente quando `TEST_DATABASE_URL` estiver disponivel.
- Repository deve tratar ausencia completa e estados parciais como dados nao encontrados quando `TEST_DATABASE_URL` estiver disponivel.
- Erros publicos continuam genericos e sem detalhes internos.

## Definition of Done

- `templ generate` executado.
- `npm run css:build` executado.
- `gofmt -w .` executado.
- `go test ./...` passa.
- `go vet ./...` passa.
- `go build ./...` passa.
- Nenhuma migration criada.
- Nenhum secret ou PII real versionado.
- Commit e push feitos conforme `AGENTS.md`.
- Fase 8 permanece planejada.

## Validacao local

- Baseline antes das alteracoes: `go test ./...` passou.
- `go test ./...`: passou apos as alteracoes.
- `TEST_DATABASE_URL` com Supabase local em `internal/customers`: passou, cobrindo leitura completa, ausencia, estados parciais e rollback.
- Nenhuma migration foi criada.
- Validacoes finais de `templ generate`, `npm run css:build`, `gofmt -w .`, `go test ./...`, `go vet ./...` e `go build ./...`: passaram.
