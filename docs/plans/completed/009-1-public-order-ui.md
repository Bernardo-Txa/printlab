# Fase 9.1 — Separacao entre Dados Operacionais e Interface Publica de Pedidos

Status: CONCLUIDA.

## Objetivo

Separar a experiencia publica do comprador dos dados operacionais preservados nos snapshots de pedido.

## Escopo

- Remover da UI publica de revisao e pedido dados de producao 3D.
- Remover da UI publica de revisao e pedido SKU interno.
- Remover da UI publica de pedido dados de embalagem fisica.
- Preservar dominio, repository, schema e snapshots persistidos.
- Atualizar documentacao de produto, roadmap e changelog.

## Fora de escopo

- Migration.
- Alteracao de snapshot persistido.
- InfinitePay.
- Webhooks.
- Painel administrativo.

## Definition of Done

- [x] `GET /checkout/revisao` mostra apenas dados comercialmente relevantes ao comprador.
- [x] `GET /pedido/{id}` nao exibe dados operacionais de producao, embalagem ou SKU interno.
- [x] Snapshots operacionais permanecem preservados no dominio/repository/banco.
- [x] Testes cobrem ausencia de labels e dados operacionais no HTML publico.
- [x] Documentacao registra a separacao entre snapshot operacional e interface publica.
- [x] Nenhuma migration e criada.
- [x] Validacoes aplicaveis passam.

## Resultado

Fase 9.1 concluida em 2026-09-11.

Validacoes finais:

- `templ generate`: passou.
- `npm run css:build`: passou.
- `gofmt -w .`: passou.
- `go test ./...`: passou.
- `go vet ./...`: passou.
- `go build ./...`: passou.

Nenhuma migration foi criada nesta fase.
