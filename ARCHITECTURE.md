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
- Inicio de pagamento InfinitePay em `POST /pedido/{id}/pagar`.
- Retorno de pagamento em `GET /pagamento/retorno`, validado por `payment_check` server-side.
- Webhook InfinitePay em `POST /webhooks/infinitepay`, validado por `payment_check` server-side.
- Acompanhamento seguro de pedido em `GET /acompanhar/{public_tracking_id}`.
- Painel administrativo em `/admin` com login Supabase Auth, autorizacao por UUID, sessao propria da PrintLab, dashboard, lista/detalhe de pedidos e mutacoes auditadas de producao/envio.
- Hardening base de seguranca com headers globais, CSP, limite global de body, timeouts HTTP e validacao/canonicalizacao de `SITE_URL`.
- Fase 16 concluida e validada em producao: SEO tecnico com canonical absoluto por `SITE_URL`, metadados publicos, `robots.txt`, sitemap de produtos ativos, `noindex` transacional, imagens WebP otimizadas e cache seletivo.
- MFA TOTP obrigatorio para Admin, validado em producao: senha -> Supabase AAL1 -> TOTP -> AAL2 -> sessao propria PrintLab.
- Fase 14.3 configurada e validada no Vercel Firewall/WAF antes da funcao Go, com rate limit de login e regras operacionais de Log no Hobby.
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
- Fase 13.4 do painel administrativo para imagens e Supabase Storage concluida e validada em producao.
- Fase 14.1 com Supabase Cron para limpeza diaria de `admin_sessions` e `carts` expirados.
- Selecao publica de configuracao por query string em `GET /produtos/{slug}?variante=<variant-slug>` quando houver mais de uma configuracao ativa.
- Supabase CLI local e estrutura `supabase/`.
- Vercel configurada para `gru1`.
- Estrutura inicial de diretorios e documentacao.

PLANEJADO:

- Fase 17.1 e Fase 17.2 concluidas e validadas em producao; Fase 17.3 em execucao, com 17.3.2 e 17.3.3 implementadas aguardando validacao manual; Fases 18 a 20 planejadas.
- HTMX quando houver interacao real que justifique sua presenca.

## Evolucao planejada pre-go-live

A Fase 17.1 foi concluida e validada em producao: slugs sao gerados no servidor para cadastros administrativos, preservados em renomeacoes e recebem sufixos determinísticos em colisoes. A Fase 17.2 foi concluida e validada em producao: `products.shipping_*` e a fonte autoritativa, configuracoes nao alteram frete e caixas usam um conjunto dimensional operacional no Admin. Nenhuma migration foi necessaria. A Fase 17.3.1 associa cores comerciais separadas da receita. A Fase 17.3.2 permite escolha publica e a Fase 17.3.3 persiste `cart_items.color_id` opcional, com unicidade por produto/variante/cor e snapshot de ID/nome/slug em `order_items`, exibido no Admin. A selecao e revalidada pelo backend e nao altera receita, frete ou pagamento; validacao manual da 17.3.3 permanece pendente. A Fase 18 planeja uma conta de cliente opcional com Magic Link, mantendo checkout convidado; o Admin continuara separado por senha, TOTP, AAL2 e allowlist. O SMTP inicial planejado para e-mails de autenticacao e conta e iCloud+ Custom Email Domain integrado futuramente ao Custom SMTP do Supabase Auth, sem configuracao realizada neste momento. A Fase 19 planeja revisao textual e experiencia, e a Fase 20 fara a auditoria final de producao somente apos essas funcionalidades estabilizarem.

## Diagrama textual

```text
Browser
   |
   v
Security middleware
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
Go Backend -> InfinitePay Checkout
Go Backend -> InfinitePay payment_check
InfinitePay -> Go Backend webhook
Go Backend -> Supabase Auth
Go Backend -> Supabase Storage
```

## Arquitetura server-side

A PrintLab sera uma aplicacao server-side. O backend Go recebera requisicoes HTTP, aplicara regras de negocio, acessara o banco e renderizara respostas HTML quando apropriado.

