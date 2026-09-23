# Diagnostico de embalagem: no_fitting_box

Status: instrumentacao implementada e testada; esta revisao nao declara nova validacao em producao. Sem deploy automatico por este documento.

## Semantica atual

`no_fitting_box` significa que o empacotador da PrintLab nao conseguiu colocar fisicamente todos os itens em nenhuma caixa ativa cadastrada. A SuperFrete nao participa mais da escolha de embalagem no checkout comercial.

A PrintLab:

- expande produtos por quantidade;
- testa rotacoes axis-aligned;
- tenta posicionar cuboides sem sobreposicao por pontos extremos deterministicos;
- escolhe a menor caixa fisica real compativel;
- envia para a SuperFrete somente o `package` final com dimensoes externas e peso total.

A SuperFrete retorna preco, prazo e disponibilidade de servicos. O payload `products` continua existindo no cliente HTTP apenas para compatibilidade/teste isolado da API, nao para o fluxo comercial.

## Instrumentacao segura

O fluxo registra:

- antes da selecao de caixa: `shipping packaging request product_lines=N units=N candidate_boxes=N`;
- por linha: quantidade, peso em gramas e dimensoes em milimetros;
- por caixa candidata: `shipping packaging candidate index=N fits=true/false internal_h_mm=... internal_w_mm=... internal_l_mm=...`;
- caixa escolhida: `shipping packaging selected box_index=N ...`;
- antes da cotacao: `shipping quote request stage=final package_weight_g=... package_h_mm=... package_w_mm=... package_l_mm=... services=N`;
- apos a cotacao: `shipping quote response stage=final final_valid_quotes=N`.

Somente no caminho de falha da selecao, um registro `shipping packaging diagnostic reason=no_fitting_box selection=real_box_packing data=...` contem:

- `product_lines` e `units`;
- `products`: indice da linha, quantidade, peso e dimensoes H/W/L em mm;
- `candidate_boxes`;
- `boxes`: indice, dimensoes internas H/W/L, eixos ordenados, validade e resultado de encaixe;
- `packing_algorithm`;
- contagens `products_omitted` e `boxes_omitted`, com limite de 32 entradas em cada lista.

Os indices sao posicionais e nao sao IDs de carrinho, pedido, produto ou caixa. Nao entram CEP, CPF, telefone, e-mail, endereco, token, nomes livres, IDs persistentes, corpo bruto da SuperFrete ou dados do cliente.

## Registro operacional

1. Apos deploy, validar manualmente em producao antes de declarar o incidente encerrado.
2. Se o erro voltar, reproduzir o carrinho afetado e consultar os logs seguros de embalagem e cotacao, sem compartilhar dados pessoais.
3. Conferir `products_omitted=0` e `boxes_omitted=0`, dimensoes dos produtos e `fits` das caixas.
4. Se nenhuma caixa comportar os itens, confirmar fisicamente as medidas cadastradas e decidir operacionalmente sobre embalagens reais. Nao alterar dados para forcar sucesso.
5. Se houver necessidade futura de suporte multi-volume, validar esse contrato separadamente antes de mudar o fluxo de caixa unica.
