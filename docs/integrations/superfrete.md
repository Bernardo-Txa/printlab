# SuperFrete

Status: PLANEJADO. Nenhuma chamada HTTP real foi implementada.

## Arquitetura planejada

```text
Go Backend
    |
    v
SuperFrete API
```

## Comportamento futuro do backend

O backend devera futuramente:

- receber CEP;
- validar parametros;
- consultar fretes;
- retornar opcoes;
- validar novamente o frete no checkout;
- associar opcao escolhida ao pedido.

## Contratos de API

Confirmar na documentacao oficial durante a implementacao.

Este documento nao define endpoints, payloads, codigos de erro ou formato de resposta da SuperFrete.

## Praticas proibidas

- Chamar SuperFrete diretamente do navegador usando credenciais.
- Aceitar valor de frete do frontend como valor final.
- Criar pedidos usando frete sem revalidacao server-side.
- Inventar endpoints ou contratos antes de consultar a documentacao oficial.
