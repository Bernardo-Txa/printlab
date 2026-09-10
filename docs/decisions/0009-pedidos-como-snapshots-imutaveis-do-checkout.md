# ADR-0009 — Pedidos como snapshots imutaveis do checkout

Status: Aprovado
Data: 2026-09-09

## Contexto

A PrintLab ja possui catalogo, variantes, receita estimada de producao 3D, carrinho anonimo, dados temporarios de checkout e selecao de frete SuperFrete.

Carrinho e um estado mutavel: preco, disponibilidade, perfil logistico, endereco e selecao de frete podem mudar antes da confirmacao.

Pedido e um registro historico: depois de criado, deve continuar representando exatamente o que foi confirmado, mesmo se produto, variante, receita, material, cor, caixa ou selecao temporaria mudarem depois.

## Decisao

Criar pedidos como snapshots imutaveis do checkout.

Ao confirmar a revisao, o backend relera e revalidara o estado atual em uma transacao PostgreSQL, fara lock do carrinho, gravara `orders`, dados de cliente, endereco, frete, itens e componentes de receita, convertera o carrinho com `carts.converted_at` e removera dados temporarios do carrinho.

O pedido preserva:

- nome, slug, SKU, quantidade, preco unitario e subtotal de cada item;
- tempo estimado de impressao por unidade;
- peso estimado de filamento por unidade;
- componentes de receita por material, cor, peso e label;
- dados de cliente e endereco como snapshot privado;
- servico, transportadora, prazo, caixa, peso, dimensoes externas e preco de frete.

`order_number` e sequencial e serve apenas como referencia humana. A rota publica do pedido usa UUID.

## Alternativas consideradas

- Referenciar diretamente `products`, `product_variants`, `materials`, `colors`, `shipping_boxes` e tabelas temporarias do carrinho como fonte do pedido.
- Guardar apenas IDs de produto e variante, recalculando dados no momento de exibicao.
- Criar pedido somente depois da confirmacao de pagamento.

## Consequencias

- Pedidos antigos nao mudam quando catalogo, receitas, materiais, cores ou caixas forem editados.
- A Fase 10 podera iniciar pagamento a partir de um pedido ja criado com valores congelados.
- Dados temporarios de checkout podem ser apagados depois da criacao do pedido, reduzindo duplicacao de PII no carrinho.
- Mudancas futuras de status, pagamento, cancelamento, envio e painel administrativo devem evoluir o schema de pedidos sem substituir o snapshot original.

## Referencias

- [ARCHITECTURE.md](../../ARCHITECTURE.md)
- [docs/product/orders.md](../product/orders.md)
- [docs/database/schema.md](../database/schema.md)
- [docs/architecture/security.md](../architecture/security.md)
