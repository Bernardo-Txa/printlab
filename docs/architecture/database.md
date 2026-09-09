# Banco de dados

Status: fundacao PostgreSQL/Supabase IMPLEMENTADA; schema de negocio PLANEJADO.

## Responsabilidade

O banco armazenara dados persistentes de produtos, clientes, enderecos, carrinhos, pedidos, pagamentos, envios e informacoes operacionais aprovadas.

## Limites

- Nao ha tabelas criadas.
- Nao ha migrations funcionais.
- Ha workflow GitHub Actions para aplicar futuras migrations versionadas ao Supabase de desenvolvimento.
- Ha acesso PostgreSQL server-side com `pgx/v5` e `pgxpool`.
- A conexao depende de `DATABASE_URL` em runtime.
- Sem `DATABASE_URL`, a aplicacao inicia, mas `/ready` retorna HTTP 503.

## Decisoes

- Usar PostgreSQL.
- Hospedar o PostgreSQL no Supabase.
- Acessar o banco pelo backend Go usando `pgx/v5` e `pgxpool`.
- Nao usar Supabase Data API como interface primaria da aplicacao.
- Manter frontend sem acesso direto a tabelas sensiveis.
- Usar `supabase/migrations/` para migrations versionadas quando o schema for aprovado.
- Aplicar migrations remotas pelo GitHub Actions com `supabase db push --dry-run` antes de `supabase db push`.
- Usar `DATABASE_URL` como unica fonte de verdade da conexao PostgreSQL em runtime.
- Nao montar connection string manualmente no codigo.
- Configurar `pgx.QueryExecModeExec` como `DefaultQueryExecMode` para compatibilidade com Supabase Transaction Pooler.
- Manter migrations separadas do startup da aplicacao.

## Runtime de conexao

```text
Vercel Go
   |
   v
pgxpool
   |
   v
Supabase Transaction Pooler
   |
   v
PostgreSQL
```

Pool padrao por instancia:

- `MaxConns = 4`
- `MinConns = 0`

`DB_MAX_CONNS` permite ajuste explicito, mas valores invalidos ou menores que 1 sao erro de configuracao.

## Convencoes de schema futuras

- Nomes de tabelas, colunas, constraints e indices devem usar `snake_case`.
- Datas e horas persistidas devem usar `timestamptz`.
- Horarios devem ser tratados em UTC no banco.
- Colunas obrigatorias devem usar `NOT NULL`.
- Relacionamentos devem usar foreign keys explicitas.
- Invariantes importantes devem ser reforcadas por constraints no banco.
- Indices devem nascer de queries reais ou requisitos claros.
- Evitar `SELECT *` em codigo de producao.
- Migrations aplicadas em ambientes compartilhados nao devem ser alteradas silenciosamente.

## Dinheiro

Valores financeiros futuros nao devem usar `float32` ou `float64` como representacao canonica. A preferencia inicial e armazenar valores inteiros em centavos, por exemplo `R$ 39,90` como `3990`.

Nenhum preco e implementado nesta fase.

## IDs

Nao ha estrategia universal de IDs aprovada nesta fase. `uuid` e `bigint identity` serao avaliados conforme cada entidade. Nenhuma extensao PostgreSQL deve ser habilitada sem necessidade atual.

## RLS e Data API

Supabase Data API nao e a interface principal da PrintLab. O browser nao deve acessar tabelas sensiveis diretamente. O backend Go controla regras criticas.

RLS continua util como camada complementar futura, mas regras financeiras nunca devem depender somente de frontend ou RLS.

## Praticas recomendadas

- Toda mudanca de schema deve passar por migration versionada.
- Migrations aplicadas em ambientes compartilhados devem ser tratadas como imutaveis.
- Consultas devem ser claras, revisaveis e testaveis.
- Transacoes devem proteger criacao de pedidos e mudancas financeiras.
- Valores monetarios nao devem usar `float32` ou `float64` como representacao canonica.
- Producao deve ter separacao explicita e politica de aprovacao antes de receber migrations automaticas.

## Praticas proibidas

- Criar ou alterar tabelas manualmente em producao sem registro.
- Versionar credenciais de banco.
- Permitir que o navegador escreva diretamente em tabelas sensiveis.
- Gerar schema antes de aprovacao das entidades e regras.
- Executar reset remoto automatico.
- Aplicar seed automaticamente no workflow de migrations.
- Rodar `supabase db push`, DDL automatico ou migration runner no startup da aplicacao Go.
- Logar connection string, senha ou `DATABASE_URL`.
