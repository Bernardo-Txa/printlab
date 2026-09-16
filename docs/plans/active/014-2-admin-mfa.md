# Fase 14.2 - MFA obrigatorio para Admin

Status: implementada; validacao real pendente.

## Escopo

- Supabase Auth password -> autorizacao por UUID -> AAL1 -> TOTP -> AAL2 -> sessao opaca PrintLab.
- Provider net/http existente ampliado; publishable key e bearer do usuario, sem secret key.
- Cookie pending HttpOnly, Strict, host-only, Path=/admin/mfa, Secure conforme ambiente e TTL de 10 minutos. Somente access_token; refresh_token descartado.
- Setup SSR com QR em img data URI, chave manual efemera e limpeza somente de TOTP unverified.
- Challenge SSR com escolha explicita quando houver varios fatores verified.
- Validacao remota do token atualizado e claim aal2 antes de criar qualquer admin_session.
- Migration append-only com mfa_verified_at nullable; sessoes antigas recusadas.
- Testes httptest sem Supabase real, documentacao e runbook de recuperacao.

## Validacao e entrega

- Gerar templ/CSS; gofmt, tidy, test, vet, build, govulncheck e npm audit.
- Validar migration com Supabase local quando disponivel e workflow oficial dry-run -> push.
- Revisar diff, commit e push seletivos.
- Validacao real de enrollment, login TOTP e recuperacao permanece pendente.

## Fora do escopo

14.3, Fase 15, WAF, rate limiting local, RBAC, outros fatores, recovery codes e reset publico de MFA.

## Resultados locais

- `templ generate`: binario ausente no PATH; geracao equivalente concluida com `go run github.com/a-h/templ/cmd/templ@v0.3.1020 generate`.
- `npm run css:build`, `gofmt -w .` e `go mod tidy`: concluidos, sem dependencia nova.
- `go test ./...`, `go vet ./...` e `go build ./...`: passaram. Testes HTTP usam apenas httptest, sem Supabase real; sockets locais exigiram execucao fora do sandbox restrito.
- `govulncheck`: binario ausente no PATH; `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` concluiu sem vulnerabilidades.
- `npm audit`: zero vulnerabilidades.
- Supabase local: `db push --local --dry-run` seguido de `db push --local` aplicou somente `20260916120000_require_admin_session_mfa.sql`.
- PostgreSQL local confirmou coluna nullable sem default, filtro de sessao sem timestamp e gravacao de timestamp em nova sessao; dados sinteticos usados em transacao com rollback. Job diario de limpeza preservado.
- Fluxos testados: senha sem sessao, UUID nao autorizado, setup/desafio, escolha e revalidacao de fatores, QR e privacidade, cookies, expiracao, Origin/Referer, codigo invalido, 429/5xx, token atualizado recusado, ausencia de AAL2, sessao completa, sessao legada e logout.
- Workflow remoto de migrations executa dry-run antes de db push apos o push para main. Validacao real de MFA no projeto de destino ainda deve ser realizada pelo responsavel.
