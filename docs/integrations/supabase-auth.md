# Supabase Auth

Status: senha e sessao propria implementadas na 13.1; MFA TOTP obrigatorio implementado na 14.2 e validado em producao. Fase 14.1 validada em producao pelo responsavel.

## Escopo

A PrintLab usa Supabase Auth para senha e segundo fator TOTP do unico administrador autorizado por UUID.

Rotas implementadas:

- `GET /admin/login`
- `POST /admin/login`
- `GET /admin/mfa/setup`
- `POST /admin/mfa/setup`
- `GET /admin/mfa/challenge`
- `POST /admin/mfa/challenge`
- `POST /admin/mfa/cancel`
- `GET /admin`
- `POST /admin/logout`

Nao existem signup, cadastro de administrador, login social, lembrar de mim ou recuperacao de senha pela aplicacao.

## Configuracao

Variaveis:

- `SUPABASE_URL`
- `SUPABASE_PUBLISHABLE_KEY`
- `ADMIN_SUPABASE_USER_ID`

`SUPABASE_PUBLISHABLE_KEY` identifica a aplicacao perante o Supabase. Ela nao e credencial administrativa e nao substitui autorizacao.

`ADMIN_SUPABASE_USER_ID` e o UUID do usuario criado manualmente no Dashboard Supabase em Authentication -> Users. A PrintLab autoriza acesso comparando esse UUID com `user.id` retornado pelo Auth.

Se a configuracao estiver ausente ou incompleta, a loja publica continua iniciando e `/admin/login` mostra indisponibilidade segura.

## Fluxo

```text
POST /admin/login
  -> POST {SUPABASE_URL}/auth/v1/token?grant_type=password
  -> Supabase Auth valida e-mail/senha
  -> PrintLab verifica user.id == ADMIN_SUPABASE_USER_ID
  -> AAL1: somente cookie MFA temporario, sem sessao PrintLab
  -> GET /auth/v1/user: usuario e fatores revalidados
  -> zero TOTP verified: limpar TOTP unverified e fazer enrollment
  -> um ou varios verified: desafio, com escolha explicita quando varios
  -> POST /auth/v1/factors/{factor_id}/challenge
  -> POST /auth/v1/factors/{factor_id}/verify
  -> GET /auth/v1/user com token atualizado + exigir claim aal2
  -> PrintLab cria sessao opaca propria com mfa_verified_at
  -> cookie HttpOnly printlab_admin_session
```

O backend usa `net/http`, timeout de 8 segundos, body de resposta limitado a 1 MiB e header `apikey: SUPABASE_PUBLISHABLE_KEY`. Chamadas do usuario usam `Authorization: Bearer <access_token>`, nunca `SUPABASE_SECRET_KEY`. Redirects externos do provider nao sao seguidos. 401/422, 429, 5xx, resposta invalida e configuracao invalida sao categorias seguras; logs nunca incluem body externo.

Contratos confirmados no OpenAPI e implementacao oficiais do Supabase Auth em 2026-09-16:

- Password: `POST /auth/v1/token?grant_type=password`, JSON email/password; ler user e access_token, descartar refresh_token.
- Usuario: `GET /auth/v1/user`, incluindo lista `factors` com id, factor_type, status e friendly_name.
- Enrollment: `POST /auth/v1/factors`, JSON `factor_type: totp`, `friendly_name: PrintLab Admin`, `issuer: PrintLab`; retorno id, type e totp.qr_code/secret. A URI nao e retida.
- Cleanup: `DELETE /auth/v1/factors/{factor_id}`, somente TOTP unverified do usuario atual. Supabase exige AAL2 para remover verified. Falha interrompe novo enrollment sem remover fator valido.
- Challenge: POST com JSON vazio; retorno id.
- Verify: POST com challenge_id e codigo de seis digitos; retorno access_token atualizado e user.

QR SVG e codificado em base64 e exibido por img data URI, sem HTML cru. O provider aceita respostas com whitespace ou declaracao XML, mas exige QR nao vazio contendo abertura `<svg` e fechamento `</svg>`; texto arbitrario ou SVG incompleto e rejeitado. A CSP existente permite `img-src data:`. Secret manual e mostrado somente na resposta inicial privada/no-store, sem cookie/DB/logs. Codigo incorreto reexibe somente formulario e factor ID; reload cria novo enrollment apos limpeza. Evitar abas simultaneas durante setup.

## Sessao PrintLab

A PrintLab nao persiste senha, access token ou refresh token no banco. Refresh token e descartado no password e no verify. O access_token AAL1 fica exclusivamente em `printlab_admin_mfa_pending`, HttpOnly, SameSite=Strict, host-only, Path=/admin/mfa, Secure em producao/HTTPS, com TTL de 10 minutos. Nao aparece em HTML, query string, JS ou logs. A cada etapa, GetUser autentica o token antes de ler sub, aal, iat e exp; token antigo exige nova senha mesmo se o cookie for reenviado.

Depois de MFA AAL2 confirmado e UUID autorizado:

