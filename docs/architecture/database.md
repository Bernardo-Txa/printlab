# Banco de dados

Status: PLANEJADO para schema. Workflow de CI/CD para migrations Supabase configurado na Fase 3.

## Responsabilidade

O banco armazenara dados persistentes de produtos, clientes, enderecos, carrinhos, pedidos, pagamentos, envios e informacoes operacionais aprovadas.

## Limites

- Nao ha tabelas criadas.
- Nao ha migrations funcionais.
- Nao ha acesso `pgx` implementado.
- Nao ha conexao com Supabase nesta fase.
- Ha workflow GitHub Actions para aplicar futuras migrations versionadas ao Supabase de desenvolvimento.

## Decisoes

- Usar PostgreSQL.
- Hospedar o PostgreSQL no Supabase.
- Acessar o banco pelo backend Go usando `pgx`.
- Nao usar Supabase Data API como interface primaria da aplicacao.
- Manter frontend sem acesso direto a tabelas sensiveis.
- Usar `supabase/migrations/` para migrations versionadas quando o schema for aprovado.
- Aplicar migrations remotas pelo GitHub Actions com `supabase db push --dry-run` antes de `supabase db push`.

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