O navegador nao deve acessar diretamente tabelas sensiveis nem enviar valores financeiros como fonte autoritativa. IDs, quantidades e escolhas do usuario podem ser enviados pelo cliente, mas preco, subtotal, total, frete validado, status de pagamento e status de pedido pertencem ao servidor.

O carrinho anonimo usa cookie opaco no navegador e persistencia server-side. O banco armazena somente o hash SHA-256 do token do cookie, enquanto itens armazenam produto, variante opcional e quantidade.

A etapa de dados do checkout continua sem login. Contato e endereco pertencem ao carrinho anonimo atual e nao criam uma identidade permanente de cliente. Esses dados sao PII e devem ser tratados com minimizacao, validacao server-side, leitura consistente, `Cache-Control: private, no-store` em respostas HTML que possam conter PII e erros genericos. A consulta de CEP e uma melhoria progressiva feita pelo backend contra ViaCEP; o navegador nao chama ViaCEP diretamente e o preenchimento manual continua valido.

A etapa de frete tambem e server-side. O navegador envia somente a escolha da opcao de frete, por `service_code`. O backend recalcula a cotacao no POST, escolhe a menor caixa fisica real compativel por dimensoes internas com rotacao, persiste somente a cotacao final usando dimensoes externas e peso final, e invalida selecoes antigas por expiracao ou `input_hash`.

A revisao de checkout e server-side e nao recota a SuperFrete. Ela valida o carrinho atual, dados de checkout, selecao de frete, expiracao e `input_hash`. O POST recalcula subtotal, frete e total no backend, compara `review_fingerprint` apenas para detectar tela antiga, cria pedido em transacao PostgreSQL, converte o carrinho e remove dados temporarios. Pedido e snapshot historico e nao depende futuramente de catalogo, receita, dados temporarios ou caixa de frete.

O pagamento InfinitePay tambem e server-side. A pagina do pedido inicia `POST /pedido/{id}/pagar`; o backend monta o payload a partir do snapshot do pedido, confere o total, envia `redirect_url` e `webhook_url` gerados no servidor e redireciona o comprador para checkout hospedado. O retorno em `/pagamento/retorno` e o webhook em `/webhooks/infinitepay` nunca confirmam pagamento diretamente: ambos chamam `payment_check` e so marcam o pedido como `paid` quando a InfinitePay confirma pagamento e valor.

O painel administrativo tambem e server-side. `POST /admin/login` envia e-mail e senha ao Supabase Auth pelo backend usando `SUPABASE_PUBLISHABLE_KEY`, verifica se `user.id` corresponde a `ADMIN_SUPABASE_USER_ID` e inicia MFA TOTP. AAL1 nunca cria sessao PrintLab. O access_token temporario fica exclusivamente no cookie HttpOnly `printlab_admin_mfa_pending`, Strict, host-only, Path=/admin/mfa, Secure em producao e TTL de 10 minutos; refresh_token e descartado e nenhum token Supabase vai ao banco. Apos challenge/verify, o backend valida o token atualizado junto ao Supabase, confirma usuario e claim AAL2 e cria sessao propria. Requests autenticadas usam cookie HttpOnly opaco e resolvem a sessao por hash SHA-256 em `public.admin_sessions`, exigindo `mfa_verified_at` nao nulo. A Fase 13.2 adicionou operacao administrativa de pedidos com lista minimizada, detalhe protegido, mutacoes sequenciais de producao/envio e auditoria transacional em `public.admin_order_events`. A Fase 13.3 adicionou gestao SSR protegida de catalogo, variantes, receita, materiais, cores e caixas sobre as tabelas existentes, sem hard delete. A Fase 13.3A refinou a UX para tratar `product_variants` como configuracoes do produto na UI, exibindo escolha publica apenas quando houver duas ou mais configuracoes ativas. A Fase 13.4 adicionou gestao administrativa de imagens com upload direto ao Supabase Storage e finalizacao server-side em `product_images`.

