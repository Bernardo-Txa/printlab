# Setup de desenvolvimento

Status: fundacao visual IMPLEMENTADA.

## Requisitos

- Go 1.26.0 ou versao compativel.
- Node.js e npm.
- CLI do `templ` v0.3.1020.
- Terminal com acesso ao diretorio do projeto.

Nenhuma conta externa e necessaria nesta fase. Nao conecte Supabase, Vercel, SuperFrete ou InfinitePay durante o bootstrap.

## Configuracao local

`.env.example` lista variaveis previstas sem credenciais reais. Caso seja necessario testar configuracao local futuramente, crie um `.env` local fora do Git.

## Instalar tooling

```sh
npm install
go install github.com/a-h/templ/cmd/templ@v0.3.1020
```

Garanta que `$(go env GOPATH)/bin` esteja no `PATH` para executar `templ`.

## Gerar templates e CSS

```sh
templ generate
npm run css:build
```

Durante desenvolvimento visual, o CSS pode ser observado com:

```sh
npm run css:watch
```

Se necessario, use terminais separados para `templ generate` ou watch, Tailwind watch e servidor Go. Nao ha ferramenta de orquestracao de processos nesta fase.

## Executar aplicacao

```sh
go run ./cmd/server
```

Porta customizada:

```sh
PORT=3000 go run ./cmd/server
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
templ generate
npm run css:build
gofmt -w cmd/server web/components web/templates
go test ./...
go vet ./...
go build ./...
```
