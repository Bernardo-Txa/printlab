# Fase 14.3 - WAF / anti-abuse na Vercel

Status: em configuracao operacional; regras reais e validacao em producao pendentes de acesso autenticado ao projeto Vercel.

## Objetivo e limite

Reduzir brute force, credential stuffing e abuso de endpoints caros no edge da Vercel, antes da funcao Go. Esta fase nao cria rate limiter em memoria, Redis, banco, cookie anti-bot, CAPTCHA proprio, fingerprinting ou migration. MFA, InfinitePay e SuperFrete permanecem inalterados.

`vercel.json` foi revisado e continua minimo, somente com `regions: ["gru1"]`. Ele nao e o mecanismo desta fase: a documentacao atual da Vercel suporta nele apenas `challenge` e `deny`, enquanto o rollout requer `log` e rate limiting configurados no Firewall.

## Analise de rotas

As rotas reais foram confirmadas em `cmd/server/main.go`:

- `POST /admin/login`;
- `POST /admin/mfa/setup` e `POST /admin/mfa/challenge`;
- `POST /checkout/frete`;
- `POST /pedido/{id}/pagar`;
- `POST /webhooks/infinitepay`.

`GET /checkout/frete` e `POST /checkout/frete` chamam `shipping.Service.prepareQuotes`; quando SuperFrete esta configurada, cada um pode fazer uma chamada de planejamento e outra de cotacao final. O GET nao entra no limite inicial porque e navegacao normal do checkout. Observar seu volume no Firewall antes de criar uma regra adicional, sem restringir catalogo ou demais GETs publicos.

## Regras a publicar

Criar regras de projeto, nesta ordem, com condicoes combinadas por `AND`, chave `ip`, algoritmo `fixed_window`, janela `600` segundos e acao ao exceder `rate_limit` (HTTP `429`). Nao usar acao persistente inicialmente.

| Nome | Metodo e path | Limite | Acao | Razao |
| --- | --- | --- | --- | --- |
| `rate-limit-admin-login` | `POST`; path exato `/admin/login` | 10 / 600 s | `429 rate_limit` | reduzir brute force e credential stuffing |
| `rate-limit-admin-mfa` | `POST`; path exato `/admin/mfa/setup` OR `/admin/mfa/challenge` | 20 / 600 s | `429 rate_limit` | reduzir abuso adicional de TOTP sem apertar demais o operador |
| `rate-limit-checkout-shipping-post` | `POST`; path exato `/checkout/frete` | 30 / 600 s | `429 rate_limit` | conter recalculos repetidos e chamadas SuperFrete |
| `rate-limit-order-payment-post` | `POST`; path que casa exatamente `/pedido/{uuid}/pagar` | 20 / 600 s | `429 rate_limit` | conter inicio repetido de checkout/pagamento |

Para a ultima regra, use no construtor visual de condicoes o equivalente a metodo `POST` e path com inicio `/pedido/` e fim `/pagar`, ou uma expressao/operador de path que case somente o padrao completo. Revise a condicao gerada antes de publicar para ela nao incluir `GET /pedido/{id}` nem outros POSTs sob `/pedido/`.

Nao criar regra para `POST /webhooks/infinitepay`: nao aplicar challenge humano, allowlist IP inventada ou rate limit generico apertado. O webhook e servidor-servidor, pode receber retries e continua usando a validacao server-side e `payment_check` existente. Tambem nao limitar apertadamente `/`, `GET /produtos`, `GET /produtos/{slug}`, `GET /static/*`, `GET /acompanhar/*` ou `GET /pedido/*`.

## Bot Protection

No projeto Vercel, habilitar/editar `Bot Protection` para a acao `Log`. Nao usar `Challenge` ou `Deny` nesta fase e nao bloquear crawlers de busca. Observar por pelo menos 10 minutos antes de propor mudanca de acao.

## Procedimento operacional

O CLI `vercel` nao esta instalado globalmente, mas `npx --yes vercel@latest` forneceu Vercel CLI `59.20.0` e os comandos `firewall rules add`, `diff`, `publish`, `status`, `traffic` e `bot-management`. A sessao local retornou `login_required`; por isso nenhuma regra foi criada, publicada ou alegada como configurada.

