# ADR-0021 — Cor comercial separada da receita de producao

Status: Proposta

Data: 2026-09-17

## Contexto

`materials`, `colors` e `variant_filaments` implementados representam receita e operacao de producao: filamento, peso, material, cor e snapshots. O cliente pode futuramente escolher uma cor comercial de um produto sem que isso seja uma variante de producao nem exija duplicar receita.

## Decisao

Planejar na Fase 17.3 um conceito de cor comercial selecionavel a partir das cores cadastradas, separado da cor de filamento usada pela receita. A escolha devera seguir de forma autoritativa produto -> carrinho -> pedido -> producao.

O modelo futuro devera evitar conceitos duplicados ou logica ambigua entre cor de receita, cor de filamento e opcao comercial. Nao serao criadas variantes artificiais apenas para representar escolha de cor do comprador.

## Alternativas consideradas

- Remover materiais e cores: rejeitado, pois sao usados por receita, snapshots e operacao de producao.
- Reusar diretamente cor de receita como escolha comercial: rejeitado, pois mistura semanticas distintas.
- Criar schema agora: rejeitado; a modelagem pertence a fase futura.

## Consequencias

- A futura modelagem devera preservar snapshots de pedidos e a semantica atual de receita.
- Nenhuma cor comercial ou mudanca de schema existe nesta ADR.

## Referencias

- [Roadmap](../plans/roadmap.md)
- [Catalogo](../product/catalog.md)
- [ADR-0005](0005-modelagem-de-variantes-e-receita-de-producao-3d.md)
