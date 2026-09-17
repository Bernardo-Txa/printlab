# Fase 15.1 — Testes criticos e observabilidade operacional base

Status: concluida localmente; observacao operacional depende de trafego real.

## Inventario

| Fluxo | Cobertura | Decisao |
| --- | --- | --- |
| Catalogo/carrinho e checkout | COBERTO | handlers, services, validacao e recalculo ja testados. |
| SuperFrete | COBERTO | sucesso, timeout, JSON invalido, sem cotacao e falhas parciais com `httptest`. |
| Pedidos | COBERTO | snapshots, total server-side, tracking e transicoes Admin. |
| InfinitePay | COBERTO | checkout, URL, 303, retorno/payment_check, webhook, idempotencia, divergencia e provider failures. |
| Admin/MFA e mutacoes | COBERTO | AAL2, sessoes legadas, logout e Origin/Referer. |
| Imagens | COBERTO | signed upload, finalize, replacement, cleanup, primary e order. |

O gap P0/P1 encontrado foi operacional: os logs criticos de pagamento podiam incluir ID de pedido e nao tinham correlacao uniforme. A fase adiciona eventos estruturados e `X-Request-ID` opaco, com testes; nao duplica cenarios de negocio ja cobertos.

## Limites

Sem migration, provider real, browser automation, rate limiter, Redis, ferramenta paga ou mudanca de regra de negocio. CI existente foi revisada: somente migrations possui workflow; um workflow Go separado fica adiado para evitar ampliar escopo, pois as validacoes locais continuam obrigatorias.
