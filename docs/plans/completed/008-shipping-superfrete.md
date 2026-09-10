# Fase 8 — Embalagem Real e Integracao de Frete SuperFrete

Status: CONCLUIDA.

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
- Diagnosticos seguros de indisponibilidade de frete por estagio e motivo, sem PII ou secrets.

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
- [x] Fase 8.1 adicionou diagnosticos seguros para configuracao, caixas, planejamento, pacote retornado, encaixe em caixa real, chamada final e cotacoes finais vazias.

### B. Migration

- [x] Migration criada em `supabase/migrations/20260909220454_add_shipping_profiles_and_selections.sql`.
- [x] `products` e `product_variants` possuem shipping profile all-or-none.
- [x] `shipping_boxes` criada sem seed.
- [x] `cart_shipping_selections` criada com snapshot logistico, `input_hash` e validade.
- [x] RLS habilitado sem policies publicas.

### C. Sandbox SuperFrete real

- [x] Token Sandbox real configurado fora do repositorio.
- [x] CEP de origem operacional da PrintLab configurado fora do repositorio.
- [x] Produto real de desenvolvimento com perfil logistico cadastrado.
- [x] Caixa fisica real cadastrada em `shipping_boxes`.
- [x] Cotacao Sandbox real validada com primeira chamada `products`.
- [x] Pacote ideal retornado pela SuperFrete validado.
- [x] Mecanismo rejeitou corretamente caixa que nao comportava o pacote (`no_fitting_box`).
- [x] Caixa real compativel permitiu a cotacao final.
- [x] Pelo menos um servico valido retornou, foi apresentado e selecionado.
- [x] `cart_shipping_selections` recebeu a selecao persistida.
- [x] Nenhuma credencial exposta.

## Validacao manual

A validacao Sandbox real foi confirmada manualmente pelo responsavel do projeto antes da Fase 9:

- cotacao Sandbox executada;
- chamada de planejamento retornou pacote;
- caixa pequena incompativel foi rejeitada corretamente;
- caixa compativel permitiu cotacao final;
- modalidades de frete foram apresentadas;
- uma modalidade foi selecionada;
- `cart_shipping_selections` persistiu a selecao.

Valores de secrets, connection strings, CEPs, tokens ou dados pessoais nao foram registrados na documentacao.

## Decisoes

- A PrintLab nao implementa bin packing 3D proprio nesta fase.
- Volume isolado nao determina encaixe.
- Caixa fisica real e obrigatoria para cotacao final.
- O preco exibido vem somente da segunda chamada SuperFrete com `package`.
- Multi-volume fica adiado.
- Erros publicos de frete permanecem genericos; diagnosticos internos nao podem registrar CEP, CPF, telefone, e-mail, endereco, token ou corpo bruto externo.
