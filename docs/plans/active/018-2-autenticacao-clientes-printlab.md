# Fase 18.2 — Autenticação de clientes PrintLab

Status: Implementada no código; aguardando validação manual em produção.

## Objetivo

Implementar a fundação da autenticação pública de clientes da PrintLab com Supabase Auth, mantendo Admin separado e checkout convidado intacto.

## Escopo implementado

- `GET /cadastro` e `POST /cadastro` para criar conta com nome, e-mail, senha e confirmação de senha.
- Cadastro enviado ao Supabase Auth com `full_name` em `user_metadata`.
- Confirmação de e-mail usando o fluxo oficial do Supabase Auth e retorno para `/auth/callback`.
- `GET /login` e `POST /login` para autenticação por e-mail e senha.
- `POST /logout` para encerrar sessão no Supabase Auth e limpar cookies HttpOnly locais dos tokens da sessão Supabase.
- `GET /recuperar-senha` e `POST /recuperar-senha` usando recuperação oficial do Supabase Auth.
- `GET /recuperar-senha/nova` e `POST /recuperar-senha/nova` para definir nova senha após callback de recuperação.
- `GET /auth/callback` troca `?code=...` retornado pelo Supabase por sessão no backend e grava cookies HttpOnly.
- `POST /auth/session` permanece como fallback compatível para retornos com `access_token` e `refresh_token` no hash da URL.
- `GET /conta` com versão inicial da área de conta: nome, e-mail, status ativo e botão Sair.
- Header público alterna entre `Entrar` e `Minha conta`/`Sair` quando há sessão de cliente válida.
- Guard de rota para `/conta` e nova senha.
- Redirects internos seguros com bloqueio de URL externa arbitrária.
- Compra como visitante preservada: catálogo, produto, carrinho, checkout, frete, retirada no local, pagamento e pedidos não exigem login.

## Arquitetura

A identidade, senha, confirmação, recuperação, sessão e refresh pertencem ao Supabase Auth. A PrintLab não cria tabela de cliente, não cria hash de senha, não cria token próprio de recuperação e não cria sessão persistida própria para cliente.

Para SSR em Go, o backend armazena apenas os tokens de sessão emitidos pelo Supabase em cookies HttpOnly, SameSite=Lax, Path=/ e Secure em produção/HTTPS:

- `printlab_customer_access_token`;
- `printlab_customer_refresh_token`.

Esses cookies não representam uma sessão própria da PrintLab no banco; eles transportam a sessão oficial do Supabase para que o servidor valide o usuário em `GET /auth/v1/user`, renove por `grant_type=refresh_token` quando necessário e execute logout em `POST /auth/v1/logout`.

Admin continua isolado:

- `/admin` segue usando senha + MFA TOTP + AAL2 + allowlist de UUID + `admin_sessions` própria;
- cliente autenticado nunca autoriza Admin;
- cookies de cliente usam nomes e escopo diferentes dos cookies administrativos.

## Segurança

- Senhas são enviadas somente ao Supabase Auth.
- Nenhuma senha, access token, refresh token, recovery token, SMTP password, API key ou Authorization é registrado em log.
- Mensagem de login inválido é genérica: `E-mail ou senha inválidos.`
- Recuperação não enumera usuários: `Se existir uma conta para este e-mail, enviaremos as instruções de recuperação.`
- Redirect de login aceita somente caminhos internos seguros.
- Páginas de conta/autenticação usam `Cache-Control: private, no-store` e noindex.
- `/auth/callback` usa JavaScript estático em `/static/js/auth-callback.js`; não há script inline.

## Banco e migrations

Nenhuma migration foi criada nesta fase.

Nenhuma tabela de clientes foi adicionada porque o escopo atual não precisa persistir dados fora do metadata do Supabase Auth. Associação de pedidos, endereços, histórico e retomada de pagamento ficam para fases posteriores.

## E-mail e Supabase Dashboard

A implementação depende das configurações externas do Supabase Auth:

- Site URL de produção: `https://www.printlab3d.com.br`;
- Redirect URLs para `/auth/callback` nos ambientes usados;
- confirmação de e-mail habilitada;
- recuperação de senha habilitada;
- Custom SMTP configurado no Supabase Auth com os dados validados da Fase 18.1.

Os templates preparados em `internal/email` continuam como fundação e referência técnica. A PrintLab só deve afirmar que templates customizados estão ativos quando eles forem conectados no Dashboard do Supabase.

## Fora de escopo

Não foi implementado:

- Minha Conta completa;
- histórico de pedidos;
- associação de pedido existente à conta;
- endereços salvos;
- alteração do checkout;
- alteração de carrinho;
- alteração de SuperFrete;
- alteração de InfinitePay;
- favoritos, avaliações ou fidelidade.

## Validação automática executada

- `templ generate`
- `gofmt`
- `go test ./...`
- `go vet ./...`
- `go build ./...`
- `git diff --check`

Validações ainda obrigatórias antes de concluir a fase:

- validação manual real em produção conforme briefing da fase.

## Validação manual pendente

Antes de mover para `completed`, validar em produção:

1. Abrir `/cadastro`.
2. Criar conta de teste.
3. Receber e-mail real.
4. Confirmar o cadastro.
5. Retornar para a PrintLab.
6. Acessar `/conta`.
7. Fazer logout.
8. Confirmar que `/conta` exige login.
9. Fazer login novamente.
10. Solicitar recuperação de senha.
11. Receber e-mail real.
12. Definir nova senha.
13. Fazer login com a nova senha.
14. Confirmar que compra como convidado continua funcionando.
