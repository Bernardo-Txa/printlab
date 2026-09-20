# Fase 17.3.2 — Cores comerciais na pagina do produto

Status: Em execucao; aguardando validacao manual em producao.

- Secao "Escolha a cor" somente para produtos com cores comerciais ativas vinculadas.
- Swatches circulares usam hex validado; ausencia de hex usa fundo neutro e nome visivel.
- Links acessiveis por teclado, com destaque por contorno e `aria-current`, nome da cor selecionada e quebra de linha em telas menores.
- Refinamento visual: componente `ProductColorSelector` alinhado ao conteudo do card de produto, com o mesmo espacamento do bloco de variantes; swatches de 48 px sem lista textual, contorno azul e marca de selecao. Nomes mantidos nos atributos acessiveis e tooltip; somente o nome escolhido aparece abaixo. Sem escolha, nao exibe instrucao generica nem rotulo vazio.
- Selecao opcional mantida em `?cor=slug`, independente de `?variante=slug`; nao cria combinacoes nem altera variantes.
- Cor desconhecida ou antiga e ignorada, sem bloquear produto. Canonical permanece `/produtos/{slug}`.
- HTML SSR sem JavaScript novo. Nenhuma cor e enviada ao carrinho ou pedido nesta fase.
- Nenhuma tabela, migration, alteracao de checkout ou receita.

## Validacao

- Testes de selecao, troca independente, hex, renderizacao, ausencia de cores e contrato de formulario do carrinho.
- Geracao templ/CSS, gofmt, suite Go, vet e build.
- Previa local do template inspecionada em Chromium: 390x1400 e 1440x1100, com cores claras/escuras/sem hex e selecao destacada.
- Em producao: testar desktop/mobile, foco por teclado, cores claras/escuras/sem hex, selecao e troca de variantes, produto sem cores e adicionar ao carrinho.
