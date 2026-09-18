# Fase 17.2 — Logistica simplificada

Status: Concluida; validada manualmente em producao.

## Objetivo

Simplificar o cadastro logistico para que cada produto tenha um unico perfil autoritativo e para que o Admin exponha um unico conjunto operacional de dimensoes por caixa.

## Implementado

- O perfil em `products.shipping_*` e a unica fonte usada por frete, checkout e revisao de pedido.
- Campos `product_variants.shipping_*` permanecem no schema como legado inerte; nao houve migration nem recalculo em lote.
- Produto ativo exige peso e as tres dimensoes positivos. Produto inativo aceita perfil ausente ou completo, nunca parcial.
- Novos produtos iniciam inativos e o formulario de produto mostra os quatro campos diretamente.
- Formularios e POSTs de configuracao nao leem nem alteram perfil logistico.
- A lista administrativa diferencia perfil ausente e destaca produto ativo inconsistente.
- O formulario de caixa mostra altura, largura e comprimento operacionais, peso da embalagem, status e ordem.
- A lista de caixas exibe um unico conjunto operacional de dimensoes, usando as dimensoes externas nos registros legados.
- Nova caixa grava as dimensoes operacionais nos campos internos e externos.
- Ao editar, mudar qualquer dimensao sincroniza os dois conjuntos; sem mudanca dimensional, os valores internos e externos persistidos sao preservados pelo servidor.
- O contrato HTTP da SuperFrete e a estrategia de duas cotacoes foram preservados.

## Validacao automatizada

- Regras de perfil completo/ausente por status do produto.
- Ignorar perfil legado de configuracao no calculo de frete.
- Validacao das dimensoes operacionais da caixa.
- Suite Go, vet e build antes do fechamento tecnico.

## Validacao manual concluida

- Novos produtos iniciam inativos e produto ativo exige perfil logistico completo.
- Configuracoes nao possuem autoridade logistica; valores legados nao sobrescrevem o produto.
- Frete usa somente o perfil logistico do produto.
- Criacao de caixa grava dimensoes internas e externas iguais.
- Edicao somente de peso preserva diferencas internas/externas antigas.
- Alteracao dimensional sincroniza dimensoes internas e externas.
- A lista administrativa mostra um unico conjunto operacional de dimensoes.

## Definition of Done

- Implementacao, testes e documentacao versionados.
- Nenhuma migration criada.
- Validacao manual em producao registrada antes de mover este plano para `completed/`.
