# Carrinho

Status: PLANEJADO.

O carrinho devera permitir que visitantes escolham produtos e quantidades antes do checkout.

## Comportamento planejado

- Adicionar itens por ID de produto ou variante.
- Atualizar quantidades.
- Remover itens.
- Persistir carrinho de visitante de forma apropriada.
- Recalcular valores no backend sempre que necessario.

## Regras de seguranca

O carrinho no navegador nao sera fonte autoritativa de preco, subtotal, desconto, frete ou total. O backend devera recalcular valores usando dados persistidos e regras aprovadas.

## Limites

- Nao ha carrinho implementado nesta fase.
- Nao ha decisao final sobre persistencia de carrinho anonimo.
- Nao ha schema aprovado para `carts` ou `cart_items`.
