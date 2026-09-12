# Supabase Auth

Status: IMPLEMENTADO NA FASE 13.1 para autenticacao administrativa.

## Escopo

A PrintLab usa Supabase Auth somente para validar e-mail e senha do acesso administrativo inicial.

Rotas implementadas:

- `GET /admin/login`
- `POST /admin/login`
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
  -> PrintLab cria sessao opaca propria
  -> cookie HttpOnly printlab_admin_session
```

O backend usa `net/http`, timeout explicito e header `apikey: SUPABASE_PUBLISHABLE_KEY`.

## Sessao PrintLab

A PrintLab nao persiste senha, access token ou refresh token do Supabase.

Depois do login autorizado:

- gera token aleatorio de 32 bytes com `crypto/rand`;
- codifica em `base64.RawURLEncoding`;
- grava somente `SHA-256(token)` em `public.admin_sessions.token_hash`;
- define `expires_at` com TTL de 8 horas;
- envia cookie `printlab_admin_session`.

Cookie:

- `HttpOnly`;
- `SameSite=Strict`;
- `Path=/admin`;
- host-only;
- `Secure` em producao ou quando `SITE_URL` usa HTTPS.

Logout remove a sessao persistida quando houver token valido e sempre limpa o cookie.

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

Supabase Auth possui rate limits proprios. Revisao de abuso, brute force, CAPTCHA/WAF e protecoes adicionais fica planejada para a Fase 14.

## Referencias oficiais

- https://supabase.com/docs/guides/auth
- https://supabase.com/docs/guides/auth/passwords
- https://supabase.com/docs/guides/getting-started/api-keys
