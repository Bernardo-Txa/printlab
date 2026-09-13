# ADR-0015 — Gestao administrativa de catalogo sem hard delete

Status: Aceita
Data: 2026-09-13

## Contexto

A Fase 13.3 permite que o Admin altere catalogo, variantes, receita estimada de producao, materiais, cores e caixas fisicas de envio.

Essas entidades sao referenciadas por:

- catalogo publico;
- carrinhos ativos;
- variantes e receitas;
- caixas usadas em cotacoes;
- pedidos historicos preservados como snapshots.

Apagar registros definitivamente pelo painel poderia quebrar referencias, dificultar diagnostico operacional e criar divergencia entre a oferta atual e o historico de compras.

## Decisao

Entidades principais de catalogo e logistica serao inativadas, nao apagadas, no Admin:

- `categories`;
- `products`;
- `product_variants`;
- `materials`;
- `colors`;
- `shipping_boxes`.

O painel usa `is_active` para retirar registros de novas ofertas, escolhas ou cotacoes sem remover o registro do banco.

`variant_filaments` e excecao operacional. Componentes da receita atual podem ser adicionados, editados ou removidos porque pedidos ja criados preservam snapshots historicos em `order_item_filaments`.

A Fase 13.3 usa as tabelas existentes e nao cria migration. Tambem nao reutiliza `admin_order_events`, que tem semantica exclusiva de pedidos, nem cria `admin_catalog_events` por antecipacao.

## Alternativas consideradas

- Hard delete no Admin: rejeitado por risco de quebrar referencias e historico.
- Soft delete com coluna nova diferente de `is_active`: rejeitado porque o schema atual ja possui a semantica necessaria.
- Auditoria generica de catalogo nesta fase: rejeitada por escopo maior que a necessidade atual.
- Escolher automaticamente nova variante default ao desativar uma default: rejeitado porque a interface publica ja possui fallback deterministico para primeira variante ativa.

## Consequencias

- Historico e referencias permanecem preservados.
- Registros inativos continuam no banco e precisam ser considerados em telas administrativas.
- Novas escolhas devem preferir ou restringir registros ativos, preservando referencias inativas ja existentes quando aplicavel.
- Limpeza definitiva futura exige processo especifico e revisao de impactos.
- Uma auditoria de catalogo futura deve ser planejada separadamente, com semantica propria e minimizacao de dados.
