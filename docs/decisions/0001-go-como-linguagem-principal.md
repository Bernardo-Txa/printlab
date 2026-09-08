# ADR-0001 — Go como linguagem principal

Status: Aprovado
Data: 2026-09-08

## Contexto

A PrintLab precisa de um backend simples, performatico, com baixo custo operacional e boa manutencao no longo prazo.

## Decisao

Usar Go como linguagem principal do backend.

## Alternativas consideradas

- Linguagens dinamicas com frameworks web completos.
- Runtime JavaScript full-stack.
- Servicos separados desde o inicio.

## Consequencias

- O projeto ganha binario simples e bom suporte da biblioteca padrao.
- A equipe deve manter codigo Go idiomatico e explicito.
- Dependencias externas devem ser adicionadas com criterio.

## Referencias

- [ARCHITECTURE.md](../../ARCHITECTURE.md)
- [docs/architecture/backend.md](../architecture/backend.md)