A Fase 14.1 adiciona hardening HTTP transversal. O servidor usa `http.Server` com timeouts explicitos, aplica headers globais de seguranca, CSP restritiva, teto global de 1 MiB para corpo de requests e validacao central de `SITE_URL`. Admin e acompanhamento publico preservam seus headers privados/noindex/referrer especificos. Rate limiting permanece uma responsabilidade operacional futura por Vercel Firewall/WAF, apos observacao de trafego real.

## Responsabilidades do frontend

O frontend e responsavel por apresentar HTML, formularios e interacoes progressivas. A stack atual e planejada e:

- IMPLEMENTADO: `templ` para templates tipados em Go.
- IMPLEMENTADO: Tailwind CSS para estilos utilitarios e design tokens.
- PLANEJADO: HTMX para interacoes HTTP parciais quando houver necessidade real.
- IMPLEMENTADO: JavaScript proprio minimo para mascaras progressivas e consulta interna de CEP na etapa de dados.
- IMPLEMENTADO: UI administrativa SSR para login, dashboard, lista/detalhe de pedidos, acoes de producao/envio, catalogo e imagens; a tela de imagens usa JavaScript nativo somente para upload direto ao Supabase Storage.

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

- A Fase 8 criou perfis logisticos opcionais em `products` e `product_variants`; desde a Fase 17.2, somente `products.shipping_*` e autoritativo e as colunas de variante sao legado inerte.
- Produto ativo exige perfil completo e positivo; produto inativo aceita perfil ausente ou completo.
- `shipping_boxes` guarda caixas fisicas reais, com medidas internas para encaixe, medidas externas para transportadora e `packaging_weight_g` para embalagem/protecao padrao.
- `cart_shipping_selections` guarda selecao de entrega 1:1 por carrinho, com `delivery_method`. Para `shipping`, grava provider, servico, preco em centavos, prazo, snapshot do pacote real, `input_hash`, `quoted_at` e `expires_at`; para `pickup`, grava retirada no local com preco zero e sem caixa/SuperFrete.
- A escolha da menor caixa valida usa menor volume interno, menor peso de embalagem, menor `sort_order`, `name` e `id`.
- Multi-volume permanece fora do escopo.

A Fase 9 adiciona pedidos:

- `carts.converted_at` diferencia carrinho ativo de carrinho convertido.
- `orders` guarda status `pending_payment`, moeda `BRL`, subtotal, frete e total em centavos.
- `orders.order_number` e sequencial e serve apenas como referencia humana.
- `orders.source_cart_id` e unique quando preenchido, impedindo pedido duplicado para o mesmo carrinho.
- `order_customer_details` e `order_shipping_addresses` guardam snapshots privados.
- `order_shipping_details` guarda `delivery_method`. Para envio, preserva servico, transportadora, prazo, caixa, peso e dimensoes externas cotadas; para retirada, preserva a modalidade explicita com frete zero.
- `order_items` guarda snapshots de produto, variante, SKU, preco, quantidade, subtotal e producao por unidade.
- `order_item_filaments` guarda componentes de receita sem FK para materiais, cores ou receita original.
- RLS fica habilitado nas tabelas de pedido, sem policies publicas.

A Fase 10 adiciona pagamento:

- `order_payments` guarda pagamento 1:1 por pedido.
- `order_nsu` e derivado do UUID canonico do pedido.
- Status de pagamento e `pending` ou `paid`.
- `orders.status` permite `pending_payment` e `paid`.
- `transaction_nsu` possui unique parcial quando preenchido.
- RLS fica habilitado em `order_payments`, sem policies publicas.

A Fase 13.1 adiciona `admin_sessions`, tabela transitoria de sessoes administrativas. Ela armazena somente `auth_user_id`, `SHA-256(token)`, timestamps e expiracao; nao referencia `auth.users`, nao guarda token bruto, senha, access token ou refresh token, e tem RLS habilitado sem policies publicas.

A Fase 13.2 adiciona `admin_order_events`, tabela de auditoria operacional para mutacoes administrativas de producao/envio. Cada evento registra pedido, ator administrativo, tipo, status anterior, status novo e horario, sem PII de cliente.

