# Fase 1 — Bootstrap da aplicacao Go

Status: Concluida

## Objetivo

Criar uma aplicacao Go minima que compile, inicie e exponha um health check.

## O que foi implementado

- `go.mod` com module path do GitHub.
- Entrada HTTP em `cmd/web/main.go`.
- Rota `GET /health`.
- Teste automatizado do health check.

## Validacoes realizadas

- `gofmt`
- `go test ./...`
- `go vet ./...`
- Execucao local do servidor e chamada a `/health`.

## Arquivos principais

- `go.mod`
- `cmd/web/main.go`
- `cmd/web/main_test.go`

## Resultado final

Aplicacao Go minima funcionando, sem banco, e-commerce, carrinho, checkout, admin ou integracoes.
