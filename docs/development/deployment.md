# Deployment

Status: PLANEJADO.

## Ambientes

- `local`: maquina de desenvolvimento.
- `development`: ambiente remoto para validacao durante desenvolvimento.
- `production`: ambiente de operacao comercial, ainda nao ativo.

## Fluxo planejado

```text
Git
  |
  v
GitHub
  |
  v
Vercel
```

Banco planejado:

```text
Supabase PostgreSQL
```

Migrations Supabase em desenvolvimento:

```text
Git
  |
  v
GitHub Actions
  |
  v
Supabase CLI
  |
  v
Supabase de desenvolvimento
```

## Situacao atual

- Desenvolvimento inicial.
- Vercel Hobby pode ser usado durante desenvolvimento.
- Entrada Go compativel com zero-config da Vercel em `cmd/server/main.go`.
- Assets estaticos servidos via `embed.FS`, reduzindo dependencia de filesystem local no runtime da Vercel.
- Vercel configurada em `vercel.json` para executar na regiao `gru1`.
- Sem operacao comercial.
- Conexao PostgreSQL da aplicacao via `DATABASE_URL` quando configurada.
- Catalogo publico depende do PostgreSQL e retorna indisponibilidade generica quando o banco ou schema nao estiverem acessiveis.
- Variantes e imagens de catalogo dependem do PostgreSQL e do bucket `product-images`.
- Carrinho anonimo depende do PostgreSQL para persistencia e usa cookie host-only `printlab_cart`.
- Dados de checkout dependem do carrinho e do PostgreSQL, sem criar conta permanente de cliente.
- Consulta de CEP usa endpoint interno no Go e chamada server-side ao ViaCEP, sem depender do banco.
- Frete depende de PostgreSQL para perfis logisticos, caixas reais e selecao de frete. Cotacao externa depende de configuracao SuperFrete.
- Pedidos dependem do PostgreSQL para criar snapshots historicos, converter carrinho e exibir `/pedido/{id}` por UUID.
- Pagamentos InfinitePay dependem de PostgreSQL para `order_payments`, de `SITE_URL` HTTPS e de `INFINITEPAY_HANDLE` para iniciar checkout hospedado, gerar `redirect_url`, gerar `webhook_url` e confirmar por `payment_check`.
- Admin 13.1 depende de PostgreSQL para `admin_sessions` e de `SUPABASE_URL`, `SUPABASE_PUBLISHABLE_KEY` e `ADMIN_SUPABASE_USER_ID` para login real.
- Sem secrets reais.
- Workflow de CI/CD para migrations Supabase configurado em `.github/workflows/supabase-migrations.yml`.

`gru1` foi escolhida porque o projeto Supabase da PrintLab esta em South America (Sao Paulo). Isso reduz a latencia entre o runtime Go na Vercel e o PostgreSQL no Supabase.

## Validacao remota de desenvolvimento

Status da Fase 3.1: CONCLUIDA.

Validacoes feitas em 2026-09-09:

- GitHub CLI autenticada sem leitura de token.
- Secrets `SUPABASE_ACCESS_TOKEN`, `SUPABASE_DB_PASSWORD` e `SUPABASE_PROJECT_ID` presentes por nome no repositorio.
- Workflow `Supabase Migrations` ativo.
- Run manual `34302139793` por `workflow_dispatch` passou, incluindo `supabase link`, `supabase db push --dry-run` e `supabase db push`.
- Nenhuma migration de negocio foi criada para essa validacao.
- Deploy publico `https://printlab-pied.vercel.app` respondeu HTTP 200 em `/`, `/health`, `/ready` e `/static/css/app.css`.
- O responsavel do projeto confirmou manualmente que `DATABASE_URL` do Supabase Transaction Pooler e `DB_MAX_CONNS` estao configuradas na Vercel.
- A resposta HTTP 200 de `/ready` valida a conectividade Vercel -> Go -> `pgxpool` -> Supabase Transaction Pooler -> PostgreSQL.

Nenhum valor secreto ou connection string foi registrado na documentacao.

## Assets estaticos

O CSS compilado em `web/static/css/app.css` e embutido no binario Go e servido em `/static/css/app.css`. Essa abordagem evita falhas em deploys onde o runtime nao encontra o diretorio `web/static/` no filesystem local.

