# Fase 2.1 — Brand Experience

Status: Concluida

## Objetivo

Refinar a experiencia visual da homepage para fortalecer a identidade da PrintLab sem alterar a arquitetura Go + templ + Tailwind.

## Escopo

- Hero mais expressivo e menos artificialmente alto.
- Logo oficial com mais protagonismo.
- Tokens alinhados a paleta da marca.
- Linguagem grafica com elementos de laboratorio, particulas e camadas de impressao 3D.
- Secoes mais editoriais para DNA PrintLab, processo, catalogo futuro, storytelling e CTA.
- Testes aplicaveis preservados e ajustados.

## Fora de escopo

- Banco de dados.
- Supabase.
- Catalogo real.
- Carrinho.
- Checkout.
- Pagamentos.
- SuperFrete.
- Autenticacao.
- Painel administrativo.
- Fase 3.

## Validacoes previstas

- `templ generate`
- `npm run css:build`
- `gofmt -w .`
- `go test ./...`
- `go vet ./...`
- `go build ./...`
- busca por referencias antigas ao entrypoint Go

## O que foi implementado

- Header com logo oficial em tamanho mais reconhecivel.
- Hero editorial com copy em tres linhas e palavras de destaque.
- Componente de grafismo molecular e camadas de impressao 3D.
- Secao DNA PrintLab com quatro blocos coloridos.
- Secao de processo com caminho visual de filamento.
- Area de catalogo futuro com placeholders abstratos, sem produtos falsos.
- Secao "Por que Lab?" com storytelling curto.
- Bloco de impacto em navy.
- CTA final simples, sem formulario ou integracao.
- Testes para manter rotas, assets e copy da homepage sem termos tecnicos internos.

## Resultado final

Homepage mais autoral, colorida e alinhada a identidade visual inicial da PrintLab, preservando SSR com `templ`, Tailwind CSS, assets embutidos via `go:embed` e ausencia de funcionalidades comerciais.
