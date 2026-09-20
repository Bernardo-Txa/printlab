# Fase 17.3.1 — Separacao de cor comercial e producao

Status: Em execucao; aguardando validacao manual em producao.

## Objetivo

Preparar a escolha de cor comercial do produto sem transformar cor comercial em variante tecnica e sem alterar a receita de producao.

## Implementado

- A relacao `product_colors` associa produtos a cores comerciais disponiveis, com ordenacao e unicidade por produto/cor.
- A UI administrativa do cadastro de produto (novo e edicao) lista todas as cores, permite selecao multipla, remocao e ordem `sort_order`.
- Produto inativo pode existir sem cores comerciais; produtos ativos tambem podem existir sem cores.
- `colors` continua representando cores usadas na producao e `variant_filaments` continua sendo a receita.
- Nenhuma variante existente foi convertida, removida ou recalculada.

## Migration

- Criada `product_colors` sem alterar ou remover tabelas existentes.
- Rollback possivel removendo a nova tabela; nenhuma carga automatica foi executada.

## Ajuste de compatibilidade

- Removida a exigencia anterior de cor para ativacao: criar, ativar e editar produtos permite zero ou varias cores.
- IDs, duplicidade, disponibilidade e ordem das cores selecionadas continuam validados no servidor.
- O detalhe publico recebe cores comerciais ativas ordenadas, sem seletor novo ou alteracao no checkout.
- Nenhuma migration adicional; reutilizada a relacao existente.

## Validacao manual pendente

- Selecionar, ordenar e remover cores comerciais no Admin.
- Confirmar que a receita de cada configuracao continua independente da selecao comercial.
- Confirmar que produtos ativos e inativos podem permanecer sem cores.

## Limites desta fase

- A experiencia de escolha de cor na loja, checkout, pedido e producao permanece para fases posteriores.
- Nenhum snapshot de pedido, carrinho, frete, pagamento ou fluxo de producao foi alterado.