A logo em `web/static/images/branding/logo-printlab-primary.png` tambem e embutida e deve ser validada no deploy pela rota `/static/images/branding/logo-printlab-primary.png`.

O JavaScript progressivo do checkout em `web/static/js/checkout.js` tambem e embutido no binario e deve ser servido por `/static/js/checkout.js`.

Validacao local:

```sh
curl -I http://localhost:8080/static/css/app.css
```

## Migrations Supabase via GitHub Actions

O workflow `.github/workflows/supabase-migrations.yml` executa em push para `main`, mas apenas quando arquivos de banco mudarem:

- `supabase/migrations/**`
- `supabase/config.toml`

Tambem existe `workflow_dispatch` para execucao manual emergencial pelo GitHub.

O responsavel pelo projeto deve configurar estes GitHub Actions Secrets:

- `SUPABASE_ACCESS_TOKEN`
- `SUPABASE_DB_PASSWORD`
- `SUPABASE_PROJECT_ID`

O job instala `supabase/setup-cli@v1` com Supabase CLI `2.117.0`, valida que os secrets existem, executa `supabase link --project-ref "$SUPABASE_PROJECT_ID"`, roda `supabase db push --dry-run` e somente depois aplica `supabase db push`.

O workflow nao usa `--include-seed`, nao executa reset remoto e nao deve imprimir valores de secrets nos logs.

Este workflow aponta para o projeto Supabase de desenvolvimento da PrintLab. Antes da operacao comercial sera necessario separar development/staging e production, com politica de aprovacao propria para producao.

A Fase 4 cria a primeira migration real, `create_catalog`, com `categories` e `products`. A Fase 5 cria `create_product_variants`, com materiais, cores, variantes, receita de producao, imagens e bucket `product-images`. A Fase 6 cria `create_carts`, com `carts` e `cart_items`. A Fase 7 cria `create_cart_customer_details`, com contato e endereco temporarios por carrinho. Elas devem ser aplicadas pelo workflow apos `supabase db push --dry-run`, sem seed e sem dados ficticios de catalogo, carrinho ou PII.

A Fase 8 cria `add_shipping_profiles_and_selections`, com perfis logisticos em produtos/variantes, `shipping_boxes` e `cart_shipping_selections`. Ela deve ser aplicada pelo workflow sem seed de caixas, produtos, cotacoes ou dados ficticios.

A Fase 9 cria `create_orders`, adicionando `carts.converted_at` e tabelas de pedido como snapshots historicos. Ela deve ser aplicada pelo workflow sem seed de pedidos, clientes, enderecos, itens ou dados ficticios.

A Fase 10 cria `add_order_payments`, adicionando `order_payments` e permitindo `orders.status = 'paid'`. Ela deve ser aplicada pelo workflow sem seed de pagamentos, checkout URLs, transacoes ou dados ficticios.

## Runtime PostgreSQL

```text
Vercel Go em gru1
  |
  v
pgxpool
  |
  v
Supabase Transaction Pooler em South America (Sao Paulo)
  |
  v
PostgreSQL
```

Secrets de runtime no ambiente de hosting:

- `DATABASE_URL`
- `INFINITEPAY_HANDLE`, necessario para habilitar checkout InfinitePay
- `DB_MAX_CONNS`, opcional, default `4`
- `SUPABASE_URL`, opcional e nao secret, usada para montar URLs publicas do bucket `product-images`
- `SUPABASE_PUBLISHABLE_KEY`, opcional e nao administrativa, usada para login Admin via Supabase Auth
- `ADMIN_SUPABASE_USER_ID`, opcional, UUID do unico usuario Supabase Auth autorizado no Admin
- `SITE_URL`, URL publica HTTPS usada como origem permitida e como base do `redirect_url` e `webhook_url` InfinitePay
- `SUPERFRETE_ENV`, opcional ate habilitar cotacao real, aceitando `sandbox` ou `production`
- `SUPERFRETE_API_TOKEN`, secret da SuperFrete
- `SUPERFRETE_ORIGIN_POSTAL_CODE`, CEP operacional de origem da PrintLab
- `SUPERFRETE_CONTACT_EMAIL`, e-mail operacional usado no `User-Agent`
- `SUPERFRETE_SERVICES`, lista de codigos de servico solicitados

