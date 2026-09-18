# Roadmap

Status: Fases 0 a 16 concluidas. Fase 17 em execucao. Fases 18 a 20 planejadas.

## Status das fases

| Fase | Status |
| --- | --- |
| Fase 0 — Fundacao e documentacao | Concluida |
| Fase 1 — Bootstrap da aplicacao Go | Concluida |
| Fase 2 — Design system e layout | Concluida |
| Fase 2.1 — Brand Experience | Concluida |
| Fase 3 — Banco de dados | Concluida |
| Fase 3.1 — Validacao do ambiente remoto de desenvolvimento | Concluida |
| Fase 4 — Catalogo | Concluida |
| Fase 5 — Produtos e variantes | Concluida |
| Fase 5.1 — Semantica da receita de producao | Concluida |
| Fase 6 — Carrinho | Concluida |
| Fase 7 — Dados do cliente e endereco | Concluida |
| Fase 7.1 — Hardening de privacidade e consistencia do checkout | Concluida |
| Fase 8 — Embalagem real e integracao SuperFrete | Concluida |
| Fase 8.1 — UX do checkout, consulta de CEP e diagnostico seguro de frete | Concluida |
| Fase 9 — Pedidos | Concluida |
| Fase 9.1 — Interface publica de pedidos | Concluida |
| Fase 10 — Integracao InfinitePay | Concluida |
| Fase 10.1 — Diagnostico Seguro InfinitePay | Concluida |
| Fase 11 — Webhooks de pagamento | Concluida |
| Fase 12 — Acompanhamento do pedido | Concluida |
| Fase 13 — Painel administrativo | Concluida |
| Fase 13.1 — Autenticacao administrativa | Concluida |
| Fase 13.2 — Pedidos, producao, envio e auditoria | Concluida |
| Fase 13.3 — Catalogo, variantes, materiais, cores e caixas | Concluida |
| Fase 13.4 — Imagens e Supabase Storage | Concluida |
| Fase 14 — Seguranca | Concluida |
| Fase 14.1 — Hardening base de seguranca | Concluida; validada em producao |
| Fase 14.2 — MFA obrigatorio para Admin | Concluida; validada em producao |
| Fase 14.3 — WAF / anti-abuse | Concluida; configurada e validada em producao |
| Fase 15 — Testes e observabilidade | Concluida |
| Fase 16 — SEO e performance | Concluida; validada em producao |
| Fase 16.1 — SEO tecnico e baseline de performance | Concluida; validada em producao |
| Fase 16.2 — Performance de imagens, cache estatico e acessibilidade | Concluida; validada em producao |
| Fase 17 — Simplificacao do catalogo e administrativo | Em execucao |
| Fase 17.1 — Cadastro e slugs | Concluida; validada em producao |
| Fase 17.2 — Logistica simplificada | Concluida; validada em producao |
| Fase 17.3 — Cores e producao | Planejada |
| Fase 18 — Conta do cliente e comunicacao transacional | Planejada |
| Fase 18.1 — E-mail transacional PrintLab | Planejada |
| Fase 18.2 — Autenticacao do cliente | Planejada |
| Fase 18.3 — Minha conta / Meus pedidos | Planejada |
| Fase 18.4 — Pagamentos pendentes e retomada | Planejada |
| Fase 19 — Experiencia e acabamento comercial | Planejada |
| Fase 19.1 — Revisao textual completa | Planejada |
| Fase 19.2 — Motion e experiencia premium | Planejada |
| Fase 19.3 — Regressao de UX/performance | Planejada |
| Fase 20 — Preparacao final para producao | Planejada |
| Fase 20.1 — Auditoria | Planejada |
| Fase 20.2 — Correcao de bloqueadores | Planejada |
| Fase 20.3 — Go-live | Planejada |

## Processo de planos

Planos em execucao devem ficar em `docs/plans/active/`.

Planos concluidos e verificados devem ser movidos para `docs/plans/completed/`.

Uma fase so deve ser considerada concluida quando sua Definition of Done for atendida e as validacoes aplicaveis forem executadas.

## Fase 0 — Fundacao e documentacao

Objetivo: estabelecer a estrutura inicial do projeto e a documentacao base.

Principais entregas:

- Estrutura de diretorios.
- Documentacao raiz.
- Documentacao por area.
- Definition of Done global.
- ADRs iniciais aprovados.

Dependencias: nenhuma.

Definition of Done:

- Estrutura criada.
- Documentos iniciais escritos em portugues brasileiro.
- Escopo planejado separado de escopo implementado.
- Nenhuma credencial adicionada.

## Fase 1 — Bootstrap da aplicacao Go

Objetivo: garantir que a aplicacao Go compila e inicia com infraestrutura HTTP minima.

