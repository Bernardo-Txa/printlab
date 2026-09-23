# Checkout

Status: etapas de dados, frete, revisao, criacao de pedido, pagamento InfinitePay e webhook redundante IMPLEMENTADOS e validados.

O checkout transforma uma intencao de compra em pedido pendente de pagamento, com validacao server-side de produtos, endereco e frete. A etapa de pagamento usa checkout hospedado InfinitePay iniciado pelo backend.

## Comportamento implementado

- `GET /checkout/dados` renderiza formulario SSR de contato e entrega.
- `POST /checkout/dados` valida, normaliza e salva contato + endereco em transacao.
- A rota exige carrinho anonimo existente, nao vazio e sem itens indisponiveis.
- Se nao houver carrinho valido, a rota redireciona para `/carrinho`.
- Dados salvos sao pre-preenchidos para edicao posterior.
- Salvamento bem-sucedido renova a validade do carrinho e do cookie.
- Salvamento bem-sucedido redireciona para `/checkout/frete`.
- O formulario funciona sem JavaScript e aceita preenchimento manual de endereco.
- Quando JavaScript esta disponivel, a etapa de dados aplica mascaras progressivas de CPF, telefone brasileiro e CEP.
- Quando JavaScript esta disponivel, a etapa de dados consulta CEP via endpoint interno do backend e preenche rua, bairro, cidade e UF.
- Respostas HTML que podem conter PII usam `Cache-Control: private, no-store`.
- `GET /checkout/frete` renderiza a escolha de modalidade de entrega e, quando `delivery_method=shipping`, opcoes de frete SSR quando a cotacao esta disponivel.
- `POST /checkout/frete` recebe `delivery_method` e, para envio, somente `service_code`; o backend revalida a escolha e persiste a selecao.
- A etapa de entrega oferece `pickup` (retirada no local), uma modalidade gratuita persistida server-side. Ela nao consulta a SuperFrete nem depende de perfil logistico ou caixa.
- A selecao bem-sucedida redireciona para `/checkout/revisao`.
- `GET /checkout/revisao` revisa produtos, dados, entrega, frete e total sem recotar SuperFrete.
- `POST /checkout/revisao` cria pedido com snapshot imutavel e status `pending_payment`.
- `GET /pedido/{id}` exibe o pedido por UUID, sem CPF completo, endereco completo, telefone ou e-mail completo.
- `POST /pedido/{id}/pagar` inicia ou reutiliza checkout InfinitePay server-side.
- `GET /pagamento/retorno` confirma pagamento apenas por `payment_check` server-side.

Fluxo atual:

```text
Carrinho
  -> Dados
  -> Frete
  -> Revisao
  -> Pedido criado
  -> Pagamento InfinitePay
```

## Dados coletados

Contato:

- nome completo;
- e-mail;
- telefone/WhatsApp;
- CPF.

Endereco de entrega:

- CEP;
- rua/logradouro;
- numero;
- complemento opcional;
- bairro;
- cidade;
- UF;
- pais fixo `BR`.

Nao sao coletados senha, login, data de nascimento, genero, redes sociais, profissao, newsletter ou consentimento de marketing.

## Normalizacao e validacao

- `full_name`: trim, obrigatorio e limitado a 120 caracteres.
- `email`: trim, lowercase, validado com biblioteca padrao e limitado a 254 caracteres.
- `phone`: aceita formatos humanos brasileiros com `+55` opcional e persiste em formato canonico E.164, como `+5527999999999`.
- `cpf`: aceita valor com ou sem mascara, persiste 11 digitos ASCII e valida os dois digitos verificadores no backend.
- `postal_code`: aceita CEP com ou sem hifen e persiste 8 digitos ASCII.
- `number`: texto, para suportar `12`, `12A`, `120-B` e `s/n`.
- `state`: normalizado para uppercase e validado contra UFs brasileiras oficiais, incluindo DF.
- `country_code`: somente `BR` nesta fase.

As mascaras de CPF, telefone e CEP sao apenas melhoria visual no navegador. A fonte autoritativa continua sendo a normalizacao e validacao server-side no `POST /checkout/dados`.

## Consulta de CEP

