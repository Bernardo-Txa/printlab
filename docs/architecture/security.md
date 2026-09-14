# Seguranca

Status: diretrizes obrigatorias aprovadas; catalogo publico com variantes, carrinho, dados de checkout, frete, pedidos, pagamento InfinitePay, webhook, acompanhamento seguro, operacao administrativa com imagens e hardening base da Fase 14.1 IMPLEMENTADOS.

## Responsabilidade

Seguranca deve orientar arquitetura, codigo, banco, integracoes e operacao. As regras abaixo sao obrigatorias para qualquer fase futura.

## Hardening base HTTP

A Fase 14.1 adiciona uma camada global de seguranca antes do roteador HTTP.

Todas as respostas devem receber, salvo decisao especifica documentada:

- `X-Content-Type-Options: nosniff`;
- `X-Frame-Options: DENY`;
- `Permissions-Policy: camera=(), microphone=(), geolocation=()`;
- `Referrer-Policy: strict-origin-when-cross-origin`;
- `Content-Security-Policy` restritiva.

A CSP base e:

```text
default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: <SUPABASE_ORIGIN>; connect-src 'self' <SUPABASE_ORIGIN>; object-src 'none'; base-uri 'self'; frame-ancestors 'none'; form-action 'self'
```

`<SUPABASE_ORIGIN>` deve ser derivado exclusivamente de `SUPABASE_URL`, usando apenas esquema e host. Quando `SUPABASE_URL` estiver ausente ou invalida, nenhuma origem externa deve ser adicionada a `img-src` ou `connect-src`. A CSP nao deve usar `unsafe-eval`.

Headers especificos de paginas sensiveis continuam prevalecendo:

- Admin preserva `Cache-Control: private, no-store`, `X-Robots-Tag: noindex, nofollow, noarchive` e `Referrer-Policy: same-origin`;
- acompanhamento publico preserva `Cache-Control: private, no-store`, `X-Robots-Tag: noindex, nofollow, noarchive` e `Referrer-Policy: no-referrer`.

`Strict-Transport-Security` e `preload` nao foram habilitados na 14.1. A decisao fica para preparacao final de producao, depois de dominio definitivo, HTTPS validado e confirmacao de que nao ha subdominio ou fluxo legitimo dependente de HTTP.

## Limites de request e timeouts

A aplicacao possui um teto global de 1 MiB para corpo de request. Requests com `Content-Length` conhecido acima desse limite devem receber `413 Request Entity Too Large`. Requests sem tamanho confiavel sao envelopados com `http.MaxBytesReader` antes do roteador.

Limites menores por rota continuam sendo a fonte de controle especifica:

- formularios Admin: 256 KiB;
- JSON de imagens Admin: 64 KiB;
- webhook InfinitePay: 64 KiB.

O servidor HTTP deve ser criado com `http.Server` e timeouts explicitos:

- `ReadHeaderTimeout`: 5s;
- `ReadTimeout`: 15s;
- `WriteTimeout`: 30s;
- `IdleTimeout`: 60s.

Clientes externos tambem possuem timeouts explicitos:

- ViaCEP: 3s;
- SuperFrete: 8s;
- InfinitePay: 8s;
- Supabase Auth: 8s;
- Supabase Storage: 10s.

## Configuracao de origem publica

`SITE_URL` e opcional, mas quando preenchida deve ser uma URL absoluta com esquema `http` ou `https`, host obrigatorio, sem userinfo e sem fragment.

Em ambientes de producao (`APP_ENV=production`/`prod` ou `VERCEL_ENV=production`/`prod`), `SITE_URL` deve usar `https`. URLs locais `http://localhost` e equivalentes continuam permitidas fora de producao.

Comparacoes de origem baseadas em `SITE_URL` devem usar `scheme://host`. Comparar apenas host nao e suficiente quando a origem configurada diferencia HTTP e HTTPS.

## Retencao de dados transientes

A Fase 14.1 adiciona uma migration Supabase Cron para limpar diariamente, em UTC, dados transientes expirados:

- `public.admin_sessions where expires_at <= now()`;
- `public.carts where expires_at <= now()`.

