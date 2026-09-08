# InfinitePay

Status: PLANEJADO. Nenhuma integracao de pagamento foi implementada.

## Arquitetura planejada

```text
PrintLab
   |
   v
InfinitePay Checkout
   |
   v
Pagamento
   |
   v
Webhook
   |
   v
PrintLab
```

## Conceitos planejados

- Criacao server-side do fluxo de pagamento.
- Uso de order ID interno.
- Redirect do usuario para fluxo de pagamento quando aplicavel.
- Recebimento de webhook.
- Confirmacao de pagamento somente apos validacao server-side.
- Idempotencia para evitar processamento duplicado.
- Nunca confiar apenas no navegador.

## Contratos de API

Detalhes da API devem ser verificados na documentacao oficial quando a fase de integracao comecar.

Este documento nao define endpoints, payloads, eventos ou assinaturas.

## Praticas proibidas

- Confirmar pagamento por redirect.
- Processar webhook sem validacao.
- Expor credenciais InfinitePay no frontend.
- Inventar contratos sem documentacao oficial.
