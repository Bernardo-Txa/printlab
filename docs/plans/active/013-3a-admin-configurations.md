# Fase 13.3A — Refinamento de configuracoes do produto

Status: implementacao concluida; validacao real pendente.

## Objetivo

Melhorar a semantica publica e administrativa de `product_variants` sem trocar o modelo interno, sem criar schema novo e sem iniciar a Fase 13.4.

## Decisao

`product_variants` continua sendo a entidade interna para:

- SKU;
- preco especifico opcional;
- tempo estimado de impressao;
- receita de filamentos;
- perfil logistico;
- status ativo;
- imagens especificas futuras;
- snapshots de pedidos.

Na interface administrativa, o conceito deve ser apresentado como "Configuracao" ou "Configuracoes do produto".

Na loja publica, uma escolha so aparece quando houver duas ou mais configuracoes ativas.

## Regras implementadas

- Produto sem configuracao ativa continua funcionando como produto simples.
- Produto com exatamente uma configuracao ativa seleciona essa configuracao automaticamente.
- Produto com exatamente uma configuracao ativa nao mostra seletor, resumo "Variante" nem copy que indique escolha publica.
- O formulario publico continua enviando a configuracao selecionada internamente quando ela existe.
- Produto com duas ou mais configuracoes ativas mostra "Escolha uma opcao".
- `is_default` define a opcao inicialmente selecionada quando houver mais de uma configuracao ativa.
- Se nao houver default em dados legados, o fallback deterministico usa a primeira configuracao ativa pela ordenacao publica.
- Carrinho auto-resolve a unica configuracao ativa quando `variant_slug` nao e enviado.
- Carrinho continua recusando configuracao inexistente, inativa ou pertencente a outro produto.
- Admin evita "Nao default" para uma unica configuracao ativa e mostra "Unica configuracao".
- Admin usa "Usar como opcao padrao" no formulario de configuracao.

## Fora do escopo

- Renomear tabelas, rotas, structs, packages ou foreign keys de `product_variants`.
- Criar sistema generico de atributos, option groups ou matriz Cor x Tamanho.
- Criar migration ou atualizar dados antigos por script.
- Upload ou gestao de imagens.
- Alterar InfinitePay, webhook, SuperFrete ou autenticacao.

## Definition of Done

- Sem migration criada.
- Templates publicos e Admin atualizados.
- Carrinho preserva preco server-side e resolve unica configuracao ativa.
- Testes de produto, carrinho, Admin e handlers atualizados.
- Documentacao atualizada.
- Validacoes locais executadas antes do commit.
- Validacao real pos-deploy pendente.
