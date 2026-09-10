# Fase 8.1 — UX do Checkout, Consulta de CEP e Diagnostico Seguro de Frete

Status: CONCLUIDA.

## Objetivo

Melhorar a etapa de dados do checkout e a observabilidade segura da cotacao de frete, sem implementar pedido, pagamento, migrations ou novas tabelas.

## Escopo implementado

- Mascaras progressivas de CPF, telefone brasileiro e CEP em JavaScript vanilla.
- Asset `/static/js/checkout.js` embutido junto com os demais arquivos de `web/static/`.
- Endpoint interno `GET /api/cep/{cep}` para consulta server-side ao ViaCEP.
- Cliente ViaCEP com `net/http`, timeout explicito de aproximadamente 3 segundos e respeito ao contexto da request.
- Resposta interna de CEP limitada a `street`, `district`, `city` e `state`.
- Fallback manual preservado quando o CEP nao e encontrado ou a consulta externa falha.
- Diagnosticos seguros de frete por estagio e motivo.
- Erros do cliente SuperFrete com categoria/status seguros e sem corpo bruto externo.

## Fora de escopo

- Migration.
- Novas tabelas.
- Pedido.
- Pagamento.
- Checkout InfinitePay.
- Webhook.
- Criacao de caixa, produto, carrinho ou cotacao ficticia.
- Validacao Sandbox real da SuperFrete.

## Definition of Done

- [x] Checkout continua funcional sem JavaScript obrigatorio.
- [x] Mascaras de CPF, telefone e CEP sao apenas UX, com validacao autoritativa no backend.
- [x] Navegador nao chama ViaCEP diretamente.
- [x] Endpoint interno de CEP nao retorna IBGE, DDD, SIAFI, GIA ou regiao.
- [x] Falha de CEP nao bloqueia preenchimento manual.
- [x] Frete registra motivos seguros: `shipping_not_configured`, `no_active_boxes`, `planning_request_failed`, `planning_no_valid_quotes`, `planning_no_package`, `no_fitting_box`, `final_request_failed` e `final_no_valid_quotes`.
- [x] Logs nao registram CEP, CPF, telefone, e-mail, endereco, token, Authorization ou corpo bruto externo.
- [x] Testes de ViaCEP usam `httptest`, sem chamada externa real.
- [x] Na conclusao desta fase, a validacao Sandbox real da SuperFrete permanecia pendente.

## Validacoes

Executar antes de mover para `docs/plans/completed/`:

- `templ generate`
- `npm run css:build`
- `gofmt -w .`
- `go mod tidy`
- `go test ./...`
- `go vet ./...`
- `go build ./...`

## Resultado

Fase 8.1 concluida em 2026-09-09.

Validacoes finais:

- `templ generate`: passou.
- `npm run css:build`: passou.
- `gofmt -w .`: passou.
- `go mod tidy`: passou.
- `go test ./...`: passou.
- `go vet ./...`: passou.
- `go build ./...`: passou.

A validacao Sandbox real da SuperFrete foi concluida posteriormente antes da Fase 9, conforme registrado em `docs/plans/completed/008-shipping-superfrete.md`.
