# ADR-0018 — Perfil logistico unico por produto

Status: Proposta

Data: 2026-09-17

## Contexto

O modelo implementado permite perfil logistico opcional em produto e override completo em configuracao. A operacao planejada precisa de cadastro mais simples e, no modelo comercial esperado, as configuracoes nao alteram a embalagem de envio.

## Decisao

Na Fase 17.2, planejar um unico perfil logistico por produto. Configuracoes nao deverao sobrescreve-lo. Produto ativo/comercial devera ter perfil completo; rascunho ou produto inativo podera permanecer incompleto sem valores ficticios.

O formulario futuro de caixa recebera altura, largura, comprimento, peso da embalagem, ativo e ordem. A semantica atual sera preservada: medida interna serve ao encaixe e externa e enviada a transportadora. Inicialmente, a mesma medida operacional podera preencher ambas as colunas atuais; remocao de colunas exigira decisao e migration posteriores.

## Alternativas consideradas

- Manter override por configuracao: rejeitado para o modelo operacional planejado, pois amplia cadastro sem necessidade atual.
- Remover colunas e criar migration agora: rejeitado; esta ADR nao implementa mudanca alguma e a compatibilidade precisa ser avaliada na fase propria.

## Consequencias

- A implementacao futura precisara de migration e estrategia de transicao avaliadas separadamente.
- O comportamento implementado permanece inalterado ate a Fase 17.2.
- Cotacao continua sem estimativa ficticia quando faltar perfil ou caixa compativel.

## Referencias

- [Roadmap](../plans/roadmap.md)
- [Catalogo](../product/catalog.md)
- [ADR-0008](0008-selecao-de-embalagem-fisica-para-frete.md)
