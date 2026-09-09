# Checkout

Status: etapas de dados e frete IMPLEMENTADAS; pedido e pagamento PLANEJADOS.

O checkout devera transformar uma intencao de compra em pedido, com validacao server-side de produtos, endereco, frete e pagamento. As etapas reais atuais sao dados de contato/endereco e frete vinculados ao carrinho anonimo.

## Comportamento implementado

- `GET /checkout/dados` renderiza formulario SSR de contato e entrega.
- `POST /checkout/dados` valida, normaliza e salva contato + endereco em transacao.
- A rota exige carrinho anonimo existente, nao vazio e sem itens indisponiveis.
- Se nao houver carrinho valido, a rota redireciona para `/carrinho`.
- Dados salvos sao pre-preenchidos para edicao posterior.
- Salvamento bem-sucedido renova a validade do carrinho e do cookie.
- Salvamento bem-sucedido redireciona para `/checkout/frete`.
- O formulario funciona sem JavaScript e sem busca externa de CEP.
- Respostas HTML que podem conter PII usam `Cache-Control: private, no-store`.
- `GET /checkout/frete` renderiza opcoes de frete SSR quando a cotacao esta disponivel.
- `POST /checkout/frete` recebe somente `service_code`, revalida a cotacao atual e persiste a selecao.
- A selecao bem-sucedida permanece em `/checkout/frete?selecionado=1` e informa que a revisao do pedido sera a proxima etapa.

Fluxo atual:

```text
Carrinho
  -> Dados
  -> Frete
  -> Revisao futura
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

## Snapshot futuro

Na Fase 9, o pedido devera copiar contato e endereco para snapshots definitivos de pedido. Isso evita depender do carrinho depois que a compra for criada.

O pedido tambem devera copiar ou revalidar a selecao de frete vigente antes de congelar valores definitivos.

## Comportamento planejado

- Recalcular subtotal e total no backend.
- Criar pedido antes ou durante o inicio do pagamento, conforme decisao futura.

## Regras obrigatorias

- O frontend nao determina preco final.
- O frontend nao determina frete; ele envia somente a escolha `service_code`.
- O frontend nao confirma pagamento.
- Redirect de pagamento nao confirma pedido pago.
- Webhook validado sera necessario para confirmacao server-side.

## Limites

- Nao ha pedido implementado nesta fase.
- Nao ha pagamento implementado nesta fase.
- Nao ha etiqueta, postagem, rastreio ou multi-volume.
- Nao ha contrato aprovado com InfinitePay.
- Nao ha schema aprovado para pedidos ou pagamentos.