O navegador nao chama ViaCEP diretamente. A pagina de dados usa `/api/cep/{cep}` como endpoint interno da aplicacao, e o backend consulta `https://viacep.com.br/ws/{cep}/json/` com timeout explicito.

O endpoint interno normaliza CEP para exatamente 8 digitos e responde apenas:

- `street`;
- `district`;
- `city`;
- `state`.

Campos como IBGE, DDD, SIAFI, GIA ou regiao do provedor externo nao sao repassados ao navegador.

Se o CEP nao for encontrado ou a consulta estiver indisponivel, a compra nao e bloqueada. A interface informa o erro de forma acessivel e o cliente pode preencher ou corrigir o endereco manualmente. Numero e complemento nunca sao preenchidos pela consulta.

## Privacidade e retencao

Os dados desta etapa sao PII e pertencem ao carrinho anonimo atual. A PrintLab nao cria entidade permanente de cliente nesta fase.

O backend nao deve logar CPF, e-mail completo, telefone, endereco ou token do carrinho. Erros publicos devem ser genericos e nao expor dados internos de PostgreSQL ou connection strings.

Contato e endereco sao lidos por uma unica consulta SQL com `JOIN`, para observar um snapshot consistente do PostgreSQL. Se houver estado parcial anomalo, como contato sem endereco ou endereco sem contato, o backend trata como dados ausentes e nao preenche o formulario com PII incompleta.

Quando o carrinho for removido, `ON DELETE CASCADE` remove `cart_customer_details` e `cart_shipping_addresses`. Carrinhos expirados e PII temporaria associada sao removidos pelo job diario `printlab_transient_data_cleanup`.

## Entrega

A etapa de entrega exige:

- carrinho anonimo existente;
- carrinho nao vazio;
- nenhum item indisponivel;
- contato e endereco ja salvos.

Quando o cliente escolhe receber em casa (`delivery_method=shipping`), o frete exige tambem:

- perfil logistico completo no produto de todos os itens;
- pelo menos uma caixa real ativa cadastrada;
- SuperFrete configurada para cotacao real.

Se nao houver carrinho valido, a rota redireciona para `/carrinho`. Se os dados ainda nao existirem, redireciona para `/checkout/dados`. Produto sem perfil logistico ou ausencia de caixa real retorna estado operacional de indisponibilidade sem estimar peso, dimensoes ou preco.

O backend calcula frete com a PrintLab como fonte de verdade da embalagem fisica: expande os itens por quantidade, testa rotacoes e posicoes sem sobreposicao nas caixas ativas, escolhe a menor caixa real compativel, soma `packaging_weight_g` ao peso dos produtos e faz uma unica cotacao SuperFrete com `package` usando medidas externas da caixa. A SuperFrete define preco, prazo e servicos disponiveis; ela nao escolhe a caixa da PrintLab.

Quando `delivery_method=pickup`, o backend ignora a cotacao de frete, salva preco zero e campos operacionais vazios. A revisao e o pedido exibem “Retirada no local” e “Grátis”; a modalidade `shipping` continua com as validacoes e o fingerprint existentes.

A retirada nao mostra endereco publico nesta versao. A mensagem exibida ao comprador e: “Após a confirmação do pedido, entraremos em contato para combinar o horário da retirada.”

Selecao de entrega expira em 30 minutos. No envio, ela tambem e invalidada por `input_hash` quando carrinho, quantidade, variante, perfil logistico do produto, CEP, servicos ou caixa mudam.

Falhas de frete mantem mensagem publica generica. Internamente, a aplicacao diferencia indisponibilidade de configuracao, ausencia de caixas, falha na chamada de planejamento, ausencia de pacote retornado, caixa inexistente para o pacote, falha na chamada final e ausencia de cotacoes finais validas, sem logar PII ou secrets.

## Revisao e pedido

`GET /checkout/revisao` exige carrinho valido e nao convertido, carrinho nao vazio, itens disponiveis, dados completos e selecao de entrega existente e nao expirada. Para envio, o `input_hash` do frete tambem precisa continuar valido; para retirada, o backend exige `delivery_method=pickup`, preco zero e campos de transportadora/servico vazios.

Se dados faltarem, redireciona para `/checkout/dados`. Se frete faltar, expirar ou divergir do carrinho atual, redireciona para `/checkout/frete`. Se o carrinho faltar, estiver vazio ou possuir item indisponivel, redireciona para `/carrinho`.

