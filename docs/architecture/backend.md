# Backend

Status: fundacao HTTP, banco, catalogo, variantes, producao, carrinho, dados de checkout, frete, pedidos, pagamentos, webhook InfinitePay, acompanhamento seguro e operacao administrativa basica IMPLEMENTADOS.

## Responsabilidade

O backend Go sera a camada autoritativa da aplicacao. Ele recebera requisicoes HTTP, validara entradas, aplicara regras de negocio, acessara o PostgreSQL e renderizara respostas server-side.

Nesta fase, o backend implementa:

- `GET /` para homepage server-side.
- `GET /produtos` para catalogo publico.
- `GET /produtos/{slug}` para detalhe publico de produto ativo.
- `GET /produtos/{slug}?variante=<slug>` para detalhe com variante selecionada por slug.
- `GET /carrinho` para carrinho anonimo SSR.
- `POST /carrinho/adicionar` para adicionar ou incrementar item.
- `POST /carrinho/itens/{id}/quantidade` para alterar quantidade.
- `POST /carrinho/itens/{id}/remover` para remover item.
- `GET /checkout/dados` para formulario SSR de contato e endereco.
- `POST /checkout/dados` para validar e salvar dados temporarios de checkout.
- `GET /checkout/frete` para calcular e renderizar cotacoes atuais de frete.
- `POST /checkout/frete` para revalidar e persistir a selecao de frete por carrinho.
- `GET /checkout/revisao` para revisar checkout sem recotar frete.
- `POST /checkout/revisao` para criar pedido pendente de pagamento.
- `GET /pedido/{id}` para exibir pedido por UUID.
- `POST /pedido/{id}/pagar` para iniciar ou reutilizar checkout hospedado InfinitePay.
- `GET /pagamento/retorno` para validar retorno com `payment_check`.
- `POST /webhooks/infinitepay` para receber webhook InfinitePay e confirmar por `payment_check`.
- `GET /acompanhar/{public_tracking_id}` para acompanhamento seguro e minimizado do pedido.
- `GET /admin/login` para formulario SSR de login administrativo ou indisponibilidade segura quando config Admin estiver ausente.
- `POST /admin/login` para autenticar e-mail/senha no Supabase Auth, autorizar por UUID e criar sessao propria.
- `GET /admin` para dashboard administrativo com contagens agregadas.
- `GET /admin/pedidos` para listagem administrativa autenticada de pedidos.
- `GET /admin/pedidos/{orderID}` para detalhe administrativo autenticado de pedido.
- `POST /admin/pedidos/{orderID}/producao` para avancar status de producao.
- `POST /admin/pedidos/{orderID}/envio` para avancar status de envio.
- `GET /admin/produtos`, `GET /admin/produtos/novo`, `POST /admin/produtos`, `GET /admin/produtos/{productID}` e `POST /admin/produtos/{productID}` para gestao administrativa de produtos.
- Rotas administrativas de variantes e receita abaixo de `/admin/produtos/{productID}/variantes`.
- Rotas administrativas de categorias, materiais, cores e caixas em `/admin/categorias`, `/admin/materiais`, `/admin/cores` e `/admin/caixas`.
- `POST /admin/logout` para apagar sessao administrativa e limpar cookie.
- `GET /health` para liveness.
- `GET /ready` para readiness de banco.
- `/static/...` para assets embutidos.
- `internal/config` para ler configuracao.
- `internal/database` para criar `pgxpool.Pool`.
- `internal/products` para modelos, service e repository PostgreSQL do catalogo, variantes, receita estimada e imagens.
- `internal/cart` para token/cookie, service e repository PostgreSQL do carrinho.
- `internal/customers` para dados temporarios de checkout, validacoes brasileiras e repository PostgreSQL transacional.
- `internal/shipping` para perfis logisticos, caixas fisicas, cotacao SuperFrete, selecao de frete e repository PostgreSQL.
- `internal/orders` para revisao, fingerprint, criacao transacional e snapshot de pedidos.
- `internal/orders` tambem expoe a view minimizada de acompanhamento por `public_tracking_id`.
- `internal/payments` para client InfinitePay, regras de pagamento e repository PostgreSQL.
- `internal/admin` para cliente Supabase Auth, token/cookie administrativo, sessao server-side, dashboard agregado, consultas administrativas de pedido, mutacoes auditadas de producao/envio e gestao administrativa de catalogo.

## Limites

