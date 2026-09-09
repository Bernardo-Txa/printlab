# Fase 5.1 — Correcao semantica da receita de producao

Status: CONCLUIDA.

## Contexto

A Fase 5 implementou `materials`, `colors`, `product_variants`, `variant_filaments` e `product_images`. A leitura da receita de producao filtrava `materials.is_active = true` e `colors.is_active = true` ao buscar `variant_filaments`.

Esse comportamento removia componentes de receitas ja cadastradas quando um material ou cor era desativado. A desativacao deve impedir novas escolhas operacionais futuras, mas nao deve apagar semanticamente receitas existentes.

## Escopo

- Corrigir a query de leitura de `variant_filaments` para preservar material e cor referenciados mesmo quando inativos.
- Manter `products.is_active` e `product_variants.is_active` como controle de visibilidade publica.
- Garantir que o peso total estimado some todos os componentes carregados da receita.
- Atualizar documentacao de produto, schema e roadmap.
- Adicionar regressao automatizada.

## Fora de escopo

- Migration.
- Alteracao de schema.
- Carrinho.
- Checkout.
- Estoque.
- Admin.
- Custos de producao.
- Novos metodos para cadastro ou listagem operacional de materiais e cores.

## Semantica final

- `materials.is_active = false` significa que o material nao deve ser oferecido para novas configuracoes operacionais futuras.
- `colors.is_active = false` significa que a cor nao deve ser oferecida para novas configuracoes operacionais futuras.
- Receitas existentes em `variant_filaments` continuam exibindo os nomes de material e cor referenciados, mesmo quando esses registros estiverem inativos.
- O peso total estimado de uma variante soma todos os componentes carregados da receita.

## Validacoes executadas

- `templ generate` passou.
- `npm run css:build` passou.
- `gofmt -w .` passou.
- `go test ./...` passou.
- `go vet ./...` passou.
- `go build ./...` passou.
- Nenhuma migration foi criada.
- Nenhuma alteracao de schema foi feita.
- Nenhum secret foi versionado.

## Definition of Done

- Query de receita preserva material e cor inativos.
- Produto ativo e variante ativa continuam controlando visibilidade publica.
- Peso total estimado soma todos os componentes carregados.
- Documentacao atualizada.
- Fase 6 permanece planejada.
