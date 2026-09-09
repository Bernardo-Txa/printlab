# ADR 0008: Selecao de embalagem fisica para cotacao de frete

Status: Aceita

Data: 2026-09-09

## Contexto

Produtos impressos em 3D possuem geometrias variadas e podem ser combinados no carrinho. Somar dimensoes diretamente ou comparar apenas volume gera conclusoes incorretas sobre encaixe.

A PrintLab tambem nao posta caixas virtuais: a cotacao final precisa refletir uma caixa fisica real cadastrada, com medidas externas e peso de embalagem/protecao.

Implementar bin packing 3D proprio nesta fase adicionaria complexidade e risco antes de existir necessidade operacional comprovada.

## Decisao

A cotacao de frete usa duas etapas:

```text
Produtos do carrinho
  -> SuperFrete calculator com products
  -> pacote ideal retornado pela SuperFrete
  -> menor caixa fisica real compativel
  -> SuperFrete calculator com package real
  -> cotacao final apresentada ao cliente
```

O pacote ideal retornado pela SuperFrete e convertido para milimetros de forma conservadora. A escolha de caixa usa dimensoes internas e permite rotacao por ordenacao dos tres eixos.

Entre caixas compativeis, a escolha e deterministica:

1. menor volume interno;
2. menor `packaging_weight_g`;
3. menor `sort_order`;
4. `name` ascendente;
5. `id` como desempate final.

A segunda chamada ao calculator usa as dimensoes externas da caixa real e o peso final: soma do peso logistico dos produtos com `packaging_weight_g`.

## Alternativas consideradas

- Somar dimensoes dos produtos: simples, mas geometricamente incorreto.
- Comparar somente volume: rejeitado porque volume suficiente nao garante encaixe em tres eixos.
- Hardcode de caixa por produto: nao atende carrinhos mistos e cria manutencao operacional fragil.
- Bin packing proprio: adiado por complexidade desnecessaria nesta fase.
- Aceitar a caixa virtual da SuperFrete como final: rejeitado porque pode nao corresponder a uma caixa fisica real da PrintLab.

## Consequencias

- Cada cotacao de frete pode fazer duas chamadas ao calculator.
- O preco apresentado reflete uma caixa fisica real cadastrada.
- A qualidade da cotacao depende de perfis logisticos e caixas reais bem cadastrados.
- Sem perfil logistico ou sem caixa compativel, o checkout mostra indisponibilidade em vez de inventar dados.
- Multi-volume permanece como melhoria futura.
