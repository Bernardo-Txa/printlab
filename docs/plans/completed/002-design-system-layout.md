# Fase 2 — Design system e layout

Status: Concluida

## Objetivo

Criar a fundacao visual server-side da aplicacao PrintLab, com `templ`, Tailwind CSS, layout base, homepage, componentes essenciais, assets estaticos, responsividade, acessibilidade basica e testes aplicaveis.

## Escopo

- Renderizacao server-side da homepage.
- Layout HTML base.
- Header, footer, container, botoes/link e empty state.
- Pipeline de CSS com Tailwind CLI.
- Design tokens iniciais.
- Servico seguro de `/static/` via `embed.FS`.
- Documentacao atualizada.

## Fora de escopo

- Banco de dados.
- Catalogo real.
- Carrinho.
- Checkout.
- SuperFrete.
- InfinitePay.
- Autenticacao.
- Fase 3.

## Validacoes previstas

- `templ generate`
- `npm run css:build`
- `gofmt`
- `go test ./...`
- `go vet ./...`
- `go build ./...`

## O que foi implementado

- Homepage server-side em `GET /`.
- Layout HTML base com `lang="pt-BR"`, metadados configuraveis, skip link, landmarks e CSS da aplicacao.
- Header e footer responsivos.
- Componentes visuais essenciais.
- Design tokens iniciais, refinados posteriormente com base na identidade visual da logo oficial.
- Tailwind CSS 4 via CLI npm.
- CSS fonte em `web/assets/css/app.css`.
- CSS compilado em `web/static/css/app.css`.
- Assets estaticos embutidos no binario Go para compatibilidade com Vercel.
- Servico de assets estaticos em `/static/`.
- Logo oficial inicial integrada ao header e ao hero, sem redesenho do asset.
- Testes de homepage, health check, CSS estatico e rotas desconhecidas.

## Arquivos principais

- `cmd/server/main.go`
- `cmd/server/main_test.go`
- `web/templates/home.templ`
- `web/components/`
- `web/assets/css/app.css`
- `web/static/css/app.css`
- `web/static/images/branding/logo-printlab-primary.png`
- `package.json`
- `package-lock.json`
- `go.mod`
- `go.sum`

## Resultado final

Fundacao visual implementada e refinada com a identidade inicial da marca, sem banco, catalogo real, carrinho, checkout, SuperFrete, InfinitePay, autenticacao ou Fase 3.