A Fase 13.3 nao adiciona tabelas. O Admin opera `categories`, `products`, `product_variants`, `materials`, `colors`, `variant_filaments` e `shipping_boxes` existentes, usando `is_active` em vez de hard delete para entidades principais. A Fase 13.4 tambem nao adiciona tabelas; ela usa `product_images.storage_path`, `sort_order` e `is_primary` existentes para gerenciar imagens do bucket `product-images`.

A Fase 14.1 habilita `pg_cron` e agenda `printlab_transient_data_cleanup` para remover diariamente `admin_sessions` expiradas e `carts` expirados. A limpeza de carrinhos apaga somente dados temporarios por FKs `ON DELETE CASCADE`; pedidos sao preservados porque `orders.source_cart_id` usa `ON DELETE SET NULL`.

Ainda nao existem tabelas de clientes permanentes nem tabelas de auditoria de catalogo.

## Comunicacao com servicos externos

Integracoes externas serao chamadas pelo backend, nunca diretamente pelo navegador quando houver credenciais, valores financeiros ou estados sensiveis envolvidos.

A integracao SuperFrete usa `net/http`, timeout explicito, `Authorization: Bearer <token>` e `User-Agent` operacional. O backend mapeia internamente `sandbox` para `https://sandbox.superfrete.com` e `production` para `https://api.superfrete.com`; nao ha base URL arbitraria por environment variable. Primeiro envia `products` ao calculator para obter pacote ideal, depois escolhe uma caixa real cadastrada e envia `package` com dimensoes externas e peso final para obter o preco apresentado ao cliente. Falhas de cotacao sao classificadas internamente por estagio e motivo seguro, sem registrar CEP, CPF, e-mail, telefone, endereco, token ou corpo bruto externo.

A integracao ViaCEP usa `net/http`, timeout explicito de aproximadamente 3 segundos e contexto da request original. O backend consulta `https://viacep.com.br/ws/{cep}/json/` apos normalizar CEP com exatamente 8 digitos e responde ao navegador somente `street`, `district`, `city` e `state`.

A integracao InfinitePay usa `net/http`, timeout explicito, base URL interna fixa `https://api.checkout.infinitepay.io`, `POST /links` para checkout hospedado e `POST /payment_check` para confirmacao server-side. O handle vem de `INFINITEPAY_HANDLE`; nao ha token/API secret no frontend. O webhook InfinitePay e aceito em `POST /webhooks/infinitepay`, mas serve apenas como gatilho para `payment_check`.

A integracao Supabase Auth para Admin usa `net/http`, timeout explicito e `POST {SUPABASE_URL}/auth/v1/token?grant_type=password` com header `apikey: SUPABASE_PUBLISHABLE_KEY`. A publishable key identifica a aplicacao, nao concede autorizacao administrativa. A autorizacao da PrintLab compara o UUID retornado por Supabase Auth com `ADMIN_SUPABASE_USER_ID`.

A integracao Supabase Storage para Admin usa `SUPABASE_SECRET_KEY` apenas no backend, depois da sessao Admin e da validacao `Origin`/`Referer`. O backend gera signed upload URL para o bucket `product-images`; o navegador envia os bytes diretamente ao Supabase e depois chama a finalizacao server-side. Na finalizacao, o backend confirma o objeto com `GET /storage/v1/object/info/{bucket}/{path}` e usa o JSON de metadata (`size` e `content_type`) antes de gravar `product_images`.

O uso de Supabase Auth no fluxo `Browser -> Go -> Supabase Auth` pode concentrar tentativas no IP server-side da aplicacao para fins de rate limit do provedor. IP forwarding oficial exige `Sb-Forwarded-For`, secret API key `sb_secret` e habilitacao explicita no Supabase; isso permanece documentado como avaliacao futura e nao foi adicionado na 14.1.

## Boundaries

Os pacotes em `internal/` devem representar areas de responsabilidade:

