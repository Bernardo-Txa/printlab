# ADR-0017 - MFA TOTP antes da sessao administrativa

Status: aceita; implementacao com validacao real pendente.
Data: 2026-09-16.

## Contexto

A Fase 14.1 foi validada em producao pelo responsavel. A sessao propria do unico Admin precisa representar senha e segundo fator concluidos, inclusive recusando sessoes anteriores a esta mudanca.

## Decisao

Preservar Supabase Auth, SSR/forms e sessao opaca PrintLab. O provider net/http usa publishable key e bearer do usuario para TOTP. Nenhum endpoint MFA normal usa secret key.

O primeiro fator autoriza UUID, mas nao cria admin_session. Access token AAL1 fica exclusivamente em cookie HttpOnly/Strict, host-only, Path=/admin/mfa, Secure em producao, por ate 10 minutos. Nao se persiste refresh token; nenhum token Supabase vai ao banco. A idade e validada por `iat` somente depois de GetUser autenticar o token.

Zero TOTP verified exige enrollment. Antes de enrollment, remover somente TOTP unverified do mesmo usuario, usando sua sessao AAL1; Supabase impede remover fatores verified com AAL1. Recarregar setup substitui o enrollment abandonado. Mais de um verified exige selecao explicita, sempre revalidada no backend.

Verify deve retornar token que GetUser autentique; UUID autorizado e claim aal2 sao obrigatorios. Somente entao criar token opaco PrintLab, persistir SHA-256 e gravar `mfa_verified_at = now()`. Coluna nullable sem default ou backfill invalida sessoes legadas. TTL de 8 horas e cron preservados.

## Alternativas consideradas

- Sessao Supabase ou refresh token persistente: desnecessarios para o Admin SSR com sessao propria.
- Estado em memoria: inadequado para instancias serverless e desnecessario.
- Token Supabase no banco: rejeitado pelo escopo e pela exposicao adicional.
- Confiar apenas em verify HTTP 200: insuficiente para comprovar AAL2.

## Consequencias

O token AAL1 e um bearer sensivel mesmo sem acesso ao Admin. Exige HTTPS em producao, TTL curto e ausencia de logs/HTML. Cookie removido em sucesso, cancelamento, logout ou erro terminal. A sessao propria nao e automaticamente revogada por mudancas futuras no Supabase; recuperacao emergencial deve invalidar sessoes PrintLab tambem.

Perder o autenticador pode bloquear o unico operador. Nao ha reset publico, bypass ou recovery codes. Recuperacao administrativa oficial esta no runbook. WAF fica na 14.3.

## Referencias

- [TOTP oficial](https://supabase.com/docs/guides/auth/auth-mfa/totp)
- [Contrato Auth OpenAPI](https://github.com/supabase/auth/blob/master/openapi.yaml)
- [Runbook PrintLab](../integrations/supabase-auth.md#recuperacao-de-emergencia)