- O schema de negocio implementado cobre catalogo, variantes, receita estimada de producao, imagens, carrinho, dados temporarios de checkout, perfis logisticos, caixas fisicas, selecao de frete, pedidos, pagamentos, acompanhamento, sessoes administrativas e auditoria operacional de pedidos.
- Admin implementa autenticacao, autorizacao, sessao, logout, dashboard, listagem/detalhe de pedidos, mutacoes auditadas de producao/envio e gestao de catalogo, variantes, receita, materiais, cores e caixas. Alteracao de valores/dados do pedido, integracao de postagem/rastreio e upload de imagens permanecem fora do escopo.
- A integracao comercial externa implementada nesta fase e somente cotacao SuperFrete. Etiqueta, postagem e rastreio permanecem fora do escopo.
- A homepage ainda nao depende obrigatoriamente do PostgreSQL.
- Nao ha upload de imagens pelo app.

## Decisoes

- Usar `net/http` como base HTTP.
- Usar `pgx/v5` e `pgxpool` para PostgreSQL.
- Separar areas futuras em `internal/`, sem codigo artificial.
- Centralizar regras financeiras no backend.
- Usar `DATABASE_URL` como unica fonte de verdade da conexao PostgreSQL.
- Usar `DB_MAX_CONNS` com default `4` para limitar conexoes por instancia.
- Configurar `pgx.QueryExecModeExec` para compatibilidade com Supabase Transaction Pooler.
- Usar `internal/products` como primeira vertical slice: handler HTTP, service, repository `pgxpool` e templates SSR.
- Expor produto publicamente por slug, nunca por UUID.
- Exibir publicamente apenas produtos ativos.
- Tratar produto inativo como inexistente.
- Representar preco-base como inteiro em centavos.
- Representar override de preco de variante como inteiro em centavos opcional.
- Calcular preco efetivo no service, usando `product_variants.price_cents` quando preenchido e `products.price_cents` como fallback.
- Representar peso estimado de filamento em miligramas como inteiro.
- Representar tempo estimado de maquina em minutos como inteiro.
- Construir URL publica de imagem em um helper de dominio a partir de `SUPABASE_URL`, bucket `product-images` e `storage_path`.
- Persistir carrinho anonimo server-side, usando cookie opaco e `SHA-256` no banco.
- Recalcular preco e subtotal do carrinho a partir do catalogo atual.
- Manter mutacoes de item limitadas por `cart_id` e `item_id`.
- Persistir dados temporarios de contato e endereco vinculados ao carrinho, sem entidade permanente de cliente.
- Validar CPF, telefone, CEP, UF e pais no backend.
- Salvar contato e endereco em transacao PostgreSQL.
- Calcular frete no backend, nunca a partir de preco enviado pelo navegador.
- Usar duas chamadas ao calculator da SuperFrete: `products` para obter pacote ideal e `package` com caixa fisica real para cotacao final.
- Escolher a menor caixa real ativa que comporte o pacote ideal usando dimensoes internas e rotacao.
- Persistir selecao de frete com snapshot do pacote real, preco em centavos, validade de 30 minutos e `input_hash`.
- Criar pedidos como snapshots imutaveis de checkout.
- Usar `orders.source_cart_id` como defesa de idempotencia para confirmacao duplicada.
- Usar UUID em `/pedido/{id}` e `order_number` apenas como referencia humana.
- Iniciar pagamento hospedado a partir do pedido congelado, nunca a partir do carrinho.
- Confirmar pagamento somente por `payment_check` server-side.
- Autenticar Admin por Supabase Auth sem armazenar senha, access token ou refresh token na PrintLab.
- Autorizar Admin por `ADMIN_SUPABASE_USER_ID`, nunca por e-mail.
- Usar sessao propria com token opaco, hash SHA-256 no PostgreSQL, cookie HttpOnly `SameSite=Strict` e TTL de 8 horas.
- Validar `Origin`/`Referer` em POSTs administrativos e rejeitar `Origin: null` independentemente de `Referer`.
- Usar `Session.AuthUserID` como ator de auditoria para mutacoes administrativas.
- Atualizar producao/envio e inserir auditoria na mesma transacao PostgreSQL com lock do pedido.

## Catalogo

`GET /produtos` lista produtos ativos e aceita filtro opcional `categoria=<slug>`. O filtro e validado antes da consulta ao banco e usa slug publico.

`GET /produtos/{slug}` valida o slug e busca apenas produto ativo. Slug invalido, produto inexistente e produto inativo retornam HTTP 404.

`GET /produtos/{slug}?variante=<slug>` valida o slug da variante antes de chamar o service. Variante invalida, inexistente, inativa ou pertencente a outro produto retorna HTTP 404.

