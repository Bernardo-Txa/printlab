# Architecture Decision Records

Architecture Decision Records, ou ADRs, registram decisoes arquiteturais relevantes, o contexto em que foram tomadas e suas consequencias.

## Padrao de nome

```text
0001-nome-da-decisao.md
0002-outra-decisao.md
```

## Estrutura

```markdown
# ADR-NNNN — Titulo

Status:
Data:

## Contexto

## Decisao

## Alternativas consideradas

## Consequencias

## Referencias
```

## ADRs iniciais

- [ADR-0001 — Go como linguagem principal](0001-go-como-linguagem-principal.md)
- [ADR-0002 — Frontend server-side com templ e HTMX](0002-frontend-server-side-com-templ-e-htmx.md)
- [ADR-0003 — PostgreSQL/Supabase com pgx e Transaction Pooler](0003-postgresql-supabase-via-pgx.md)
- [ADR-0004 — Catalogo basico com categories e products](0004-catalogo-basico-com-categories-products.md)
- [ADR-0005 — Modelagem de variantes e receita de producao 3D](0005-modelagem-de-variantes-e-receita-de-producao-3d.md)
- [ADR-0006 — Carrinho anonimo persistido server-side](0006-carrinho-anonimo-persistido-server-side.md)
- [ADR-0007 — Dados temporarios de checkout vinculados ao carrinho](0007-dados-temporarios-checkout-vinculados-ao-carrinho.md)
- [ADR-0008 — Selecao de embalagem fisica para frete](0008-selecao-de-embalagem-fisica-para-frete.md)
- [ADR-0009 — Pedidos como snapshots imutaveis do checkout](0009-pedidos-como-snapshots-imutaveis-do-checkout.md)
- [ADR-0010 — Pagamento hospedado via InfinitePay](0010-pagamento-hospedado-via-infinitepay.md)