A limpeza de carrinhos depende dos FKs ja existentes com `ON DELETE CASCADE` para `cart_items`, `cart_customer_details`, `cart_shipping_addresses` e `cart_shipping_selections`. `orders.source_cart_id` usa `ON DELETE SET NULL`, portanto pedidos, snapshots, pagamentos, fulfillment e eventos administrativos devem ser preservados.

Jobs de limpeza nao devem fazer chamadas HTTP, usar secrets ou apagar tabelas historicas de pedidos.

## Rate limiting e WAF

A Fase 14.1 nao implementa rate limiter dentro do Go. Endpoints sensiveis e/ou caros devem ser protegidos operacionalmente por Vercel Firewall/WAF depois de medir trafego real.

Endpoints prioritarios:

- `POST /admin/login`;
- `GET /api/cep/{cep}`;
- `GET /checkout/frete`;
- `POST /checkout/frete`;
- `POST /webhooks/infinitepay`;
- operacoes administrativas de Storage;
- chamadas server-side a SuperFrete, ViaCEP, InfinitePay e Supabase Auth.

Rollout recomendado:

1. criar regras por path/metodo em modo observacao/log;
2. medir volume real, falsos positivos, origens e padroes de erro;
3. definir limites diferentes por criticidade e custo;
4. ativar acao de rate limit, challenge ou deny de forma incremental;
5. validar checkout, webhook financeiro e login administrativo apos ativacao.

## Supabase Auth e MFA

O fluxo `Browser -> Go -> Supabase Auth` pode fazer tentativas de login parecerem concentradas no IP server-side perante os limites do Supabase Auth. A mitigacao oficial de IP forwarding exige header `Sb-Forwarded-For`, secret API key com prefixo `sb_secret` e habilitacao explicita do recurso no projeto. A Fase 14.1 apenas documenta esse risco; ela nao adiciona o header nem altera o fluxo de credenciais.

MFA nao foi implementado na 14.1. Uma Fase 14.2 pode avaliar TOTP/AAL2 se o risco justificar. Pontos obrigatorios dessa avaliacao:

- usar challenge/verify antes de criar a sessao propria da PrintLab;
- decidir como manter temporariamente o estado entre primeiro fator e desafio sem persistir access token ou refresh token desnecessariamente;
- evitar lockout do unico administrador;
- testar credenciais invalidas, fator ausente, challenge expirado e sessao nao elevada.

## Banco, logs e dependencias

Todas as tabelas de negocio e operacionais criadas ate a Fase 14.1 mantem RLS habilitado sem policies publicas. A aplicacao acessa dados server-side via connection string do backend; novas policies publicas exigem ADR ou documentacao especifica de arquitetura.

Como `DATABASE_URL` real e secret e nao deve ser lida nem impressa, a Fase 14.1 nao altera privilegios do usuario de banco em runtime. Antes do go-live comercial, a Fase 17 deve verificar operacionalmente se a connection string usa uma role com privilegios minimos suficientes para a aplicacao, em vez de uma role ampla demais.

Logs devem continuar sem PII, secrets, tokens, connection strings, URLs de checkout completas, signed upload URLs, `transaction_nsu`, `invoice_slug`, CPF, e-mail completo, telefone, endereco ou `public_tracking_id`.

Dependencias devem ser revisadas com `go mod tidy`, `go test ./...`, `go vet ./...`, `go build ./...`, `govulncheck ./...` quando disponivel e `npm audit` antes do fechamento da fase.

## Regras financeiras

Nunca confiar em dados financeiros recebidos do navegador.

O frontend nunca deve determinar de maneira autoritativa:

- preco do produto;
- desconto;
- subtotal;
- total;
- valor de frete;
- status de pagamento;
- status de pedido.

Antes de finalizar uma compra, o backend deve:

1. receber IDs e quantidades;
2. buscar produtos e precos no banco;
3. validar disponibilidade;
4. recalcular subtotal;
5. validar frete;
6. calcular total;
7. criar o pedido.

## Pagamentos

