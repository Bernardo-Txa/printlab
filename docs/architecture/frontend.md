# Frontend

Status: fundacao visual, catalogo SSR, selecao SSR de variantes e carrinho SSR IMPLEMENTADOS; interacoes HTMX e funcionalidades comerciais futuras PLANEJADAS.

## Responsabilidade

O frontend apresenta paginas HTML renderizadas no servidor. A experiencia deve ser simples, rapida e acessivel.

Nesta fase, a homepage em `GET /`, o catalogo em `GET /produtos`, o detalhe de produto em `GET /produtos/{slug}` e o carrinho em `GET /carrinho` sao renderizados com `templ`, usando Tailwind CSS compilado localmente. O detalhe aceita `?variante=<slug>` para trocar variante por links SSR, sem JavaScript obrigatorio.

A logo oficial inicial da PrintLab foi integrada ao header e ao hero da homepage. Ela deve ser tratada como fonte de verdade visual nesta etapa, sem redesenho ou alteracao do conteudo da imagem.

A Fase 2.1 refinou a homepage para ter mais presenca de marca, com hero editorial, blocos de DNA PrintLab, processo visual, catalogo futuro abstrato, storytelling e CTA final. Isso continua dentro do escopo visual da Fase 2.

## Limites

- O frontend nao acessa diretamente tabelas sensiveis.
- O frontend nao decide preco, desconto, subtotal, total, frete, status de pedido ou status de pagamento.
- O frontend nao armazena credenciais de integracoes.
- O frontend nao calcula preco de produto ou variante; recebe o preco efetivo ja formatado pelo backend.
- O frontend do carrinho nao envia preco, subtotal, total ou nome de produto como fonte de verdade.

## Decisoes

- Usar renderizacao server-side.
- Usar `templ` v0.3.1020 para templates tipados.
- Usar Tailwind CSS v4.3.3 via CLI, sem CDN.
- Usar design tokens em `web/assets/css/app.css`.
- Refinar tokens com base na paleta da logo: navy, azul vivo, teal, magenta e amarelo/laranja.
- Usar cores vibrantes estruturalmente por secao, mantendo navy como ancora visual.
- Servir CSS compilado por `/static/css/app.css` usando assets embutidos via `embed.FS`.
- Usar HTMX futuramente para atualizacoes parciais baseadas em HTTP, apenas quando houver interacao real.
- Manter JavaScript proprio no minimo necessario.
- Exibir catalogo e detalhe de produto sem JavaScript obrigatorio.
- Exibir seletor de variantes como links navegaveis por teclado.
- Exibir carrinho com forms HTML e redirects 303, sem JavaScript obrigatorio.
- Usar input numerico de quantidade apenas como melhoria de UX; o backend valida `1..99`.
- Usar imagem geral primaria em cards quando existir.
- Priorizar imagens da variante selecionada no detalhe; quando nao existirem, usar imagens gerais do produto.
- Usar placeholder visual de marca quando nao houver imagem publica renderizavel.
- Manter canonical do detalhe como `/produtos/{slug}`, sem depender de query string de variante.

## Estrutura implementada

```text
web/components/          componentes templ reutilizaveis
web/templates/           paginas templ
web/assets/css/app.css   CSS fonte e design tokens
web/static/css/app.css   CSS compilado, embutido no binario e servido pela aplicacao
web/static/images/branding/logo-printlab-primary.png   logo oficial inicial da marca
```

Templates de catalogo implementados:

- `web/templates/catalog.templ` para catalogo, detalhe, indisponibilidade e 404 de produto.
- `web/templates/cart.templ` para carrinho vazio, linhas, resumo e indisponibilidade.
- `web/components/product_card.templ` para card reutilizavel, media de produto, galeria SSR e placeholder visual de produto.

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
- Homepage sem copy de implementacao tecnica voltada a desenvolvedores.
- Empty state honesto quando o catalogo estiver vazio.
- Slug de categoria como filtro publico em links server-side.
- Slug de variante como query parameter opcional em links server-side.
- Galeria sem carousel, slider ou dependencia JavaScript.

## Praticas proibidas

- Duplicar regra financeira no navegador.
- Criar SPA pesada sem decisao arquitetural registrada.
- Usar JavaScript para contornar validacao server-side.
- Expor tokens, chaves ou endpoints sensiveis no cliente.
- Usar CDN do Tailwind.
- Adicionar HTMX sem interacao que justifique sua presenca.
- Redesenhar, alterar ou substituir a logo oficial sem decisao do responsavel pelo projeto.
- Usar cores vibrantes da marca de forma aleatoria ou excessiva.
- Criar controles falsos de quantidade, estoque, carrinho ou checkout antes das fases aprovadas.
- Criar botao de checkout funcional falso.
- Usar imagens falsas, stock photo ou placeholders externos para produtos.