- `products`: catalogo, variantes e atributos de produto;
- `cart`: carrinho e itens;
- `checkout`: orquestracao futura de compra;
- `orders`: pedidos e itens de pedido;
- `shipping`: calculo e validacao de frete;
- `payments`: pagamentos e webhooks;
- `customers`: dados de cliente e endereco;
- `admin`: autenticacao, sessao e operacao interna administrativa;
- `database`: infraestrutura de acesso ao banco;
- `config`: leitura de configuracao.

Diretorios sem implementacao permanecem vazios com `.gitkeep`. Nao devem receber codigo artificial apenas para preencher estrutura.

## Seguranca transversal

Todas as rotas passam por middleware global que aplica:

- `X-Content-Type-Options: nosniff`;
- `X-Frame-Options: DENY`;
- `Permissions-Policy: camera=(), microphone=(), geolocation=()`;
- `Referrer-Policy: strict-origin-when-cross-origin`;
- CSP restritiva sem `unsafe-eval`;
- limite global de 1 MiB para body.

A CSP permite scripts apenas de `self`; estilos de `self` e inline existente; imagens de `self`, `data:` e origem Supabase quando `SUPABASE_URL` estiver configurada; conexoes de `self` e origem Supabase quando aplicavel; bloqueia objetos, frames ancestrais e base URI externa.

Limites especificos menores continuam ativos para formularios Admin, JSON de imagens Admin e webhook InfinitePay.

## Fluxo HTTP esperado

Fluxo atual:

```text
GET / -> homepage HTML renderizada com templ
GET /health -> HTTP 200
GET /ready -> HTTP 200 quando banco configurado e acessivel; HTTP 503 quando ausente ou indisponivel
GET /produtos -> catalogo publico SSR; HTTP 503 quando banco estiver indisponivel
GET /produtos/{slug} -> detalhe publico de produto ativo; HTTP 404 para inexistente, inativo ou slug invalido
GET /produtos/{slug}?variante={variant-slug} -> detalhe com configuracao ativa selecionada; HTTP 404 para configuracao invalida ou indisponivel
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
POST /pedido/{id}/pagar -> cria ou reutiliza checkout InfinitePay e redireciona 303 para checkout hospedado
GET /pagamento/retorno -> valida payment_check; pago redireciona 303 para /pedido/{uuid}?pagamento=confirmado
POST /webhooks/infinitepay -> valida identificadores, chama payment_check e responde JSON
GET /acompanhar/{public_tracking_id} -> acompanhamento SSR minimizado, sem PII, valores ou IDs internos
GET /admin/login -> formulario SSR de login administrativo ou indisponibilidade segura
POST /admin/login -> autentica via Supabase Auth, autoriza por user UUID e inicia AAL1 sem sessao PrintLab
GET/POST /admin/mfa/setup -> enrollment e verificacao inicial TOTP
GET/POST /admin/mfa/challenge -> escolha de fator e challenge/verify -> valida token e AAL2 -> cria sessao PrintLab
POST /admin/mfa/cancel -> limpa cookie pending e volta ao login
GET /admin -> dashboard administrativo inicial somente leitura, protegido por sessao
POST /admin/logout -> remove sessao e limpa cookies administrativo e pending
GET /static/... -> assets embutidos a partir de web/static/
```

Todas as respostas desse fluxo recebem headers globais de seguranca. Rotas Admin e `/acompanhar/{public_tracking_id}` adicionam seus headers privados/noindex/referrer especificos.

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
- Preco efetivo de configuracao deve ser calculado no backend a partir de `product_variants.price_cents` ou `products.price_cents`.
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

## Observabilidade

A Fase 15 implementa eventos operacionais pesquisaveis com `event`, `level`, `reason` seguro e `request_id` opaco. Sucessos usam `info`, rejeicoes esperadas usam `warning` e indisponibilidades operacionais usam `error`, sem PII, tokens ou identificadores de pedido/carrinho. O procedimento de consulta dos Runtime Logs Vercel, a limitacao de retencao no Hobby, a estrategia de alertas e o runbook inicial estao em `docs/operations/`.

Antes de operacao comercial, metricas adicionais, retencao maior e alertas automatizados devem ser reavaliados conforme trafego e plano de hosting, sem alterar a fonte de verdade financeira `payment_check`.

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