`DATABASE_URL` deve ser configurada como secret e nunca impressa em logs. Se estiver ausente, `GET /ready` retorna 503, mas `GET /` e `GET /health` continuam funcionando temporariamente nesta fase.

Se a configuracao SuperFrete estiver ausente, a aplicacao continua iniciando e as rotas publicas existentes continuam funcionando. A etapa `/checkout/frete`, quando acessada com carrinho e dados validos, apresenta indisponibilidade segura em vez de panic ou exposicao de erro interno. Se qualquer variavel SuperFrete for preenchida, a configuracao precisa estar completa e valida.

Se `INFINITEPAY_HANDLE` ou `SITE_URL` HTTPS estiverem ausentes, a aplicacao continua iniciando e as rotas existentes continuam funcionando. A pagina de pedido nao exibe botao falso de pagamento e mostra indisponibilidade segura.

Se `SUPABASE_PUBLISHABLE_KEY`, `ADMIN_SUPABASE_USER_ID`, `SUPABASE_URL` ou `DATABASE_URL` estiverem ausentes, a aplicacao publica continua iniciando. `/admin/login` apresenta indisponibilidade segura e nao revela qual configuracao esta faltando.

O cookie do carrinho e marcado como `Secure` quando `APP_ENV=production`, `VERCEL_ENV=production` ou `SITE_URL` usa HTTPS.

`SUPABASE_URL` pode ficar ausente enquanto nao houver imagens reais cadastradas. Nesse caso, catalogo e detalhe continuam funcionando com placeholder visual. Bucket/schema implementados nao significam imagem real validada.

Na validacao final da Fase 3.1, a URL publica retornou HTTP 200 em `/ready`, confirmando a conexao runtime com o Supabase Transaction Pooler sem expor detalhes internos.

Na validacao remota da Fase 4, tambem foram validados:

- `GET /produtos`: HTTP 200 com empty state.
- `GET /produtos/nao-existe`: HTTP 404.

O catalogo pode estar vazio e ainda assim responder HTTP 200. Produto inexistente responde HTTP 404.

Depois da Fase 5, validar tambem:

```sh
curl -i https://printlab-pied.vercel.app/produtos
```

O catalogo remoto pode continuar vazio. Nao inserir produto, variante ou imagem ficticia apenas para testar UI de variante em producao/desenvolvimento remoto.

Na validacao remota da Fase 5, foram validados:

- GitHub Actions `Supabase Migrations` run `34378986883`: sucesso.
- `supabase db push --dry-run`: sucesso.
- `supabase db push`: sucesso.
- `GET /`: HTTP 200.
- `GET /health`: HTTP 200 com body `ok`.
- `GET /ready`: HTTP 200 com body `ok`.
- `GET /produtos`: HTTP 200 com empty state.
- `GET /static/css/app.css`: HTTP 200.

Schema e bucket foram implementados. Imagem real de produto nao foi validada remotamente porque nao ha `product_images` cadastradas.

Depois da Fase 6, validar tambem:

```sh
curl -i https://printlab-pied.vercel.app/carrinho
```

Sem cookie, a resposta esperada e HTTP 200 com carrinho vazio. Nao inserir produto, variante, carrinho ou item ficticio apenas para validar POST remoto.

Depois da Fase 7, validar tambem:

```sh
curl -i https://printlab-pied.vercel.app/checkout/dados
```

Sem carrinho valido, a resposta esperada e redirect para `/carrinho`. Nao criar carrinho/produto fake nem enviar PII ficticia em ambiente remoto apenas para testar checkout.

Depois da Fase 8, validar tambem:

```sh
curl -i https://printlab-pied.vercel.app/checkout/frete
```

Sem carrinho valido, a resposta esperada e redirect para `/carrinho`. Nao criar produto, caixa, carrinho, endereco ou cotacao ficticia em ambiente remoto apenas para validar frete.

Quando existirem dados reais de desenvolvimento:

