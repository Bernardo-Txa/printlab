# Schema de banco

Status: PLANEJADO. Fundacao PostgreSQL/Supabase implementada, sem tabelas de negocio.

Schema ainda nao aprovado.

Nenhuma tabela deve ser criada nesta fase. As entidades abaixo sao candidatas provaveis para fases futuras e precisam de revisao antes de virar migration.

## Convencoes futuras

- Usar `snake_case` para tabelas, colunas, constraints e indices.
- Usar `timestamptz` para datas e horas.
- Tratar horarios em UTC no banco.
- Usar `NOT NULL` quando a coluna for obrigatoria.
- Declarar foreign keys explicitas para relacionamentos.
- Colocar constraints no banco para invariantes importantes.
- Criar indices a partir de queries reais ou necessidades claras.
- Evitar `SELECT *` em codigo de producao.
- Alteracoes de schema devem usar migrations versionadas em `supabase/migrations/`.
- Migrations aplicadas nao devem ser alteradas silenciosamente.

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
- Regras financeiras nunca devem depender somente de frontend ou RLS.

## Dinheiro

Valores financeiros futuros nao devem usar `float32` ou `float64` como representacao canonica.

A preferencia atual e armazenar dinheiro como inteiro em centavos:

```text
R$ 39,90 -> 3990
```

Nenhum preco e implementado nesta fase.

## IDs

Nao ha estrategia universal de IDs aprovada. `uuid` e `bigint identity` serao avaliados conforme cada entidade.

Nenhuma extensao PostgreSQL deve ser habilitada sem necessidade atual.

## RLS e Data API

Supabase Data API nao e a interface principal da PrintLab. O browser nao acessa tabelas sensiveis diretamente; o backend Go controla regras criticas.

RLS continua util como camada complementar futura, mas nao substitui validacao server-side para precos, frete, pagamentos, pedidos ou permissoes sensiveis.

## Pendencias

- Definir representacao monetaria.
- Definir status de pedido.
- Definir status de pagamento.
- Definir modelo de variantes.
- Definir estrategia para produtos sob demanda.
- Definir dados minimos de cliente e endereco.
