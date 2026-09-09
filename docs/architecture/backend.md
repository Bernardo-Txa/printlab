# Backend

Status: fundacao HTTP e banco IMPLEMENTADA; funcionalidades de negocio PLANEJADAS.

## Responsabilidade

O backend Go sera a camada autoritativa da aplicacao. Ele recebera requisicoes HTTP, validara entradas, aplicara regras de negocio, acessara o PostgreSQL e renderizara respostas server-side.

Nesta fase, o backend implementa:

- `GET /` para homepage server-side.
- `GET /health` para liveness.
- `GET /ready` para readiness de banco.
- `/static/...` para assets embutidos.
- `internal/config` para ler configuracao.
- `internal/database` para criar `pgxpool.Pool`.

## Limites

- Nao ha schema de negocio.
- Nao ha produtos, carrinho, checkout, pedidos, pagamentos ou admin.
- Nao ha integracoes externas.
- A homepage ainda nao depende obrigatoriamente do PostgreSQL.

## Decisoes

- Usar `net/http` como base HTTP.
- Usar `pgx/v5` e `pgxpool` para PostgreSQL.
- Separar areas futuras em `internal/`, sem codigo artificial.
- Centralizar regras financeiras no backend.
- Usar `DATABASE_URL` como unica fonte de verdade da conexao PostgreSQL.
- Usar `DB_MAX_CONNS` com default `4` para limitar conexoes por instancia.
- Configurar `pgx.QueryExecModeExec` para compatibilidade com Supabase Transaction Pooler.

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

## Praticas proibidas

- Confiar em valores financeiros vindos do navegador.
- Adicionar dependencia sem justificativa.
- Criar interfaces prematuras sem multiplos consumidores ou necessidade clara de teste.
- Implementar integracoes reais sem plano aprovado.
- Colocar secrets no codigo, testes ou documentacao.
- Executar migrations no startup da aplicacao.
- Logar `DATABASE_URL`, senha, token ou connection string.