Principais entregas:

- `go.mod`.
- Entrada `cmd/server/main.go`.
- Health check `GET /health`.
- Teste aplicavel do health check.

Dependencias: Fase 0.

Definition of Done:

- `go test ./...` passa.
- `go vet ./...` passa.
- `gofmt` aplicado.
- Nenhuma funcionalidade de negocio adicionada.

## Fase 2 — Design system e layout

Objetivo: criar a base visual server-side da aplicacao.

Principais entregas:

- Integracao inicial de `templ`.
- Pipeline de Tailwind CSS.
- Layout base.
- Componentes reutilizaveis essenciais.

Dependencias: Fase 1.

Definition of Done:

- Paginas base renderizam server-side.
- CSS e gerado por processo documentado.
- Acessibilidade basica revisada.
- Sem regra de negocio duplicada no frontend.

## Fase 3 — Banco de dados

Objetivo: preparar a fundacao PostgreSQL/Supabase sem criar tabelas de negocio.

Principais entregas:

- Escolha da ferramenta de migrations.
- Estrutura `supabase/` com `config.toml` e `supabase/migrations/`.
- Configuracao de conexao via environment variables.
- Acesso inicial via `pgx/v5` e `pgxpool`.
- Workflow de CI/CD para migrations Supabase com dry-run antes da aplicacao.
- `GET /ready` para readiness de banco.
- Vercel configurada para `gru1`.

Dependencias: Fases 0 e 1.

Definition of Done:

- Fundacao de conexao documentada.
- Migrations sem duas fontes de verdade.
- Testes aplicaveis de config, database e HTTP executados.
- Nenhuma alteracao manual sem registro.
- Nenhuma tabela de negocio criada.

## Fase 3.1 — Validacao do ambiente remoto de desenvolvimento

Objetivo: validar as conexoes operacionais entre GitHub Actions, Supabase DEV, Vercel e runtime Go sem criar schema de negocio.

Principais entregas:

- Preflight de Git, GitHub CLI, Supabase CLI, Vercel CLI e `vercel.json`.
- Verificacao por nome dos GitHub Secrets exigidos para Supabase.
- Execucao segura do workflow `Supabase Migrations` sem migration ficticia.
- Validacao da URL publica da Vercel em `/`, `/health`, `/ready` e `/static/css/app.css`.
- Registro explicito dos resultados sem expor secrets.

Dependencias: Fase 3.

Definition of Done:

- GitHub Actions -> Supabase DEV validado.
- Vercel autenticada e projeto PrintLab confirmado.
- `DATABASE_URL` configurada com o Supabase Transaction Pooler sem versionar valor.
- `DB_MAX_CONNS=4` configurado nos ambientes Vercel relevantes.
- `/ready` remoto retorna HTTP 200.
- Nenhuma migration de teste ou tabela ficticia criada.

## Fase 4 — Catalogo

Objetivo: permitir exibicao publica de produtos publicados.

Principais entregas:

- Migration `create_catalog` com `categories` e `products`.
- Listagem publica em `GET /produtos`.
- Filtro server-side por categoria.
- Pagina publica em `GET /produtos/{slug}`.
- Repository PostgreSQL com `pgxpool`.
- Service de catalogo com validacao de slug e formatacao BRL.
- Estados vazios e erros genericos.

Dependencias: Fases 2 e 3.

Definition of Done:

- Produtos ativos exibidos a partir do banco.
- Produto inativo tratado como inexistente.
- Precos renderizados a partir de dados server-side em centavos.
- Testes de handlers, service, slug e dinheiro executados.
- Migration aplicada ao Supabase DEV pelo workflow.
- Documentacao de catalogo e schema atualizada.
- Nenhum produto ficticio criado.
- Fase 5 permanece planejada.

## Fase 5 — Produtos e variantes

Objetivo: modelar variantes, imagens e receita estimada de producao 3D.

Principais entregas:

- Migration `create_product_variants`.
- Materiais logicos e cores logicas.
- Variantes com preco opcional, default e tempo estimado de maquina.
- Receita estimada por `variant_filaments` para multicolor e multimaterial.
- Imagens gerais de produto e especificas de variante.
- Bucket publico `product-images` para imagens de catalogo.
- Selecao SSR de variante por query string.
- Regras de preco efetivo no backend.

Dependencias: Fases 3 e 4.

Definition of Done:

- Variantes persistidas por schema aprovado.
- Receita de producao persistida por peso em miligramas e tempo em minutos.
- Imagens associadas por caminho relativo no Storage.
- Regras de preco, variante default, peso, tempo e imagem testadas.
- Sem precos autoritativos no cliente.
- Sem estoque, carrinho, checkout, upload ou admin.
- Documentacao atualizada.

