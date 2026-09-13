# Painel administrativo

Status: Fases 13.1 e 13.2 CONCLUIDAS; Fase 13.3 com implementacao concluida e validacao real pendente; Fase 13.4 PLANEJADA.

O painel administrativo concentrara funcionalidades internas de operacao da PrintLab em subfases pequenas.

## Status por subfase

- 13.1 — autenticacao, autorizacao, sessao e shell administrativo: concluida.
- 13.2 — pedidos, producao, envio e auditoria: concluida com validacao real em producao.
- 13.3 — catalogo, variantes, materiais, cores e caixas: implementacao concluida; validacao real pendente.
- 13.4 — imagens e Supabase Storage: planejada.

## Fase 13.1 implementada

Rotas:

- `GET /admin/login`
- `POST /admin/login`
- `GET /admin`
- `POST /admin/logout`

O login usa Supabase Auth com e-mail e senha. A PrintLab autoriza somente o usuario cujo `user.id` corresponde a `ADMIN_SUPABASE_USER_ID`.

A sessao administrativa e propria da PrintLab:

- token aleatorio opaco;
- cookie `printlab_admin_session`;
- `HttpOnly`;
- `SameSite=Strict`;
- `Path=/admin`;
- `Secure` em producao;
- TTL de 8 horas;
- `SHA-256(token)` persistido em `public.admin_sessions`.

O dashboard inicial e somente leitura e mostra contagens agregadas:

- pedidos aguardando pagamento;
- pedidos pagos aguardando producao;
- pedidos em producao;
- pedidos aguardando envio.

`pending_payment` nao entra como aguardando producao.

## Fase 13.2 implementada

Rotas:

- `GET /admin/pedidos`
- `GET /admin/pedidos/{order_id}`
- `POST /admin/pedidos/{order_id}/producao`
- `POST /admin/pedidos/{order_id}/envio`

A listagem administrativa de pedidos e protegida por sessao admin, usa SSR e nao carrega PII de cliente nem identificadores tecnicos de pagamento. Ela mostra somente numero humano, data, status de pagamento, status de producao, status de envio, total e contagens de itens/unidades.

O detalhe administrativo de pedido e protegido por sessao admin e pode exibir dados necessarios para operacao:

- nome, e-mail, telefone e CPF do cliente;
- endereco completo de entrega;
- frete selecionado, caixa, peso e dimensoes;
- itens, variantes, SKU e snapshots de producao;
- resumo de pagamento sem `checkout_url`, `transaction_nsu` ou `invoice_slug`;
- trilha de auditoria operacional.

Mutacoes de producao e envio sao POSTs administrativos, validam `Origin`/`Referer`, exigem sessao valida e usam `Session.AuthUserID` como ator do evento de auditoria.

Transicoes permitidas:

- producao: `waiting` -> `in_production` -> `completed`;
- envio: `waiting` -> `preparing` -> `shipped` -> `delivered`.

Regras:

- pedido `pending_payment` nao pode iniciar producao;
- envio so pode sair de `waiting` depois de producao `completed`;
- nao ha regressao de status;
- `delivered` e terminal para operacoes de envio;
- transicao repetida ou stale e tratada como conflito;
- a atualizacao de `order_fulfillment` e o insert em `admin_order_events` ocorrem na mesma transacao PostgreSQL com lock do pedido.

## Validacao real da Fase 13.2

A validacao real em producao foi concluida sem registrar dados pessoais reais, UUID real do admin, `order_id` interno real ou identificadores privados.

Evidencias funcionais confirmadas manualmente:

- `/admin/pedidos` carregou corretamente;
- detalhe administrativo de pedido carregou corretamente;
- pedido pago avancou producao em sequencia: `waiting` -> `in_production` -> `completed`;
- envio avancou em sequencia: `waiting` -> `preparing` -> `shipped` -> `delivered`;
- `public.order_fulfillment` terminou com `production_status = completed` e `shipping_status = delivered`;
- `public.admin_order_events` registrou um evento por transicao;
- auditoria preservou `from_status`, `to_status` e `created_at`;
- acompanhamento publico refletiu os estados operacionais;
- transicoes invalidas foram bloqueadas;
- pedido `pending_payment` nao pode iniciar producao;
- `delivered` e terminal.

## Fase 13.3 implementada

Rotas de catalogo:

