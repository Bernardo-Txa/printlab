# Fase 21 — Endereços salvos e checkout recorrente

Status: Planejada; executar somente após Fase 20/go-live.

## Objetivo

Permitir que clientes autenticados salvem múltiplos endereços, definam um endereço padrão e selecionem o endereço desejado durante o checkout, preservando checkout guest e snapshots históricos de pedidos.

Esta fase é futura. Este documento é somente planejamento e não implementa código, migration, schema, checkout ou Minha Conta.

## Princípios de arquitetura

- Supabase Auth UUID continua sendo a identidade oficial do cliente.
- Nunca associar endereço por e-mail.
- `customer_profiles` permanece responsável por dados pessoais:
  - nome;
  - telefone;
  - CPF.
- Endereço deixa futuramente de ser uma coleção de colunas conceitualmente pertencente ao perfil.
- A futura entidade separada proposta é `customer_addresses`.

Schema conceitual proposto, sem migration nesta tarefa:

```text
id uuid
auth_user_id uuid
label text
recipient_name text
postal_code text
street text
number text
complement text nullable
district text
city text
state text
country_code text
is_default boolean
created_at timestamptz
updated_at timestamptz
```

## Regras de negócio futuras

- Cliente autenticado pode ter múltiplos endereços.
- Limite inicial sugerido: 10 endereços ativos por conta.
- Labels sugeridos:
  - Casa;
  - Trabalho;
  - Principal;
  - texto personalizado.
- `label` deve ter limite e validação server-side.
- Exatamente zero ou um endereço padrão por usuário.
- Ao criar o primeiro endereço, ele pode virar padrão automaticamente.
- Usuário pode alterar qual endereço é padrão.
- Usuário pode editar endereço próprio.
- Usuário pode remover endereço próprio.
- Nunca permitir acesso a endereço de outro `auth_user_id`.

Comportamento de exclusão a definir explicitamente na implementação:

- remover endereço salvo não altera pedidos anteriores;
- se remover o padrão e houver outros endereços, a aplicação deve escolher e documentar a regra do novo padrão;
- não deixar a regra implícita ou dependente apenas da UI.

## Migração futura do perfil atual

Hoje `customer_profiles` contém campos de endereço. A migration futura deve ser segura e em etapas.

Plano proposto:

1. Para cada `customer_profile` que possua endereço válido, criar um registro correspondente em `customer_addresses`.
2. Usar label inicial `Principal`.
3. Marcar `is_default=true` para esse endereço inicial.
4. Preservar todos os dados existentes.
5. Tornar a migration idempotente e segura contra duplicação conforme o padrão do projeto.

Somente depois de a nova arquitetura estar validada em produção, avaliar remoção das colunas de endereço de `customer_profiles`.

Não planejar `DROP` de colunas na primeira migration. Preferir rollout em etapas:

- A. adicionar tabela;
- B. migrar/copiar dados;
- C. aplicação passa a ler nova tabela;
- D. validar produção;
- E. limpar schema em fase posterior, se realmente necessário.

## Minha Conta futura

Planejar uma seção `Meus endereços`, mobile-first, sem exibir `auth_user_id` na UI.

Exemplo conceitual:

```text
Casa                      Padrão
Rua X, 123
Vila Velha - ES

[Editar]

Trabalho
Av. Y, 900
Vitória - ES

[Editar] [Definir como padrão]

[+ Adicionar endereço]
```

Ações futuras:

- adicionar;
- editar;
- definir padrão;
- excluir.

## Checkout futuro

Para usuário autenticado com endereços salvos, a etapa Dados/Entrega deve permitir escolher o endereço.

Exemplo conceitual:

```text
Onde devemos entregar?

( ) Casa
    Rua X, 123
    Vila Velha - ES

( ) Trabalho
    Av. Y, 900
    Vitória - ES

( ) Usar outro endereço
```

O endereço padrão deve iniciar selecionado, mas o cliente pode alterar.

Selecionar endereço salvo não deve confiar em dados de endereço enviados pelo browser como fonte autoritativa. O browser deve enviar somente o identificador do endereço. O backend deve:

- validar sessão;
- buscar endereço por `address.id` e `auth_user_id = profile.ID`;
- usar os dados recuperados server-side.

## Usar outro endereço

Preservar opção de preencher um endereço apenas para aquela compra.

Não obrigar o cliente a salvar todo endereço digitado.

Pode futuramente existir a opção explícita:

```text
[ ] Salvar este endereço na minha conta
```

Não salvar silenciosamente.

## Guest checkout

Checkout sem conta deve permanecer funcionando.

Não exigir autenticação para comprar.

Guest continua preenchendo endereço normalmente.

A nova funcionalidade não deve reduzir conversão em troca de conveniência para clientes recorrentes.

## SuperFrete

O endereço selecionado deve ser convertido para os mesmos dados usados atualmente pelo checkout.

SuperFrete continua server-side.

Mudança de endereço:

- invalida/recalcula cotação de frete quando aplicável;
- nunca reutiliza cotação de outro CEP/endereço indevidamente;
- mantém regras de `input_hash` e validade existentes.

A implementação precisará de testes específicos para recotação, validade e troca entre endereço salvo e endereço digitado.

## Pedidos e snapshots

Regra crítica: pedidos continuam congelando endereço como snapshot.

Exemplo:

```text
Cliente compra usando:
Casa — Rua A, 10

Depois altera Casa para:
Rua B, 20

Pedido antigo continua mostrando:
Rua A, 10
```

Nunca fazer pedido histórico depender de `customer_addresses`.

## Segurança

Planejar a implementação com as seguintes regras:

- ownership sempre por Supabase Auth UUID;
- mutations protegidas contra cross-site como o restante do projeto;
- nenhum address ID de outro usuário pode ser acessado;
- respostas públicas sem detalhes SQL;
- logs sem endereço, CEP completo ou PII;
- RLS habilitado sem policies públicas conforme arquitetura atual, se aplicável;
- validações brasileiras reaproveitadas;
- sem confiança em campos hidden para ownership.

## Performance

Endereços salvos são poucos; não criar arquitetura complexa.

Listagem simples por `auth_user_id` deve ser suficiente.

Planejar índice futuro apropriado por:

```text
auth_user_id
```

Planejar também constraint ou índice para garantir unicidade lógica do default por usuário quando `is_default=true`.

Não implementar agora.

## Fora de escopo inicial

Não planejar inicialmente:

- geolocalização;
- mapa;
- autocomplete pago;
- endereço internacional;
- sincronização externa;
- endereço de cobrança separado;
- carteira/cartões;
- complexidade excessiva estilo marketplace grande.

País continua `BR` inicialmente.

## Subfases sugeridas

Nenhuma subfase está iniciada.

- 21.1 — Modelo e persistência de endereços.
- 21.2 — Minha Conta / gerenciamento de endereços.
- 21.3 — Seleção de endereço no checkout.
- 21.4 — Migração/compatibilidade e validação em produção.

## Dependências

Fase 21 depende de:

- Fase 19 concluída;
- Fase 20 concluída;
- go-live estabilizado.

Não iniciar antes de resolver bloqueadores de produção.

## Definition of Done futura

A Definition of Done deve ser detalhada quando a fase for ativada. No mínimo, deverá cobrir:

- schema/migration revisados;
- ownership por `auth_user_id` testado;
- migração segura de dados existentes;
- Minha Conta com gerenciamento mobile-first;
- checkout autenticado selecionando endereço salvo server-side;
- checkout guest preservado;
- SuperFrete recotando corretamente ao trocar endereço;
- pedidos preservando snapshots históricos;
- logs sem PII;
- validação manual em produção.