Pagamento somente podera ser considerado confirmado apos validacao server-side.

Redirect do navegador apos pagamento nunca devera ser considerado prova suficiente de pagamento.

Na integracao InfinitePay implementada, `POST /pedido/{id}/pagar` valida origem, monta payload apenas com snapshots de pedido, compara o total em centavos com `orders.total_cents` e aceita redirect apenas para checkout hospedado em host explicitamente autorizado da InfinitePay.

`GET /pagamento/retorno` usa somente `order_nsu`, `transaction_nsu` e `slug` para chamar `payment_check` server-side. Query params como `receipt_url` e `capture_method` nao sao fonte de autoridade.

Webhooks InfinitePay sao apenas gatilho para `payment_check` server-side e nao confirmam pagamento diretamente pelo payload recebido.

## Secrets

Nunca armazenar dentro do Git:

- senha;
- API key;
- token;
- database password;
- service role key;
- credencial de producao.

Use environment variables para configuracoes sensiveis. `.env.example` deve conter apenas nomes de variaveis e comentarios, sem valores reais.

`DATABASE_URL` e secret e deve ser configurada apenas em `.env` local ignorado pelo Git ou em secrets do ambiente de hosting. A aplicacao nao deve imprimir `DATABASE_URL`, senha, host privado, project ref ou connection string em logs, respostas HTTP ou mensagens publicas de erro.

`SUPABASE_SERVICE_ROLE_KEY` nao e usada para conexao PostgreSQL da aplicacao nem como credencial primaria da aplicacao.

`SUPABASE_SECRET_KEY` e usada somente no backend para operacoes administrativas de Supabase Storage da Fase 13.4. Ela deve usar o prefixo `sb_secret_`, ficar apenas em `.env` local ignorado pelo Git ou nas variaveis do hosting, e nunca ser logada, renderizada, enviada ao navegador, colocada em URL ou documentada com valor real.

`SUPERFRETE_API_TOKEN` e secret operacional. Ele deve existir somente em `.env` local ignorado pelo Git ou nas variaveis de ambiente do hosting. O token nao deve ser logado, renderizado, armazenado no banco, enviado ao navegador, colocado em URL ou exposto em mensagens de erro.

`SUPABASE_PUBLISHABLE_KEY` identifica a aplicacao perante o Supabase e nao concede autorizacao administrativa. Mesmo assim, valores reais nao devem ser escritos em documentacao, testes ou logs.

## Catalogo publico

- Produtos publicos exigem `products.is_active = true`.
- Produto inativo retorna publicamente como inexistente.
- Slugs de produto e categoria sao validados antes de consulta ao banco.
- Slug invalido retorna resposta 404, sem revelar detalhes.
- Queries de catalogo usam parametros PostgreSQL.
- Erros de PostgreSQL nao sao retornados ao usuario.
- Preco-base e definido pelo backend a partir de `products.price_cents`.
- Preco efetivo de variante e definido pelo backend a partir de `product_variants.price_cents` ou fallback para `products.price_cents`.
- Variante inativa, inexistente, invalida ou pertencente a outro produto retorna publicamente como inexistente.
- Templates recebem preco formatado e nao fazem calculo financeiro.
- `product_images.storage_path` deve ser caminho relativo de bucket, nunca URL absoluta.
- URLs publicas de imagens sao montadas centralizadamente no backend a partir de `SUPABASE_URL` e do bucket `product-images`.
- O bucket `product-images` e publico para leitura de imagens de catalogo, mas nao existe policy publica de upload, update ou delete.
- Upload administrativo de imagens usa signed upload URL gerada pelo backend apos sessao Admin, `Origin`/`Referer`, produto, configuracao, MIME e tamanho serem validados.
- O backend gera o object path; o navegador nao escolhe bucket/path arbitrario.
- `SUPABASE_SERVICE_ROLE_KEY` nao e usada pela aplicacao.

## Carrinho anonimo

