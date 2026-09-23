# SuperFrete

Status: IMPLEMENTADO E VALIDADO EM SANDBOX PARA COTACAO.

A Fase 8 implementa cotacao server-side com SuperFrete para o checkout da PrintLab. A validacao Sandbox real foi confirmada manualmente pelo responsavel do projeto antes da Fase 9, sem registrar secrets, CEPs, tokens ou dados pessoais.

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

O `Error()` nao inclui token, corpo bruto da SuperFrete, payload de cotacao, CEP ou dados do cliente. Rejeicoes por servico retornadas com HTTP 200 e `has_error=true` sao classificadas sem expor a mensagem externa bruta:

- `service_unavailable`;
- `invalid_package`;
- `invalid_postal_code`;
- `unsupported_dimensions`;
- `unknown_service_error`.

## Estrategia de cotacao

A PrintLab e a fonte de verdade da embalagem fisica. A SuperFrete e responsavel por preco, prazo e disponibilidade dos servicos.

Fluxo aprovado do checkout:

```text
Produtos do carrinho
  -> PrintLab tenta selecionar a menor caixa fisica real compativel
  -> se nao houver encaixe, monta embalagem estimada conservadora
  -> peso final = produtos + embalagem real ou estimada
  -> SuperFrete calculator com package final
  -> cotacao final exibida ao cliente
```

Antes da chamada externa, o backend:

- expande cada `QuoteProduct` por quantidade;
- testa rotacoes axis-aligned dos cuboides;
- usa um empacotador conservador de pontos extremos, sem sobreposicao;
- aceita falso negativo conservador, mas nao aceita falso positivo;
- escolhe caixas por menor volume interno compativel, com desempates deterministicos por `sort_order`, volume externo, peso da embalagem, nome, slug e ID;
- quando nenhuma caixa real comporta os itens, calcula uma embalagem estimada conservadora por unidade, soma comprimentos, aplica margem externa, arredonda dimensoes para 10 mm e peso para 50 g;
- calcula peso final em gramas com `TotalPackageWeightG` para caixa real ou com a regra conservadora de fallback.

A chamada comercial da SuperFrete:

- envia `from.postal_code`;
- envia `to.postal_code`;
- envia `services`;
- envia `options` com adicionais desabilitados;
- envia somente `package`;
- usa dimensoes externas da caixa fisica selecionada ou da embalagem estimada de fallback;
- usa peso final em kg;
- nao envia `products` simultaneamente.

O cliente HTTP ainda suporta payload `products` para compatibilidade e testes isolados de contrato da API, mas o checkout real nao usa essa etapa.

Logs seguros do fluxo:

- antes da selecao: `shipping packaging request product_lines=N units=N candidate_boxes=N`;
- por linha logistica: quantidade, peso em gramas e dimensoes em milimetros;
- por caixa candidata: indice, `fits`, dimensoes internas;
- apos selecionar: `shipping packaging selected source=real_box ...` com dimensoes internas/externas e peso de embalagem, ou `shipping packaging selected source=fallback ...` com pacote estimado;
- antes da chamada externa: `shipping quote request stage=final ...`;
- apos a chamada externa: `shipping quote response stage=final final_valid_quotes=N`.

Somente o preco da chamada com `package` real e apresentado ao cliente.

A modalidade `pickup` (retirada no local) nao pertence ao fluxo SuperFrete. Ela e selecionada e persistida pelo backend com `delivery_method=pickup`, preco zero e campos de servico vazios, sem chamada externa, sem perfil logistico e sem caixa.

## Embalagem

Produto cru, perfil logistico protegido e caixa fisica sao conceitos diferentes.

O perfil logistico autoritativo em `products` representa uma unidade preparada para acondicionamento:

- `shipping_weight_g`;
- `shipping_height_mm`;
- `shipping_width_mm`;
- `shipping_length_mm`.

Internamente, peso usa gramas inteiras e dimensoes usam milimetros inteiros. Conversao para kg/cm acontece somente na borda HTTP da SuperFrete.

As colunas equivalentes em `product_variants` permanecem no schema como legado inerte e nao sao lidas pela cotacao. O contrato com a SuperFrete e o fluxo de duas chamadas permanecem inalterados.

`shipping_boxes` representa caixas fisicas reais:

- `internal_*`: espaco util usado para validar encaixe.
- `external_*`: dimensoes enviadas a transportadora.
- `packaging_weight_g`: peso da caixa/protecao/enchimento padrao.

A escolha da caixa permite rotacao por comparacao dos tres eixos ordenados. Volume sozinho nunca determina encaixe.

## Erros e limites

Servicos com erro especifico na resposta da SuperFrete sao ignorados quando outros servicos validos existem. Se nenhuma cotacao valida existir, o checkout mostra indisponibilidade generica.

Quando a resposta HTTP 200 contem servico com `has_error=true`, a aplicacao ignora esse servico no resultado publico e registra somente `service_code`, `service_name` quando disponivel e uma categoria segura. A mensagem bruta externa do campo `error` nao deve ser logada. Cotacoes descartadas por erro do servico ou preco invalido registram apenas codigo, nome seguro do servico e motivo fixo. Quando a cotacao e valida, mas o pacote vem ausente ou invalido, o log usa `shipping quote package unavailable` com os mesmos campos seguros.

Falhas de frete sao classificadas internamente por estagio e motivo seguro:

- `shipping_not_configured`;
- `final_request_failed`;
- `final_no_valid_quotes`.

A ausencia de caixas ativas e o caso em que caixas existem mas nenhuma comporta os itens acionam fallback conservador, desde que todos os produtos tenham perfil logistico valido. Esses caminhos usam logs de embalagem (`reason=no_active_boxes` ou `reason=no_fitting_box`) e so viram indisponibilidade se o fallback nao puder ser construido ou se a cotacao final nao retornar servico valido.

O runtime emite no startup somente `superfrete configured environment=<env> services=<lista>` ou `superfrete not configured`. Esse log nao inclui token, CEP de origem nem e-mail operacional.

Mensagens publicas continuam genericas e nao devem revelar CEP, CPF, telefone, e-mail, endereco, token, corpo externo ou detalhes de configuracao.

Nao ha nesta fase:

- multi-volume;
- etiqueta/postagem;
- rastreio;
- seguro/valor declarado;
- mao propria;
- aviso de recebimento;
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

## Validacao Sandbox

A validacao Sandbox real confirmou:

- cotacao com token Sandbox real configurado fora do repositorio;
- chamada de planejamento com `products` retornando pacote;
- rejeicao correta de caixa que nao comportava o pacote;
- cotacao final com `package` apos uso de caixa compativel;
- modalidades apresentadas ao usuario;
- modalidade selecionada;
- persistencia em `cart_shipping_selections`.

Nao criar dados ficticios em migration nem registrar secrets na documentacao. A validacao de Sandbox nao inclui compra de etiqueta, postagem, rastreio ou pagamento.

## Diagnostico temporario de encaixe

`no_fitting_box` agora aciona uma embalagem estimada conservadora quando nenhuma caixa real comporta os itens. O registro e limitado, sem PII ou identificadores persistentes. A validacao real em producao deve ser feita manualmente apos deploy. Consulte o [runbook do incidente](../operations/shipping-packaging-diagnostic.md) para interpretar diagnosticos, se o erro voltar.
