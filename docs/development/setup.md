# Setup de desenvolvimento

Status: fundacao minima IMPLEMENTADA.

## Requisitos

- Go 1.22.2 ou versao compativel.
- Terminal com acesso ao diretorio do projeto.

Nenhuma conta externa e necessaria nesta fase. Nao conecte Supabase, Vercel, SuperFrete ou InfinitePay durante o bootstrap.

## Configuracao local

`.env.example` lista variaveis previstas sem credenciais reais. Caso seja necessario testar configuracao local futuramente, crie um `.env` local fora do Git.

## Executar aplicacao

```sh
go run ./cmd/web
```

Porta customizada:

```sh
PORT=3000 go run ./cmd/web
```

## Validar health check

```sh
curl http://localhost:8080/health
```

Resposta esperada:

```text
ok
```

## Comandos de validacao

```sh
gofmt -w cmd/web
go test ./...
go vet ./...
```
