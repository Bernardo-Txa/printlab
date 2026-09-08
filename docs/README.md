# Documentacao da PrintLab

Este diretorio concentra a documentacao operacional, tecnica e de produto do projeto PrintLab.

## Indice

- Produto:
  - [Regras de negocio](product/business-rules.md)
  - [Catalogo](product/catalog.md)
  - [Carrinho](product/cart.md)
  - [Checkout](product/checkout.md)
  - [Pedidos](product/orders.md)
  - [Admin](product/admin.md)
- Arquitetura:
  - [Backend](architecture/backend.md)
  - [Frontend](architecture/frontend.md)
  - [Banco](architecture/database.md)
  - [Seguranca](architecture/security.md)
  - [Integracoes](architecture/integrations.md)
- Integracoes:
  - [Supabase](integrations/supabase.md)
  - [SuperFrete](integrations/superfrete.md)
  - [InfinitePay](integrations/infinitepay.md)
- Banco:
  - [Schema](database/schema.md)
  - [Migrations](database/migrations.md)
- Desenvolvimento:
  - [Setup](development/setup.md)
  - [Testes](development/testing.md)
  - [Deployment](development/deployment.md)
  - [Padroes de codigo](development/coding-standards.md)
- Decisoes:
  - [ADRs](decisions/README.md)
- Planos:
  - [Roadmap](plans/roadmap.md)

## Planos ativos e concluidos

`docs/plans/active/` sera usado para planos em execucao.

`docs/plans/completed/` sera usado para planos concluidos.

Planos movidos para `completed/` devem representar entregas verificadas, nao apenas ideias discutidas.

## Definition of Done global

Uma tarefa so pode ser considerada concluida quando:

- requisitos foram atendidos;
- codigo compila;
- `gofmt` foi aplicado quando houver Go alterado;
- testes aplicaveis passam;
- `go vet` aplicavel passa;
- nenhuma credencial foi adicionada;
- comportamento importante foi documentado;
- documentacao afetada foi atualizada;
- nenhuma funcionalidade fora do escopo foi adicionada;
- nenhuma regressao conhecida foi introduzida.
