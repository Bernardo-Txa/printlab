# Roadmap

Status: Fases 0, 1, 2, 2.1, 3, 3.1, 4, 5, 5.1, 6, 7, 7.1, 8, 8.1, 9, 9.1, 10, 10.1, 11, 12 e 13.1 concluidas. Fase 13 em andamento; Fases 13.2 a 17 planejadas.

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
| Fase 13 — Painel administrativo | Em andamento |
| Fase 13.1 — Autenticacao administrativa | Concluida |
| Fase 13.2 — Pedidos, producao, envio e auditoria | Planejada |
| Fase 13.3 — Catalogo, variantes, materiais, cores e caixas | Planejada |
| Fase 13.4 — Imagens e Supabase Storage | Planejada |
| Fase 14 — Seguranca | Planejada |
| Fase 15 — Testes e observabilidade | Planejada |
| Fase 16 — SEO e performance | Planejada |
| Fase 17 — Preparacao para producao | Planejada |

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
- 13.2: operacoes criticas auditaveis quando necessario: planejado
- 13.2 a 13.4: funcionalidades internas de operacao: planejado

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
- Headers privados/noindex/no-referrer.
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

Status: Planejada.

## Fase 13.3 — Catalogo, variantes, materiais, cores e caixas

Objetivo: permitir gestao interna do catalogo e dados operacionais basicos relacionados.

Status: Planejada.

## Fase 13.4 — Imagens e Supabase Storage

Objetivo: permitir upload e gestao segura de imagens de catalogo.

Status: Planejada.

## Fase 14 — Seguranca

Objetivo: revisar seguranca antes de ampliar uso real.

Principais entregas:

- Revisao de secrets.
- Revisao de autenticacao e autorizacao.
- Revisao de webhooks.
- Revisao de dependencias.
- Checklist de operacao segura.

Dependencias: fases com dados sensiveis ou financeiros implementadas.

Definition of Done:

- Nenhuma credencial no repositorio.
- Fluxos financeiros revisados.
- Permissoes revisadas.
- Pendencias criticas enderecadas ou bloqueadas explicitamente.

## Fase 15 — Testes e observabilidade

Objetivo: aumentar confiabilidade operacional.

Principais entregas:

- Cobertura dos fluxos criticos.
- Logs estruturados quando aprovados.
- Monitoramento de erros.
- Alertas para falhas de pagamento e webhook.

Dependencias: Fases 9, 10 e 11.

Definition of Done:

- Testes criticos documentados e executados.
- Logs nao expõem dados sensiveis.
- Falhas relevantes sao observaveis.
- Plano de resposta a incidentes inicial definido.

## Fase 16 — SEO e performance

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

## Fase 17 — Preparacao para producao

Objetivo: preparar operacao comercial.

Principais entregas:

- Revisao de hospedagem.
- Revisao de environment variables e secrets.
- Revisao de dominio.
- Revisao de backups.
- Revisao de webhooks.
- Revisao de observabilidade.
- Revisao de seguranca.

Dependencias: fases comerciais essenciais concluidas.

Definition of Done:

- Plano de Vercel revisado.
- Banco e migrations revisados.
- Backups avaliados.
- Pagamentos e webhooks verificados.
- Responsavel pelo projeto aprova entrada em producao.
