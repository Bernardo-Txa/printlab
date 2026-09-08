# Frontend

Status: fundacao visual IMPLEMENTADA; interacoes HTMX e funcionalidades comerciais PLANEJADAS.

## Responsabilidade

O frontend apresenta paginas HTML renderizadas no servidor. A experiencia deve ser simples, rapida e acessivel.

Nesta fase, a homepage em `GET /` e renderizada com `templ`, usando Tailwind CSS compilado localmente.

A logo oficial inicial da PrintLab foi integrada ao header e ao hero da homepage. Ela deve ser tratada como fonte de verdade visual nesta etapa, sem redesenho ou alteracao do conteudo da imagem.

## Limites

- O frontend nao acessa diretamente tabelas sensiveis.
- O frontend nao decide preco, desconto, subtotal, total, frete, status de pedido ou status de pagamento.
- O frontend nao armazena credenciais de integracoes.

## Decisoes

- Usar renderizacao server-side.
- Usar `templ` v0.3.1020 para templates tipados.
- Usar Tailwind CSS v4.3.3 via CLI, sem CDN.
- Usar design tokens em `web/assets/css/app.css`.
- Refinar tokens com base na paleta da logo: navy, azul vivo, teal, magenta e amarelo/laranja.
- Servir CSS compilado por `/static/css/app.css` usando assets embutidos via `embed.FS`.
- Usar HTMX futuramente para atualizacoes parciais baseadas em HTTP, apenas quando houver interacao real.
- Manter JavaScript proprio no minimo necessario.

## Estrutura implementada

```text
web/components/          componentes templ reutilizaveis
web/templates/           paginas templ
web/assets/css/app.css   CSS fonte e design tokens
web/static/css/app.css   CSS compilado, embutido no binario e servido pela aplicacao
web/static/images/branding/logo-printlab-primary.png   logo oficial inicial da marca
```

Arquivos Go gerados pelo `templ` permanecem versionados para que `go build ./...` funcione sem geracao implicita durante a execucao.

Arquivos em `web/static/` sao embutidos no binario Go. Essa estrategia deixa o servidor autossuficiente para entregar CSS, imagens e JavaScript futuro sem depender de caminhos de filesystem no runtime da Vercel.

## Design tokens

Tokens iniciais cobrem conceitos semanticos:

- background;
- surface;
- foreground;
- muted;
- border;
- primary;
- primary foreground;
- accent;
- accent foreground;
- highlight blue;
- highlight pink;
- highlight teal;
- highlight yellow;
- secondary;
- danger;
- radius;
- container width.

Componentes devem usar tokens e classes semanticas, evitando hex colors arbitrarias espalhadas por templates.

## Praticas recomendadas

- Formularios sem dependencia obrigatoria de JavaScript.
- Interacoes HTMX que preservem semantica HTTP.
- Componentes reutilizaveis apenas quando reduzirem duplicacao real.
- Estados de erro claros vindos do backend.
- Acessibilidade considerada desde os primeiros layouts.
- Skip link para o conteudo principal.
- Apenas um H1 por pagina.
- `focus-visible` perceptivel.
- Alt adequado para imagens da marca quando a imagem comunica conteudo.

## Praticas proibidas

- Duplicar regra financeira no navegador.
- Criar SPA pesada sem decisao arquitetural registrada.
- Usar JavaScript para contornar validacao server-side.
- Expor tokens, chaves ou endpoints sensiveis no cliente.
- Usar CDN do Tailwind.
- Adicionar HTMX sem interacao que justifique sua presenca.
- Redesenhar, alterar ou substituir a logo oficial sem decisao do responsavel pelo projeto.
- Usar cores vibrantes da marca de forma aleatoria ou excessiva.