Com credencial de operador e projeto explicitamente selecionado, prefira a CLI auditavel abaixo. Substituir apenas `<project>` pelo nome ou ID confirmado, sem versionar token:

```bash
npx --yes vercel@latest firewall rules add "rate-limit-admin-login" --project <project> --condition '{"type":"method","op":"eq","value":"POST"}' --condition '{"type":"path","op":"eq","value":"/admin/login"}' --action rate_limit --rate-limit-keys ip --rate-limit-algo fixed_window --rate-limit-requests 10 --rate-limit-window 600 --rate-limit-action rate_limit --yes
npx --yes vercel@latest firewall rules add "rate-limit-checkout-shipping-post" --project <project> --condition '{"type":"method","op":"eq","value":"POST"}' --condition '{"type":"path","op":"eq","value":"/checkout/frete"}' --action rate_limit --rate-limit-keys ip --rate-limit-algo fixed_window --rate-limit-requests 30 --rate-limit-window 600 --rate-limit-action rate_limit --yes
npx --yes vercel@latest firewall rules list --project <project> --expand
npx --yes vercel@latest firewall diff --project <project>
npx --yes vercel@latest firewall publish --project <project>
npx --yes vercel@latest firewall status --project <project> --json
```

Criar as regras MFA e pagamento pelo construtor do Dashboard quando a expressao de path precisao nao puder ser confirmada pelo CLI, e publicar somente apos revisar o diff. No Dashboard: projeto -> `Firewall` -> `Configure` -> `Add New` -> `Rule`; adicionar method/path, selecionar `Rate Limit`, `fixed window`, IP, janela e resposta `429`; `Save Rule` -> `Review Changes` -> `Publish`. Para Bot Protection: projeto -> `Firewall` -> `Configure` -> regra gerenciada `Bot Protection` -> `Log` -> `Save Rule` -> `Review Changes` -> `Publish`.

## Validacao real e rollback

Depois de publicar, registrar somente evidencias sem PII, tokens, IPs, pedidos ou credenciais:

1. `firewall rules list --expand`, `firewall status` e o diff vazio apos publish confirmam nomes, condicoes, chaves, janelas e acoes.
2. Validar uma requisicao normal de cada POST protegido abaixo do limite; ela deve chegar ao handler e manter a validacao atual de `Origin`/`Referer`.
3. Em ambiente/conta de teste, exceder cada limite com requests repetidas e confirmar HTTP `429`; aguardar a janela e confirmar recuperacao.
4. Confirmar que `POST /webhooks/infinitepay` nao recebe challenge ou regra apertada e que o fluxo financeiro normal continua intacto.
5. Conferir no Firewall traffic/logs os matches e falsos positivos; Bot Protection fica em `Log` por no minimo 10 minutos antes de qualquer escalonamento.

Para rollback imediato, no Firewall desabilitar ou remover somente a regra afetada, revisar e publicar; pelo CLI, usar `firewall rules disable <name-or-id> --project <project>` seguido de `firewall publish --project <project>`. `firewall discard --project <project>` descarta apenas mudancas ainda nao publicadas. Nao modificar `vercel.json`, Go, InfinitePay, SuperFrete ou MFA como rollback.

## Referencias oficiais consultadas

- https://vercel.com/docs/vercel-firewall/vercel-waf/custom-rules
- https://vercel.com/docs/vercel-firewall/vercel-waf/rate-limiting
- https://vercel.com/docs/bot-management
- https://vercel.com/changelog/manage-vercel-firewall-in-the-cli

## Definition of Done

- Regras reais acima existentes no projeto Vercel e revisadas apos publicacao.
- Bot Protection em `Log` e observado.
- Login, MFA, frete, pagamento e webhook validados sem regressao; `429` confirmado em conta/ambiente de teste.
- Documentacao operacional, rollback e referencias atualizados.
- Nenhuma migration, limiter local ou alteracao em MFA, InfinitePay ou SuperFrete.