- Carrinho anonimo usa cookie opaco `printlab_cart`.
- O cookie e `HttpOnly`, `SameSite=Lax`, `Path=/`, host-only e `Secure` em producao.
- O token do cookie e aleatorio, gerado com `crypto/rand` com 32 bytes.
- O token bruto nao e persistido, logado, renderizado em HTML ou enviado em URL.
- O banco armazena somente `SHA-256(token)` em `carts.token_hash`.
- Carrinho com `carts.converted_at` preenchido nao deve ser tratado como carrinho ativo.
- `cart_items` armazena somente produto, variante opcional e quantidade.
- Preco unitario, subtotal e total sao sempre recalculados server-side.
- Mutacoes de item usam escopo `cart_id + item_id`; nunca atualizam ou removem apenas por `item_id`.
- Mutacoes usam POST e validacao centralizada de `Origin`/`Referer`.
- Requests cross-site com origem conhecida e incompatibil devem ser rejeitados.
- Checkout revalida todos os itens antes de criar pedido.

## Dados pessoais de checkout

- A etapa `GET/POST /checkout/dados` coleta somente dados necessarios para compra, entrega e contato relacionado ao pedido.
- Dados de contato e endereco pertencem ao carrinho anonimo atual.
- Nao ha conta, senha, username, data de nascimento, genero, newsletter ou marketing consent nesta fase.
- CPF e necessario para documentacao futura de envio/DC-e, mas nao e identificador publico e nao possui indice ou unique.
- CPF e CEP sao armazenados como digitos ASCII normalizados.
- Telefone brasileiro e armazenado em formato canonico E.164.
- E-mail e normalizado com trim e lowercase para uso operacional atual.
- Contato e endereco sao persistidos em transacao para evitar estado parcial.
- Contato e endereco sao lidos em uma unica consulta SQL consistente; estado parcial anomalo nao e retornado como checkout valido.
- Respostas HTML de checkout que podem conter PII usam `Cache-Control: private, no-store`.
- Erros publicos devem ser genericos e nao conter CPF, e-mail completo, telefone, endereco, token de carrinho ou detalhes PostgreSQL.
- Logs nao devem registrar CPF, e-mail completo, telefone, endereco, token de carrinho, `DATABASE_URL` ou connection strings.
- A consulta progressiva de CEP deve ser server-side. O navegador chama apenas endpoint interno, e logs nao devem registrar CEP consultado nem endereco retornado.
- O endpoint interno de CEP deve retornar somente rua, bairro, cidade e UF, sem repassar codigos administrativos do provedor externo.
- Dados temporarios sao removidos por `ON DELETE CASCADE` quando o carrinho for removido.
- Carrinhos expirados e PII temporaria associada sao removidos pelo job diario de Supabase Cron da Fase 14.1.

## Frete

- Cotacao de frete e autoritativa no backend.
- O frontend envia apenas `service_code`; preco, prazo, transportadora, peso e dimensoes vindos do navegador sao ignorados.
- O backend reexecuta a cotacao no POST antes de persistir uma selecao.
- A integracao SuperFrete usa `Authorization: Bearer <token>` apenas server-side.
- O `User-Agent` da SuperFrete usa identificacao operacional da aplicacao e `SUPERFRETE_CONTACT_EMAIL`, nunca e-mail do cliente.
- `SUPERFRETE_ENV` aceita somente `sandbox` ou `production`.
- A base URL e mapeada internamente para `https://sandbox.superfrete.com` ou `https://api.superfrete.com`; environment variable nao pode redirecionar Authorization para host arbitrario.
- O cliente HTTP possui timeout explicito e respeita cancelamento de contexto.
- Erros publicos de frete sao genericos e nao expoem token, payload externo, CEP, CPF, e-mail, telefone ou endereco.
- Logs comuns nao devem registrar token, CPF, e-mail completo, telefone, endereco, connection strings ou payloads completos de cotacao.
- Logs operacionais de frete podem registrar somente estagio, motivo seguro, status HTTP seguro e identificacao generica de servico externo.
- Categorias internas de indisponibilidade de frete incluem configuracao ausente, ausencia de caixas ativas, falha de planejamento, ausencia de pacote retornado, ausencia de caixa compativel, falha de cotacao final e ausencia de cotacoes finais validas.
- Erros do cliente SuperFrete preservam categoria segura como `400`, `401`, `429`, `500`, `timeout` ou `invalid_json`, sem corpo bruto, token ou payload externo em `Error()`.
- `GET /checkout/frete` e re-renderizacoes de POST usam `Cache-Control: private, no-store`.
- Selecoes de frete expiram em 30 minutos.
- `input_hash` invalida selecoes quando carrinho, quantidade, variante, perfil logistico, CEP, servicos ou caixa mudam, sem incluir PII desnecessaria.
- Caixas fisicas reais sao obrigatorias para cotacao final; o sistema nao inventa caixas nem divide em multi-volume nesta fase.

