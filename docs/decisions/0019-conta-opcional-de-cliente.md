# ADR-0019 — Conta opcional de cliente

Status: Aceita

Data: 2026-09-17

## Contexto

O checkout implementado e anonimo para reduzir atrito. Uma conta so agrega valor quando permite consultar pedidos, acompanhar producao/envio, retomar pagamento e recomprar, sem prejudicar visitantes.

## Decisao

Implementar na Fase 18 uma conta opcional de cliente com Supabase Auth separado do Admin. A Fase 18.2 adotou e-mail/senha com confirmação de e-mail e recuperação oficial do Supabase, preservando checkout convidado e acompanhamento publico seguro.

Autenticacao de cliente sera conceitualmente e tecnicamente separada da autenticacao administrativa. Um cliente autenticado nao recebe acesso a `/admin`; autorizacao administrativa continua dependente de senha, TOTP, AAL2, allowlist de UUID e sessao propria PrintLab.

Pedidos serao associados e autorizados server-side. Eventual reivindicacao de pedido antigo de convidado exigira prova de posse do e-mail e nao sera requisito inicial.

## Alternativas consideradas

- Conta obrigatoria antes do checkout: rejeitada por aumentar atrito e contradizer o fluxo atual.
- Reutilizar autenticacao de cliente como autorizacao Admin: rejeitada por risco de escalacao de privilegio.

## Consequencias

- O modelo futuro precisara de associacao segura entre identidade de cliente e pedido.
- A Fase 18.2 cria signup/login/recuperação usando Supabase Auth, sem tabela de cliente e sem associação de pedidos nesta etapa.

## Referencias

- [Roadmap](../plans/roadmap.md)
- [Pedidos](../product/orders.md)
- [Supabase Auth](../integrations/supabase-auth.md)
