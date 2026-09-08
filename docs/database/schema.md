# Schema de banco

Status: PLANEJADO.

Schema ainda nao aprovado.

Nenhuma tabela deve ser criada nesta fase. As entidades abaixo sao candidatas provaveis para fases futuras e precisam de revisao antes de virar migration.

## Entidades candidatas

- `products`: produtos publicados ou administrados pela PrintLab.
- `product_variants`: variacoes de produto, como cor, material ou tamanho.
- `product_images`: imagens associadas a produtos.
- `categories`: organizacao de catalogo.
- `customers`: dados minimos de clientes.
- `addresses`: enderecos de entrega ou cobranca quando necessario.
- `carts`: carrinhos de visitantes ou clientes.
- `cart_items`: itens dentro de carrinhos.
- `orders`: pedidos criados pelo backend.
- `order_items`: itens persistidos de pedido com valores calculados pelo backend.
- `payments`: registros de pagamento, tentativas e status validados.
- `shipments`: dados de frete e envio.

## Regras iniciais

- Valores financeiros devem ter representacao segura e deterministica.
- `float32` e `float64` nao devem ser usados como representacao canonica de dinheiro.
- Pedidos devem preservar os valores calculados no momento da compra.
- Pagamentos e webhooks exigem desenho de idempotencia antes da implementacao.
- Mudancas de schema devem usar migrations versionadas.

## Pendencias

- Definir representacao monetaria.
- Definir status de pedido.
- Definir status de pagamento.
- Definir modelo de variantes.
- Definir estrategia para produtos sob demanda.
- Definir dados minimos de cliente e endereco.
