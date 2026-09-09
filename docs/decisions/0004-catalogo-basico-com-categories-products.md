# ADR-0004 — Catalogo basico com categories e products

Status: aprovado
Data: 2026-09-09

## Contexto

A PrintLab precisa da primeira funcionalidade de negocio persistida no PostgreSQL: catalogo publico com categorias e produtos basicos.

A fundacao anterior ja definiu Go, `net/http`, SSR com `templ`, PostgreSQL/Supabase, `pgx/v5`, `pgxpool`, Supabase Transaction Pooler, migrations por GitHub Actions e Vercel em `gru1`.

## Decisao

Criar `public.categories` e `public.products` por migration versionada em `supabase/migrations/`.

Categorias e produtos usam UUID no banco e slug unico para URLs publicas. Produtos guardam `price_cents` como `bigint`, representando o preco-base comercial em centavos. O backend Go formata o preco para BRL antes de renderizar templates.

O catalogo publico mostra apenas produtos ativos. Produto inativo se comporta como inexistente. Categorias sao opcionais; `products.category_id` usa `on delete set null`.

RLS fica habilitado nas duas tabelas sem policies publicas nesta fase. A interface primaria segue sendo o backend Go via `pgxpool`, nao Supabase Data API.

## Alternativas consideradas

- Seed ou produto ficticio para demonstracao: rejeitado para evitar dados falsos no ambiente remoto.
- UUID nas URLs publicas: rejeitado; slug e mais adequado para catalogo publico e SEO basico.
- Dinheiro como decimal ou float: rejeitado nesta fase; inteiro em centavos e suficiente e deterministico.
- Trigger automatico para `updated_at`: adiado; updates futuros poderao alterar o campo explicitamente.
- Modelar variantes agora: rejeitado; variantes pertencem a Fase 5.

## Consequencias

- A primeira migration real passa a depender do workflow GitHub Actions para aplicacao no Supabase DEV.
- `/produtos` depende do PostgreSQL e retorna indisponibilidade generica se o banco ou schema nao estiver acessivel.
- O catalogo pode iniciar vazio sem seed.
- A Fase 5 pode adicionar variantes, materiais, cores, metricas de producao e estoque sem alterar o contrato de preco-base atual.

## Referencias

- [Schema](../database/schema.md)
- [Catalogo](../product/catalog.md)
- [Migrations](../database/migrations.md)
