# Banco de dados

Status: PLANEJADO. Nenhum schema foi aprovado nesta fase.

## Responsabilidade

O banco armazenara dados persistentes de produtos, clientes, enderecos, carrinhos, pedidos, pagamentos, envios e informacoes operacionais aprovadas.

## Limites

- Nao ha tabelas criadas.
- Nao ha migrations funcionais.
- Nao ha acesso `pgx` implementado.
- Nao ha conexao com Supabase nesta fase.

## Decisoes

- Usar PostgreSQL.
- Hospedar o PostgreSQL no Supabase.
- Acessar o banco pelo backend Go usando `pgx`.
- Nao usar Supabase Data API como interface primaria da aplicacao.
- Manter frontend sem acesso direto a tabelas sensiveis.

## Praticas recomendadas

- Toda mudanca de schema deve passar por migration versionada.
- Migrations aplicadas em ambientes compartilhados devem ser tratadas como imutaveis.
- Consultas devem ser claras, revisaveis e testaveis.
- Transacoes devem proteger criacao de pedidos e mudancas financeiras.
- Valores monetarios nao devem usar `float32` ou `float64` como representacao canonica.

## Praticas proibidas

- Criar ou alterar tabelas manualmente em producao sem registro.
- Versionar credenciais de banco.
- Permitir que o navegador escreva diretamente em tabelas sensiveis.
- Gerar schema antes de aprovacao das entidades e regras.
