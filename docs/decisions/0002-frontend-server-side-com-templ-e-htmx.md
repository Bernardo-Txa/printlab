# ADR-0002 — Frontend server-side com templ e HTMX

Status: Aprovado
Data: 2026-09-08

## Contexto

O projeto prioriza seguranca, simplicidade e baixa complexidade no frontend. As regras sensiveis devem permanecer no backend.

## Decisao

Usar renderizacao server-side com `templ`, HTMX para interacoes incrementais e Tailwind CSS para estilos.

## Alternativas consideradas

- SPA com framework JavaScript pesado.
- HTML renderizado manualmente sem templates tipados.
- Frontend acessando diretamente APIs de banco.

## Consequencias

- A maior parte da logica permanece no servidor.
- JavaScript proprio deve ser minimo.
- O design system e os componentes serao introduzidos em fase propria.

## Referencias

- [ARCHITECTURE.md](../../ARCHITECTURE.md)
- [docs/architecture/frontend.md](../architecture/frontend.md)