- configurar `SUPERFRETE_ENV=sandbox`;
- configurar `SUPERFRETE_API_TOKEN` como secret;
- configurar `SUPERFRETE_ORIGIN_POSTAL_CODE`;
- configurar `SUPERFRETE_CONTACT_EMAIL`;
- configurar `SUPERFRETE_SERVICES`;
- cadastrar pelo menos um produto real com perfil logistico;
- cadastrar pelo menos uma caixa fisica real ativa;
- validar uma cotacao Sandbox real sem criar etiqueta/postagem.

A Fase 8.1 adiciona diagnosticos seguros para diferenciar falhas de configuracao, caixas, planejamento, pacote retornado, encaixe em caixa real, chamada final e cotacoes finais vazias. Esses logs nao devem registrar CEP, CPF, telefone, e-mail, endereco, token ou corpo bruto da SuperFrete.

A validacao Sandbox real da Fase 8 foi confirmada manualmente antes da Fase 9: planejamento retornou pacote, caixa pequena foi rejeitada, caixa compativel permitiu cotacao final, modalidades foram exibidas, uma modalidade foi selecionada e `cart_shipping_selections` persistiu a selecao. Nenhum secret, CEP ou dado pessoal deve ser registrado.

Validar consulta de CEP no deploy:

```sh
curl -i https://printlab-pied.vercel.app/api/cep/01001000
curl -I https://printlab-pied.vercel.app/static/js/checkout.js
```

A resposta do endpoint de CEP deve conter somente `street`, `district`, `city` e `state`. CEP inexistente deve retornar 404 e a interface deve permitir preenchimento manual.

Depois da Fase 9, validar tambem sem criar pedido remoto automaticamente:

```sh
curl -i https://printlab-pied.vercel.app/checkout/revisao
curl -i https://printlab-pied.vercel.app/pedido/00000000-0000-0000-0000-000000000000
```

Sem carrinho valido, `/checkout/revisao` deve redirecionar para `/carrinho`. Pedido inexistente por UUID deve retornar 404. Nao criar produto, caixa, carrinho, endereco, frete ou pedido ficticio apenas para validar rota remota.

Depois da Fase 10, validar sem criar pagamento real automaticamente:

```sh
curl -i https://printlab-pied.vercel.app/pagamento/retorno
curl -i -X POST https://printlab-pied.vercel.app/pedido/00000000-0000-0000-0000-000000000000/pagar \
  -H 'Origin: https://printlab-pied.vercel.app'
```

Retorno sem parametros deve responder erro seguro sem refletir query string. Pedido inexistente por UUID deve retornar 404 ou redirecionar com indisponibilidade segura conforme configuracao. Nao executar pagamento real sem roteiro controlado.

Depois da Fase 13.1, validar Admin sem configurar credenciais reais automaticamente:

```sh
curl -i https://printlab-pied.vercel.app/admin/login
```

Sem config Admin, a resposta esperada e HTTP 503 com indisponibilidade segura. Com config Admin real feita manualmente na Vercel e usuario criado no Supabase Auth, validar login pelo navegador sem registrar e-mail, senha, user UUID real, token de sessao ou tokens Supabase em logs/documentacao.

## Vercel

`vercel.json` contem apenas:

```json
{
  "$schema": "https://openapi.vercel.sh/vercel.json",
  "regions": ["gru1"]
}
```

Nao ha builds, rewrites, routes, outputDirectory, installCommand ou Docker customizado.

## Antes da loja operar comercialmente

- Revisar plano de hospedagem.
- Revisar environment variables.
- Revisar secrets.
- Revisar banco.
- Revisar migrations.
- Separar ambientes Supabase de desenvolvimento/staging e producao.
- Definir politica de aprovacao para migrations de producao.
- Validar webhook InfinitePay real apos deploy de mudancas de pagamento.
- Revisar dominio.
- Revisar observabilidade.
- Revisar backups.
- Revisar seguranca.
- Implementar limpeza programada de carrinhos expirados e PII associada.
- Implementar limpeza operacional de sessoes administrativas expiradas.

## Praticas proibidas

- Publicar ambiente de producao com credenciais expostas.
- Operar comercialmente sem validar webhooks de pagamento e alertas de falha.
- Alterar banco de producao manualmente sem registro.
- Fazer deploy de funcionalidade financeira sem testes aplicaveis.
- Executar migrations automaticas em producao sem politica de aprovacao.
