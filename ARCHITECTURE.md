# Arquitetura da PrintLab

Este e o documento principal de arquitetura do projeto PrintLab. Ele descreve a direcao aprovada para a fundacao tecnica, sem declarar como implementadas funcionalidades que ainda pertencem ao roadmap.

## Status atual

IMPLEMENTADO:

- Aplicacao Go em `cmd/server`.
- Homepage server-side em `GET /`.
- Catalogo publico em `GET /produtos`.
- Pagina publica de produto em `GET /produtos/{slug}`.
- Rota `GET /health` para verificar que o processo HTTP esta funcionando.
- Rota `GET /ready` para readiness de banco.
- Servico de assets estaticos em `/static/` via `embed.FS`.
- Frontend server-side com `templ`.
- Tailwind CSS via CLI npm.
- Identidade visual da homepage refinada na Fase 2.1.
- Workflow de CI/CD para migrations Supabase de desenvolvimento.
- Configuracao centralizada em `internal/config`.
- Acesso PostgreSQL com `pgx/v5` e `pgxpool` em `internal/database`.
- Primeiro schema de negocio com `public.categories` e `public.products`.
- Vertical slice de catalogo em `internal/products`.
- Fase 5 com `materials`, `colors`, `product_variants`, `variant_filaments` e `product_images`.
- Bucket publico `product-images` no Supabase Storage para imagens de catalogo.
- Selecao publica de variante por query string em `GET /produtos/{slug}?variante=<variant-slug>`.
- Supabase CLI local e estrutura `supabase/`.
- Vercel configurada para `gru1`.
- Estrutura inicial de diretorios e documentacao.

PLANEJADO:

- Carrinho, checkout, pedidos, painel administrativo e integracoes externas.
- HTMX quando houver interacao real que justifique sua presenca.

## Diagrama textual

```text
Browser
   |
   v
Go Backend
   |
   v
pgx
   |
   v
Supabase PostgreSQL

Runtime de banco:

Vercel Go
   |
   v
pgxpool
   |
   v
Supabase Transaction Pooler
   |
   v
PostgreSQL

Servicos externos planejados:

Go Backend -> SuperFrete API
Go Backend -> InfinitePay Checkout/Webhooks
```

## Arquitetura server-side

A PrintLab sera uma aplicacao server-side. O backend Go recebera requisicoes HTTP, aplicara regras de negocio, acessara o banco e renderizara respostas HTML quando apropriado.

O navegador nao deve acessar diretamente tabelas sensiveis nem enviar valores financeiros como fonte autoritativa. IDs, quantidades e escolhas do usuario podem ser enviados pelo cliente, mas preco, subtotal, total, frete validado, status de pagamento e status de pedido pertencem ao servidor.

## Responsabilidades do frontend

O frontend e responsavel por apresentar HTML, formularios e interacoes progressivas. A stack atual e planejada e:

- IMPLEMENTADO: `templ` para templates tipados em Go.
- IMPLEMENTADO: Tailwind CSS para estilos utilitarios e design tokens.
- PLANEJADO: HTMX para interacoes HTTP parciais quando houver necessidade real.
- PLANEJADO: minimo possivel de JavaScript proprio.

O frontend pode melhorar a experiencia do usuario, mas nao decide regras financeiras, disponibilidade final, status de pedido ou confirmacao de pagamento.

## Responsabilidades do backend

O backend sera a autoridade para:

- buscar produtos e precos;
- validar disponibilidade;
- recalcular subtotais e totais;
- validar frete;
- criar pedidos;
- iniciar fluxos de pagamento;
- processar webhooks;
- persistir eventos relevantes;
- expor HTML ou respostas HTTP adequadas para o frontend.

As regras de negocio devem ficar no backend para evitar duplicacao insegura no navegador.

## Banco de dados

O banco planejado e PostgreSQL hospedado no Supabase. O acesso principal e feito pelo backend Go usando `pgx/v5` e `pgxpool`, por `DATABASE_URL`.

`DATABASE_URL` e a unica fonte de verdade da conexao PostgreSQL em runtime. A aplicacao nao monta connection string manualmente e nao deve logar host, usuario, senha, project ref ou URL de conexao.

Para compatibilidade com Supabase Transaction Pooler, `pgxpool.Config.ConnConfig.DefaultQueryExecMode` e configurado como `pgx.QueryExecModeExec`. A aplicacao nao depende de cache de prepared statements e nao executa `SQL PREPARE` explicito.

O Supabase Data API nao sera a interface primaria da aplicacao. Supabase Storage e usado para imagens publicas de catalogo no bucket `product-images`; metadados e relacoes continuam no PostgreSQL.

Migrations Supabase devem ser versionadas em `supabase/migrations/` e aplicadas ao ambiente de desenvolvimento pelo GitHub Actions apos dry-run bem-sucedido. A aplicacao Go nao executa migrations no startup.

O primeiro schema de negocio cria `public.categories` e `public.products`. Produtos publicos dependem de `products.is_active = true`, usam slug como URL publica e armazenam preco-base em `price_cents` como inteiro em centavos.

A Fase 5 adiciona variantes, materiais, cores, receitas estimadas de producao e imagens:

- `product_variants.price_cents` pode sobrescrever `products.price_cents`.
- `variant_filaments.estimated_weight_mg` usa inteiro em miligramas.
- `product_variants.print_time_minutes` usa inteiro em minutos e nao representa prazo de entrega.
- `product_images.storage_path` guarda caminho relativo no bucket `product-images`.

Ainda nao existem tabelas de carrinho, pedidos, pagamentos, frete, clientes ou admin.

