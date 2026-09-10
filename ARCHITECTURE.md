# Arquitetura da PrintLab

Este e o documento principal de arquitetura do projeto PrintLab. Ele descreve a direcao aprovada para a fundacao tecnica, sem declarar como implementadas funcionalidades que ainda pertencem ao roadmap.

## Status atual

IMPLEMENTADO:

- Aplicacao Go em `cmd/server`.
- Homepage server-side em `GET /`.
- Catalogo publico em `GET /produtos`.
- Pagina publica de produto em `GET /produtos/{slug}`.
- Carrinho anonimo em `GET /carrinho`.
- Mutacoes de carrinho por POST com redirects 303.
- Dados de checkout em `GET /checkout/dados` e `POST /checkout/dados`, vinculados ao carrinho anonimo.
- Consulta interna de CEP em `GET /api/cep/{cep}` para melhoria progressiva da etapa de dados.
- Frete em `GET /checkout/frete` e `POST /checkout/frete`, com cotacao server-side pela SuperFrete quando configurada.
- Revisao de checkout em `GET /checkout/revisao`.
- Criacao de pedido pendente de pagamento em `POST /checkout/revisao`.
- Exibicao de pedido por UUID em `GET /pedido/{id}`.
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
- Fase 6 com `carts` e `cart_items`.
- Fase 7 com `cart_customer_details` e `cart_shipping_addresses`.
- Fase 8 com perfis logisticos, `shipping_boxes`, `cart_shipping_selections` e cliente SuperFrete.
- Fase 9 com `orders`, snapshots de pedido e conversao de carrinho por `converted_at`.
- Bucket publico `product-images` no Supabase Storage para imagens de catalogo.
- Selecao publica de variante por query string em `GET /produtos/{slug}?variante=<variant-slug>`.
- Supabase CLI local e estrutura `supabase/`.
- Vercel configurada para `gru1`.
- Estrutura inicial de diretorios e documentacao.

PLANEJADO:

- Painel administrativo e integracoes externas de pagamento.
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

Servicos externos:

Go Backend -> SuperFrete API
Go Backend -> ViaCEP API
Go Backend -> InfinitePay Checkout/Webhooks
```

## Arquitetura server-side

A PrintLab sera uma aplicacao server-side. O backend Go recebera requisicoes HTTP, aplicara regras de negocio, acessara o banco e renderizara respostas HTML quando apropriado.

O navegador nao deve acessar diretamente tabelas sensiveis nem enviar valores financeiros como fonte autoritativa. IDs, quantidades e escolhas do usuario podem ser enviados pelo cliente, mas preco, subtotal, total, frete validado, status de pagamento e status de pedido pertencem ao servidor.

O carrinho anonimo usa cookie opaco no navegador e persistencia server-side. O banco armazena somente o hash SHA-256 do token do cookie, enquanto itens armazenam produto, variante opcional e quantidade.

A etapa de dados do checkout continua sem login. Contato e endereco pertencem ao carrinho anonimo atual e nao criam uma identidade permanente de cliente. Esses dados sao PII e devem ser tratados com minimizacao, validacao server-side, leitura consistente, `Cache-Control: private, no-store` em respostas HTML que possam conter PII e erros genericos. A consulta de CEP e uma melhoria progressiva feita pelo backend contra ViaCEP; o navegador nao chama ViaCEP diretamente e o preenchimento manual continua valido.

A etapa de frete tambem e server-side. O navegador envia somente a escolha da opcao de frete, por `service_code`. O backend recalcula a cotacao no POST, escolhe a menor caixa fisica real compativel por dimensoes internas com rotacao, persiste somente a cotacao final usando dimensoes externas e peso final, e invalida selecoes antigas por expiracao ou `input_hash`.

A revisao de checkout e server-side e nao recota a SuperFrete. Ela valida o carrinho atual, dados de checkout, selecao de frete, expiracao e `input_hash`. O POST recalcula subtotal, frete e total no backend, compara `review_fingerprint` apenas para detectar tela antiga, cria pedido em transacao PostgreSQL, converte o carrinho e remove dados temporarios. Pedido e snapshot historico e nao depende futuramente de catalogo, receita, dados temporarios ou caixa de frete.

## Responsabilidades do frontend

O frontend e responsavel por apresentar HTML, formularios e interacoes progressivas. A stack atual e planejada e:

- IMPLEMENTADO: `templ` para templates tipados em Go.
- IMPLEMENTADO: Tailwind CSS para estilos utilitarios e design tokens.
- PLANEJADO: HTMX para interacoes HTTP parciais quando houver necessidade real.
- IMPLEMENTADO: JavaScript proprio minimo para mascaras progressivas e consulta interna de CEP na etapa de dados.

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

A Fase 6 adiciona carrinho anonimo:

- `carts.token_hash` armazena `SHA-256` do token bruto do cookie.
- `carts.expires_at` controla validade de 30 dias.
- `cart_items` armazena `product_id`, `variant_id` opcional e `quantity`.
- `cart_items.quantity` e limitado a `1..99`.
- Indices unique parciais impedem linhas duplicadas para produto sem variante e produto com variante.
- Precos e subtotais sao recalculados em leitura, sem persistir `unit_price`.

A Fase 7 adiciona dados temporarios de checkout vinculados ao carrinho:

- `cart_customer_details` armazena `full_name`, `email`, `phone` e `cpf`.
- `cart_shipping_addresses` armazena endereco de entrega brasileiro.
- CPF e CEP sao armazenados apenas como digitos ASCII normalizados.
- Telefone e armazenado em formato canonico brasileiro E.164.
- Contato e endereco sao salvos em transacao e removidos por `ON DELETE CASCADE` quando o carrinho for removido.
- Contato e endereco sao lidos por uma unica consulta SQL com `JOIN`; estados parciais anomalos sao tratados como dados ausentes.
- Nao ha indice ou unique em CPF; uma pessoa pode ter carrinhos diferentes.

A Fase 8 adiciona frete:

- `products` e `product_variants` possuem perfil logistico opcional em gramas e milimetros.
- O perfil logistico e atomico: variante completa sobrescreve produto; campos parciais nao sao misturados.
- `shipping_boxes` guarda caixas fisicas reais, com medidas internas para encaixe, medidas externas para transportadora e `packaging_weight_g` para embalagem/protecao padrao.
- `cart_shipping_selections` guarda selecao de frete 1:1 por carrinho, com provider, servico, preco em centavos, prazo, snapshot do pacote real, `input_hash`, `quoted_at` e `expires_at`.
- A escolha da menor caixa valida usa menor volume interno, menor peso de embalagem, menor `sort_order`, `name` e `id`.
- Multi-volume permanece fora do escopo.

A Fase 9 adiciona pedidos:

- `carts.converted_at` diferencia carrinho ativo de carrinho convertido.
- `orders` guarda status `pending_payment`, moeda `BRL`, subtotal, frete e total em centavos.
- `orders.order_number` e sequencial e serve apenas como referencia humana.
- `orders.source_cart_id` e unique quando preenchido, impedindo pedido duplicado para o mesmo carrinho.
- `order_customer_details` e `order_shipping_addresses` guardam snapshots privados.
- `order_shipping_details` guarda servico, transportadora, prazo, caixa, peso e dimensoes externas cotadas.
- `order_items` guarda snapshots de produto, variante, SKU, preco, quantidade, subtotal e producao por unidade.
- `order_item_filaments` guarda componentes de receita sem FK para materiais, cores ou receita original.
- RLS fica habilitado nas tabelas de pedido, sem policies publicas.

Ainda nao existem tabelas de pagamentos, clientes permanentes ou admin.

## Comunicacao com servicos externos

Integracoes externas serao chamadas pelo backend, nunca diretamente pelo navegador quando houver credenciais, valores financeiros ou estados sensiveis envolvidos.

A integracao SuperFrete usa `net/http`, timeout explicito, `Authorization: Bearer <token>` e `User-Agent` operacional. O backend mapeia internamente `sandbox` para `https://sandbox.superfrete.com` e `production` para `https://api.superfrete.com`; nao ha base URL arbitraria por environment variable. Primeiro envia `products` ao calculator para obter pacote ideal, depois escolhe uma caixa real cadastrada e envia `package` com dimensoes externas e peso final para obter o preco apresentado ao cliente. Falhas de cotacao sao classificadas internamente por estagio e motivo seguro, sem registrar CEP, CPF, e-mail, telefone, endereco, token ou corpo bruto externo.

