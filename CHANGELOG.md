# Changelog

Este arquivo segue a ideia de [Keep a Changelog](https://keepachangelog.com/), com secoes organizadas por versao.

## [Unreleased]

### Added

- Fundacao inicial do projeto.
- Estrutura documental.
- Arquitetura inicial.
- Fase 2 concluida com homepage server-side usando `templ`.
- Pipeline Tailwind CSS 4 via CLI npm.
- Design tokens iniciais e componentes visuais fundamentais.
- Servico de assets estaticos em `/static/`.
- Testes de homepage, health check, static CSS e rotas desconhecidas.
- Integracao da logo oficial inicial da PrintLab ao header e hero.
- Teste para entrega do asset de marca em `/static/images/branding/logo-printlab-primary.png`.
- Fase 2.1 — Brand Experience, com homepage mais editorial e linguagem grafica da marca.
- Workflow de GitHub Actions para migrations Supabase de desenvolvimento com dry-run antes da aplicacao.
- Fase 3 — Fundacao do Banco de Dados, com `pgx/v5`, `pgxpool`, `internal/config`, `internal/database`, `GET /ready`, Supabase CLI local e `supabase/config.toml`.
- `vercel.json` minimo configurando a regiao `gru1`.
- Politica permanente de Git do projeto com fluxo implementar, validar, commit e push.
- Fase 3.1 — Remote Environment Validation concluida, validando GitHub Actions -> Supabase DEV e Vercel -> PostgreSQL.
- Fase 4 — Catalogo, com primeiro schema de negocio, categorias, produtos, listagem SSR, filtro por categoria, detalhe de produto e empty state.
- Migration `create_catalog` para `public.categories` e `public.products`.
- Testes de service, handlers de catalogo, slug e formatacao BRL.
- Fase 5 — Produtos, Variantes e Producao, com materiais, cores, variantes, receita estimada de producao, imagens e bucket publico `product-images`.
- Migration `create_product_variants` para `public.materials`, `public.colors`, `public.product_variants`, `public.variant_filaments` e `public.product_images`.
- Supabase Storage para imagens publicas de catalogo, sem policy publica de upload.
- Selecao publica de variante por `?variante=<slug>` sem JavaScript obrigatorio.
- Preco efetivo de variante com override opcional e fallback para `products.price_cents`.
- Helpers de dominio para peso em miligramas, apresentacao em gramas, tempo de maquina e URL publica de imagem.
- Fase 5.1 — Correcao semantica da receita de producao.
- Testes de regressao para preservar componentes de receita com material ou cor inativos.
- Fase 6 — Carrinho anonimo persistido no PostgreSQL.
- Migration `create_carts` para `public.carts` e `public.cart_items`.
- Cookie opaco `printlab_cart` com token aleatorio, `HttpOnly`, `SameSite=Lax` e hash SHA-256 no banco.
- Rotas `GET /carrinho`, `POST /carrinho/adicionar`, `POST /carrinho/itens/{id}/quantidade` e `POST /carrinho/itens/{id}/remover`.
- Formulario real de adicionar ao carrinho no detalhe de produto, sem campos de preco enviados pelo frontend.
- Testes de carrinho para token, cookie, service, disponibilidade, subtotal, overflow, escopo de item e handlers HTTP.
- Fase 7 — Dados do Cliente e Endereco, com contato e endereco vinculados ao carrinho anonimo.
- Migration `create_cart_customer_details` para `public.cart_customer_details` e `public.cart_shipping_addresses`.
- Validacoes brasileiras de CPF, telefone, CEP, UF e pais `BR`, sem dependencia externa.
- Rotas `GET /checkout/dados` e `POST /checkout/dados` para salvar dados temporarios de checkout em transacao.
- Testes de dados de checkout para validacao, normalizacao, service, handlers, migration e transacao.
- Fase 7.1 — Hardening de Privacidade e Consistencia do Checkout.
- Testes de regressao para cache privado de checkout e estado parcial de contato/endereco.
- Fase 8 — Embalagem Real e Integracao de Frete SuperFrete, com implementacao, testes e validacao Sandbox real concluidos.
- Perfis logisticos opcionais em `products` e `product_variants`, com peso em gramas, dimensoes em milimetros e constraints all-or-none.
- Migration `add_shipping_profiles_and_selections` para `shipping_boxes` e `cart_shipping_selections`, sem seed de caixas ficticias.
- Escolha da menor caixa fisica real compativel com pacote ideal, usando dimensoes internas, rotacao e desempates deterministicos.
- Cliente SuperFrete server-side com Bearer token, `User-Agent`, timeout, base URLs controladas e DTOs isolados.
- Estrategia de duas cotacoes: `products` para pacote ideal e `package` com caixa real para preco final.
- Rotas `GET /checkout/frete` e `POST /checkout/frete` para cotacao e selecao de frete sem JavaScript obrigatorio.
- `input_hash` e validade de 30 minutos para invalidar selecoes de frete obsoletas.
- Fase 8.1 — UX do Checkout, Consulta de CEP e Diagnostico Seguro de Frete.
- JavaScript progressivo em `/static/js/checkout.js` para mascaras de CPF, telefone brasileiro e CEP.
- Endpoint interno `GET /api/cep/{cep}` com consulta server-side ao ViaCEP e resposta limitada a rua, bairro, cidade e UF.
- Diagnosticos seguros de frete por estagio e motivo, sem PII, secrets ou corpo bruto externo.
- Categorias seguras de erro do cliente SuperFrete para status HTTP, timeout e JSON invalido.
- Fase 9 — Revisao e Criacao de Pedidos, com snapshots imutaveis, `orders`, `order_items`, `order_item_filaments` e status inicial `pending_payment`.
- Migration `create_orders` adicionando `carts.converted_at` e tabelas historicas de pedido, sem seed ou dados ficticios.
- Rotas `GET /checkout/revisao`, `POST /checkout/revisao` e `GET /pedido/{id}`.
- Snapshot historico de cliente, endereco, frete, itens, preco, receita de producao e filamentos no momento de criacao do pedido.
- Fase 9.1 — Separacao entre dados operacionais preservados em snapshot e interface publica de pedidos.
- Fase 10 — Pagamentos InfinitePay, com checkout hospedado server-side, retorno por `payment_check` e validacao real pendente.
- Migration `add_order_payments` para `public.order_payments` e status `paid` em pedidos.
- Pacote `internal/payments` com client HTTP InfinitePay, service, repository PostgreSQL e testes.
- Rotas `POST /pedido/{id}/pagar` e `GET /pagamento/retorno`.
- ADR-0010 — Pagamento hospedado via InfinitePay.
- Fase 10.1 — Diagnostico Seguro da Integracao InfinitePay, preservando status HTTP, operacao e categoria sem expor PII, payload, checkout URL completa, transaction NSU ou secrets.
- Migration `allow_current_infinitepay_checkout_host`, alinhando a constraint `order_payments_checkout_url_host` aos hosts `checkout.infinitepay.io` e `checkout.infinitepay.com.br` aceitos pelo dominio Go.
- Fase 11 — Webhook InfinitePay, com `webhook_url` no checkout, endpoint `POST /webhooks/infinitepay`, confirmacao por `payment_check` server-side, idempotencia e logs seguros sem PII.
- ADR-0011 — Confirmacao redundante de pagamentos InfinitePay.

### Changed

- Versao minima de Go atualizada para 1.26.0.
- Assets estaticos passaram a ser servidos via `embed.FS` para melhorar compatibilidade com deploy na Vercel.
- Tokens de design refinados com base na paleta visual da marca.
- Homepage revisada para remover copy tecnica e ampliar presenca estrutural das cores da PrintLab.
- Fonte oficial de migrations alterada de `migrations/` para `supabase/migrations/`.
- Workflow Supabase atualizado para usar CLI `2.117.0`.
- Definition of Done atualizada para commit e push automaticos apos validacoes aplicaveis.
- GitHub Actions -> Supabase DEV validado por `workflow_dispatch` sem migration de negocio.
- `/ready` remoto validado com HTTP 200 apos configuracao segura de `DATABASE_URL` e `DB_MAX_CONNS` na Vercel.
- Navegacao principal atualizada com link real para `/produtos`.
- Migration de catalogo aplicada ao Supabase DEV e `/produtos` validado na Vercel com empty state.
- Catalogo e detalhe de produto atualizados para exibir menor preco efetivo, texto "A partir de", galeria SSR e imagem de produto/variante quando existir.
- `supabase/config.toml` passou a habilitar Storage local para validacao do bucket de imagens de catalogo.
- Migration de variantes/producao aplicada ao Supabase DEV e `/produtos` validado na Vercel com catalogo vazio.
- Receita de producao passou a carregar `variant_filaments` mesmo quando material ou cor referenciados estiverem inativos.
- `materials.is_active` e `colors.is_active` agora documentam somente a oferta para novas escolhas futuras, sem alterar receitas existentes.
- Navegacao principal atualizada com link real para `/carrinho`.
- Carrinho recalcula preco atual e subtotal no backend, preservando itens indisponiveis sem inclui-los no subtotal.
- Carrinho com itens disponiveis passa a apontar para a etapa real de dados em `/checkout/dados`.
- Respostas HTML de checkout com PII passam a usar `Cache-Control: private, no-store`.
- Leitura de contato e endereco de checkout consolidada em uma unica consulta SQL consistente.
- Salvamento de dados de checkout passa a redirecionar para a etapa real de frete.
- Etapa de dados passa a oferecer consulta opcional de CEP via backend, mantendo preenchimento manual e validacao server-side como fonte autoritativa.
- Selecao de frete passa a redirecionar para a etapa real de revisao do pedido.
- Carrinhos convertidos deixam de ser reutilizados no fluxo ativo por `carts.converted_at`.
- Revisao e pagina publica de pedido deixam de exibir SKU interno, dados de producao 3D e dados de embalagem fisica, preservando essas informacoes internamente.
- Pagina publica de pedido passa a exibir CTA real de pagamento quando InfinitePay esta configurada e `Pagamento confirmado` quando o pedido esta `paid`.
- Validacao de checkout URL da InfinitePay passa a aceitar por allowlist explicita `checkout.infinitepay.io` e `checkout.infinitepay.com.br`, mantendo rejeicao de HTTP, wildcard, subdominios e sufixos parecidos.