## Pedidos

- `GET /checkout/revisao` e `GET /pedido/{id}` usam `Cache-Control: private, no-store`.
- Revisao nao chama SuperFrete novamente; ela valida expiracao e `input_hash` da selecao persistida.
- `review_fingerprint` e somente deteccao de tela antiga. Ele nao e secret e nao define preco, frete, subtotal, total ou status.
- `POST /checkout/revisao` reutiliza validacao centralizada de `Origin`/`Referer`.
- O pedido e criado em transacao PostgreSQL unica, com lock do carrinho por `SELECT ... FOR UPDATE`.
- `orders.source_cart_id` unique impede que o mesmo carrinho crie pedidos duplicados.
- Depois do commit, o backend expira o cookie `printlab_cart`.
- O pedido preserva snapshot de itens, precos, frete, cliente, endereco e receita de producao.
- `order_item_filaments` nao referencia `materials`, `colors` ou `variant_filaments`, para preservar historico.
- A rota publica `/pedido/{id}` aceita somente UUID e nao deve expor pedido por `order_number`.
- `order_number` nao e mecanismo de autorizacao.
- A pagina publica de pedido nao deve renderizar CPF completo, endereco completo, telefone ou e-mail completo.
- A pagina publica de pedido pode renderizar link `Acompanhar pedido`, mas esse link deve usar `orders.public_tracking_id`, nunca `orders.id` nem `order_number`.
- Logs de pedido podem conter UUID, `order_number`, status e conversao de carrinho; nao devem conter CPF, e-mail, telefone, endereco ou token de carrinho.

## Acompanhamento de pedido

- `/acompanhar/{public_tracking_id}` trata o link como capability URL.
- `public_tracking_id` e UUID aleatorio persistido, unico e obrigatorio.
- `public_tracking_id` nao substitui autenticacao e nao deve ser logado.
- UUID invalido e UUID desconhecido retornam 404.
- A consulta usa `orders.public_tracking_id` e lista colunas explicitamente.
- A view publica nao deve conter CPF, e-mail, telefone, endereco, UUID interno do pedido, `source_cart_id`, `transaction_nsu`, `invoice_slug`, checkout URL, itens, produtos, valores, peso, dimensoes, filamento, material ou cor.
- A resposta usa `Cache-Control: private, no-store`.
- A resposta usa `X-Robots-Tag: noindex, nofollow, noarchive`.
- O HTML usa meta robots `noindex, nofollow, noarchive`.
- A resposta usa `Referrer-Policy: no-referrer`.
- Nao existe endpoint publico de mutacao de producao ou envio.

## Admin

