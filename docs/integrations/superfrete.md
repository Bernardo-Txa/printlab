# SuperFrete

Status: IMPLEMENTACAO CONCLUIDA; validacao Sandbox real PENDENTE.

A Fase 8 implementa cotacao server-side com SuperFrete para o checkout da PrintLab. A conclusao funcional local nao significa que o Sandbox real foi validado: isso depende de token Sandbox, CEP de origem operacional, produto real com perfil logistico e caixa real cadastrada.

Documentacao oficial consultada:

- Primeiros passos: <https://superfrete.readme.io/reference/primeiros-passos>
- Cotacao de frete: <https://superfrete.readme.io/reference/cotacao-de-frete>

## Ambientes

O backend nao aceita base URL arbitraria por environment variable.

Mapeamento interno:

- `SUPERFRETE_ENV=sandbox` -> `https://sandbox.superfrete.com`
- `SUPERFRETE_ENV=production` -> `https://api.superfrete.com`

Testes automatizados podem injetar endpoint `httptest`, mas essa opcao nao existe na configuracao de runtime.

## Configuracao

Variaveis:

- `SUPERFRETE_ENV`: `sandbox` ou `production`.
- `SUPERFRETE_API_TOKEN`: secret da SuperFrete.
- `SUPERFRETE_ORIGIN_POSTAL_CODE`: CEP de origem da PrintLab, normalizado para 8 digitos.
- `SUPERFRETE_CONTACT_EMAIL`: e-mail operacional usado no `User-Agent`.
- `SUPERFRETE_SERVICES`: codigos de servico solicitados, separados por virgula.

Se nenhuma variavel SuperFrete estiver preenchida, a aplicacao inicia e a rota de frete mostra indisponibilidade segura. Se apenas parte da configuracao for preenchida ou valores forem invalidos, o startup falha para evitar configuracao ambigua.

Servicos conhecidos nesta fase:

- `1`: PAC.
- `2`: SEDEX.
- `17`: Mini Envios.
- `3`: Jadlog.
- `33`: J&T.

O codigo `31` nao e enviado para forcar Loggi; a documentacao oficial atual indica que a disponibilidade da Loggi depende da configuracao do token.

## Autenticacao e User-Agent

Requests usam:

- `Authorization: Bearer <token>`.
- `User-Agent: PrintLab/1.0 (<SUPERFRETE_CONTACT_EMAIL>)`.
- `Content-Type: application/json`.
- `Accept: application/json`.

O token nunca deve ser logado, versionado, armazenado no banco, renderizado no HTML, enviado ao navegador ou incluido em URL.

## Endpoint

O calculator e chamado por `POST /api/v0/calculator`.

O cliente HTTP usa `net/http`, timeout explicito de 8 segundos e contexto da request original. Nao ha retry automatico nesta fase.

Erros do cliente preservam classificacao segura para diagnostico operacional:

- `400`;
- `401`;
- `429`;
- `500`;
- `timeout`;
- `invalid_json`;
- `request_failed`;
- `response_read_failed`.

O `Error()` nao inclui token, corpo bruto da SuperFrete, payload de cotacao, CEP ou dados do cliente.

## Estrategia de cotacao

A PrintLab nao implementa bin packing 3D proprio nesta fase.

Fluxo aprovado:

```text
Produtos do carrinho
  -> SuperFrete calculator com products
  -> pacote ideal retornado pela SuperFrete
  -> menor caixa fisica real compativel em shipping_boxes
  -> SuperFrete calculator com package real
  -> cotacao final exibida ao cliente
```

Primeira chamada:

- envia `from.postal_code`;
- envia `to.postal_code`;
- envia `services`;
- envia `options` com adicionais desabilitados;
- envia `products`, um item por linha logistica do carrinho;
- usa peso em kg e dimensoes em cm somente no DTO externo.
- registra diagnostico seguro da quantidade de pacotes validos retornados e se diferentes modalidades retornaram dimensoes diferentes.

Segunda chamada:

- envia `package`;
- usa dimensoes externas da caixa fisica selecionada;
- usa peso final em kg, calculado por peso logistico dos produtos + `packaging_weight_g`;
- nao envia `products` simultaneamente.

Somente o preco da segunda chamada e apresentado ao cliente.

## Embalagem

Produto cru, perfil logistico protegido e caixa fisica sao conceitos diferentes.

O perfil logistico em `products` e `product_variants` representa uma unidade preparada para acondicionamento:

- `shipping_weight_g`;
- `shipping_height_mm`;
- `shipping_width_mm`;
- `shipping_length_mm`.

Internamente, peso usa gramas inteiras e dimensoes usam milimetros inteiros. Conversao para kg/cm acontece somente na borda HTTP da SuperFrete.

`shipping_boxes` representa caixas fisicas reais:

- `internal_*`: espaco util usado para validar encaixe.
- `external_*`: dimensoes enviadas a transportadora.
- `packaging_weight_g`: peso da caixa/protecao/enchimento padrao.

A escolha da caixa permite rotacao por comparacao dos tres eixos ordenados. Volume sozinho nunca determina encaixe.

## Erros e limites

Servicos com erro especifico na resposta da SuperFrete sao ignorados quando outros servicos validos existem. Se nenhuma cotacao valida existir, o checkout mostra indisponibilidade generica.

Quando a resposta HTTP 200 contem servico com `has_error=true`, a aplicacao ignora esse servico no resultado publico e registra somente `service_code`, `service_name` quando disponivel e categoria generica de indisponibilidade. A mensagem bruta externa do campo `error` nao deve ser logada.

Falhas de frete sao classificadas internamente por estagio e motivo seguro:

- `shipping_not_configured`;
- `no_active_boxes`;
- `planning_request_failed`;
- `planning_no_valid_quotes`;
- `planning_no_package`;
- `no_fitting_box`;
- `final_request_failed`;
- `final_no_valid_quotes`.

Mensagens publicas continuam genericas e nao devem revelar CEP, CPF, telefone, e-mail, endereco, token, corpo externo ou detalhes de configuracao.

Nao ha nesta fase:

- multi-volume;
- etiqueta/postagem;
- rastreio;
- seguro/valor declarado;
- mao propria;
- aviso de recebimento;
- criacao de pedido;
- pagamento.

Adicionais enviados no calculator:

- `own_hand = false`;
- `receipt = false`;
- `use_insurance_value = false`;
- `insurance_value = 0`.

## Validacao local

Sem configuracao SuperFrete, a aplicacao deve iniciar normalmente e manter homepage, catalogo, carrinho, `/health` e `/ready` conforme o banco configurado.

Com banco, carrinho real, dados de checkout, perfis logisticos, caixas reais e configuracao SuperFrete:

```sh
go run ./cmd/server
curl -i http://localhost:8080/checkout/frete
```

Sem carrinho valido, a rota redireciona para `/carrinho`. Sem dados de checkout, redireciona para `/checkout/dados`.

## Validacao Sandbox pendente

Para marcar a Fase 8 como concluida de ponta a ponta, ainda e necessario validar uma cotacao Sandbox real com:

- token Sandbox real;
- CEP de origem operacional da PrintLab;
- pelo menos um produto real com perfil logistico;
- pelo menos uma caixa fisica real cadastrada;
- carrinho real de desenvolvimento;
- endereco de entrega de teste controlado.

Nao criar dados ficticios em migration nem registrar secrets na documentacao.
