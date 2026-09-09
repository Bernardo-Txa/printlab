# Setup de desenvolvimento

Status: fundacao visual, banco, catalogo, variantes, carrinho, dados de checkout e frete IMPLEMENTADA.

## Requisitos

- Go 1.26.0 ou versao compativel.
- Node.js e npm.
- CLI do `templ` v0.3.1020.
- Terminal com acesso ao diretorio do projeto.
- Supabase CLI instalada via npm (`supabase` v2.117.0).

Nenhuma conta externa e necessaria para executar a aplicacao local basica. A cotacao real de frete exige configuracao SuperFrete opcional de desenvolvimento; sem ela, a rota de frete apresenta indisponibilidade segura.

Para o workflow remoto de migrations Supabase, o responsavel pelo projeto deve configurar secrets diretamente no GitHub Actions. Nao coloque credenciais em `.env`, documentacao ou codigo.

## Configuracao local

`.env.example` lista variaveis previstas sem credenciais reais. Caso seja necessario testar configuracao local futuramente, crie um `.env` local fora do Git.

Variaveis de runtime:

- `APP_ENV`: ambiente da aplicacao.
- `PORT`: porta HTTP, default `8080`.
- `SITE_URL`: URL publica da aplicacao quando necessaria, usada tambem como origem permitida em mutacoes de carrinho.
- `DATABASE_URL`: secret PostgreSQL. Deve apontar para o Supabase Transaction Pooler.
- `DB_MAX_CONNS`: maximo de conexoes do pool por instancia, default `4`.
- `SUPABASE_URL`: URL publica do projeto Supabase. Opcional e nao secret, usada somente para montar URLs publicas de imagens do bucket `product-images`.
- `SUPERFRETE_ENV`: `sandbox` ou `production`, obrigatoria somente quando a cotacao real estiver habilitada.
- `SUPERFRETE_API_TOKEN`: secret da SuperFrete, nunca versionado.
- `SUPERFRETE_ORIGIN_POSTAL_CODE`: CEP operacional de origem da PrintLab.
- `SUPERFRETE_CONTACT_EMAIL`: e-mail operacional do `User-Agent` da SuperFrete.
- `SUPERFRETE_SERVICES`: codigos de servico solicitados, separados por virgula.

Sem `DATABASE_URL`, o servidor inicia, `GET /` funciona, `GET /health` retorna 200, `GET /ready` retorna 503, catalogo fica indisponivel, `GET /carrinho` funciona apenas como carrinho vazio quando nao ha cookie, e `/checkout/dados` redireciona para `/carrinho` sem carrinho valido.

Sem `SUPABASE_URL`, catalogo e detalhe continuam funcionando; imagens cadastradas caem no placeholder visual porque a URL publica nao pode ser montada.

Sem configuracao SuperFrete, a rota `/checkout/frete` nao faz chamada externa e exibe estado de indisponibilidade depois que carrinho e dados forem resolvidos. Se qualquer variavel SuperFrete for preenchida, todas as variaveis obrigatorias precisam estar validas para evitar configuracao parcial.

Na Vercel, `VERCEL_ENV=production` tambem e considerado para marcar o cookie do carrinho como `Secure`. Localmente, `SITE_URL=http://localhost:8080` permite validar formularios sem exigir HTTPS.

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

Selecao de variante por SSR:

```sh
curl -i "http://localhost:8080/produtos/<produto>?variante=<variante>"
```

O slug de variante e opcional. Variante invalida, inexistente, inativa ou de outro produto retorna HTTP 404. Produto sem variantes continua valido.

## Validar carrinho

Com `DATABASE_URL` configurada e migrations aplicadas:

```sh
curl -i http://localhost:8080/carrinho
```

Sem cookie, a resposta esperada e HTTP 200 com carrinho vazio. A rota nao cria carrinho apenas por leitura.

Adicionar item exige produto real ativo no banco:

```sh
curl -i \
  -X POST \
  -H "Origin: http://localhost:8080" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data "product_slug=<produto>&variant_slug=<variante>&quantity=1" \
  http://localhost:8080/carrinho/adicionar
```

Nao inserir produto, variante, carrinho ou item ficticio apenas para validar UI local. Use dados reais de desenvolvimento quando existirem.

## Validar dados de checkout

Com `DATABASE_URL` configurada, migrations aplicadas e um carrinho real com itens disponiveis:

```sh
curl -i http://localhost:8080/checkout/dados
```

Sem carrinho valido, a resposta esperada e redirect para `/carrinho`. Nao inserir PII ficticia em migration nem criar carrinho/produto falso apenas para validar a rota. O formulario aceita preenchimento manual de contato e endereco, sem ViaCEP, BrasilAPI, Google Maps ou autocomplete externo.

Depois de salvar dados validos, o fluxo normal redireciona para `/checkout/frete`.

## Validar frete

Com `DATABASE_URL`, migrations aplicadas, carrinho real com itens disponiveis, dados de checkout salvos, perfis logisticos completos, caixas reais cadastradas e SuperFrete configurada:

```sh
curl -i http://localhost:8080/checkout/frete
```

A resposta esperada e HTTP 200 com opcoes de frete cotadas server-side. O POST envia somente `service_code`:

```sh
curl -i \
  -X POST \
  -H "Origin: http://localhost:8080" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data "service_code=1" \
  http://localhost:8080/checkout/frete
```

Sem carrinho valido, a rota redireciona para `/carrinho`. Sem dados de checkout, redireciona para `/checkout/dados`. Sem perfil logistico, caixa real ou configuracao SuperFrete, a rota mostra indisponibilidade honesta, sem inventar peso, caixa ou preco.

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

A segunda migration real e `create_product_variants`, criando `materials`, `colors`, `product_variants`, `variant_filaments`, `product_images` e o bucket publico `product-images`. Ela nao insere produtos, materiais, cores, variantes ou imagens ficticias.

A terceira migration real e `create_carts`, criando `carts` e `cart_items`. Ela nao insere carrinhos, itens ou dados ficticios.

A quarta migration real e `create_cart_customer_details`, criando `cart_customer_details` e `cart_shipping_addresses`. Ela nao insere contato, endereco, CPF, PII ou dados ficticios.

A quinta migration real e `add_shipping_profiles_and_selections`, adicionando perfis logisticos, `shipping_boxes` e `cart_shipping_selections`. Ela nao insere caixas, produtos, cotacoes ou dados ficticios.

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
gofmt -w .
go test ./...
go vet ./...
go build ./...
npx supabase --version
npx supabase db reset
```
