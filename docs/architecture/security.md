# Seguranca

Status: diretrizes obrigatorias aprovadas; catalogo publico com variantes, carrinho e dados de checkout IMPLEMENTADOS.

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

## Catalogo publico

- Produtos publicos exigem `products.is_active = true`.
- Produto inativo retorna publicamente como inexistente.
- Slugs de produto e categoria sao validados antes de consulta ao banco.
- Slug invalido retorna resposta 404, sem revelar detalhes.
- Queries de catalogo usam parametros PostgreSQL.
- Erros de PostgreSQL nao sao retornados ao usuario.
- Preco-base e definido pelo backend a partir de `products.price_cents`.
- Preco efetivo de variante e definido pelo backend a partir de `product_variants.price_cents` ou fallback para `products.price_cents`.
- Variante inativa, inexistente, invalida ou pertencente a outro produto retorna publicamente como inexistente.
- Templates recebem preco formatado e nao fazem calculo financeiro.
- `product_images.storage_path` deve ser caminho relativo de bucket, nunca URL absoluta.
- URLs publicas de imagens sao montadas centralizadamente no backend a partir de `SUPABASE_URL` e do bucket `product-images`.
- O bucket `product-images` e publico para leitura de imagens de catalogo, mas nao existe policy publica de upload, update ou delete.
- `SUPABASE_SERVICE_ROLE_KEY` nao e usada pela aplicacao nesta fase.

## Carrinho anonimo

- Carrinho anonimo usa cookie opaco `printlab_cart`.
- O cookie e `HttpOnly`, `SameSite=Lax`, `Path=/`, host-only e `Secure` em producao.
- O token do cookie e aleatorio, gerado com `crypto/rand` com 32 bytes.
- O token bruto nao e persistido, logado, renderizado em HTML ou enviado em URL.
- O banco armazena somente `SHA-256(token)` em `carts.token_hash`.
- `cart_items` armazena somente produto, variante opcional e quantidade.
- Preco unitario, subtotal e total sao sempre recalculados server-side.
- Mutacoes de item usam escopo `cart_id + item_id`; nunca atualizam ou removem apenas por `item_id`.
- Mutacoes usam POST e validacao centralizada de `Origin`/`Referer`.
- Requests cross-site com origem conhecida e incompatibil devem ser rejeitados.
- Checkout futuro devera revalidar todos os itens antes de criar pedido.

## Dados pessoais de checkout

- A etapa `GET/POST /checkout/dados` coleta somente dados necessarios para compra, entrega e contato relacionado ao pedido.
- Dados de contato e endereco pertencem ao carrinho anonimo atual.
- Nao ha conta, senha, username, data de nascimento, genero, newsletter ou marketing consent nesta fase.
- CPF e necessario para documentacao futura de envio/DC-e, mas nao e identificador publico e nao possui indice ou unique.
- CPF e CEP sao armazenados como digitos ASCII normalizados.
- Telefone brasileiro e armazenado em formato canonico E.164.
- E-mail e normalizado com trim e lowercase para uso operacional atual.
- Contato e endereco sao persistidos em transacao para evitar estado parcial.
- Contato e endereco sao lidos em uma unica consulta SQL consistente; estado parcial anomalo nao e retornado como checkout valido.
- Respostas HTML de checkout que podem conter PII usam `Cache-Control: private, no-store`.
- Erros publicos devem ser genericos e nao conter CPF, e-mail completo, telefone, endereco, token de carrinho ou detalhes PostgreSQL.
- Logs nao devem registrar CPF, e-mail completo, telefone, endereco, token de carrinho, `DATABASE_URL` ou connection strings.
- Dados temporarios sao removidos por `ON DELETE CASCADE` quando o carrinho for removido.
- Limpeza programada de carrinhos expirados e PII associada e requisito obrigatorio antes do go-live comercial.

## Limites

- Nao ha autenticacao implementada.
- Nao ha autorizacao implementada.
- Nao ha webhooks implementados.
- Nao ha processamento de pagamento implementado.
- As tabelas de negocio implementadas cobrem catalogo, variantes, receita estimada de producao, imagens, carrinho e dados temporarios de checkout.
- `GET /ready` nao expoe detalhes internos do PostgreSQL.
- Nao ha upload de imagens, autenticacao administrativa ou escrita publica em Storage.

## Praticas recomendadas

- Validar entradas no servidor.
- Registrar eventos importantes sem expor dados sensiveis.
- Usar transacoes para alteracoes financeiras.
- Projetar idempotencia antes de processar webhooks.
- Revisar dependencias antes de adiciona-las.
- Usar `TEST_DATABASE_URL` para testes opcionais de integracao com banco, nunca `DATABASE_URL` de producao.
- Validar paths de Storage antes de montar URL publica de imagem.

## Praticas proibidas

- Confirmar pagamento por parametro de URL ou redirect.
- Salvar secrets em codigo, fixtures, logs, documentacao ou exemplos.
- Aceitar preco, desconto ou frete do cliente como valor final.
- Processar webhook sem validacao e protecao contra duplicidade.
- Executar migrations automaticamente no startup do servidor web.
- Usar Table Editor ou SQL Editor remoto como workflow normal de mudanca de schema.
- Criar policy publica de `INSERT`, `UPDATE` ou `DELETE` em `storage.objects` para imagens de produto.
- Armazenar URL externa em `product_images.storage_path`.
