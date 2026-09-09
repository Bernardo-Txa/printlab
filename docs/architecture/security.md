# Seguranca

Status: diretrizes obrigatorias aprovadas; implementacoes especificas ainda PLANEJADAS.

## Responsabilidade

Seguranca deve orientar arquitetura, codigo, banco, integracoes e operacao. As regras abaixo sao obrigatorias para qualquer fase futura.

## Regras financeiras

Nunca confiar em dados financeiros recebidos do navegador.

O frontend nunca deve determinar de maneira autoritativa:

- preco do produto;
- desconto;
- subtotal;
- total;
- valor de frete;
- status de pagamento;
- status de pedido.

Antes de finalizar uma compra, o backend devera futuramente:

1. receber IDs e quantidades;
2. buscar produtos e precos no banco;
3. validar disponibilidade;
4. recalcular subtotal;
5. validar frete;
6. calcular total;
7. criar o pedido.

## Pagamentos

Pagamento somente podera ser considerado confirmado apos validacao server-side.

Redirect do navegador apos pagamento nunca devera ser considerado prova suficiente de pagamento.

Webhooks deverao futuramente possuir:

- validacao;
- idempotencia;
- protecao contra processamento duplicado;
- logs adequados.

## Secrets

Nunca armazenar dentro do Git:

- senha;
- API key;
- token;
- database password;
- service role key;
- credencial de producao.

Use environment variables para configuracoes sensiveis. `.env.example` deve conter apenas nomes de variaveis e comentarios, sem valores reais.

`DATABASE_URL` e secret e deve ser configurada apenas em `.env` local ignorado pelo Git ou em secrets do ambiente de hosting. A aplicacao nao deve imprimir `DATABASE_URL`, senha, host privado, project ref ou connection string em logs, respostas HTTP ou mensagens publicas de erro.

`SUPABASE_SERVICE_ROLE_KEY` nao e usada para conexao PostgreSQL da aplicacao.

## Limites

- Nao ha autenticacao implementada.
- Nao ha autorizacao implementada.
- Nao ha webhooks implementados.
- Nao ha processamento de pagamento implementado.
- Nao ha tabelas de negocio implementadas.
- `GET /ready` nao expoe detalhes internos do PostgreSQL.

## Praticas recomendadas

- Validar entradas no servidor.
- Registrar eventos importantes sem expor dados sensiveis.
- Usar transacoes para alteracoes financeiras.
- Projetar idempotencia antes de processar webhooks.
- Revisar dependencias antes de adiciona-las.
- Usar `TEST_DATABASE_URL` para testes opcionais de integracao com banco, nunca `DATABASE_URL` de producao.

## Praticas proibidas

- Confirmar pagamento por parametro de URL ou redirect.
- Salvar secrets em codigo, fixtures, logs, documentacao ou exemplos.
- Aceitar preco, desconto ou frete do cliente como valor final.
- Processar webhook sem validacao e protecao contra duplicidade.
- Executar migrations automaticamente no startup do servidor web.
- Usar Table Editor ou SQL Editor remoto como workflow normal de mudanca de schema.