## Fase 5.1 — Semantica da receita de producao

Objetivo: corrigir a leitura de receitas para preservar componentes que referenciem material ou cor inativos.

Principais entregas:

- Query de `variant_filaments` sem filtro por `materials.is_active` ou `colors.is_active`.
- Preservacao de nomes de material/cor em receitas existentes.
- Peso total estimado somando todos os componentes carregados.
- Documentacao da semantica final de `materials.is_active` e `colors.is_active`.

Dependencias: Fase 5.

Definition of Done:

- Receitas existentes nao perdem componentes por material ou cor inativos.
- Produto e variante ativos continuam controlando visibilidade publica.
- Sem migration ou alteracao de schema.
- Testes de regressao executados.
- Fase 6 permanece planejada.

## Fase 6 — Carrinho

Objetivo: permitir selecao de itens antes do checkout.

Principais entregas:

- Carrinho anonimo persistido no PostgreSQL.
- Token opaco em cookie `printlab_cart` e `SHA-256` no banco.
- `GET /carrinho` com carrinho vazio, linhas, indisponibilidade e subtotal.
- `POST /carrinho/adicionar` com add/increment e redirect 303.
- `POST /carrinho/itens/{id}/quantidade` com validacao `1..99`.
- `POST /carrinho/itens/{id}/remover` idempotente na experiencia publica.
- Recalculo server-side de preco e subtotal.
- Itens indisponiveis preservados sem entrar no subtotal.
- Protecao cross-site por SameSite=Lax e validacao Origin/Referer.

Dependencias: Fases 4, 5 e 5.1.

Definition of Done:

- Carrinho nao confia em preco vindo do cliente.
- Testes cobrem token, cookie, service, disponibilidade, dinheiro, handlers e escopo por carrinho.
- Estados de carrinho vazio, invalido e indisponivel documentados.
- Migration `create_carts` criada.
- RLS habilitado sem policies publicas.
- Sem checkout implementado dentro da Fase 6.
- Fase 7 permaneceu planejada ao final da Fase 6.

## Fase 7 — Dados do cliente e endereco

Objetivo: coletar dados necessarios para entrega e contato.

Principais entregas:

- Dados temporarios de contato vinculados ao carrinho.
- Endereco de entrega vinculado ao carrinho.
- Formularios server-side em `GET /checkout/dados`.
- Persistencia transacional em `POST /checkout/dados`.
- Validacoes brasileiras de CPF, telefone, CEP, UF e pais.
- Minimizacao de PII, sem conta obrigatoria e sem cliente permanente.

Dependencias: Fases 3 e 6.

Definition of Done:

- Dados minimos aprovados.
- Validacoes server-side implementadas.
- Tratamento de erro documentado.
- Privacidade e seguranca revisadas.
- Migration `create_cart_customer_details` criada.
- RLS habilitado sem policies publicas.
- Fase 8 foi iniciada posteriormente.

## Fase 7.1 — Hardening de privacidade e consistencia do checkout

Objetivo: reforcar privacidade e consistencia operacional da etapa de dados do checkout.

Principais entregas:

- `Cache-Control: private, no-store` em respostas HTML de checkout que podem conter PII.
- Leitura de contato e endereco consolidada em uma unica consulta SQL consistente.
- Estado parcial anomalo tratado como dados ausentes, sem retornar PII incompleta.
- Testes de regressao para headers, query consolidada e estado parcial.

Dependencias: Fase 7.

Definition of Done:

- Sem migration ou alteracao de schema.
- Sem mudanca na escrita transacional existente.
- Testes, vet e build executados.
- Documentacao de checkout, seguranca e schema atualizada.
- Fase 8 foi iniciada posteriormente.

## Fase 8 — Embalagem real e integracao SuperFrete

Objetivo: calcular opcoes de frete usando perfis logisticos, caixas fisicas reais e SuperFrete.

Principais entregas:

- Perfis logisticos opcionais em produtos e variantes.
- Tabela `shipping_boxes` para caixas fisicas reais, sem seed ficticio.
- Tabela `cart_shipping_selections` para selecao de frete por carrinho.
- Cliente HTTP server-side da SuperFrete.
- Configuracao por environment variables.
- Estrategia de duas chamadas: `products` para pacote ideal e `package` com caixa real para preco final.
- Escolha da menor caixa real compativel por dimensoes internas e rotacao.
- Rotas `GET /checkout/frete` e `POST /checkout/frete`.
- Revalidacao server-side no POST.
- `input_hash` e validade de 30 minutos para selecao de frete.

Dependencias: Fases 6 e 7; confirmacao da documentacao oficial da SuperFrete.