- Supabase Auth autentica e-mail e senha em `POST /admin/login`.
- A PrintLab autoriza separadamente comparando `user.id` com `ADMIN_SUPABASE_USER_ID`.
- E-mail nao e regra de autorizacao administrativa.
- A aplicacao nao possui signup administrativo, criacao de conta, login social, lembrar de mim ou recuperacao de senha nesta subfase.
- Senha administrativa vai somente para Supabase Auth, nunca e persistida, logada, colocada em query string ou armazenada em sessao.
- Access token e refresh token do Supabase nao sao persistidos pela PrintLab.
- Depois de autenticar e autorizar, a PrintLab cria sessao propria com token opaco aleatorio de 32 bytes.
- `public.admin_sessions.token_hash` armazena somente `SHA-256(token)`, com constraint de 32 bytes.
- `admin_sessions.expires_at` controla TTL inicial de 8 horas; nao ha renovacao automatica nesta fase.
- `admin_sessions` tem RLS habilitado e nenhuma policy publica.
- Nao ha FK para `auth.users`; a autorizacao e feita no backend com o UUID retornado pelo Auth.
- Cookie administrativo `printlab_admin_session` usa `HttpOnly`, `SameSite=Strict`, `Path=/admin`, host-only e `Secure` em producao ou quando `SITE_URL` usa HTTPS.
- Todas as paginas `/admin` usam `Cache-Control: private, no-store`, `X-Robots-Tag: noindex, nofollow, noarchive` e `Referrer-Policy: same-origin`, preservando referrer apenas em navegacoes same-origin.
- `POST /admin/login` e `POST /admin/logout` reutilizam validacao administrativa estrita de `Origin`/`Referer`; `Origin: null`, cross-site e ausencia simultanea de `Origin` e `Referer` sao rejeitados. `Referer` same-origin e apenas fallback quando `Origin` estiver ausente. `SameSite=Strict` e camada adicional, nao substituta.
- `GET /admin/pedidos`, `GET /admin/pedidos/{order_id}`, `POST /admin/pedidos/{order_id}/producao` e `POST /admin/pedidos/{order_id}/envio` exigem sessao administrativa valida.
- Rotas de catalogo Admin em `/admin/produtos`, `/admin/categorias`, `/admin/materiais`, `/admin/cores` e `/admin/caixas` tambem exigem sessao administrativa valida.
- POSTs administrativos de producao, envio, catalogo e imagens reutilizam validacao administrativa estrita de `Origin`/`Referer`; `Origin: null`, cross-site e ausencia simultanea de `Origin` e `Referer` sao rejeitados.
- Formularios administrativos possuem limite de body de 256 KiB. Endpoints JSON de imagens possuem limite de 64 KiB e recebem somente metadados; bytes de imagem vao direto do navegador ao Supabase Storage por signed upload URL.
- O ator de auditoria operacional vem de `Session.AuthUserID`, nunca de campo de formulario, query string ou header enviado pelo navegador.
- Mutacoes administrativas de producao/envio atualizam `order_fulfillment` e inserem `admin_order_events` em uma unica transacao PostgreSQL com lock do pedido.
- Listagem administrativa de pedidos nao deve carregar PII nem identificadores tecnicos de pagamento.
- Detalhe administrativo de pedido pode renderizar PII operacional somente apos autenticacao e autorizacao administrativas.
- `admin_order_events` registra pedido, UUID do usuario Auth, tipo de evento, status anterior, status novo e horario, sem PII de cliente.
- Catalogo Admin nao deve consultar pedidos, PII de clientes, dados InfinitePay, checkout URL, `transaction_nsu`, `invoice_slug` ou UUID interno de pedido.
- Mutacoes de catalogo nao reutilizam `admin_order_events` e nao criam auditoria de catalogo antecipada.
- Logs administrativos podem registrar somente eventos genericos como login bem-sucedido, login falho, logout, sessao expirada e erro de repository.
- Logs administrativos nao devem registrar e-mail, senha, token de sessao, token hash, access token, refresh token, publishable key, secret key, PII de clientes ou connection strings.
- Dashboard e listagem de pedidos mostram apenas informacoes agregadas ou operacionais minimizadas e nao carregam CPF, endereco, telefone, e-mail de cliente, `transaction_nsu`, `invoice_slug` ou checkout URL.
- Supabase Auth possui rate limits proprios; protecoes adicionais contra abuso devem ser configuradas operacionalmente por Vercel Firewall/WAF apos observacao de trafego real.

## Pagamentos InfinitePay

