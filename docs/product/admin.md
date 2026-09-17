# Painel administrativo

Status: Fases 13.1, 13.2, 13.3 e 13.4 CONCLUIDAS; Fase 13 concluida.

O painel administrativo concentrara funcionalidades internas de operacao da PrintLab em subfases pequenas.

## Status por subfase

- 13.1 — autenticacao, autorizacao, sessao e shell administrativo: concluida.
- 13.2 — pedidos, producao, envio e auditoria: concluida com validacao real em producao.
- 13.3 — catalogo, variantes, materiais, cores e caixas: concluida com validacao real em producao.
- 13.4 — imagens e Supabase Storage: concluida com validacao real em producao.
- 14.2 - MFA TOTP obrigatorio: concluido; validado em producao.
- 14.3 - WAF / anti-abuse: concluida e validada no Vercel Hobby; o login tem rate limit por IP e MFA nao recebe limite customizado adicional no edge.

## Fase 13.1 implementada

Rotas:

- `GET /admin/login`
- `POST /admin/login`
- `GET /admin`
- `POST /admin/logout`

O login usa Supabase Auth com e-mail e senha. A PrintLab autoriza somente o usuario cujo `user.id` corresponde a `ADMIN_SUPABASE_USER_ID`.

Desde a Fase 14.2, a senha inicia somente o fluxo MFA. No primeiro acesso, `/admin/mfa/setup` apresenta QR e chave manual para cadastrar um autenticador. Nos acessos seguintes, `/admin/mfa/challenge` pede o codigo; havendo varios TOTP verificados, o administrador escolhe qual usar. A sessao PrintLab so nasce depois de AAL2 confirmado no servidor.

O fluxo temporario dura ate 10 minutos. Codigo invalido permite nova tentativa sem reapresentar o secret; recarregar setup descarta fatores TOTP nao verificados e gera outro QR. Cancelar limpa o estado temporario. A tela funciona com forms SSR, sem JavaScript.

