# Roadmap

Status: Fases 0, 1, 2, 2.1, 3, 3.1 e 4 concluidas. Fases 5 a 17 planejadas.

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
| Fase 5 — Produtos e variantes | Planejada |
| Fase 6 — Carrinho | Planejada |
| Fase 7 — Dados do cliente e endereco | Planejada |
| Fase 8 — Integracao SuperFrete | Planejada |
| Fase 9 — Pedidos | Planejada |
| Fase 10 — Integracao InfinitePay | Planejada |
| Fase 11 — Webhooks de pagamento | Planejada |
| Fase 12 — Acompanhamento do pedido | Planejada |
| Fase 13 — Painel administrativo | Planejada |
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

Objetivo: modelar variantes como cor, material, tamanho ou atributos aprovados.

Principais entregas:

- Modelo de variantes.
- Regras de disponibilidade.
- Imagens por produto ou variante, se aprovado.
- Documentacao de regras de produto.

Dependencias: Fases 3 e 4.

Definition of Done:

- Variantes persistidas por schema aprovado.
- Regras testadas.
- Sem precos autoritativos no cliente.
- Documentacao atualizada.

## Fase 6 — Carrinho

Objetivo: permitir selecao de itens antes do checkout.

Principais entregas:

- Adicionar, alterar e remover itens.
- Persistencia de carrinho aprovada.
- Recalculo server-side.
- Testes de quantidades e erros.

Dependencias: Fases 4 e 5.

Definition of Done:

- Carrinho nao confia em preco vindo do cliente.
- Testes cobrem alteracao de itens.
- Estados de carrinho vazio e invalido documentados.
- Sem checkout implementado fora de escopo.

## Fase 7 — Dados do cliente e endereco

Objetivo: coletar dados necessarios para entrega e contato.

Principais entregas:

- Modelo de cliente.
- Modelo de endereco.
- Formularios server-side.
- Validacoes basicas.

Dependencias: Fases 3 e 6.

Definition of Done:

- Dados minimos aprovados.
- Validacoes server-side implementadas.
- Tratamento de erro documentado.
- Privacidade e seguranca revisadas.

## Fase 8 — Integracao SuperFrete

Objetivo: calcular opcoes de frete usando SuperFrete.

Principais entregas:

- Cliente HTTP server-side.
- Configuracao por environment variables.
- Validacao de CEP e parametros.
- Revalidacao no checkout.

Dependencias: Fases 6 e 7; confirmacao da documentacao oficial da SuperFrete.

Definition of Done:

- Contrato oficial usado.
- Timeouts e erros tratados.
- Testes de contrato ou mocks deterministico.
- Valor de frete validado server-side.

## Fase 9 — Pedidos

Objetivo: criar pedidos a partir de carrinho, cliente, endereco e frete validados.

Principais entregas:

- Modelo de pedido.
- Itens de pedido com valores congelados.
- Transacao de criacao.
- Status iniciais.

Dependencias: Fases 3, 6, 7 e 8.

Definition of Done:

- Pedido criado somente com dados recalculados no backend.
- Transacoes testadas.
- Status documentados.
- Falhas nao criam estado financeiro inconsistente.

## Fase 10 — Integracao InfinitePay

Objetivo: iniciar pagamentos com InfinitePay de forma server-side.

Principais entregas:

- Cliente HTTP server-side.
- Criacao de checkout ou fluxo equivalente.
- Relacao com order ID interno.
- Tratamento de erros.

Dependencias: Fase 9; confirmacao da documentacao oficial da InfinitePay.

Definition of Done:

- Contrato oficial usado.
- Credenciais fora do Git.
- Testes aplicaveis.
- Redirect nao e tratado como confirmacao de pagamento.

## Fase 11 — Webhooks de pagamento

Objetivo: confirmar pagamentos por eventos validados no backend.

Principais entregas:

- Endpoint de webhook.
- Validacao de autenticidade conforme contrato oficial.
- Idempotencia.
- Logs adequados.

Dependencias: Fases 9 e 10.

Definition of Done:

- Webhook duplicado nao duplica processamento.
- Eventos invalidos sao rejeitados.
- Status de pagamento muda apenas por evento confiavel.
- Testes de idempotencia passam.

## Fase 12 — Acompanhamento do pedido

Objetivo: permitir que cliente acompanhe status basico do pedido.

Principais entregas:

- Consulta segura de pedido.
- Tela de acompanhamento.
- Estados de pagamento, producao e envio aprovados.

Dependencias: Fases 9 e 11.

Definition of Done:

- Acesso a pedido e protegido por regra aprovada.
- Informacoes sensiveis nao sao expostas indevidamente.
- Status exibidos refletem fonte server-side.
- Documentacao atualizada.

## Fase 13 — Painel administrativo

Objetivo: apoiar operacao interna da PrintLab.

Principais entregas:

- Autenticacao e autorizacao administrativa.
- Gestao de produtos.
- Consulta de pedidos.
- Fluxos basicos de producao.

Dependencias: Fases 3, 4, 9 e decisao de seguranca administrativa.

Definition of Done:

- Acesso administrativo protegido.
- Permissoes documentadas.
- Operacoes criticas auditaveis quando necessario.
- Testes aplicaveis passam.

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
