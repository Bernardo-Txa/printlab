# Fase 8 — Embalagem Real e Integracao de Frete SuperFrete

Status: IMPLEMENTACAO CONCLUIDA; VALIDACAO SANDBOX SUPERFRETE PENDENTE.

## Objetivo

Implementar a primeira etapa real de frete da PrintLab:

- perfil logistico de produtos e variantes;
- caixas fisicas reais;
- escolha automatica da menor caixa compativel;
- cotacao server-side via SuperFrete;
- selecao de servico de frete pelo cliente.

## Escopo implementado

- Campos logisticos opcionais em `products` e `product_variants`.
- Constraints all-or-none e positivas para perfis logisticos.
- Tabela `shipping_boxes` para caixas fisicas reais, sem seed ficticio.
- Tabela `cart_shipping_selections` para selecao 1:1 por carrinho.
- RLS habilitado em tabelas de frete, sem policies publicas.
- Dominio `internal/shipping` para perfis, caixas, conversoes, hash, service e repository.
- Cliente SuperFrete com `net/http`, Bearer token, `User-Agent`, timeout e DTOs isolados.
- Cotacao em duas etapas: `products` para pacote ideal e `package` com caixa real para preco final.
- Rotas `GET /checkout/frete` e `POST /checkout/frete`.
- UI SSR sem JavaScript obrigatorio, com radios HTML.
- Revalidacao server-side no POST.
- `input_hash` para invalidar selecao por mudanca de carrinho, CEP, perfil, caixa ou servicos.
- Validade operacional de 30 minutos para cotacoes selecionadas.

## Fora de escopo

- Pedido.
- InfinitePay.
- Webhook de pagamento.
- Etiqueta/postagem.
- Rastreio.
- Multi-volume.
- Admin para cadastro de caixas/profiles.
- Seed de produtos ou caixas.

## Definition of Done

### A. Implementacao e testes

- [x] Perfis logisticos implementados em dominio e repository.
- [x] Escolha de menor caixa real compativel implementada com rotacao.
- [x] Conversoes g/kg e mm/cm centralizadas.
- [x] Conversao cm -> mm conservadora para resposta externa.
- [x] Parsing decimal seguro de preco para centavos.
- [x] Cliente SuperFrete implementado conforme documentacao oficial.
- [x] Rotas e templates de frete implementados sem JavaScript obrigatorio.
- [x] POST revalida cotacao atual e persiste preco server-side.
- [x] Testes automatizados cobrem embalagem, cliente, service, repository, migration e handlers.

### B. Migration

- [x] Migration criada em `supabase/migrations/20260909220454_add_shipping_profiles_and_selections.sql`.
- [x] `products` e `product_variants` possuem shipping profile all-or-none.
- [x] `shipping_boxes` criada sem seed.
- [x] `cart_shipping_selections` criada com snapshot logistico, `input_hash` e validade.
- [x] RLS habilitado sem policies publicas.

### C. Sandbox SuperFrete real

- [ ] Token Sandbox real configurado.
- [ ] CEP de origem operacional da PrintLab configurado.
- [ ] Produto real de desenvolvimento com perfil logistico cadastrado.
- [ ] Caixa fisica real cadastrada em `shipping_boxes`.
- [ ] Cotacao Sandbox real validada com primeira chamada `products`.
- [ ] Pacote ideal retornado pela SuperFrete validado.
- [ ] Menor caixa real compativel selecionada.
- [ ] Segunda chamada `package` validada.
- [ ] Pelo menos um servico valido retornado e apresentado.
- [ ] Nenhuma credencial exposta.

## Pendencias para concluir a fase de ponta a ponta

Para mover este plano para `docs/plans/completed/`, ainda falta validar uma cotacao Sandbox real com dados reais de desenvolvimento:

- `SUPERFRETE_ENV=sandbox`;
- `SUPERFRETE_API_TOKEN` real configurado como secret;
- `SUPERFRETE_ORIGIN_POSTAL_CODE` real da PrintLab;
- `SUPERFRETE_CONTACT_EMAIL` operacional;
- `SUPERFRETE_SERVICES` com servicos autorizados;
- ao menos um produto real com perfil logistico;
- ao menos uma caixa fisica real ativa cadastrada;
- carrinho real com dados de checkout.

Nao criar esses dados em migration e nao registrar secrets.

## Decisoes

- A PrintLab nao implementa bin packing 3D proprio nesta fase.
- Volume isolado nao determina encaixe.
- Caixa fisica real e obrigatoria para cotacao final.
- O preco exibido vem somente da segunda chamada SuperFrete com `package`.
- Multi-volume fica adiado.
