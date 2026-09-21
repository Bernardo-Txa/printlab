# Fase 17.3.3 — Persistencia de cor no carrinho e pedido

Status: Concluida; validada manualmente em producao.

## Escopo implementado

- O formulario do produto envia `color_slug` opcional, derivado da selecao visual por `?cor=`.
- O backend resolve o ID somente entre cores ativas vinculadas em `product_colors`. Nome, disponibilidade e preco enviados pelo navegador nao sao autoridade.
- `cart_items.color_id` referencia `colors`. Produto + variante opcional + cor opcional identificam a linha; a mesma escolha incrementa quantidade, e cores diferentes geram linhas separadas.
- Carrinhos antigos e produtos sem cor continuam aceitos. Ter cores disponiveis nao torna a selecao obrigatoria.
- Cores desativadas ou desvinculadas deixam a linha indisponivel e impedem confirmar pedido. O cliente pode remover o item e selecionar novamente.
- Carrinho, resumos de dados/frete, revisao, pedido publico e detalhe administrativo exibem a cor comercial separada da variante.
- A confirmacao revalida a cor e grava `order_items.color_id`, `color_name` e `color_slug` como referencia e snapshot historico. O fingerprint da revisao inclui ID, nome e slug.
- Renomear ou excluir a cor no catalogo nao altera o nome historico do pedido. Receitas e seus snapshots continuam independentes.
- Nenhuma combinacao de variantes e criada. Nenhuma regra financeira, integracao InfinitePay, SuperFrete ou receita foi alterada.

## Migration e publicacao

Migration: `supabase/migrations/20260920224834_persist_commercial_colors.sql`.

Adiciona colunas opcionais nas tabelas existentes e quatro indices unique parciais para carrinho com/sem variante e com/sem cor. Nenhuma nova tabela ou backfill. RLS e policies atuais permanecem iguais.

Aplicar a migration antes de disponibilizar o novo backend. O codigo anterior usa predicados antigos de upsert, portanto a troca requer coordenacao entre schema e aplicacao. O push segue o workflow de migrations existente; confirmar sucesso desse workflow antes de liberar o backend.

Rollback preferido: corrigir adiante preservando os dados. Remover as colunas perderia cores historicas e recriar a unicidade antiga pode falhar quando houver linhas de cores diferentes; nao executar rollback destrutivo automatico.

## Validacao automatizada

- Servico de carrinho: selecao opcional, cores diferentes, repeticao, cor invalida e indisponibilidade.
- Handler e template: envio do slug selecionado e compatibilidade sem selecao.
- Revisao: cor independente da receita e alteracoes de cor detectadas no fingerprint.
- Integracao PostgreSQL: adicionar, agregar, separar, limitar quantidade, bloquear cor inativa/desvinculada e confirmar pedido nos quatro cenarios com/sem variante e com/sem cores.
- Integracao verifica snapshot apos renomeacao/exclusao, leitura do Admin, receita preservada e confirmacao idempotente. Segue skip quando `TEST_DATABASE_URL` nao esta definida.
- Executados com sucesso: `templ generate`, `gofmt`, `go test ./...`, `go vet ./...` e `go build ./...`. A suite completa foi executada com `TEST_DATABASE_URL` apontando para PostgreSQL 16 descartavel, incluindo as integracoes.
- Migration aplicada tambem sobre fixtures legadas anteriores a 17.3.3: quantidades, valores, ausencia de cor e RLS preservados. Nenhum banco remoto foi acessado nessa verificacao.

## Validacao manual em producao

Validado em producao:

- Escolher duas cores do mesmo produto gera linhas separadas.
- Checkout conclui preservando a cor comercial escolhida.
- Pedido e Admin exibem cor e variante separadas.
- Produto sem cores continua funcionando.