Quando um produto possui variantes ativas, o service escolhe automaticamente a variante default ativa; se nao existir, usa a primeira variante ativa pela ordenacao publica. Produto sem variantes continua valido e usa o preco-base.

As consultas de catalogo evitam N+1 em Go. A listagem calcula o menor preco efetivo e busca imagem geral primaria em uma consulta. O detalhe carrega produto, variantes, receitas e imagens em consultas separadas e coesas.

Se o banco estiver indisponivel, rotas de catalogo retornam HTTP 503 com resposta generica, sem detalhes do PostgreSQL.

## Carrinho

`GET /carrinho` renderiza carrinho vazio quando nao ha cookie valido. A rota nao cria registro no banco apenas por leitura.

`POST /carrinho/adicionar` recebe `product_slug`, `variant_slug` opcional e `quantity`. O backend valida slugs, quantidade, produto ativo, variante ativa quando exigida e nao aceita preco do frontend.

Produto com variantes ativas exige variante valida. Produto sem variantes ativas pode ser adicionado sem variante.

Adicionar produto/variante ja existente incrementa a linha de forma atomica no SQL e respeita o limite 99.

`POST /carrinho/itens/{id}/quantidade` e `POST /carrinho/itens/{id}/remover` atuam somente quando o item pertence ao carrinho atual. Ambas usam `cart_id` junto de `item_id`.

Itens que ficam indisponiveis continuam aparecendo no carrinho, podem ser removidos e nao entram no subtotal.

Mutacoes bem-sucedidas renovam a expiracao do carrinho e do cookie para 30 dias.

## Dados de checkout

`GET /checkout/dados` exige cookie de carrinho valido, carrinho ativo, pelo menos um item e nenhum item indisponivel. Sem essa condicao, a rota redireciona para `/carrinho` e nao coleta PII.

`POST /checkout/dados` reutiliza a validacao centralizada de `Origin`/`Referer`, valida o carrinho atual e normaliza os campos em `internal/customers`.

O backend aceita entradas humanas de CPF, telefone e CEP com mascara, mas persiste valores canonicos. O pais e limitado a `BR`. Contato e endereco sao salvos por `INSERT ... ON CONFLICT (cart_id) DO UPDATE` dentro de uma transacao.

Salvamento bem-sucedido renova a validade do carrinho e do cookie e redireciona para `/checkout/frete`. Se houver erro de validacao, o formulario e renderizado novamente com mensagens por campo. Se houver erro de infraestrutura, a resposta e generica e nao expoe CPF, e-mail, telefone, endereco ou detalhes PostgreSQL.

## Frete

`GET /checkout/frete` exige cookie de carrinho valido, carrinho ativo, pelo menos um item, nenhum item indisponivel e dados de checkout ja salvos. Sem carrinho valido, redireciona para `/carrinho`. Sem dados, redireciona para `/checkout/dados`.

O service resolve o perfil logistico efetivo de cada linha: variante com perfil completo sobrescreve o produto; caso contrario usa o perfil completo do produto. Campos parciais nao sao misturados. Produto sem perfil efetivo torna a cotacao indisponivel sem estimativa ficticia.

A cotacao usa duas chamadas SuperFrete:

1. Planejamento com `products`, usando peso em kg e dimensoes em cm convertidos a partir dos valores internos em gramas e milimetros.
2. Cotacao final com `package`, usando peso dos produtos somado a `shipping_boxes.packaging_weight_g` e dimensoes externas da menor caixa real compativel.

Somente o resultado da segunda chamada e apresentado ao cliente. O POST recebe apenas `service_code`, reexecuta a cotacao atual, persiste a opcao se ela ainda existir e ignora qualquer preco ou dimensao que o navegador tente enviar.

Selecoes antigas sao consideradas invalidas se expiraram ou se o `input_hash` atual diverge por mudanca de carrinho, variante, perfil logistico, CEP, servicos ou caixa.

## Pedidos

`GET /checkout/revisao` exige carrinho valido, nao convertido, nao vazio, sem itens indisponiveis, com dados completos e frete selecionado valido. Se faltar dados, redireciona para `/checkout/dados`; se faltar frete valido, redireciona para `/checkout/frete`; se faltar carrinho valido, redireciona para `/carrinho`.

A revisao nao chama a SuperFrete. Ela recalcula o `input_hash` esperado para a selecao persistida e compara com o valor salvo em `cart_shipping_selections`.

