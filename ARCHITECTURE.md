# Arquitetura da PrintLab

Este e o documento principal de arquitetura do projeto PrintLab. Ele descreve a direcao aprovada para a fundacao tecnica, sem declarar como implementadas funcionalidades que ainda pertencem ao roadmap.

## Status atual

IMPLEMENTADO:

- Aplicacao Go em `cmd/server`.
- Homepage server-side em `GET /`.
- Rota `GET /health` para verificar que o processo HTTP esta funcionando.
- Servico de assets estaticos em `/static/` via `embed.FS`.
- Frontend server-side com `templ`.
- Tailwind CSS via CLI npm.
- Identidade visual da homepage refinada na Fase 2.1.
- Workflow de CI/CD para migrations Supabase de desenvolvimento.
- Estrutura inicial de diretorios e documentacao.

PLANEJADO:

- Catalogo, carrinho, checkout, pedidos, painel administrativo e integracoes externas.
- Acesso ao PostgreSQL via `pgx`.
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

O banco planejado e PostgreSQL hospedado no Supabase. O acesso principal sera feito pelo backend Go usando `pgx`, por conexao PostgreSQL apropriada para ambiente hospedado.

O Supabase Data API nao sera a interface primaria da aplicacao. O uso futuro de Supabase Storage para imagens podera ser avaliado quando catalogo e midia de produto forem implementados.

Migrations Supabase futuras devem ser versionadas em `supabase/migrations/` e aplicadas ao ambiente de desenvolvimento pelo GitHub Actions apos dry-run bem-sucedido. Schema e conexao Go com Supabase ainda dependem de aprovacao nas proximas entregas da Fase 3.

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
GET /static/... -> assets embutidos a partir de web/static/
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
- `tailwindcss` e `@tailwindcss/cli` para CSS.

Dependencias planejadas, mas ainda nao adicionadas:

- `pgx` para PostgreSQL;
- HTMX quando houver interacao real.

## Seguranca

Regras obrigatorias:

- Nunca confiar em dados financeiros recebidos do navegador.
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
