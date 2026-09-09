# Supabase

Status: fundacao, catalogo, variantes e Storage de catalogo IMPLEMENTADOS.

## Arquitetura planejada

```text
Go Backend
   |
   v
pgx
   |
   v
Supabase Transaction Pooler
   |
   v
Supabase PostgreSQL
```

## Decisoes

- Supabase hospedara o PostgreSQL.
- A aplicacao Go usa `pgx/v5` e `pgxpool`.
- O acesso principal ao banco sera server-side.
- O banco sera acessado por `DATABASE_URL`, que deve apontar para o Supabase Transaction Pooler.
- Credenciais virao de environment variables.
- Nenhuma credencial sera colocada em Git.
- Supabase Data API nao sera a interface primaria da aplicacao.
- Supabase Storage armazena imagens publicas de catalogo no bucket `product-images`.
- Migrations versionadas serao aplicadas ao Supabase remoto de desenvolvimento pelo GitHub Actions quando houver alteracao em `supabase/migrations/**` ou `supabase/config.toml` na branch `main`.
- O workflow usa `supabase/setup-cli@v1` com Supabase CLI `2.117.0` fixado, executa `supabase link`, roda `supabase db push --dry-run` e so depois executa `supabase db push`.
- `pgx.QueryExecModeExec` e usado para evitar dependencia de prepared statement cache incompativel com transaction pooling.
- A aplicacao Go nao executa migrations no startup.
- A primeira migration de negocio cria `categories` e `products`, sem seed ficticio.
- A segunda migration de negocio cria materiais, cores, variantes, receita estimada de producao, imagens e o bucket `product-images`, sem seed ficticio.
- Nenhuma policy publica de upload, update ou delete em `storage.objects` e criada.

## Variaveis previstas

As variaveis abaixo existem como placeholders em `.env.example` e devem ser revisadas quando a configuracao oficial for feita:

- `DATABASE_URL`
- `DB_MAX_CONNS`
- `SUPABASE_URL`

`DATABASE_URL` e a unica fonte de verdade da conexao PostgreSQL.

`SUPABASE_URL` e opcional e nao e secret. A aplicacao usa essa URL apenas para montar URLs publicas do Storage quando existirem imagens cadastradas em `product_images`.

`SUPABASE_SERVICE_ROLE_KEY` nao e usada pela aplicacao nesta fase.

## Supabase CLI local

A CLI esta instalada como devDependency npm:

```sh
npx supabase --version
```

Scripts locais:

```sh
npm run db:start
npm run db:stop
npm run db:status
npm run db:reset
npm run db:push:dry-run
npm run db:push
```

`db:push` e manual e nao faz parte do build da aplicacao.

## GitHub Actions Secrets

O responsavel pelo projeto deve configurar estes secrets diretamente no GitHub, sem registrar valores no repositorio:

- `SUPABASE_ACCESS_TOKEN`
- `SUPABASE_DB_PASSWORD`
- `SUPABASE_PROJECT_ID`

Esses secrets sao usados apenas pelo workflow `.github/workflows/supabase-migrations.yml`. O workflow nao deve imprimir valores de secrets nos logs.

## Ambiente remoto de migrations

O workflow aponta para o projeto Supabase de desenvolvimento da PrintLab identificado por `SUPABASE_PROJECT_ID`.

Antes da operacao comercial, deve existir separacao explicita entre ambientes de desenvolvimento/staging e producao. O banco de producao nao deve receber migrations automaticas sem uma politica de aprovacao propria.

## Validacao remota Fase 3.1

Em 2026-09-09, a integracao remota da Fase 3.1 foi concluida sem expor valores de secrets:

- `gh secret list` confirmou por nome `SUPABASE_ACCESS_TOKEN`, `SUPABASE_DB_PASSWORD` e `SUPABASE_PROJECT_ID`.
- O workflow `Supabase Migrations` foi executado manualmente por `workflow_dispatch` no run `34302139793`.
- O run passou pelas etapas de setup da CLI, checagem de secrets, `supabase link`, `supabase db push --dry-run` e `supabase db push`.
- O responsavel do projeto confirmou `DATABASE_URL` do Supabase Transaction Pooler e `DB_MAX_CONNS` configuradas na Vercel.
- `GET /ready` remoto retornou HTTP 200 com body `ok`, validando Vercel -> Go -> `pgxpool` -> Supabase Transaction Pooler -> PostgreSQL.
- Nenhuma migration de negocio foi criada para acionar a validacao.
- Nenhum schema de negocio foi criado ou alterado por esta tarefa.

Nenhum valor de `DATABASE_URL`, senha, token, project ref ou connection string foi registrado.

## Catalogo e variantes

O catalogo publico usa o PostgreSQL do Supabase via backend Go e `pgxpool`.

- `categories` e `products` ficam no schema `public`.
- `materials`, `colors`, `product_variants`, `variant_filaments` e `product_images` tambem ficam no schema `public`.
- RLS fica habilitado nas tabelas de catalogo/variantes sem policies publicas nesta fase.
- `DATABASE_URL` deve apontar para o Transaction Pooler.
- O backend consulta somente produtos ativos.
- O backend consulta somente variantes ativas para exibicao publica.
- O backend calcula o preco efetivo da variante usando `product_variants.price_cents` ou fallback para `products.price_cents`.
- O Data API nao e a interface primaria do catalogo.
- `product_images.storage_path` guarda caminho relativo no bucket, nao URL absoluta.

A migration `20260909153625_create_catalog.sql` foi aplicada ao Supabase DEV pelo workflow `Supabase Migrations` no run `34372918466`, com dry-run antes da aplicacao.

## Storage

PostgreSQL armazena dados estruturados. Supabase Storage armazena arquivos publicos de imagem do catalogo.

Bucket:

- `product-images`

Configuracao da migration:

- bucket publico para leitura;
- limite de 5 MB por arquivo;
- MIME types permitidos: `image/avif`, `image/webp`, `image/jpeg`, `image/png`;
- `avif_autodetection` habilitado quando suportado pela versao atual.

Uso permitido:

- imagens publicas de produto;
- imagens publicas especificas de variante.

Uso proibido:

- documentos;
- dados de clientes;
- notas;
- arquivos privados;
- invoices;
- secrets.

Nao ha policy publica de upload. Upload, admin autenticado e regras de escrita permanecem futuros.

## Estrutura local

```text
supabase/
├── .gitignore
├── config.toml
└── migrations/
```

`supabase/config.toml` nao contem secrets. Storage local fica habilitado para validar o bucket de catalogo. Auth, Realtime, Edge Runtime, Analytics e seed ficam desabilitados nesta fase.

## Praticas proibidas

- Conectar ao Supabase sem plano aprovado.
- Versionar senha do banco ou service role key.
- Dar ao frontend acesso direto a tabelas sensiveis.
- Criar schema manualmente sem migration registrada.
- Usar Table Editor ou SQL Editor remoto como workflow normal para mudancas de schema.
- Executar `supabase db reset --linked` contra ambiente remoto.
- Aplicar seed automaticamente em deploy de migrations.
- Fazer `supabase login`, `supabase link` ou `supabase db push` manual como workflow normal de desenvolvimento.
- Inserir seed ou produto ficticio apenas para validar deploy.
- Criar policy publica de escrita em `storage.objects`.
