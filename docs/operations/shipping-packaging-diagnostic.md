# Diagnostico de embalagem: no_fitting_box

Status: instrumentacao temporaria implementada e testada; causa dimensional do incidente de producao ainda nao confirmada. Sem deploy automatico.

## Evidencia e limites

O incidente informado apresentou `shipping package planning packages=2 dimension_variants=1` e `shipping quote unavailable stage=packaging reason=no_fitting_box`.

Esse log nao armazena dimensoes. `packages` conta cotacoes com `Package != nil`, nao volumes fisicos de um envio. `mapSuperFreteQuotes` conserva somente `packages[0]` de cada modalidade, e `firstReturnedPackage` escolhe o primeiro pacote mapeado. Assim, duas modalidades com dimensoes iguais geram exatamente esse log. Ele nao prova que houve dois volumes em uma modalidade, nem permite recuperar as dimensoes descartadas. Esse comportamento foi preservado; suporte multi-volume permanece fora do fluxo atual.

A conexao Vercel consultada nao listou projetos acessiveis e nao permitiu obter o deployment do incidente. Sem dimensoes registradas ou reproducao identificada, nao e possivel classificar a causa como bug ou cadastro insuficiente. Os valores sinteticos dos testes nao sao medidas do incidente.

Uma leitura somente das caixas ativas do projeto Supabase PrintLab encontrou, em ordem H/W/L:

| Posicao na consulta | Internas (mm) | Externas (mm) | Eixos internos ordenados (mm) |
| --- | --- | --- | --- |
| 0 | 80 x 250 x 250 | 80 x 250 x 250 | 80, 250, 250 |
| 1 | 100 x 200 x 200 | 100 x 200 x 200 | 100, 200, 200 |
| 2 | 150 x 200 x 20 | 150 x 200 x 20 | 20, 150, 200 |

Essa leitura retrata o cadastro no momento da investigacao; nao comprova quais registros o deployment recebeu no instante do erro. Nenhum registro foi alterado. Em particular, o valor 20 mm foi mantido sem assumir erro de digitacao.

## Semantica verificada

A [referencia oficial de cotacao SuperFrete](https://superfrete.readme.io/reference/cotacao-de-frete) define dimensoes em centimetros e descreve o retorno de `products` como caixa ideal para acomodar os itens. O exemplo de resposta organiza `packages` dentro de cada servico. A documentacao consultada nao especifica detalhadamente a semantica de varios volumes por modalidade ou eventual folga/ajuste dimensional aplicado ao planejamento.

O sistema converte as dimensoes retornadas para milimetros, arredondando para cima; compara-as ao espaco **interno** de caixas ativas, com rotacao por ordenacao dos eixos. A cotacao final usa dimensoes **externas** da caixa escolhida e peso logistico dos itens mais embalagem. Nao foi comprovado erro de conversao, rotacao ou selecao. Nao substituir internas por externas, somar pacotes entre modalidades ou relaxar medidas.

## Instrumentacao temporaria

Somente no caminho de falha da selecao, um registro `shipping packaging diagnostic reason=no_fitting_box selection=first_returned_package data=...` contem:

- `selected_hwl_mm`, `selected_sorted_mm`, `selected_valid`: medidas efetivamente passadas a selecao;
- `quotes_with_package`: quantidade de cotacoes mapeadas com pacote;
- `planning`: `quote_index` e `dimensions_hwl_mm`, um pacote mapeado por modalidade;
- `candidate_boxes`: quantidade total de caixas consideradas;
- `boxes`: `box_index`, `internal_hwl_mm`, `internal_sorted_mm`, validade, encaixe e `deficit_sorted_mm`;
- contagens `planning_omitted` e `boxes_omitted`: limite de 32 entradas em cada lista para conter tamanho dos logs.

Os indices sao posicionais, com base zero; caixas seguem a ordem do repository (`sort_order`, nome, ID). Nao sao IDs de carrinho/pedido ou nomes livres. Um unico registro mantem pacote e caixas juntos em requests concorrentes. Somente numeros, booleanos e chaves fixas entram no JSON; nao ha CEP, PII, credenciais, nomes, IDs persistentes, corpo bruto ou dados do cliente.

`deficit_sorted_mm` corresponde ao menor, medio e maior eixo ordenado: `max(0, pacote - caixa)`. Nao corresponde necessariamente a altura/largura/comprimento original. Qualquer deficit positivo identifica um gargalo que impede encaixe, mesmo com rotacao. Dimensoes invalidas nao geram deficits.

Exemplo **sintetico de teste**, nao de producao: pacote 200 x 300 x 400 mm contra internas 240 x 120 x 160 mm resulta em eixos da caixa 120/160/240 e deficits 80/140/160 mm. Nenhuma tolerancia e aplicada, e a cotacao final nao e chamada.

## Coleta operacional pendente

1. Publicar manualmente este commit quando autorizado; esta tarefa nao faz push em main nem deploy, para evitar disparo automatico.
2. Reproduzir o mesmo carrinho e consultar o registro completo `shipping packaging diagnostic`, sem compartilhar dados pessoais.
3. Conferir `planning_omitted=0` e `boxes_omitted=0`, dimensoes selecionadas e deficits de cada caixa.
4. Se nenhuma caixa comportar o pacote, confirmar fisicamente as medidas cadastradas e decidir operacionalmente sobre embalagens reais. Nao alterar dados para forcar sucesso.
5. Se houver varios volumes reais dentro de uma modalidade, validar esse contrato separadamente antes de mudar o fluxo de caixa unica.
6. Registrar a causa confirmada e retirar esta instrumentacao detalhada apos concluir o incidente; preservar logs genericos e testes de encaixe.
