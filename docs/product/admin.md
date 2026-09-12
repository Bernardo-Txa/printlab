# Painel administrativo

Status: Fase 13.1 IMPLEMENTADA; Fases 13.2, 13.3 e 13.4 PLANEJADAS.

O painel administrativo concentrara funcionalidades internas de operacao da PrintLab em subfases pequenas.

## Status por subfase

- 13.1 — autenticacao, autorizacao, sessao e shell administrativo: concluida.
- 13.2 — pedidos, producao, envio e auditoria: planejada.
- 13.3 — catalogo, variantes, materiais, cores e caixas: planejada.
- 13.4 — imagens e Supabase Storage: planejada.

## Fase 13.1 implementada

Rotas:

- `GET /admin/login`
- `POST /admin/login`
- `GET /admin`
- `POST /admin/logout`

O login usa Supabase Auth com e-mail e senha. A PrintLab autoriza somente o usuario cujo `user.id` corresponde a `ADMIN_SUPABASE_USER_ID`.

A sessao administrativa e propria da PrintLab:

- token aleatorio opaco;
- cookie `printlab_admin_session`;
- `HttpOnly`;
- `SameSite=Strict`;
- `Path=/admin`;
- `Secure` em producao;
- TTL de 8 horas;
- `SHA-256(token)` persistido em `public.admin_sessions`.

O dashboard inicial e somente leitura e mostra contagens agregadas:

- pedidos aguardando pagamento;
- pedidos pagos aguardando producao;
- pedidos em producao;
- pedidos aguardando envio.

`pending_payment` nao entra como aguardando producao.

## Seguranca

- Nao ha signup administrativo pela aplicacao.
- O administrador inicial deve ser criado manualmente no Dashboard Supabase em Authentication -> Users.
- E-mail nao e autorizacao administrativa; o UUID do usuario e a fonte estavel.
- Senha, access token, refresh token, token de sessao e token hash nao devem aparecer em logs ou documentacao.
- Todas as respostas `/admin` usam `Cache-Control: private, no-store`, `X-Robots-Tag: noindex, nofollow, noarchive` e `Referrer-Policy: same-origin`.
- POSTs administrativos validam `Origin`/`Referer`; `Origin: null` e rejeitado independentemente de `Referer`.
- O dashboard da 13.1 nao carrega nem renderiza CPF, endereco, telefone, e-mail de cliente, `transaction_nsu` ou checkout URL.

## Limites da 13.1

- Nao ha CRUD de produtos.
- Nao ha alteracao de pedidos.
- Nao ha mutations de producao/envio.
- Nao ha auditoria operacional.
- Nao ha upload de imagens.
- Nao ha papeis multiplos.
- Nao ha MFA obrigatorio nem CAPTCHA/WAF na aplicacao.
- Nao ha uso de `SUPABASE_SECRET_KEY` ou service role.

## Decisoes pendentes

- Regras de auditoria da Fase 13.2.
- Fluxo operacional de producao e envio.
- Politica de permissoes caso existam multiplos usuarios administrativos.
- Protecoes adicionais contra abuso/brute force na Fase 14.