`POST /checkout/revisao` recebe apenas `review_fingerprint` como deteccao de tela antiga. O backend revalida disponibilidade, dados, frete, subtotal e total, cria o pedido em uma transacao PostgreSQL, converte o carrinho com `converted_at`, limpa dados temporarios e expira o cookie depois do commit.

O pedido copia snapshots de itens, preco, frete, dados de cliente, endereco e receita de producao 3D. `order_item_filaments` preserva material, cor, peso e label sem depender de `materials`, `colors` ou `variant_filaments`.

`GET /pedido/{id}` aceita somente UUID. A pagina mostra status humano, itens, frete e totais, mas nao exibe CPF completo, endereco completo, telefone ou e-mail completo.

`GET /acompanhar/{public_tracking_id}` aceita somente UUID, consulta por `orders.public_tracking_id` e renderiza apenas numero humano, data, status de pagamento, status de producao, status de envio e transportadora/servico comercial quando houver. Identificador invalido e desconhecido retornam 404. A rota usa headers `private, no-store`, `noindex` e `no-referrer` e nao loga o tracking ID.

## Admin

Rotas `/admin/*` usam headers privados/noindex e `Referrer-Policy: same-origin`.

`GET /admin/pedidos` lista pedidos com dados minimizados. `GET /admin/pedidos/{orderID}` carrega o detalhe operacional completo somente apos sessao administrativa valida.

`POST /admin/pedidos/{orderID}/producao` e `POST /admin/pedidos/{orderID}/envio` validam origem, resolvem sessao, usam o `AuthUserID` da sessao como ator e delegam as regras ao pacote `internal/admin`. O repository bloqueia pedido/fulfillment, valida transicao sequencial, atualiza `order_fulfillment` e insere `admin_order_events` na mesma transacao.

## Pagamentos

`POST /pedido/{id}/pagar` aceita somente UUID, valida `Origin`/`Referer` e exige pagamento configurado por `INFINITEPAY_HANDLE` e `SITE_URL` HTTPS. Quando o pedido esta `pending_payment`, o service monta o payload InfinitePay a partir dos snapshots historicos do pedido e confere o total antes de chamar `POST /links`.

O client InfinitePay usa `net/http`, timeout explicito, `context.Context`, base URL interna fixa `https://api.checkout.infinitepay.io` e nao adiciona SDK ou dependencia nova.

Checkout URL retornada pelo provedor e aceita somente se for HTTPS nos hosts `checkout.infinitepay.io` ou `checkout.infinitepay.com.br`.

`GET /pagamento/retorno` nao confirma pagamento por redirect. Ele valida parametros seguros, chama `POST /payment_check` e altera `orders.status` para `paid` em transacao somente quando `success=true`, `paid=true` e `amount` igual a `orders.total_cents`.

`POST /webhooks/infinitepay` nao confirma pagamento diretamente pelo payload recebido. Ele aceita JSON limitado, valida identificadores, chama `POST /payment_check` e usa a mesma regra transacional do retorno. Pagamentos ja confirmados retornam sucesso sem duplicar atualizacao.

## Health e readiness

`GET /health` e liveness. Ele sempre responde HTTP 200 com body `ok` quando o processo HTTP esta vivo e nao consulta o PostgreSQL.

`GET /ready` e readiness. Ele faz `Ping` com timeout de 3 segundos quando `DATABASE_URL` esta configurada. Sem `DATABASE_URL`, retorna HTTP 503. Esse comportamento e temporario enquanto a homepage nao depende do banco.

## Praticas recomendadas

- Handlers pequenos e explicitos.
- Validacao de entrada antes de chamar regras de dominio.
- Erros tratados explicitamente.
- `context.Context` quando a operacao envolver I/O, banco, chamadas externas ou cancelamento.
- Testes deterministico para regras criticas.
- Pacotes coesos por responsabilidade.
- Fechar `pgxpool.Pool` no encerramento do processo.
- Usar parametros PostgreSQL para entradas externas.
- Listar colunas explicitamente em SQL.

## Praticas proibidas

- Confiar em valores financeiros vindos do navegador.
- Adicionar dependencia sem justificativa.
- Criar interfaces prematuras sem multiplos consumidores ou necessidade clara de teste.
- Implementar integracoes reais sem plano aprovado.
- Colocar secrets no codigo, testes ou documentacao.
- Executar migrations no startup da aplicacao.
- Logar `DATABASE_URL`, senha, token ou connection string.
- Usar `SELECT *`.
- Concatenar valores externos em SQL.
