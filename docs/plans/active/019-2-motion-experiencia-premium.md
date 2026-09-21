# Fase 19.2 — Motion e experiência premium

Status: Implementada; aguardando validação manual.

## Objetivo

Refinar a experiência visual da PrintLab com animações, transições e microinterações sutis, preservando a identidade atual, a renderização server-side e o funcionamento sem JavaScript obrigatório.

## Escopo implementado

- Sistema centralizado de movimento em CSS com tokens `--motion-fast`, `--motion-normal`, `--motion-slow`, `--motion-enter`, `--motion-ease` e `--motion-spring`.
- Microinterações para header, navegação, logo, botões, filtros, cards de produto, mídia de produto, variantes, swatches de cores comerciais, carrinho, checkout e Admin.
- Elemento decorativo lateral próprio da PrintLab, restrito a desktop, sem bloquear interação.
- Reveal progressivo com `IntersectionObserver`, executado uma vez por elemento, usando somente `opacity` e `transform`.
- Conteúdo permanece visível quando JavaScript está indisponível.
- Feedback visual rápido em botões submetidos, sem atrasar nem bloquear ações.
- Suporte a `prefers-reduced-motion: reduce`, removendo animações decorativas e reduzindo transições.

## Limites preservados

- Nenhuma alteração de banco, schema ou migration.
- Nenhuma alteração de checkout, carrinho, pedidos, pagamentos, SuperFrete, InfinitePay, autenticação, preços, variantes, cores comerciais ou APIs.
- Nenhuma biblioteca nova de animação.
- Nenhuma dependência de JavaScript para conteúdo principal ou fluxos comerciais.

## Validação técnica

- `templ generate`
- `gofmt`
- `npm run css:build`
- `go test ./...`
- `go vet ./...`
- `go build ./...`
- `git diff --check`

## Validação manual pendente

- Desktop: home, catálogo, produto, carrinho e checkout.
- Mobile: home, catálogo, produto, seletor de cor, carrinho e checkout.
- Confirmar ausência de overflow horizontal, foco de teclado visível, botões clicáveis, ações sem atraso, conteúdo visível sem JavaScript e reduced motion sem animações decorativas.
