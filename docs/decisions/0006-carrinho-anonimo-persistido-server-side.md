# ADR-0006 — Carrinho anonimo persistido server-side

Status: aprovado
Data: 2026-09-09

## Contexto

A PrintLab precisa permitir que visitantes montem um carrinho antes do checkout sem exigir login. O carrinho deve sobreviver ao fechamento do navegador enquanto estiver valido e deve funcionar entre instancias da Vercel.

O navegador nao pode ser autoridade sobre preco, subtotal, total, disponibilidade, nome de produto ou variante. A aplicacao precisa recalcular valores server-side e preservar itens que fiquem indisponiveis para revisao futura.

## Decisao

Persistir o carrinho anonimo no PostgreSQL usando:

```text
cookie opaco
  +
token aleatorio
  +
SHA-256 no banco
  +
carts/cart_items
```

O cookie `printlab_cart` guarda somente o token bruto opaco. O banco guarda somente `carts.token_hash`, calculado com `SHA-256(token)`.

`cart_items` guarda `product_id`, `variant_id` opcional e `quantity`. Preco unitario, subtotal e total nao sao persistidos em `cart_items`; sao recalculados em leitura a partir de `products.price_cents` e `product_variants.price_cents`.

Carrinhos expiram apos 30 dias. Mutacoes bem-sucedidas renovam `expires_at` e o cookie para `agora + 30 dias`.

## Alternativas consideradas

- `localStorage`: rejeitado porque deixaria dados do carrinho e possivelmente regras de negocio no navegador.
- Carrinho completo em cookie: rejeitado porque aumentaria superficie de manipulacao e tamanho de cookie, alem de dificultar revalidacao server-side.
- Sessao em memoria: rejeitada porque nao funciona de forma confiavel entre instancias Vercel nem sobrevive a restart.
- Redis: adiado por adicionar dependencia operacional antes de necessidade real.
- Conta obrigatoria: rejeitada para preservar checkout sem login e reduzir atrito.

## Consequencias

- Visitante consegue usar carrinho sem criar conta.
- Carrinho funciona entre instancias porque a fonte de verdade e PostgreSQL.
- O banco continua sendo autoridade sobre produtos, variantes, preco e disponibilidade.
- Cookie atua como credencial anonima do carrinho e deve ser tratado com cuidado.
- Uma leitura acidental do banco nao revela diretamente tokens utilizaveis.
- Carrinhos expirados exigirao limpeza fisica futura.
- Checkout futuro deve revalidar itens, disponibilidade, frete e totais antes de criar pedido.

## Referencias

- [Carrinho](../product/cart.md)
- [Schema](../database/schema.md)
- [Seguranca](../architecture/security.md)
- [Roadmap](../plans/roadmap.md)
