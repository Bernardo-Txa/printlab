# ADR-0007 - Dados temporarios de checkout vinculados ao carrinho

Status: Aprovada
Data: 2026-09-09

## Contexto

A PrintLab precisa iniciar o checkout sem login obrigatorio, coletando os dados minimos de contato e entrega para fases futuras de frete, pedido, DC-e e pagamento.

Esses dados sao PII. Criar uma entidade permanente de cliente antes de existir autenticacao poderia gerar duplicidade, acoplamento prematuro e retencao desnecessaria.

## Decisao

Dados de contato e endereco da etapa inicial do checkout ficam vinculados ao carrinho anonimo atual:

```text
carts
  |
  +-- cart_customer_details
  |
  +-- cart_shipping_addresses
```

Cada carrinho possui no maximo um registro de contato e um endereco de entrega atual. O pedido futuro devera copiar esses dados para snapshots definitivos no momento de criacao do pedido.

## Alternativas consideradas

- Criar `customers` permanente agora.
- Manter dados somente no browser.
- Exigir conta antes do checkout.
- Gravar endereco diretamente em `orders` antes da existencia de pedido.

## Consequencias

- Menor acoplamento com autenticacao futura.
- Menos PII permanente antes de existir cliente autenticado.
- Carrinho passa a possuir dados sensiveis e exige cuidado extra de logs, acesso e retencao.
- Limpeza programada de carrinhos expirados e PII associada torna-se requisito de producao antes do go-live comercial.

## Referencias

- [docs/product/checkout.md](../product/checkout.md)
- [docs/product/cart.md](../product/cart.md)
- [docs/database/schema.md](../database/schema.md)
- [docs/architecture/security.md](../architecture/security.md)
