# ADR-0014 — Auditoria transacional de operacoes administrativas de pedidos

Status: Aceita
Data: 2026-09-12

## Contexto

A Fase 13.2 permite que o Admin altere status operacionais de producao e envio. Essas operacoes afetam a comunicacao com o comprador em `/acompanhar/{public_tracking_id}` e precisam ser rastreaveis sem expor PII ou depender de logs efemeros.

Tambem e necessario evitar estados inconsistentes, como status alterado sem evento de auditoria, evento sem mutacao real ou duas operacoes concorrentes pulando etapas.

## Decisao

Criar `public.admin_order_events` para registrar mutacoes administrativas de producao/envio.

Cada POST administrativo:

```text
resolve sessao admin
  -> usa Session.AuthUserID como ator
  -> abre transacao PostgreSQL
  -> bloqueia pedido + fulfillment com SELECT ... FOR UPDATE
  -> valida transicao permitida
  -> atualiza public.order_fulfillment
  -> insere public.admin_order_events
  -> commit
```

Eventos guardam somente:

- `order_id`;
- `actor_auth_user_id`;
- `event_type`;
- `from_status`;
- `to_status`;
- `created_at`.

A tabela tem RLS habilitado sem policies publicas e nao referencia `auth.users`, mantendo baixo acoplamento com o schema interno do Supabase Auth.

## Alternativas consideradas

- Apenas logs de aplicacao: rejeitado porque logs nao sao trilha transacional e podem ser rotacionados.
- Colunas `updated_by` em `order_fulfillment`: rejeitada porque preserva apenas o ultimo ator e perde historico.
- Tabela de eventos generica para todo o sistema: rejeitada por escopo maior que a necessidade atual.
- Trigger no banco: rejeitada nesta fase porque a aplicacao ja precisa validar regras de transicao e conhece o ator da sessao.
- Armazenar snapshots completos do pedido no evento: rejeitado para evitar PII e duplicacao desnecessaria.

## Consequencias

- Mudancas operacionais validas ficam auditaveis por pedido.
- Status e evento sao persistidos ou revertidos juntos.
- Conflitos/stale submits falham sem criar evento.
- Consultas futuras podem exibir timeline operacional sem acessar logs.
- Ainda sera necessario definir limpeza/retencao formal se houver exigencia regulatoria futura.
- Papeis multiplos e permissoes granulares continuam fora do escopo desta fase.
