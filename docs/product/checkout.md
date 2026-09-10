# Checkout

Status: etapas de dados, frete, revisao e criacao de pedido IMPLEMENTADAS; pagamento PLANEJADO.

O checkout transforma uma intencao de compra em pedido pendente de pagamento, com validacao server-side de produtos, endereco e frete. Pagamento permanece planejado para a Fase 10.

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
- `GET /checkout/frete` renderiza opcoes de frete SSR quando a cotacao esta disponivel.
- `POST /checkout/frete` recebe somente `service_code`, revalida a cotacao atual e persiste a selecao.
- A selecao bem-sucedida redireciona para `/checkout/revisao`.
- `GET /checkout/revisao` revisa produtos, dados, entrega, frete e total sem recotar SuperFrete.
- `POST /checkout/revisao` cria pedido com snapshot imutavel e status `pending_payment`.
- `GET /pedido/{id}` exibe o pedido por UUID, sem CPF completo, endereco completo, telefone ou e-mail completo.

Fluxo atual:

```text
Carrinho
  -> Dados
  -> Frete
  -> Revisao
  -> Pedido criado
  -> Pagamento futuro
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

Quando o carrinho for removido, `ON DELETE CASCADE` remove `cart_customer_details` e `cart_shipping_addresses`. A limpeza programada de carrinhos expirados e PII associada e pendencia obrigatoria antes do go-live comercial.

## Frete

A etapa de frete exige:

- carrinho anonimo existente;
- carrinho nao vazio;
- nenhum item indisponivel;
- contato e endereco ja salvos;
- perfil logistico efetivo para todos os itens;
- pelo menos uma caixa real ativa cadastrada;
- SuperFrete configurada para cotacao real.

Se nao houver carrinho valido, a rota redireciona para `/carrinho`. Se os dados ainda nao existirem, redireciona para `/checkout/dados`. Produto sem perfil logistico ou ausencia de caixa real retorna estado operacional de indisponibilidade sem estimar peso, dimensoes ou preco.

O backend calcula frete em duas etapas: envia `products` para a SuperFrete obter pacote ideal, escolhe a menor caixa fisica real compativel usando medidas internas e rotacao, soma `packaging_weight_g` ao peso dos produtos e faz a cotacao final com `package` usando medidas externas da caixa. Apenas o resultado final e apresentado ao cliente.

Frete selecionado expira em 30 minutos e e invalidado por `input_hash` quando carrinho, quantidade, variante, perfil logistico, CEP, servicos ou caixa mudam.

Falhas de frete mantem mensagem publica generica. Internamente, a aplicacao diferencia indisponibilidade de configuracao, ausencia de caixas, falha na chamada de planejamento, ausencia de pacote retornado, caixa inexistente para o pacote, falha na chamada final e ausencia de cotacoes finais validas, sem logar PII ou secrets.

## Revisao e pedido

`GET /checkout/revisao` exige carrinho valido e nao convertido, carrinho nao vazio, itens disponiveis, dados completos e selecao de frete existente, nao expirada e com `input_hash` valido.

Se dados faltarem, redireciona para `/checkout/dados`. Se frete faltar, expirar ou divergir do carrinho atual, redireciona para `/checkout/frete`. Se o carrinho faltar, estiver vazio ou possuir item indisponivel, redireciona para `/carrinho`.

A revisao mostra CPF mascarado e usa `Cache-Control: private, no-store`.

O POST de revisao recalcula produtos, disponibilidade, subtotal, frete e total no servidor. O campo oculto `review_fingerprint` serve somente para detectar revisao antiga entre GET e POST; nao e secret e nao determina preco.

Se a revisao mudou, nenhum pedido e criado e a pagina informa que os dados precisam ser revisados novamente.

Pedido criado copia contato, endereco, frete, itens e receita de producao para tabelas historicas. Depois do commit, `carts.converted_at` e preenchido, dados temporarios do carrinho sao removidos e o cookie `printlab_cart` expira.

## Regras obrigatorias

- O frontend nao determina preco final.
- O frontend nao determina frete; ele envia somente a escolha `service_code`.
- O frontend nao determina status de pedido.
- O frontend nao confirma pagamento.
- Pedido deve ser criado em transacao unica e idempotente por `source_cart_id`.
- Redirect de pagamento nao confirma pedido pago.
- Webhook validado sera necessario para confirmacao server-side.

## Limites

- Nao ha pagamento implementado nesta fase.
- Nao ha etiqueta, postagem, rastreio ou multi-volume.
- Nao ha contrato aprovado com InfinitePay.
- Nao ha webhook de pagamento.
- Nao ha painel administrativo.