Definition of Done:

- Contrato oficial usado.
- Timeouts e erros tratados.
- Testes de contrato ou mocks deterministico.
- Valor de frete validado server-side.
- Migration criada sem seed ficticio.
- Sandbox real validado com token, CEP de origem, produto real com perfil logistico e caixa real cadastrada.

## Fase 8.1 — UX do checkout, consulta de CEP e diagnostico seguro de frete

Objetivo: melhorar a experiencia da etapa de dados e a observabilidade segura do frete sem avancar para pedidos ou pagamentos.

Principais entregas:

- Mascaras progressivas de CPF, telefone brasileiro e CEP.
- Endpoint interno de consulta de CEP com chamada server-side ao ViaCEP.
- Fallback manual de endereco preservado.
- Diagnosticos seguros de frete por estagio e motivo.
- Categorias seguras para erros do cliente SuperFrete.

Dependencias: Fases 7, 7.1 e implementacao local da Fase 8.

Definition of Done:

- Checkout continua funcionando sem JavaScript obrigatorio.
- Endpoint de CEP retorna apenas rua, bairro, cidade e UF.
- Testes de ViaCEP usam `httptest`, sem chamada real em `go test`.
- Logs de CEP e frete nao registram CEP, CPF, telefone, e-mail, endereco, token ou corpo bruto externo.
- Fase 8 foi concluida posteriormente com validacao Sandbox SuperFrete confirmada manualmente.

## Fase 9 — Pedidos

Objetivo: criar pedidos a partir de carrinho, cliente, endereco e frete validados.

Principais entregas:

- Modelo de pedido com `orders` e `order_number` sequencial.
- Itens de pedido com valores congelados.
- Snapshots de cliente, endereco, frete e receita de producao 3D.
- Revisao SSR em `GET /checkout/revisao`.
- Criacao transacional em `POST /checkout/revisao`.
- Exibicao de pedido por UUID em `GET /pedido/{id}`.
- Conversao de carrinho por `carts.converted_at`.
- Idempotencia por `source_cart_id`.
- Status inicial `pending_payment`.

Dependencias: Fases 3, 6, 7 e 8.

Definition of Done:

- Pedido criado somente com dados recalculados no backend.
- Transacoes testadas.
- Status documentados.
- Falhas nao criam estado financeiro inconsistente.
- Pagamento permanece planejado para a Fase 10.

## Fase 9.1 — Interface publica de pedidos

Objetivo: separar dados operacionais preservados nos snapshots da experiencia publica do comprador.

Principais entregas:

- Remocao de SKU interno da revisao e pagina de pedido.
- Remocao de tempo de impressao, consumo de filamento e receita operacional da UI publica.
- Remocao de caixa fisica, peso e dimensoes do pacote da pagina publica de pedido.
- Preservacao dos campos operacionais no dominio, repository e banco.

Dependencias: Fase 9.

Definition of Done:

- Revisao e pedido exibem somente informacoes comercialmente relevantes ao comprador.
- Snapshots operacionais seguem preservados para painel administrativo e operacao futura.
- Nenhuma migration e criada.
- Pagamento permanece planejado para a Fase 10.

## Fase 10 — Integracao InfinitePay

Objetivo: iniciar pagamentos com InfinitePay de forma server-side.

Principais entregas:

- Cliente HTTP server-side com `net/http`, timeout e base URL interna fixa.
- Criacao/reuso de checkout hospedado InfinitePay por `POST /pedido/{id}/pagar`.
- `order_payments` 1:1 com pedido.
- Retorno `GET /pagamento/retorno` com `payment_check` server-side.
- Atualizacao transacional de pedido para `paid` somente com valor confirmado.
- UI SSR da etapa 4 de pagamento.

Dependencias: Fase 9; confirmacao da documentacao oficial da InfinitePay.

Definition of Done:

- Contrato oficial usado.
- Credenciais fora do Git.
- Testes aplicaveis.
- Redirect nao e tratado como confirmacao de pagamento.
- Link real InfinitePay: validado em producao.
- Pagamento real confirmado: validado em producao.

## Fase 11 — Webhooks de pagamento

Objetivo: confirmar pagamentos por eventos recebidos no backend sem confiar no webhook como autoridade direta.

Principais entregas:

- Endpoint `POST /webhooks/infinitepay`.
- `webhook_url` enviado em `POST /links`.
- Confirmacao redundante via `payment_check` server-side.
- Idempotencia para webhook duplicado e corrida com retorno do navegador.
- Logs seguros sem PII, checkout URL completa ou NSU de transacao.

Dependencias: Fases 9 e 10.

Definition of Done:

- A. Implementacao e testes automatizados: concluida ✅
- B. Checkout real contendo `webhook_url`: concluida ✅
- C. Webhook real recebido em producao: concluida ✅
- D. Pagamento confirmado sem redirect do comprador: concluida ✅

## Fase 12 — Acompanhamento do pedido

Objetivo: permitir que cliente acompanhe status basico do pedido.

Principais entregas:

- `orders.public_tracking_id` UUID aleatorio, unico e obrigatorio.
- `order_fulfillment` 1:1 com status de producao e envio.
- Consulta segura por `/acompanhar/{public_tracking_id}`.
- Tela SSR de acompanhamento minimizada, sem PII, valores ou IDs internos.
- Headers privados, noindex e no-referrer.
- Link `Acompanhar pedido` em `/pedido/{id}` usando `public_tracking_id`.

Dependencias: Fases 9 e 11.

Definition of Done:

- Acesso a pedido e protegido por regra aprovada: concluido ✅
- Informacoes sensiveis nao sao expostas indevidamente: concluido ✅
- Status exibidos refletem fonte server-side: concluido ✅
- Documentacao atualizada: concluido ✅

## Fase 13 — Painel administrativo

Objetivo: apoiar operacao interna da PrintLab.

Principais entregas:

- 13.1: autenticacao, autorizacao, sessao e shell administrativo.
- 13.2: pedidos, producao, envio e auditoria.
- 13.3: catalogo, variantes, materiais, cores e caixas.
- 13.4: imagens e Supabase Storage.

Dependencias: Fases 3, 4, 9 e decisao de seguranca administrativa.

Definition of Done:

- 13.1: acesso administrativo protegido: concluido ✅
- 13.1: permissoes documentadas para unico administrador por UUID: concluido ✅
- 13.1: testes aplicaveis passam: concluido ✅
- 13.2: operacoes criticas auditaveis quando necessario: concluido ✅
- 13.3: catalogo administrativo sem hard delete: concluido ✅
- 13.4: imagens e Supabase Storage: concluido ✅

## Fase 13.1 — Autenticacao administrativa

Objetivo: criar a fundacao segura de acesso ao Admin sem CRUD ou mutacoes operacionais.

Principais entregas:

- Login por Supabase Auth usando e-mail/senha.
- Autorizacao por `ADMIN_SUPABASE_USER_ID`.
- Sessao propria da PrintLab com token opaco e hash SHA-256 em `admin_sessions`.
- Cookie `printlab_admin_session` HttpOnly, SameSite Strict, Path `/admin`, host-only e Secure em producao.
- Logout administrativo.
- Guard para `/admin` e `/admin/*`.
- Dashboard inicial somente leitura com contagens agregadas.
- Headers privados/noindex e `Referrer-Policy: same-origin`.
- Protecao `Origin`/`Referer` em POSTs administrativos.
- Documentacao e ADR.

Dependencias: Fases 3, 9, 11, 12 e decisao de seguranca administrativa.

Definition of Done:

- Autenticacao e autorizacao por UUID implementadas: concluido ✅
- Sessoes administrativas seguras implementadas: concluido ✅
- Config Admin opcional sem quebrar loja publica: concluido ✅
- Testes e documentacao atualizados: concluido ✅

## Fase 13.2 — Pedidos, producao, envio e auditoria

Objetivo: permitir operacao interna de pedidos pagos, status de producao/envio e trilha de auditoria.

Status: Concluida.

Principais entregas:

- Listagem administrativa autenticada de pedidos em `/admin/pedidos`.
- Detalhe administrativo autenticado em `/admin/pedidos/{order_id}`.
- Mutacoes POST para producao e envio com validacao de origem.
- Transicoes sequenciais de producao e envio sem regressao.
- Auditoria em `public.admin_order_events` na mesma transacao da atualizacao operacional.
- Documentacao de produto, seguranca, schema, migrations e ADR.

Definition of Done:

- Admin carrega lista sem PII nem identificadores tecnicos de pagamento: concluido ✅
- Detalhe autenticado mostra dados operacionais necessarios: concluido ✅
- Producao/envio respeitam regras de transicao e pagamento: concluido ✅
- Mutacoes criam auditoria transacional: concluido ✅
- Testes, template, CSS e documentacao atualizados: concluido ✅
- Validacao real em producao concluida sem registrar dados pessoais reais: concluido ✅

## Fase 13.3 — Catalogo, variantes, materiais, cores e caixas

Objetivo: permitir gestao interna do catalogo e dados operacionais basicos relacionados.

Status: Concluida.

Principais entregas:

- Listagem, criacao e edicao de produtos em `/admin/produtos`.
- Listagem, criacao e edicao de categorias em `/admin/categorias`.
- Criacao e edicao de variantes por produto.
- Edicao da receita atual em `variant_filaments` com peso em gramas na UI e miligramas no banco.
- Listagem, criacao e edicao de materiais, cores e caixas.
- Ativacao/inativacao por `is_active`, sem hard delete das entidades principais.
- Validacao de preco BRL sem `float`, slugs canonicos, perfis logisticos atomicos, default variant ativa e caixas com dimensoes externas maiores ou iguais as internas.
- Segurança Admin preservada: sessao obrigatoria, POST para mutacoes, validacao `Origin`/`Referer`, rejeicao de `Origin: null`, headers privados/noindex e body limitado.
- Nenhuma migration nova; uso das tabelas existentes.
- Refinamento 13.3A: `product_variants` permanece como modelo interno, mas a UI Admin usa "Configuracoes do produto" e a loja publica so mostra escolha quando houver duas ou mais configuracoes ativas.

Definition of Done:

- Rotas e templates Admin implementados: concluido ✅
- Service e repository Admin implementados sobre schema existente: concluido ✅
- Testes aplicaveis de parser, validacao, handlers e regressao de repository: concluido ✅
- Documentacao, plano ativo e ADR atualizados: concluido ✅
- 13.3A — Semantica publica/Admin de configuracoes refinada sem migration: concluido ✅
- Validacao real em producao pelo responsavel: concluido ✅

## Fase 13.4 — Imagens e Supabase Storage

Objetivo: permitir upload e gestao segura de imagens de catalogo.

Status: Concluida e validada em producao.

Principais entregas:

- Subpagina de imagens em `/admin/produtos/{product_id}/imagens`.
- Upload direto navegador -> Supabase Storage com signed upload URL/token.
- Autorizacao e finalizacao pelo Go Admin antes/depois do upload.
- Uso server-side de `SUPABASE_SECRET_KEY`, somente apos sessao Admin e validacao `Origin`/`Referer`.
- Validacao de produto, configuracao, MIME e tamanho antes da autorizacao.
- Confirmacao de objeto no Storage por `GET /storage/v1/object/info/{bucket}/{path}` e metadata JSON antes de inserir/atualizar `public.product_images`.
- Remocao, substituicao, ordenacao e imagem principal.
- Preservacao de imagens legadas/manuais sem DELETE arbitrario.
- Nenhuma migration nova; uso de `product_images.storage_path`, `sort_order` e `is_primary` existentes.

Definition of Done:

- Schema `product_images` e bucket `product-images` inspecionados: concluido ✅
- Provider Supabase Storage testavel sem chamadas reais em testes: concluido ✅
- Rotas Admin e UI SSR implementadas com JavaScript nativo restrito ao upload: concluido ✅
- `SUPABASE_SECRET_KEY` documentada e validada pelo prefixo `sb_secret_`: concluido ✅
- Testes de config, Storage, service, handlers e assets estaticos: concluido ✅
- Documentacao, plano ativo e ADR atualizados: concluido ✅
- Validacao real em producao concluida: concluido ✅

## Fase 14 — Seguranca

Objetivo: revisar seguranca antes de ampliar uso real.

Status: Em andamento.

Principais entregas:

- Revisao de secrets e configuracoes sensiveis.
- Revisao de autenticacao, autorizacao e sessoes.
- Revisao de webhooks e endpoints publicos sensiveis.
- Headers globais de seguranca e CSP.
- Timeouts HTTP conservadores.
- Limites globais e especificos de corpo de request.
- Limpeza automatica de dados transientes expirados.
- Checklist de operacao segura, rate limiting e WAF.

Dependencias: fases com dados sensiveis ou financeiros implementadas.

Definition of Done:

- Nenhuma credencial no repositorio.
- Fluxos financeiros revisados.
- Permissoes revisadas.
- Pendencias criticas enderecadas ou bloqueadas explicitamente.

## Fase 14.1 — Hardening base de seguranca

Objetivo: aplicar controles basicos de seguranca sem alterar fluxos de negocio, Admin, Storage ou pagamento ja validados.

Status: Concluida; validada em producao pelo responsavel.

Principais entregas:

- Headers globais: `nosniff`, `DENY` para frame, `Permissions-Policy`, `Referrer-Policy` global e CSP restritiva.
- Preservacao de `Referrer-Policy`, `Cache-Control` e `X-Robots-Tag` especificos de Admin e acompanhamento publico.
- CSP sem `unsafe-eval`; origem Supabase adicionada somente quando derivavel de `SUPABASE_URL`; `form-action` limitado a `'self'` e origins exatos de checkout InfinitePay aceitos pelo backend.
- Teto global de 1 MiB para corpo de requests, sem remover limites menores ja existentes por rota.
- `http.Server` com `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout` e `IdleTimeout` explicitos.
- Validacao central de `SITE_URL`, com HTTPS obrigatorio em producao e HTTP local preservado em desenvolvimento.
- Canonicalizacao de `GET` e `HEAD` para a origem de `SITE_URL`, preservando path/query e mantendo `POST` sem redirect automatico.
- Comparacao de origem configurada por `SITE_URL` usando `scheme://host`.
- Migration Supabase Cron diaria para limpar `admin_sessions` e `carts` expirados.
- Auditoria documentada de rate limiting/WAF, Supabase Auth, MFA futura, HSTS futuro, webhook, tracking, Storage, logs, RLS e dependencias.

Definition of Done:

- Codigo de hardening implementado: concluido ✅
- Testes automatizados de headers, CSP, body limit, `SITE_URL` e migration: concluido ✅
- Documentacao atualizada: concluido ✅
- Validacoes locais executadas antes do commit: concluido ✅
- Validacao real apos deploy: concluida pelo responsavel

## Fase 14.2 - MFA obrigatorio para Admin

Plano concluido: [014-2-admin-mfa.md](completed/014-2-admin-mfa.md). TOTP obrigatorio, estado AAL1 temporario, sessao propria somente apos AAL2 e invalidacao de sessoes legadas por timestamp. Fluxo validado em producao pelo responsavel: senha -> Supabase AAL1 -> TOTP -> AAL2 -> sessao propria PrintLab.

## Fase 14.3 - WAF / anti-abuse

Plano concluido: [014-3-waf-anti-abuse.md](completed/014-3-waf-anti-abuse.md). A protecao esta configurada no edge da Vercel e validada em producao.

## Fase 15 — Testes e observabilidade

Plano concluido: [015-1-tests-observability.md](completed/015-1-tests-observability.md).

Objetivo: aumentar confiabilidade operacional.

Principais entregas:

- Cobertura dos fluxos criticos.
- Eventos operacionais pesquisaveis e logs sem dados sensiveis.
- Monitoramento de erros pelos Runtime Logs Vercel.
- Estrategia manual de alertas compativel com Hobby e criterio de escalonamento futuro.

Dependencias: Fases 9, 10 e 11.

Definition of Done:

- Testes criticos documentados e executados.
- Logs nao expõem dados sensiveis.
- Falhas relevantes sao observaveis.
- Plano de resposta a incidentes inicial definido.

## Fase 16 — SEO e performance

Planos concluidos: [016-1-seo-performance.md](completed/016-1-seo-performance.md) e [016-2-image-performance.md](completed/016-2-image-performance.md).

Objetivo: preparar a loja para descoberta e boa experiencia de navegacao.

Principais entregas:

- Metadados basicos.
- URLs apropriadas.
- Performance de paginas publicas.
- Otimizacao de imagens quando houver catalogo.

Dependencias: Fases 2, 4 e 5.

Definition of Done:

- Paginas publicas tem metadados adequados.
- HTML server-side funciona sem JavaScript obrigatorio.
- Performance medida.
- Imagens otimizadas conforme necessidade real.

## Fase 17 — Simplificacao do catalogo e administrativo

Objetivo: reduzir atrito no cadastro e separar com clareza configuracao comercial, producao e logistica. As Fases 17.1 e 17.2 foram concluidas e validadas em producao; a Fase 17.3 permanece planejada.

### Fase 17.1 — Cadastro e slugs

Status: Concluida; validada manualmente em producao.

- Gerar slug automaticamente a partir do nome para produtos, categorias, materiais, cores, caixas e configuracoes.
- Manter tratamento seguro e deterministico de colisao, com sufixo automatico e constraints existentes preservadas.
- Preservar slugs existentes em renomeacoes, sem recalculo em lote ou migration.
- Remover slug editavel do fluxo comum do Admin; o valor existente pode ser mostrado somente como informacao.

### Fase 17.2 — Logistica simplificada

Status: Concluida; validada em producao.

- Adotar perfil logistico unico por produto; configuracoes nao terao override logistico.
- Exigir logistica completa para produto ativo/comercial e permitir ausencia somente em rascunho ou inativo, sem inventar valores.
- Simplificar o formulario de caixas para altura, largura, comprimento, peso da embalagem, ativo e ordem.
- Preservar a semantica atual: medidas internas para encaixe e externas para transportadora. A implementacao preferida inicialmente escreve o mesmo conjunto de medidas nas colunas interna e externa atuais; eventual remocao de colunas exige justificativa posterior.
- Manter as colunas atuais sem migration; campos logisticos de configuracao permanecem como legado inerte.