- `INFINITEPAY_HANDLE` deve ser configurado por environment variable, sem valor real no Git.
- O backend nao usa token/API secret InfinitePay nesta fase.
- A pagina publica do pedido nao deve renderizar checkout URL, `transaction_nsu`, `invoice_slug` ou detalhes tecnicos.
- Logs de pagamento podem conter UUID e `order_number`, mas nao checkout URL, e-mail, telefone, endereco, query params completos, `transaction_nsu`, `invoice_slug` ou secrets.
- Falhas da InfinitePay devem preservar diagnostico seguro com `provider`, `operation`, status HTTP quando houver e categoria controlada, sem body bruto, payload completo, checkout URL completa, PII, NSU de transacao ou secrets.
- Categorias seguras de falha da InfinitePay incluem `network_error`, `timeout`, status HTTP mapeados, `invalid_json`, `invalid_checkout_url` e `unknown`.
- `order_nsu` e derivado do UUID do pedido e nao deve ser tratado como autenticacao.
- `POST /webhooks/infinitepay` aceita chamadas sem `Origin`/`Referer`, exige JSON limitado a 64 KiB e nunca confirma pagamento diretamente pelo payload recebido.
- Webhook InfinitePay e apenas gatilho para `payment_check` server-side. Pedido so muda para `paid` quando `success=true`, `paid=true` e `amount` iguala `orders.total_cents`.
- Payload de webhook nao pode gravar `receipt_url`, itens, dados de cliente ou qualquer detalhe de PII.
- `order_payments` tem RLS habilitado e nenhuma policy publica.
- Falhas de API, timeout, JSON invalido, retorno com `paid=false` ou abandono de checkout nao podem marcar pedido como pago.

## Limites

- Autenticacao e autorizacao administrativas basicas estao implementadas apenas para um usuario Supabase Auth autorizado por UUID.
- Nao ha papeis multiplos, MFA obrigatorio, CAPTCHA, rate limiter em Go ou alteracao de valores/dados de pedidos nesta subfase.
- Webhook InfinitePay esta implementado sem HMAC/IP allowlist porque o contrato publico consultado nao documenta assinatura; a autoridade permanece no `payment_check` server-side.
- Recebimento real de webhook InfinitePay em producao foi validado na Fase 11.
- Acompanhamento publico de pedido esta implementado por `public_tracking_id`, sem login e com minimizacao de dados.
- Processamento de pagamento existe como checkout hospedado InfinitePay, retorno por `payment_check` e webhook redundante por `payment_check`.
- As tabelas de negocio implementadas cobrem catalogo, variantes, receita estimada de producao, imagens, carrinho, dados temporarios de checkout, frete e pedidos.
- `GET /ready` nao expoe detalhes internos do PostgreSQL.
- Nao ha escrita publica em Storage, alteracao administrativa de valores/dados de pedidos ou postagem/rastreio externo. Upload de imagens existe apenas no Admin, com signed upload URL e finalizacao server-side.

## Praticas recomendadas

- Validar entradas no servidor.
- Registrar eventos importantes sem expor dados sensiveis.
- Usar transacoes para alteracoes financeiras.
- Projetar idempotencia antes de processar webhooks.
- Revisar dependencias antes de adiciona-las.
- Usar `TEST_DATABASE_URL` para testes opcionais de integracao com banco, nunca `DATABASE_URL` de producao.
- Validar paths de Storage antes de montar URL publica de imagem.
- Revisar headers globais, CSP e limites de body apos cada nova rota publica ou administrativa.
- Configurar protecoes de Vercel Firewall/WAF inicialmente em modo observacao antes de bloquear trafego.

## Praticas proibidas

- Confirmar pagamento por parametro de URL ou redirect.
- Salvar secrets em codigo, fixtures, logs, documentacao ou exemplos.
- Aceitar preco, desconto ou frete do cliente como valor final.
- Enviar token SuperFrete para o navegador ou para base URL configuravel por usuario/env.
- Inserir caixa ficticia ou dimensao ficticia para forcar cotacao.
- Processar webhook como autoridade direta sem `payment_check` server-side e protecao contra duplicidade.
- Executar migrations automaticamente no startup do servidor web.
- Usar Table Editor ou SQL Editor remoto como workflow normal de mudanca de schema.
- Criar policy publica de `INSERT`, `UPDATE` ou `DELETE` em `storage.objects` para imagens de produto.
- Armazenar URL externa em `product_images.storage_path`.
