# Fase 9 — Revisao e Criacao de Pedidos

Status: CONCLUIDA.

## Objetivo

Implementar a primeira criacao real de pedidos da PrintLab a partir de carrinho, dados de cliente, endereco e frete validados, sem implementar pagamento.

## Pre-condicao registrada

A Fase 8 — Embalagem Real e Integracao de Frete SuperFrete foi validada manualmente pelo responsavel do projeto antes desta fase:

- cotacao Sandbox executada;
- chamada de planejamento retornou pacote;
- caixa pequena incompativel foi rejeitada corretamente;
- caixa compativel permitiu cotacao final;
- modalidades de frete foram apresentadas;
- uma modalidade foi selecionada;
- `cart_shipping_selections` persistiu a selecao.

Nenhum secret, CEP, token ou dado pessoal foi registrado.

## Escopo implementado

- Migration `create_orders` em `supabase/migrations/`.
- `carts.converted_at` para diferenciar carrinho ativo de carrinho convertido.
- Tabela `orders` com UUID, `order_number` sequencial, `source_cart_id`, status `pending_payment`, moeda `BRL`, subtotal, frete e total.
- Tabelas `order_customer_details`, `order_shipping_addresses`, `order_shipping_details`, `order_items` e `order_item_filaments` como snapshots historicos.
- RLS habilitado nas tabelas de pedido, sem policies publicas.
- Dominio `internal/orders` com service, repository PostgreSQL, fingerprint, mascaramento de CPF e formatacao de exibicao.
- `GET /checkout/revisao` com pre-condicoes, resumo SSR e `Cache-Control: private, no-store`.
- `POST /checkout/revisao` com validacao Origin/Referer, fingerprint de revisao, transacao, lock do carrinho, snapshot, conversao, limpeza temporaria e expiracao do cookie apos commit.
- `GET /pedido/{id}` por UUID, com status humano, itens, frete, total e sem PII completa.
- POST de frete passa a redirecionar para `/checkout/revisao`.

## Decisoes

- Pedido e snapshot imutavel do checkout.
- Carrinho continua mutavel ate a confirmacao do pedido.
- `order_number` e somente referencia humana; nao e autorizacao e nao e usado como rota publica.
- `review_fingerprint` nao e secret e nao e autoridade financeira; serve apenas para detectar revisao antiga.
- A revisao nao recota SuperFrete; apenas valida expiracao e `input_hash` da selecao persistida.
- `order_item_filaments` nao referencia `materials`, `colors` ou `variant_filaments`.
- Componentes de receita entram no snapshot mesmo quando material ou cor estiverem inativos.

## Fora de escopo

- InfinitePay.
- Processamento de pagamento.
- Webhooks de pagamento.
- Compra de etiqueta SuperFrete.
- Postagem, rastreio ou multi-volume.
- Painel administrativo.
- Status de producao/logistica.

## Definition of Done

- [x] Fase 8 marcada como concluida com validacao Sandbox manual registrada.
- [x] Migration de pedidos criada sem alterar migrations anteriores.
- [x] Pedido criado somente com dados recalculados no backend.
- [x] Criacao de pedido transacional e com lock do carrinho.
- [x] Idempotencia defendida por `source_cart_id`.
- [x] Carrinho convertido nao e reutilizado como carrinho ativo.
- [x] Dados temporarios sao removidos apos snapshot na mesma transacao.
- [x] Cookie `printlab_cart` expira somente apos commit.
- [x] Revisao e pedido usam `Cache-Control: private, no-store`.
- [x] Pagina de pedido nao exibe CPF completo, endereco completo, telefone ou e-mail completo.
- [x] Testes cobrem handlers, fingerprint, migration, snapshot de receita e pontos criticos do repository.
- [x] Documentacao de produto, banco, arquitetura, seguranca, integracoes e roadmap atualizada.

## Validacoes

Executar antes do commit:

- `templ generate`
- `npm run css:build`
- `gofmt -w .`
- `go mod tidy`
- `go test ./...`
- `go vet ./...`
- `go build ./...`
- `npx supabase --version`
- `npx supabase db reset` quando Supabase local estiver disponivel.

## Resultado

Fase 9 concluida em 2026-09-10.

Validacoes finais:

- `templ generate`: passou.
- `npm run css:build`: passou.
- `gofmt -w .`: passou.
- `go mod tidy`: passou.
- `go test ./...`: passou.
- `go vet ./...`: passou.
- `go build ./...`: passou.
- `npx supabase --version`: passou com CLI `2.117.0`.
- `npx supabase db reset`: nao executado porque o Supabase local nao estava iniciado (`supabase start is not running`).
