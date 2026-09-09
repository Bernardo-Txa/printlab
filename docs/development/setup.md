# Setup de desenvolvimento

Status: fundacao visual, banco e catalogo IMPLEMENTADA.

## Requisitos

- Go 1.26.0 ou versao compativel.
- Node.js e npm.
- CLI do `templ` v0.3.1020.
- Terminal com acesso ao diretorio do projeto.
- Supabase CLI instalada via npm (`supabase` v2.117.0).

Nenhuma conta externa e necessaria para executar a aplicacao local atual. Nao conecte SuperFrete ou InfinitePay durante esta fase.

Para o workflow remoto de migrations Supabase, o responsavel pelo projeto deve configurar secrets diretamente no GitHub Actions. Nao coloque credenciais em `.env`, documentacao ou codigo.

## Configuracao local

`.env.example` lista variaveis previstas sem credenciais reais. Caso seja necessario testar configuracao local futuramente, crie um `.env` local fora do Git.

Variaveis de runtime:

- `APP_ENV`: ambiente da aplicacao.
- `PORT`: porta HTTP, default `8080`.
- `SITE_URL`: URL publica da aplicacao quando necessaria.
- `DATABASE_URL`: secret PostgreSQL. Deve apontar para o Supabase Transaction Pooler.
- `DB_MAX_CONNS`: maximo de conexoes do pool por instancia, default `4`.

Sem `DATABASE_URL`, o servidor inicia, `GET /` funciona, `GET /health` retorna 200 e `GET /ready` retorna 503. Esse comportamento e temporario enquanto a homepage nao depende do banco.

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

## Validar readiness

```sh
curl -i http://localhost:8080/ready
```

Sem `DATABASE_URL`, a resposta esperada nesta fase e HTTP 503. Com `DATABASE_URL` configurada e banco acessivel, a resposta esperada e HTTP 200 com body `ok`.

## Validar catalogo

Com `DATABASE_URL` configurada e a migration de catalogo aplicada:

```sh
curl -i http://localhost:8080/produtos
```

Resposta esperada: HTTP 200. Se ainda nao houver produtos ativos, a pagina mostra o empty state do catalogo.

Detalhe de produto inexistente:

```sh
curl -i http://localhost:8080/produtos/nao-existe
```

Resposta esperada: HTTP 404.

Sem banco configurado ou com banco indisponivel, as rotas de catalogo retornam resposta generica de indisponibilidade.

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

A primeira migration real e `create_catalog`, criando `categories` e `products` sem inserir dados ficticios.

Nao use Table Editor ou SQL Editor remoto como workflow normal para mudancas de schema. Nao rode `supabase db reset --linked` contra banco remoto.

Enquanto o workflow estiver configurado, o desenvolvedor nao precisa executar manualmente `supabase login`, `supabase link` e `supabase db push` para migrations normais de desenvolvimento.

Scripts locais:

```sh
npm run db:start
npm run db:stop
npm run db:status
npm run db:reset
npm run db:push:dry-run
npm run db:push
```

`npm run db:push` e manual e nao deve ser chamado por build ou startup da aplicacao.

## Comandos de validacao

```sh
templ generate
npm run css:build
gofmt -w cmd/server web/components web/templates
go test ./...
go vet ./...
go build ./...
npx supabase --version
```
