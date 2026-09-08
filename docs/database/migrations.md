# Migrations

Status: PLANEJADO. A pasta `migrations/` permanece sem migrations funcionais nesta fase.

## Regras

- Toda mudanca de schema deve possuir migration versionada.
- Migrations sao imutaveis depois de aplicadas em ambientes compartilhados.
- Rollback deve ser considerado antes da aplicacao.
- Migrations devem ser revisadas.
- Mudancas destrutivas precisam de cuidado adicional.
- Alteracoes de banco nao devem ser feitas manualmente em producao sem registro.

## Praticas recomendadas

- Uma migration deve representar uma mudanca coesa.
- O nome deve deixar clara a intencao da mudanca.
- Dados sensiveis nao devem aparecer em migrations.
- Migrations que alteram dados devem ser especialmente revisadas.
- Criacao de indices deve considerar impacto em tabelas grandes.

## Antes de aprovar uma migration

- O schema relacionado foi documentado.
- O impacto em codigo e dados foi entendido.
- Testes aplicaveis foram planejados ou executados.
- O caminho de rollback foi discutido quando necessario.
