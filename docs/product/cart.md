# Carrinho

Status: Fase 6 IMPLEMENTADA.

O carrinho permite que visitantes anonimos escolham produtos e quantidades antes do checkout, sem criar conta.

## Comportamento implementado

- `GET /carrinho` renderiza o carrinho server-side.
- `POST /carrinho/adicionar` adiciona ou incrementa item e redireciona com 303 para `/carrinho`.
- `POST /carrinho/itens/{id}/quantidade` atualiza quantidade e redireciona com 303.
- `POST /carrinho/itens/{id}/remover` remove item de forma idempotente na experiencia publica e redireciona com 303.
- O detalhe de produto envia formulario real de adicionar com `product_slug`, `variant_slug` opcional e `quantity`.
- Carrinho com itens disponiveis mostra CTA real para `/checkout/dados`.
- Carrinho com item indisponivel nao permite continuar para dados ate revisao/remocao.
- Produto com variantes ativas exige variante valida.
- Produto sem variantes ativas pode ser adicionado com `variant_id = null`.
- Quantidade valida: `1..99`.
- Carrinho vazio nao cria registro no banco apenas por visita.
- Carrinho expirado ou cookie desconhecido e tratado como vazio.

## Regras de seguranca

O carrinho no navegador nao e fonte autoritativa de preco, subtotal, desconto, frete ou total. O backend recalcula valores usando dados persistidos e regras aprovadas.

O navegador pode enviar:

- `product_slug`;
- `variant_slug`;
- `quantity`.

O navegador nunca deve enviar como fonte de verdade:

- `unit_price`;
- `subtotal`;
- `total`;
- nome do produto;
- disponibilidade.

## Persistencia anonima

O carrinho e persistido no PostgreSQL:

```text
Browser
   |
   | cookie printlab_cart
   v
token bruto
   |
   v
Go
   |
   | SHA-256
   v
carts.token_hash
   |
   v
PostgreSQL
```

O cookie contem um token opaco aleatorio gerado com `crypto/rand`, codificado com `base64.RawURLEncoding`. O token bruto nao e persistido no banco, nao aparece em HTML, nao aparece em URL e nao deve ser logado.

No banco, `carts.token_hash` armazena somente `SHA-256(token)`.

## Cookie

Cookie:

- nome: `printlab_cart`;
- `HttpOnly = true`;
- `SameSite = Lax`;
- `Path = /`;
- `Secure = true` em producao;
- sem `Domain` explicito.

A validade inicial e de 30 dias.

## Expiracao

Carrinhos anonimos expiram apos 30 dias.

Ao modificar o carrinho com sucesso, o backend renova `expires_at` para `agora + 30 dias` e renova o cookie.

Nao ha job de limpeza nesta fase. Remocao fisica de carrinhos expirados podera ser implementada futuramente.

## Precos e subtotais

`cart_items` nao persiste preco unitario.

Ao abrir o carrinho, o preco atual e recalculado:

```text
product_variants.price_cents != null -> preco da variante
caso contrario -> products.price_cents
```

Se o preco mudar enquanto o item estiver no carrinho, o carrinho exibe o preco atual. O checkout futuro devera revalidar tudo novamente antes de criar pedido.

Subtotal da linha:

```text
preco efetivo atual * quantity
```

Subtotal do carrinho:

```text
soma das linhas disponiveis
```

Valores usam `int64` em centavos. Se um overflow for detectado, o backend nao retorna total incorreto.

## Disponibilidade

Um item disponivel exige:

- `products.is_active = true`;
- quando `variant_id != null`, `product_variants.is_active = true`;
- quando `variant_id != null`, a variante pertence ao produto;
- quando `variant_id = null`, o produto nao possui variantes ativas no momento.

Se produto ou variante forem desativados depois da adicao, a linha continua visivel como indisponivel. Itens indisponiveis podem ser removidos, nao entram no subtotal e nao devem seguir para checkout futuro.

Se um produto que estava no carrinho sem variante passar a ter variantes ativas, a linha fica indisponivel. O sistema nao escolhe automaticamente outra variante pelo cliente.

## Protecao cross-site

Mutacoes do carrinho usam POST, cookie `SameSite=Lax` e validacao centralizada de `Origin`/`Referer`.

Quando `Origin` esta presente, a origem precisa bater com o host da request ou com `SITE_URL`. Quando `Origin` esta ausente e `Referer` esta presente, o `Referer` e usado como fallback. Requests sem ambos sao aceitos para preservar compatibilidade com navegadores/proxies, apoiados pelo `SameSite=Lax`.

Checkout e autenticacao poderao exigir protecao CSRF mais forte em fases futuras.

## Limites

- Nao ha login.
- Nao ha frete.
- Nao ha pedido.
- Nao ha pagamento.
- Nao ha cupom ou desconto.
- Nao ha estoque ou reserva.
- Nao ha contador global no header.
- Nao ha HTMX ou JavaScript obrigatorio.