- `GET /admin/produtos`
- `GET /admin/produtos/novo`
- `POST /admin/produtos`
- `GET /admin/produtos/{product_id}`
- `POST /admin/produtos/{product_id}`
- `GET /admin/produtos/{product_id}/variantes/nova`
- `POST /admin/produtos/{product_id}/variantes`
- `GET /admin/produtos/{product_id}/variantes/{variant_id}`
- `POST /admin/produtos/{product_id}/variantes/{variant_id}`
- `POST /admin/produtos/{product_id}/variantes/{variant_id}/receita`
- `POST /admin/produtos/{product_id}/variantes/{variant_id}/receita/{component_id}`
- `POST /admin/produtos/{product_id}/variantes/{variant_id}/receita/{component_id}/remover`
- `GET /admin/categorias`
- `GET /admin/categorias/nova`
- `POST /admin/categorias`
- `GET /admin/categorias/{id}`
- `POST /admin/categorias/{id}`
- `GET /admin/materiais`
- `GET /admin/materiais/novo`
- `POST /admin/materiais`
- `GET /admin/materiais/{id}`
- `POST /admin/materiais/{id}`
- `GET /admin/cores`
- `GET /admin/cores/nova`
- `POST /admin/cores`
- `GET /admin/cores/{id}`
- `POST /admin/cores/{id}`
- `GET /admin/caixas`
- `GET /admin/caixas/nova`
- `POST /admin/caixas`
- `GET /admin/caixas/{id}`
- `POST /admin/caixas/{id}`

A 13.3 torna administraveis categorias, produtos, variantes, receitas estimadas de producao, materiais logicos, cores logicas e caixas fisicas de envio usando as tabelas ja existentes. Nenhuma migration foi criada para esta fase.

Regras administrativas:

- entidades principais usam ativacao/inativacao por `is_active`; nao ha hard delete de categorias, produtos, variantes, materiais, cores ou caixas;
- componentes atuais de `variant_filaments` podem ser adicionados, editados ou removidos porque pedidos antigos usam snapshots historicos;
- alteracoes de catalogo nao atualizam pedidos historicos, itens de pedido, snapshots de receita, frete congelado ou pagamentos;
- produtos e variantes usam preco em BRL no formulario e persistem centavos inteiros, sem `float`;
- receita usa gramas na UI Admin e persiste `estimated_weight_mg` como inteiro;
- materiais e cores inativos continuam carregaveis em receitas existentes e sao indicados como inativos;
- novas escolhas de receita usam somente materiais e cores ativos;
- produto ou variante pode ter perfil logistico completo ou nenhum perfil; perfis parciais sao rejeitados;
- caixa inativa deixa de participar de novas cotacoes, sem apagar selecoes historicas;
- imagens permanecem fora do escopo e ficam para a Fase 13.4.

Mutacoes de catalogo usam POST, sessao administrativa obrigatoria, validacao centralizada de `Origin`/`Referer`, rejeicao de `Origin: null`, limite de body de 256 KiB e redirecionamento PRG com `303 See Other` em sucesso.

## Seguranca

- Nao ha signup administrativo pela aplicacao.
- O administrador inicial deve ser criado manualmente no Dashboard Supabase em Authentication -> Users.
- E-mail nao e autorizacao administrativa; o UUID do usuario e a fonte estavel.
- Senha, access token, refresh token, token de sessao e token hash nao devem aparecer em logs ou documentacao.
- Todas as respostas `/admin` usam `Cache-Control: private, no-store`, `X-Robots-Tag: noindex, nofollow, noarchive` e `Referrer-Policy: same-origin`.
- POSTs administrativos validam `Origin`/`Referer`; `Origin: null` e rejeitado independentemente de `Referer`.
- Dashboard e listagem de pedidos nao carregam nem renderizam CPF, endereco, telefone, e-mail de cliente, `transaction_nsu`, `invoice_slug` ou checkout URL.
- O detalhe de pedido pode renderizar PII operacional somente apos sessao administrativa valida.
- Eventos de auditoria guardam UUID do usuario Supabase Auth, tipo de evento, status anterior, status novo e horario; nao armazenam PII de cliente.
- Paginas de catalogo Admin nao fazem join com pedidos e nao carregam PII, dados InfinitePay, `order_id` interno ou URL de checkout.
- `admin_order_events` continua exclusivo para operacoes de pedidos; a 13.3 nao cria auditoria de catalogo.

## Limites atuais

- Nao ha alteracao de dados comerciais do pedido, valores, cliente, endereco ou pagamento.
- Nao ha upload de imagens.
- Nao ha papeis multiplos.
- Nao ha MFA obrigatorio nem CAPTCHA/WAF na aplicacao.
- Nao ha uso de `SUPABASE_SECRET_KEY` ou service role.
- Nao ha etiqueta, postagem, rastreio externo ou integracao logistica de despacho.
- Nao ha estoque fisico de filamento, marcas, lotes, carretel, custos calculados ou multiplos admins/papeis.

## Decisoes pendentes

- Politica de permissoes caso existam multiplos usuarios administrativos.
- Protecoes adicionais contra abuso/brute force na Fase 14.