A revisao mostra CPF mascarado e usa `Cache-Control: private, no-store`.

Snapshots operacionais de produção e embalagem são preservados no pedido, mas não são apresentados na experiência pública do comprador.

Na experiencia publica de revisao e pedido, o comprador ve produto, variante, cor comercial quando houver, quantidade, valores, dados necessarios, modalidade de entrega, frete e total. Para envio, tambem ve endereco, servico de frete, transportadora e prazo. Para retirada, ve “Retirada no local”, “Grátis” e a orientacao de combinacao posterior de horario. SKU interno, tempo de impressao, consumo de filamento, componentes da receita, materiais, cores de producao, caixa fisica, peso e dimensoes do pacote permanecem fora da UI publica.

O POST de revisao recalcula produtos, disponibilidade, subtotal, frete e total no servidor. O campo oculto `review_fingerprint` serve somente para detectar revisao antiga entre GET e POST; nao e secret e nao determina preco.

Se a revisao mudou, nenhum pedido e criado e a pagina informa que os dados precisam ser revisados novamente.

Pedido criado copia contato, endereco, frete, itens e receita de producao para tabelas historicas. Depois do commit, `carts.converted_at` e preenchido, dados temporarios do carrinho sao removidos e o cookie `printlab_cart` expira.

## Pagamento

A etapa de pagamento e iniciada pela pagina publica do pedido.

`POST /pedido/{id}/pagar`:

- valida `Origin`/`Referer`;
- aceita somente UUID de pedido;
- exige pedido `pending_payment`;
- monta payload somente a partir do snapshot historico do pedido;
- compara o total interno do payload com `orders.total_cents`;
- cria ou reutiliza registro `order_payments`;
- redireciona 303 somente para checkout URL `https` em host autorizado da InfinitePay (`checkout.infinitepay.io` ou `checkout.infinitepay.com.br`).

Se `INFINITEPAY_HANDLE` nao estiver configurado, a pagina informa indisponibilidade segura e nao exibe botao falso.

`GET /pagamento/retorno` nao confia no redirect. Ele valida `order_nsu`, `transaction_nsu` e `slug`, chama `payment_check` no backend e so muda o pedido para `paid` quando a resposta confirmar `success=true`, `paid=true` e `amount` igual ao total congelado do pedido.

`POST /webhooks/infinitepay` recebe eventos do provider, mas tambem nao confia no payload como autoridade. Ele valida `order_nsu`, `transaction_nsu` e `invoice_slug`, chama `payment_check` no backend e aplica a mesma regra financeira. O `webhook_url` e gerado no backend a partir de `SITE_URL`; o navegador nunca controla esse valor.

`receipt_url` e `capture_method` vindos da query string sao ignorados como fonte de autoridade.

## Regras obrigatorias

- O frontend nao determina preco final.
- O frontend nao determina frete; ele envia somente `delivery_method` e, para envio, `service_code`.
- O frontend nao determina status de pedido.
- O frontend nao confirma pagamento.
- Pedido deve ser criado em transacao unica e idempotente por `source_cart_id`.
- Redirect de pagamento nao confirma pedido pago.
- Webhook InfinitePay confirma de forma redundante apenas apos `payment_check` server-side quando o comprador paga e nao retorna ao site.

## Limites

- Link real InfinitePay e pagamento real foram validados antes da Fase 11.
- Nao ha etiqueta, postagem, rastreio ou multi-volume.
- Recebimento real de webhook InfinitePay em producao foi validado na Fase 11.
- O Admin atual opera pedidos ja criados e nao altera carrinhos ou dados temporarios de checkout.

## Cor comercial — Fase 17.3.3

A cor opcional do item acompanha os resumos do checkout, a revisao e o pedido. A confirmacao valida novamente a disponibilidade e o vinculo ao produto; o fingerprint inclui ID, nome e slug da cor. `order_items.color_id` referencia o catalogo e `color_name`/`color_slug` congelam o nome comercial para exibicao publica e no Admin. Renomeacao ou exclusao posterior nao altera esse snapshot. Pedidos antigos continuam sem cor comercial. Variantes, receitas, precos, frete e pagamentos mantem suas regras.
