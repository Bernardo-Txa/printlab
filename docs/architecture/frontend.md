# Frontend

Status: PLANEJADO.

## Responsabilidade

O frontend devera apresentar paginas HTML renderizadas no servidor, formularios e interacoes progressivas. A experiencia deve ser simples, rapida e acessivel.

## Limites

- O frontend nao acessa diretamente tabelas sensiveis.
- O frontend nao decide preco, desconto, subtotal, total, frete, status de pedido ou status de pagamento.
- O frontend nao armazena credenciais de integracoes.

## Decisoes

- Usar renderizacao server-side.
- Usar `templ` para templates tipados.
- Usar HTMX para atualizacoes parciais baseadas em HTTP.
- Usar Tailwind CSS quando a etapa de design system comecar.
- Manter JavaScript proprio no minimo necessario.

## Praticas recomendadas

- Formularios sem dependencia obrigatoria de JavaScript.
- Interacoes HTMX que preservem semantica HTTP.
- Componentes reutilizaveis apenas quando reduzirem duplicacao real.
- Estados de erro claros vindos do backend.
- Acessibilidade considerada desde os primeiros layouts.

## Praticas proibidas

- Duplicar regra financeira no navegador.
- Criar SPA pesada sem decisao arquitetural registrada.
- Usar JavaScript para contornar validacao server-side.
- Expor tokens, chaves ou endpoints sensiveis no cliente.
