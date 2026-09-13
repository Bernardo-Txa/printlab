# Fase 13.2 — Pedidos, producao, envio e auditoria

Status: implementacao concluida; validacao real pendente.

## Objetivo

Permitir que a PrintLab opere pedidos autenticada pelo Admin, com consulta de dados completos apenas no detalhe, avancos controlados de producao/envio e trilha de auditoria transacional.

## Escopo implementado

- Listagem administrativa em `GET /admin/pedidos`.
- Filtros por etapa operacional.
- Detalhe administrativo em `GET /admin/pedidos/{order_id}`.
- Mutacao de producao em `POST /admin/pedidos/{order_id}/producao`.
- Mutacao de envio em `POST /admin/pedidos/{order_id}/envio`.
- Tabela `public.admin_order_events`.
- Auditoria com ator derivado de `Session.AuthUserID`.
- Documentacao e ADR.

## Fora do escopo

- Fase 13.3.
- CRUD de catalogo.
- Alteracao de valores, cliente, endereco ou pagamento.
- Upload de imagens.
- Etiqueta, postagem ou rastreio externo.
- Papeis multiplos.

## Definition of Done

- Lista admin protegida por sessao: concluido.
- Lista sem PII e sem identificadores tecnicos de pagamento: concluido.
- Detalhe admin com dados operacionais completos apos sessao valida: concluido.
- Transicoes de producao/envio sequenciais e sem regressao: concluido.
- Pedido pendente nao inicia producao: concluido.
- Envio exige producao concluida: concluido.
- `delivered` terminal: concluido.
- Auditoria criada na mesma transacao da mutacao: concluido.
- RLS habilitado na nova tabela sem policies publicas: concluido.
- Testes, templates, CSS e docs atualizados: concluido.
- Validacao real com Supabase/Vercel: pendente.

## Validacoes locais

Executar antes de concluir o ciclo:

- `templ generate`
- `npm run css:build`
- `gofmt -w .`
- `go mod tidy`
- `go test ./...`
- `go vet ./...`
- `go build ./...`
- `npx supabase --version`
- `npx supabase db reset` quando o ambiente local estiver disponivel.