- gera token aleatorio de 32 bytes com `crypto/rand`;
- codifica em `base64.RawURLEncoding`;
- grava somente `SHA-256(token)` em `public.admin_sessions.token_hash`;
- define `expires_at` com TTL de 8 horas;
- grava `mfa_verified_at = now()`; sessao legada com NULL nao autentica;
- envia cookie `printlab_admin_session`.

Cookie:

- `HttpOnly`;
- `SameSite=Strict`;
- `Path=/admin`;
- host-only;
- `Secure` em producao ou quando `SITE_URL` usa HTTPS.

Logout remove a sessao persistida quando houver token valido e limpa ambos os cookies. Sucesso MFA, cancelamento e erro terminal limpam pending. Codigo invalido/429 antes da elevacao permite retry dentro do TTL; falha terminal exige nova senha.

## Banco

Tabela:

- `public.admin_sessions`

Ela nao possui FK para `auth.users` nesta fase. Isso reduz acoplamento com o schema interno do Supabase Auth; a autorizacao continua no backend pelo UUID retornado no login.

RLS fica habilitado e nenhuma policy publica e criada.

## Seguranca

Mensagens publicas de erro de login sao genericas:

```text
E-mail ou senha inválidos.
```

A aplicacao nao revela se o usuario existe, se a senha falhou ou se o usuario autenticado nao e administrador.

Logs nao devem registrar e-mail, senha, token de sessao, hash, access token, refresh token, publishable key, secret key ou PII.

Supabase Auth possui rate limits proprios. Como a PrintLab usa o fluxo `Browser -> Go -> Supabase Auth`, tentativas de login podem ser vistas pelo Supabase como trafego vindo do IP server-side da aplicacao. A mitigacao oficial de IP forwarding exige `Sb-Forwarded-For`, secret API key com prefixo `sb_secret` e habilitacao explicita do recurso no projeto. A Fase 14.1 documenta o risco, mas nao adiciona o header nem troca a credencial do fluxo Auth.

Protecao contra abuso e brute force e definida operacionalmente pela Fase 14.3 no Vercel Firewall/WAF, sem rate limiter local. No Vercel Hobby atual, `POST /admin/login` tem limite por IP de 10 requisicoes por 600 segundos; MFA continua protegido por senha, TOTP e rate limits do Supabase Auth, sem custom rate-limit adicional no edge. A configuracao real esta registrada em [014-3-waf-anti-abuse.md](../plans/completed/014-3-waf-anti-abuse.md).

## Configuracao e validacao real

TOTP e habilitado por padrao nos projetos Supabase segundo a documentacao oficial. Confirmar no Dashboard em Authentication/Multi-Factor Authentication que enrollment e verify TOTP estao habilitados no projeto de destino; nao habilitar phone MFA para esta entrega. Nao e necessaria variavel de ambiente nova nem secret key para o fluxo.

`20260916120000_require_admin_session_mfa.sql` foi aplicado pelo workflow oficial dry-run -> db push. O responsavel confirmou em producao `ADMIN_SUPABASE_USER_ID`, HTTPS e `SITE_URL` canonica, primeiro enrollment e novo login TOTP: senha -> AAL1 -> TOTP -> AAL2 -> sessao PrintLab. Evidencias nao registram QR, secret, codigos ou tokens.

## Recuperacao de emergencia

Perder o fator pode bloquear o unico Admin. Nao ha bypass MFA, recovery codes experimentais, reset publico ou desativacao pela PrintLab.

1. Um operador autorizado acessa o projeto pelo Dashboard Supabase com suas credenciais administrativas protegidas e confirma a identidade do responsavel fora da aplicacao.
2. Usa o mecanismo administrativo oficial de MFA para listar e remover o fator perdido: API Auth Admin `GET /admin/users/{user_id}/factors` e `DELETE /admin/users/{user_id}/factors/{factor_id}`, ou a operacao equivalente do Dashboard quando disponivel. Sao rotas administrativas do Supabase, nao rotas PrintLab. Executar somente em ambiente confiavel, seguindo a referencia oficial e sem transportar credenciais para o navegador/aplicacao.
3. Nao editar `auth.*` diretamente por SQL. Nao adicionar secret key ao provider normal para recuperar acesso.
4. Invalidar sessoes PrintLab existentes do usuario por operacao administrativa controlada em `public.admin_sessions` (remocao por auth_user_id). Revogar fator Supabase nao revoga automaticamente a sessao propria de 8 horas.
5. Fazer login novamente com senha; sem TOTP verified, a PrintLab exige novo enrollment e so libera Admin depois de AAL2. Se restar outro fator verificado acessivel, ele pode ser selecionado no login.

Manter acesso administrativo ao projeto Supabase e processo de identificacao do operador testados. Segundo TOTP de backup e fluxo administrativo dedicado sao mitigacoes futuras, nao implementadas aqui.

## Referencias oficiais

- https://supabase.com/docs/guides/auth
- https://supabase.com/docs/guides/auth/passwords
- https://supabase.com/docs/guides/getting-started/api-keys
- https://supabase.com/docs/guides/auth/rate-limits
- https://supabase.com/docs/guides/auth/auth-mfa
- https://supabase.com/docs/guides/auth/auth-mfa/totp
- https://github.com/supabase/auth/blob/master/openapi.yaml
- https://github.com/supabase/auth/blob/master/internal/api/mfa.go
