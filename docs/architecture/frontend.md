# Frontend

Status: fundacao visual, catalogo SSR, selecao SSR de variantes, carrinho SSR, checkout SSR ate revisao e pedido SSR IMPLEMENTADOS; interacoes HTMX e pagamentos futuros PLANEJADOS.

## Responsabilidade

O frontend apresenta paginas HTML renderizadas no servidor. A experiencia deve ser simples, rapida e acessivel.

Nesta fase, a homepage em `GET /`, o catalogo em `GET /produtos`, o detalhe de produto em `GET /produtos/{slug}`, o carrinho em `GET /carrinho`, as etapas de dados, frete e revisao do checkout e a pagina de pedido em `GET /pedido/{id}` sao renderizados com `templ`, usando Tailwind CSS compilado localmente. O detalhe aceita `?variante=<slug>` para trocar variante por links SSR, sem JavaScript obrigatorio.

A logo oficial inicial da PrintLab foi integrada ao header e ao hero da homepage. Ela deve ser tratada como fonte de verdade visual nesta etapa, sem redesenho ou alteracao do conteudo da imagem.

A Fase 2.1 refinou a homepage para ter mais presenca de marca, com hero editorial, blocos de DNA PrintLab, processo visual, catalogo futuro abstrato, storytelling e CTA final. Isso continua dentro do escopo visual da Fase 2.

## Limites

- O frontend nao acessa diretamente tabelas sensiveis.
- O frontend nao decide preco, desconto, subtotal, total, frete, status de pedido ou status de pagamento.
- O frontend nao armazena credenciais de integracoes.
- O frontend nao calcula preco de produto ou variante; recebe o preco efetivo ja formatado pelo backend.
- O frontend do carrinho nao envia preco, subtotal, total ou nome de produto como fonte de verdade.
- O frontend de checkout nao envia preco, subtotal, total, frete, prazo, peso, dimensoes ou decisao financeira como fonte de verdade.
- A etapa de frete envia somente `service_code`; o backend revalida a cotacao e persiste o valor atual.
- A etapa de revisao envia somente `review_fingerprint`; o backend recalcula tudo e usa o fingerprint apenas para detectar tela antiga.
- A pagina de pedido nao deve exibir CPF completo, endereco completo, telefone ou e-mail completo.
- A UI de frete nao precisa expor caixa fisica, dimensoes internas/externas ou peso operacional ao consumidor.
- O formulario de dados usa mascaras progressivas e consulta CEP por endpoint interno da aplicacao; sem JavaScript, o preenchimento manual continua funcionando.

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
- Exibir a etapa de dados com formulario HTML, autocomplete nativo, mascaras progressivas, consulta interna de CEP e redirects 303, sem JavaScript obrigatorio.
- Exibir a etapa de frete com radios HTML, POST tradicional e redirects 303, sem JavaScript obrigatorio.
- Exibir a etapa de revisao com resumo, links de edicao e POST tradicional, sem JavaScript obrigatorio.
- Exibir pedido criado por UUID, sem botao falso de pagamento.
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
web/static/js/checkout.js   melhoria progressiva de mascaras e consulta CEP para checkout
web/static/images/branding/logo-printlab-primary.png   logo oficial inicial da marca
```

Templates de catalogo implementados:

- `web/templates/catalog.templ` para catalogo, detalhe, indisponibilidade e 404 de produto.
- `web/templates/cart.templ` para carrinho vazio, linhas, resumo e indisponibilidade.
- `web/templates/checkout.templ` para etapa de dados, contato, entrega, mensagens de validacao e resumo compacto do carrinho.
- `web/templates/shipping.templ` para etapa de frete, opcoes cotadas, estado indisponivel e resumo parcial.
- `web/templates/order.templ` para revisao de checkout, criacao de pedido e pagina de pedido.
- `web/components/product_card.templ` para card reutilizavel, media de produto, galeria SSR e placeholder visual de produto.

Arquivos Go gerados pelo `templ` permanecem versionados para que `go build ./...` funcione sem geracao implicita durante a execucao.

Arquivos em `web/static/` sao embutidos no binario Go. Essa estrategia deixa o servidor autossuficiente para entregar CSS, imagens e JavaScript progressivo sem depender de caminhos de filesystem no runtime da Vercel.

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
- Inputs de contato e endereco devem usar `autocomplete`, `inputmode` e labels claros. Mascaras de CPF, telefone e CEP sao melhoria de UX, nao validacao autoritativa.
- Consulta de CEP deve chamar apenas endpoint interno do backend e preservar campos editaveis e fallback manual.
- Opcoes de frete devem usar controles nativos de radio, labels clicaveis, preco e prazo vindos do backend.
- Revisao deve oferecer links para editar dados e alterar frete, sem editar os dados diretamente nessa etapa.

## Praticas proibidas

- Duplicar regra financeira no navegador.
- Criar SPA pesada sem decisao arquitetural registrada.
- Usar JavaScript para contornar validacao server-side.
- Expor tokens, chaves ou endpoints sensiveis no cliente.
- Usar CDN do Tailwind.
- Adicionar HTMX sem interacao que justifique sua presenca.
- Chamar ViaCEP, SuperFrete ou outras integracoes diretamente do navegador.
- Redesenhar, alterar ou substituir a logo oficial sem decisao do responsavel pelo projeto.
- Usar cores vibrantes da marca de forma aleatoria ou excessiva.
- Criar controles falsos de quantidade, estoque, carrinho ou checkout antes das fases aprovadas.
- Criar botao falso de pagamento.
- Enviar preco, prazo, transportadora, peso ou dimensoes de frete como campos autoritativos do formulario.
- Enviar subtotal, frete, total ou status como campos autoritativos do formulario de revisao.
- Usar imagens falsas, stock photo ou placeholders externos para produtos.