## Comunicacao com servicos externos

Integracoes externas serao chamadas pelo backend, nunca diretamente pelo navegador quando houver credenciais, valores financeiros ou estados sensiveis envolvidos.

SuperFrete e InfinitePay estao planejados. Esta fundacao nao implementa chamadas HTTP reais, endpoints, payloads ou webhooks.

## Boundaries

Os pacotes em `internal/` devem representar areas de responsabilidade:

- `products`: catalogo, variantes e atributos de produto;
- `cart`: carrinho e itens;
- `checkout`: orquestracao de compra;
- `orders`: pedidos e itens de pedido;
- `shipping`: calculo e validacao de frete;
- `payments`: pagamentos e webhooks;
- `customers`: dados de cliente e endereco;
- `admin`: operacao interna;
- `database`: infraestrutura de acesso ao banco;
- `config`: leitura de configuracao.

Diretorios sem implementacao permanecem vazios com `.gitkeep`. Nao devem receber codigo artificial apenas para preencher estrutura.

## Fluxo HTTP esperado

Fluxo atual:

```text
GET / -> homepage HTML renderizada com templ
GET /health -> HTTP 200
GET /ready -> HTTP 200 quando banco configurado e acessivel; HTTP 503 quando ausente ou indisponivel
GET /produtos -> catalogo publico SSR; HTTP 503 quando banco estiver indisponivel
GET /produtos/{slug} -> detalhe publico de produto ativo; HTTP 404 para inexistente, inativo ou slug invalido
GET /produtos/{slug}?variante={variant-slug} -> detalhe com variante ativa selecionada; HTTP 404 para variante invalida ou indisponivel
GET /static/... -> assets embutidos a partir de web/static/
```

Vertical slice implementada para catalogo:

```text
HTTP
   |
   v
products handler
   |
   v
products service
   |
   v
products repository
   |
   v
pgxpool
   |
   v
PostgreSQL
```

Fluxo planejado para funcionalidades de negocio:

```text
Browser
   |
   v
Handler HTTP
   |
   v
Servico de dominio
   |
   v
Repositorio / pgx
   |
   v
PostgreSQL
```

Handlers devem validar entrada, chamar regras de dominio e devolver HTML ou respostas HTTP. Regras financeiras e mudancas de estado devem ser centralizadas no backend.

Assets estaticos compilados em `web/static/` sao embutidos no binario Go com `embed.FS` e expostos por `http.FileServer` sobre `http.FS`. Isso evita depender da presenca do diretorio `web/static/` no filesystem do runtime da Vercel.

## Filosofia de dependencias

A biblioteca padrao do Go e a primeira escolha. Dependencias externas so devem ser adicionadas quando resolverem uma necessidade real, com justificativa clara em documentacao ou ADR quando a decisao for arquitetural.

Dependencias implementadas:

- `github.com/a-h/templ` para templates server-side;
- `github.com/jackc/pgx/v5` para PostgreSQL server-side;
- `tailwindcss` e `@tailwindcss/cli` para CSS.
- `supabase` CLI via npm para tooling local de banco.

Dependencias planejadas, mas ainda nao adicionadas:

- HTMX quando houver interacao real.

## Seguranca

Regras obrigatorias:

- Nunca confiar em dados financeiros recebidos do navegador.
- Preco efetivo de variante deve ser calculado no backend a partir de `product_variants.price_cents` ou `products.price_cents`.
- Preco, desconto, subtotal, total, frete, status de pagamento e status de pedido devem ser definidos ou validados pelo backend.
- Pagamento so pode ser considerado confirmado apos validacao server-side.
- Redirect do navegador apos pagamento nunca e prova suficiente de pagamento.
- Webhooks devem ter validacao, idempotencia, protecao contra duplicidade e logs adequados.
- Secrets nunca devem ser versionados.

Detalhes adicionais estao em [docs/architecture/security.md](docs/architecture/security.md).

## Observabilidade futura

Antes de operacao comercial, o projeto devera definir logs estruturados, metricas essenciais, rastreamento de erros, monitoramento de webhooks e alertas para falhas em checkout, pagamento e envio.

Nesta fase, nao ha stack de observabilidade implementada.

## Escalabilidade

A arquitetura inicial favorece uma aplicacao monolitica simples, com responsabilidades bem separadas por pacote. Isso reduz custo e complexidade enquanto o produto valida necessidades reais.

Escalabilidade futura deve priorizar:

- consultas SQL bem modeladas;
- cache apenas onde houver gargalo medido;
- filas apenas quando houver necessidade operacional;
- separacao de servicos somente quando a complexidade justificar.

## Por que Go

Go foi escolhido por simplicidade operacional, desempenho, compilacao rapida, binario unico, biblioteca padrao forte e bom suporte a servidores HTTP.

## Por que SSR

Renderizacao server-side reduz JavaScript no cliente, concentra regras no servidor e simplifica seguranca para um e-commerce pequeno em evolucao.

## Por que HTMX

HTMX permite interacoes incrementais usando HTTP e HTML, sem transformar o frontend em uma aplicacao JavaScript complexa.

## Por que PostgreSQL

PostgreSQL e adequado para pedidos, pagamentos, estoque, dados relacionais, consistencia transacional e consultas analiticas futuras.

## Por que Supabase

Supabase oferece PostgreSQL hospedado, operacao simplificada e recursos futuros como Storage, mantendo a aplicacao livre para acessar o banco por conexao PostgreSQL padrao.

## Por que pgx

`pgx` e um driver PostgreSQL maduro para Go, com bom suporte a recursos nativos do PostgreSQL, controle explicito de conexoes e desempenho adequado.