Sessoes anteriores ao MFA sao recusadas por `mfa_verified_at` nulo. Perder o autenticador pode bloquear acesso: seguir o [runbook de recuperacao](../integrations/supabase-auth.md#recuperacao-de-emergencia), sem reset ou bypass pela aplicacao.

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

A 13.3A refinou a semantica visivel de variantes:

- `product_variants` continua sendo o modelo interno e as rotas continuam usando `/variantes`;
- o Admin deve chamar esse conceito de "Configuracao" ou "Configuracoes do produto";
- a pagina de produto usa a secao "Configuracoes do produto" e a acao "Adicionar configuracao";
- uma unica configuracao ativa aparece como "Unica configuracao" em vez de "Nao default";
- com duas ou mais configuracoes ativas, o Admin mostra "Padrao" ou "Nao padrao";
- o formulario usa "Usar como opcao padrao" e explica que isso define a opcao inicialmente selecionada na loja quando houver mais de uma configuracao ativa;
- nenhuma migration foi necessaria para atualizar registros antigos.

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
- na Fase 13.3, imagens permaneceram fora do escopo e foram implementadas depois na Fase 13.4.

Mutacoes de catalogo usam POST, sessao administrativa obrigatoria, validacao administrativa estrita de `Origin`/`Referer`, rejeicao de `Origin: null`, rejeicao de requests sem os dois headers, limite de body de 256 KiB e redirecionamento PRG com `303 See Other` em sucesso.

## Validacao real da Fase 13.3

A validacao real em producao da Fase 13.3 e do refinamento 13.3A foi concluida pelo responsavel.

Evidencias funcionais confirmadas:

- carrinho validado com configuracoes;
- Admin validado para catalogo/configuracoes;
- produto com uma unica configuracao ativa nao expoe seletor artificial;
- produtos com multiplas configuracoes continuam oferecendo escolha publica;
- comportamento com multiplas configuracoes e default validado;
- `product_variants` permanece modelo interno, mas a UI usa "Configuracao" para o administrador.

## Fase 13.4 implementada

Rotas de imagens:

- `GET /admin/produtos/{product_id}/imagens`
- `POST /admin/produtos/{product_id}/imagens/upload-url`
- `POST /admin/produtos/{product_id}/imagens/finalizar`
- `POST /admin/produtos/{product_id}/imagens/{image_id}/substituir-url`
- `POST /admin/produtos/{product_id}/imagens/{image_id}/finalizar-substituicao`
- `POST /admin/produtos/{product_id}/imagens/{image_id}/remover`
- `POST /admin/produtos/{product_id}/imagens/{image_id}/ordem`
- `POST /admin/produtos/{product_id}/imagens/{image_id}/principal`

A tela de imagens permite:

- visualizar thumbnails cadastradas;
- associar imagem ao produto inteiro;
- associar imagem opcionalmente a uma Configuracao;
- enviar nova imagem;
- substituir imagem sem sobrescrever o object path anterior;
- alterar ordem;
- marcar imagem principal;
- remover associacao e, quando seguro, remover objeto gerenciado no Storage.

O upload nao envia bytes pelo backend Go/Vercel. O fluxo e:

1. Browser Admin envia ao Go somente metadados do arquivo.
2. Go valida sessao Admin, `Origin`/`Referer`, produto, Configuracao, MIME e tamanho.
3. Go gera object path imprevisivel em `products/{product_uuid}/{random}.ext`.
4. Go solicita signed upload URL ao Supabase Storage usando `SUPABASE_SECRET_KEY`.
5. Browser envia o arquivo diretamente ao Supabase Storage.
6. Browser chama a finalizacao no Go.
7. Go confirma o objeto no Storage por `GET /storage/v1/object/info/{bucket}/{path}`, usa `size` e `content_type` do JSON retornado e cria/atualiza `public.product_images`.

Regras:

- MIME aceitos pela aplicacao: `image/jpeg`, `image/png`, `image/webp`.
- SVG, GIF, PDF, video e arquivos genericos nao sao aceitos.
- A extensao e gerada pelo backend a partir do MIME validado: `.jpg`, `.png` ou `.webp`.
- O nome original do arquivo nao define o object path.
- O limite da aplicacao e 5 MB por imagem, alinhado ao bucket `product-images` ja existente.
- Cada substituicao cria novo path; nao ha overwrite no mesmo objeto.
- Imagem de Configuracao exige que a Configuracao pertenca ao mesmo produto.
- Falha de INSERT/UPDATE apos upload tenta cleanup best-effort do objeto recem-enviado.
- Remocao fisica so ocorre para path gerenciado e validado no prefixo `products/{product_id}/`.
- Remocao de imagem gerenciada exige Storage administrativo configurado; sem `SUPABASE_SECRET_KEY`, a UI nao promete remocao completa desse objeto.
- Imagens legadas/manuais fora desse prefixo podem ter associacao removida do banco, mas nao geram DELETE arbitrario.

`public.product_images` ja possuia `storage_path`, `sort_order` e `is_primary`; por isso nenhuma migration foi criada para a 13.4.

## Validacao real da Fase 13.4

A validacao real em producao da Fase 13.4 foi concluida pelo responsavel sem registrar dados pessoais reais, UUID administrativo real, `SUPABASE_SECRET_KEY`, token de signed upload ou URLs privadas completas.

Evidencias funcionais confirmadas:

- upload pela tela Admin;
- envio direto do browser ao Supabase Storage por signed upload URL;
- finalizacao server-side no Go;
- exibicao publica da imagem no catalogo;
- associacao da imagem ao produto;
- associacao opcional da imagem a Configuracao;
- substituicao de imagem sem overwrite;
- marcacao de imagem principal;
- ordenacao de imagens;
- remocao de associacao;
- limpeza fisica do objeto gerenciado no Storage quando aplicavel.

## Seguranca

- Nao ha signup administrativo pela aplicacao.
- O administrador inicial deve ser criado manualmente no Dashboard Supabase em Authentication -> Users.
- E-mail nao e autorizacao administrativa; o UUID do usuario e a fonte estavel.
- Senha, access token, refresh token, token de sessao e token hash nao devem aparecer em logs ou documentacao.
- Todas as respostas `/admin` usam `Cache-Control: private, no-store`, `X-Robots-Tag: noindex, nofollow, noarchive` e `Referrer-Policy: same-origin`.
- Alem dos headers privados do Admin, a aplicacao aplica headers globais de hardening da Fase 14.1, incluindo `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Permissions-Policy` e CSP restritiva.
- POSTs administrativos exigem `Origin` same-origin valido ou, quando `Origin` estiver ausente, `Referer` same-origin como fallback; `Origin: null`, cross-site e ausencia simultanea de `Origin` e `Referer` sao rejeitados.
- Dashboard e listagem de pedidos nao carregam nem renderizam CPF, endereco, telefone, e-mail de cliente, `transaction_nsu`, `invoice_slug` ou checkout URL.
- O detalhe de pedido pode renderizar PII operacional somente apos sessao administrativa valida.
- Eventos de auditoria guardam UUID do usuario Supabase Auth, tipo de evento, status anterior, status novo e horario; nao armazenam PII de cliente.
- Paginas de catalogo Admin nao fazem join com pedidos e nao carregam PII, dados InfinitePay, `order_id` interno ou URL de checkout.
- A 13.4 usa `SUPABASE_SECRET_KEY` somente no backend e apenas apos validar sessao Admin, origem e escopo da operacao.
- `SUPABASE_SECRET_KEY` nunca deve aparecer em HTML, JavaScript, logs, responses ou documentacao com valor real.
- Endpoints de imagem nao aceitam bucket/path arbitrario enviado pelo cliente; o backend gera o path e limita ao bucket `product-images`.
- Signed upload URL/token e entregue apenas ao Admin autenticado e nao e persistido.
- `admin_order_events` continua exclusivo para operacoes de pedidos; auditoria de catalogo/imagens pode ser avaliada futuramente em estrutura propria.

## Limites atuais

- Nao ha alteracao de dados comerciais do pedido, valores, cliente, endereco ou pagamento.
- Nao ha papeis multiplos.
- MFA TOTP e obrigatorio; nao ha CAPTCHA ou rate limiter em Go na aplicacao.
- Nao ha etiqueta, postagem, rastreio externo ou integracao logistica de despacho.
- Nao ha estoque fisico de filamento, marcas, lotes, carretel, custos calculados ou multiplos admins/papeis.
- Nao ha crop, compressao avancada, bulk upload, thumbnails persistidos multiplos ou DAM.
- RBAC permanece fora do escopo; protecoes WAF/anti-abuse ficam para a Fase 14.3.

## Decisoes pendentes

- Politica de permissoes caso existam multiplos usuarios administrativos.
- Protecoes adicionais contra abuso/brute force via Vercel Firewall/WAF apos observacao de trafego real.