A integracao ViaCEP usa `net/http`, timeout explicito de aproximadamente 3 segundos e contexto da request original. O backend consulta `https://viacep.com.br/ws/{cep}/json/` apos normalizar CEP com exatamente 8 digitos e responde ao navegador somente `street`, `district`, `city` e `state`.

InfinitePay continua planejado. Webhooks ainda nao existem.

## Boundaries

Os pacotes em `internal/` devem representar areas de responsabilidade:

- `products`: catalogo, variantes e atributos de produto;
- `cart`: carrinho e itens;
- `checkout`: orquestracao futura de compra;
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
GET /carrinho -> carrinho SSR; carrinho vazio 200 sem cookie
POST /carrinho/adicionar -> adiciona/incrementa item e redireciona 303 para /carrinho
POST /carrinho/itens/{id}/quantidade -> atualiza quantidade e redireciona 303
POST /carrinho/itens/{id}/remover -> remove item e redireciona 303
GET /checkout/dados -> formulario SSR de contato/endereco; exige carrinho com itens disponiveis
POST /checkout/dados -> valida e salva dados do carrinho em transacao; redireciona 303
GET /api/cep/{cep} -> consulta CEP via backend; retorna street, district, city e state sem dados extras do provedor
GET /checkout/frete -> calcula cotacoes atuais e renderiza opcoes de frete; exige carrinho, dados, perfis logisticos e caixas reais
POST /checkout/frete -> revalida cotacao atual e persiste selecao de frete por service_code; redireciona 303 para /checkout/revisao
GET /checkout/revisao -> revisa carrinho, dados, frete e total sem recotar SuperFrete
POST /checkout/revisao -> cria pedido pendente de pagamento em transacao e redireciona 303 para /pedido/{uuid}
GET /pedido/{id} -> exibe pedido por UUID com status humano e sem PII completa
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

Vertical slice implementada para carrinho:

```text
HTTP
   |
   v
cart handler
   |
   v
cart service
   |
   v
cart repository
   |
   v
pgxpool
   |
   v
PostgreSQL
```

Vertical slice implementada para dados temporarios de checkout:

```text
HTTP
   |
   v
checkout details handler
   |
   v
customers service
   |
   v
customers repository
   |
   v
pgxpool
   |
   v
PostgreSQL
```

Vertical slice implementada para frete:

```text
HTTP
   |
   v
shipping handler
   |
   v
shipping service
   |
   v
shipping repository + SuperFrete client
   |
   v
PostgreSQL + SuperFrete API
```

Vertical slice implementada para pedidos:

```text
HTTP
   |
   v
orders handler
   |
   v
orders service
   |
   v
orders repository
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
- Carrinho nao congela preco; subtotal usa preco atual calculado no backend.
- Token bruto de carrinho nao deve ser persistido, logado, renderizado em HTML ou usado em URL.
- CPF, e-mail, telefone e endereco nao devem ser logados nem usados como identificadores publicos.
- Dados temporarios de checkout pertencem ao carrinho e devem ser removidos quando o carrinho for removido.
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