### Fase 17.3 — Cores e producao

- Preservar `materials` e `colors` como dados de producao usados por receita, `variant_filaments`, peso, snapshots e operacao.
- Modelar futuramente cor comercial escolhida pelo cliente como conceito distinto de cor de filamento/receita.
- Fazer a escolha comercial fluir de produto para carrinho, pedido e producao sem criar variantes artificiais apenas para cor.

Dependencias: Fase 16.

## Fase 18 — Conta do cliente e comunicacao transacional

Objetivo: oferecer conta opcional e comunicacao transacional sem remover o checkout convidado. Tudo nesta fase permanece planejado.

### Fase 18.1 — E-mail transacional PrintLab

- Usar inicialmente `acesso@printlab3d.com.br` como remetente PrintLab para autenticacao e conta, sem marketing.
- Planejar iCloud+ Custom Email Domain -> SMTP iCloud (`smtp.mail.me.com`, porta 587) -> Supabase Auth Custom SMTP -> cliente.
- Usar senha especifica de app somente como secret no provider apropriado; nunca em codigo, documentacao, GitHub, frontend ou logs.
- Validar DNS, SPF, DKIM, DMARC, From/Reply-To e entrega em Gmail, Outlook e iCloud antes de uso real.
- Personalizar somente templates Supabase Auth habilitados, em pt-BR, responsivos e sem conteudo promocional.
- Reavaliar SMTP se volume, entregabilidade, limites ou necessidade de analytics/retries justificarem provedor transacional dedicado.

### Fase 18.2 — Autenticacao do cliente

- Usar Supabase Auth passwordless/Magic Link como preferencia arquitetural.
- Manter o fluxo de cliente separado do Admin: sem acesso a `/admin`, sem herdar autorizacao administrativa e sem inferir papel por autenticacao.

### Fase 18.3 — Minha conta / Meus pedidos

- Permitir que cliente autenticado consulte somente pedidos associados server-side a sua identidade.
- Manter checkout convidado e acompanhamento publico seguro.
- Tratar eventual reivindicacao de pedido antigo de convidado como funcionalidade futura, somente apos prova de posse do e-mail.

### Fase 18.4 — Pagamentos pendentes e retomada

- Preservar `pending_payment` internamente: o pedido anterior ao checkout InfinitePay congela snapshots e suporta `payment_check` e webhook idempotente.
- Oferecer retomada somente para pedido ainda valido, com revalidacao server-side e sem pagamento ja confirmado.
- Definir politica para pendencias abandonadas sem apagar automaticamente evidencias financeiras ou pedidos.

Dependencias: Fase 17.

## Fase 19 — Experiencia e acabamento comercial

Objetivo: dar acabamento profissional ao site apos estabilizar produto e conta. Tudo nesta fase permanece planejado.

### Fase 19.1 — Revisao textual completa

- Auditar textos visiveis, mensagens, estados, acessibilidade, SEO e e-mails para pt-BR profissional.
- Corrigir somente texto apresentado; nao alterar slugs, identificadores, colunas, constantes, eventos ou APIs por acentuacao.

### Fase 19.2 — Motion e experiencia premium

- Adicionar interacoes proprias da PrintLab que sejam sutis, mobile-first, acessiveis, compatíveis com SSR e respeitem `prefers-reduced-motion`.
- Nao transformar o site em SPA nem bloquear checkout.

### Fase 19.3 — Regressao de UX/performance

- Revalidar mobile, desktop, teclado, foco, contraste, reduced motion, Lighthouse, CLS, LCP, TBT, SEO, checkout, Admin e conta do cliente.
- Preservar os ganhos da Fase 16.

Dependencias: Fase 18.

## Fase 20 — Preparacao final para producao

Objetivo: auditar e liberar a operacao comercial somente quando as Fases 17 a 19 estiverem congeladas.

### Fase 20.1 — Auditoria

- Auditar Vercel, dominio/DNS/HTTPS, environments e secrets, Supabase, migrations, backups/restore, Storage, Auth, SMTP, SuperFrete, InfinitePay, webhook, WAF, seguranca, observabilidade, PII/retenção, produtos e operacao.

### Fase 20.2 — Correcao de bloqueadores

- Corrigir somente problemas reais da auditoria, classificados como BLOCKER, WARNING ou MANUAL CHECK.
- Nao introduzir melhoria cosmetica nova.

### Fase 20.3 — Go-live

- Validar catalogo real, precos, imagens, configuracoes, cores, receita, logistica, pagamentos, e-mail, conta, Admin, backup e monitoramento.
- Exigir aprovacao explicita do responsavel antes de declarar a operacao comercial pronta.

Dependencias: Fases 17, 18 e 19.
