# Fase 10.1 — Diagnostico Seguro da Integracao InfinitePay

Status: CONCLUIDA.

## Objetivo

Adicionar observabilidade segura ao checkout InfinitePay antes de decidir qualquer mudanca de payload, handle, endpoint, URL de checkout, timeout ou fluxo de retorno.

## Entregas

- Erro estruturado interno para provider `infinitepay`.
- Operacoes classificadas como `create_checkout` e `payment_check`.
- Status HTTP preservado para respostas nao 2xx.
- Categorias seguras para erro de rede, timeout, status HTTP, JSON invalido, checkout URL invalida e casos desconhecidos.
- Correcao pontual posterior: validacao real em 2026 confirmou checkout URL com host `checkout.infinitepay.io`; a allowlist explicita do dominio Go e da constraint PostgreSQL passou a aceitar esse host e `checkout.infinitepay.com.br`.
- Leitura limitada e sanitizada de body de erro estruturado, sem logar body bruto.
- Logs seguros nos handlers de inicio de checkout e retorno de pagamento.
- Diferenciacao entre erro HTTP da API e resposta 2xx com checkout URL invalida.
- Testes de cliente e handler cobrindo status, categorias e ausencia de PII/URL completa/NSU nos diagnosticos.
- Documentacao da discrepancia entre endpoints InfinitePay como ponto a validar.

## Categorias

- `network_error`;
- `timeout`;
- `http_400`;
- `http_401`;
- `http_403`;
- `http_404`;
- `http_409`;
- `http_422`;
- `http_429`;
- `http_5xx`;
- `invalid_json`;
- `invalid_checkout_url`;
- `unknown`.

## Fora do escopo respeitado

- Nenhuma alteracao de schema.
- Nenhuma migration.
- Nenhum webhook.
- Nenhuma alteracao de status de pedido ou pagamento.
- Nenhuma troca de endpoint InfinitePay.
- Nenhum fallback automatico para outro endpoint.

## Validacao

- `go test ./...` passou antes da alteracao como baseline.
- Validacoes finais executadas:
  - `templ generate`;
  - `npm run css:build`;
  - `gofmt -w .`;
  - `go test ./...`;
  - `go vet ./...`;
  - `go build ./...`.
