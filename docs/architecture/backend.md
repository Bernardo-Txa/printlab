# Backend

Status: fundacao HTTP, banco, catalogo, variantes e producao IMPLEMENTADOS; demais funcionalidades de negocio PLANEJADAS.

## Responsabilidade

O backend Go sera a camada autoritativa da aplicacao. Ele recebera requisicoes HTTP, validara entradas, aplicara regras de negocio, acessara o PostgreSQL e renderizara respostas server-side.

Nesta fase, o backend implementa:

- `GET /` para homepage server-side.
- `GET /produtos` para catalogo publico.
- `GET /produtos/{slug}` para detalhe publico de produto ativo.
- `GET /produtos/{slug}?variante=<slug>` para detalhe com variante selecionada por slug.
- `GET /health` para liveness.
- `GET /ready` para readiness de banco.
- `/static/...` para assets embutidos.
- `internal/config` para ler configuracao.
- `internal/database` para criar `pgxpool.Pool`.
- `internal/products` para modelos, service e repository PostgreSQL do catalogo, variantes, receita estimada e imagens.

## Limites

- O schema de negocio implementado cobre catalogo, variantes, receita estimada de producao e imagens.
- Nao ha carrinho, checkout, pedidos, pagamentos ou admin.
- Nao ha integracoes comerciais externas como frete ou pagamento.
- A homepage ainda nao depende obrigatoriamente do PostgreSQL.
- Nao ha upload de imagens pelo app.

## Decisoes

- Usar `net/http` como base HTTP.
- Usar `pgx/v5` e `pgxpool` para PostgreSQL.
- Separar areas futuras em `internal/`, sem codigo artificial.
- Centralizar regras financeiras no backend.
- Usar `DATABASE_URL` como unica fonte de verdade da conexao PostgreSQL.
- Usar `DB_MAX_CONNS` com default `4` para limitar conexoes por instancia.
- Configurar `pgx.QueryExecModeExec` para compatibilidade com Supabase Transaction Pooler.
- Usar `internal/products` como primeira vertical slice: handler HTTP, service, repository `pgxpool` e templates SSR.
- Expor produto publicamente por slug, nunca por UUID.
- Exibir publicamente apenas produtos ativos.
- Tratar produto inativo como inexistente.
- Representar preco-base como inteiro em centavos.
- Representar override de preco de variante como inteiro em centavos opcional.
- Calcular preco efetivo no service, usando `product_variants.price_cents` quando preenchido e `products.price_cents` como fallback.
- Representar peso estimado de filamento em miligramas como inteiro.
- Representar tempo estimado de maquina em minutos como inteiro.
- Construir URL publica de imagem em um helper de dominio a partir de `SUPABASE_URL`, bucket `product-images` e `storage_path`.

## Catalogo

`GET /produtos` lista produtos ativos e aceita filtro opcional `categoria=<slug>`. O filtro e validado antes da consulta ao banco e usa slug publico.

`GET /produtos/{slug}` valida o slug e busca apenas produto ativo. Slug invalido, produto inexistente e produto inativo retornam HTTP 404.

`GET /produtos/{slug}?variante=<slug>` valida o slug da variante antes de chamar o service. Variante invalida, inexistente, inativa ou pertencente a outro produto retorna HTTP 404.

Quando um produto possui variantes ativas, o service escolhe automaticamente a variante default ativa; se nao existir, usa a primeira variante ativa pela ordenacao publica. Produto sem variantes continua valido e usa o preco-base.

As consultas de catalogo evitam N+1 em Go. A listagem calcula o menor preco efetivo e busca imagem geral primaria em uma consulta. O detalhe carrega produto, variantes, receitas e imagens em consultas separadas e coesas.

Se o banco estiver indisponivel, rotas de catalogo retornam HTTP 503 com resposta generica, sem detalhes do PostgreSQL.

## Health e readiness

`GET /health` e liveness. Ele sempre responde HTTP 200 com body `ok` quando o processo HTTP esta vivo e nao consulta o PostgreSQL.

`GET /ready` e readiness. Ele faz `Ping` com timeout de 3 segundos quando `DATABASE_URL` esta configurada. Sem `DATABASE_URL`, retorna HTTP 503. Esse comportamento e temporario enquanto a homepage nao depende do banco.

## Praticas recomendadas

- Handlers pequenos e explicitos.
- Validacao de entrada antes de chamar regras de dominio.
- Erros tratados explicitamente.
- `context.Context` quando a operacao envolver I/O, banco, chamadas externas ou cancelamento.
- Testes deterministico para regras criticas.
- Pacotes coesos por responsabilidade.
- Fechar `pgxpool.Pool` no encerramento do processo.
- Usar parametros PostgreSQL para entradas externas.
- Listar colunas explicitamente em SQL.

## Praticas proibidas

- Confiar em valores financeiros vindos do navegador.
- Adicionar dependencia sem justificativa.
- Criar interfaces prematuras sem multiplos consumidores ou necessidade clara de teste.
- Implementar integracoes reais sem plano aprovado.
- Colocar secrets no codigo, testes ou documentacao.
- Executar migrations no startup da aplicacao.
- Logar `DATABASE_URL`, senha, token ou connection string.
- Usar `SELECT *`.
- Concatenar valores externos em SQL.
