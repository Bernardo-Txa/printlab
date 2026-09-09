# Fase 7 - Dados do cliente e endereco

Status: CONCLUIDA.

## Objetivo

Permitir que visitantes com carrinho valido informem e editem dados minimos de contato e endereco de entrega, sem criar conta e sem criar entidade permanente de cliente.

## Decisoes

- Dados pessoais ficam vinculados ao carrinho anonimo atual.
- Nao ha entidade global `customers` nesta fase.
- Cada carrinho pode ter no maximo um registro em `cart_customer_details` e um em `cart_shipping_addresses`.
- Contato e endereco sao persistidos como uma unidade transacional.
- Pedido futuro devera copiar esses dados para snapshots definitivos.
- Nenhum dado de marketing, newsletter, senha, conta, data de nascimento ou perfil e coletado.

## Schema

- `public.cart_customer_details`: `cart_id` como chave primaria e FK para `carts`, com `full_name`, `email`, `phone` e `cpf`.
- `public.cart_shipping_addresses`: `cart_id` como chave primaria e FK para `carts`, com CEP, logradouro, numero, complemento opcional, bairro, cidade, UF e pais.
- CPF e armazenado com 11 digitos ASCII.
- Telefone e armazenado em formato canonico brasileiro E.164.
- CEP e armazenado com 8 digitos ASCII.
- `country_code` aceita somente `BR` nesta fase.
- RLS habilitado sem policies publicas.

## Privacidade

- CPF, e-mail, telefone, endereco e token de carrinho nao devem ser logados.
- Erros publicos sao genericos e nao expoem detalhes de PostgreSQL, connection string ou PII.
- Browser nao acessa diretamente as tabelas sensiveis.
- Limpeza programada de carrinhos expirados e PII associada e pendencia obrigatoria antes do go-live comercial.

## Escopo

- `GET /checkout/dados`.
- `POST /checkout/dados`.
- Formulario SSR de contato e entrega.
- Resumo compacto do carrinho na etapa de dados.
- CTA real do carrinho para `/checkout/dados` quando todos os itens estiverem disponiveis.
- Validacoes brasileiras de CPF, telefone, CEP, UF e pais.
- Migration `create_cart_customer_details`.
- ADR de dados temporarios de checkout vinculados ao carrinho.

## Fora de escopo

- Frete.
- SuperFrete.
- Pagamento.
- InfinitePay.
- Pedido.
- Login ou conta.
- Entidade permanente de cliente.
- Newsletter ou marketing consent.
- Busca externa de CEP.
- Autocomplete externo.
- Checkout internacional.
- Limpeza automatica por cron/job.

## Definition of Done

- `templ generate` executado.
- `npm run css:build` executado.
- `gofmt -w .` executado.
- `go mod tidy` executado.
- `go test ./...` passa.
- `go vet ./...` passa.
- `go build ./...` passa.
- `npx supabase --version` passa.
- `npx supabase db reset` passa quando Supabase local estiver disponivel.
- Migration aplicada pelo GitHub Actions apos push.
- Vercel validada quando observavel.
- Nenhum secret ou PII real versionado.
- Fase 8 permanece planejada.

## Validacao local

- `templ generate`: passou.
- `npm run css:build`: passou.
- `gofmt -w .`: executado.
- `go mod tidy`: executado sem mudancas relevantes.
- `go test ./...`: passou.
- `go vet ./...`: passou.
- `go build ./...`: passou.
- `npx supabase --version`: passou com Supabase CLI 2.117.0.
- `npx supabase db reset`: passou com a migration `20260909203845_create_cart_customer_details.sql` aplicada no ambiente local.
- Teste opcional de transacao do repository com `TEST_DATABASE_URL`: passou no banco local.
- Validacao HTTP local: `GET /checkout/dados` sem carrinho valido redireciona para `/carrinho`.

## Validacao remota

- Migration deve ser aplicada ao Supabase DEV pelo workflow `Supabase Migrations` apos o push.
- Vercel deve ser validada apos o deploy ficar observavel.
