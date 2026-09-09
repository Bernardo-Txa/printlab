# Setup de desenvolvimento

Status: fundacao visual IMPLEMENTADA.

## Requisitos

- Go 1.26.0 ou versao compativel.
- Node.js e npm.
- CLI do `templ` v0.3.1020.
- Terminal com acesso ao diretorio do projeto.

Nenhuma conta externa e necessaria para executar a aplicacao local atual. Nao conecte SuperFrete ou InfinitePay durante esta fase.

Para o workflow remoto de migrations Supabase, o responsavel pelo projeto deve configurar secrets diretamente no GitHub Actions. Nao coloque credenciais em `.env`, documentacao ou codigo.

## Configuracao local

`.env.example` lista variaveis previstas sem credenciais reais. Caso seja necessario testar configuracao local futuramente, crie um `.env` local fora do Git.

Secrets exigidos no GitHub Actions para migrations:

- `SUPABASE_ACCESS_TOKEN`
- `SUPABASE_DB_PASSWORD`
- `SUPABASE_PROJECT_ID`

Esses valores nao devem ser solicitados no chat nem versionados.

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

## Validar assets estaticos

O CSS compilado deve ser servido por `/static/css/app.css`. Os arquivos de `web/static/` sao embutidos no binario Go, entao a mesma rota deve funcionar localmente e no deploy.

```sh
curl -I http://localhost:8080/static/css/app.css
```

Resposta esperada: HTTP 200 com `Content-Type` de CSS.

Validar logo da marca:

```sh
curl -I http://localhost:8080/static/images/branding/logo-printlab-primary.png
```

Resposta esperada: HTTP 200 com `Content-Type` de imagem PNG.

A homepage tambem deve ser validada visualmente em celular, tablet e desktop para conferir logo, hero, blocos coloridos, CTA e ausencia de overflow horizontal.

## Migrations Supabase

O fluxo normal de schema deve ser: criar migration SQL em `supabase/migrations/`, revisar, versionar no Git e enviar para `main`. O GitHub Actions executara `supabase link`, `supabase db push --dry-run` e, se passar, `supabase db push` contra o projeto Supabase de desenvolvimento.

Nao use Table Editor ou SQL Editor remoto como workflow normal para mudancas de schema. Nao rode `supabase db reset --linked` contra banco remoto.

Enquanto o workflow estiver configurado, o desenvolvedor nao precisa executar manualmente `supabase login`, `supabase link` e `supabase db push` para migrations normais de desenvolvimento.

## Comandos de validacao

```sh
templ generate
npm run css:build
gofmt -w cmd/server web/components web/templates
go test ./...
go vet ./...
go build ./...
```
