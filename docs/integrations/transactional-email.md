# E-mail transacional PrintLab

Status: fundação de código implementada na Fase 18.1; SMTP externo configurado e validado fora do repositório para uso pelo Supabase Auth.

## Objetivo

A PrintLab usará e-mails transacionais com identidade própria para autenticação futura de clientes e comunicação de pedidos. A Fase 18.1 preparou templates e documentação; testes automatizados não enviam SMTP real. A Fase 18.2 passou a usar Supabase Auth para cadastro, login e recuperação de clientes, sem alterar checkout, SuperFrete ou InfinitePay.

Remetente planejado:

- Nome: `PrintLab`
- E-mail: `acesso@printlab3d.com.br`
- Reply-To planejado: `acesso@printlab3d.com.br`

Infraestrutura planejada:

```text
iCloud+ Custom Email Domain
  -> acesso@printlab3d.com.br
  -> Custom SMTP do Supabase Auth
  -> e-mails transacionais da PrintLab
```

Nenhuma senha SMTP, token, API key ou segredo deve ser versionado, documentado em claro, enviado ao frontend ou registrado em logs.

## Estado atual de autenticação

Supabase Auth está implementado para dois fluxos separados:

- Admin: `POST /admin/login` autentica e-mail/senha no Supabase Auth, autoriza por `ADMIN_SUPABASE_USER_ID` e exige MFA TOTP antes de criar a sessão administrativa própria;
- Cliente: `/cadastro`, `/login`, `/recuperar-senha`, `/auth/callback`, `/auth/session`, `/conta` e `/logout` usam Supabase Auth com e-mail/senha, confirmação de e-mail e recuperação oficial.

Checkout convidado permanece disponível e não exige conta.

## Configuração externa necessária

Checklist antes de habilitar envio real:

- [ ] domínio configurado no iCloud+ Custom Email Domain;
- [ ] endereço `acesso@printlab3d.com.br` criado;
- [ ] DNS exigido pelo iCloud+ configurado;
- [ ] SPF configurado;
- [ ] DKIM configurado;
- [ ] DMARC configurado;
- [x] Custom SMTP configurado no Supabase Auth;
- [x] SMTP host definido no Supabase;
- [x] SMTP port definido no Supabase;
- [x] SMTP username definido no Supabase;
- [x] SMTP password definido como secret externo, nunca no Git;
- [x] sender name `PrintLab` configurado;
- [x] sender email `acesso@printlab3d.com.br` configurado;
- [ ] reply-to configurado;
- [ ] URLs de redirect do Supabase Auth revisadas para o domínio real;
- [x] e-mail de teste recebido;
- [ ] Gmail validado;
- [ ] Outlook validado;
- [ ] iCloud Mail validado.

Não registrar credenciais ou evidências sensíveis. Templates customizados no Dashboard do Supabase só devem ser marcados como ativos quando forem configurados manualmente e validados.

## Templates preparados no código

O pacote `internal/email` renderiza HTML e texto simples para:

1. confirmação de cadastro;
2. Magic Link;
3. recuperação de acesso;
4. confirmação de pedido;
5. atualização de status do pedido;
6. pagamento confirmado.

Cada mensagem possui:

- subject;
- preheader;
- título;
- corpo;
- CTA quando há URL real;
- fallback textual com URL;
- rodapé transacional;
- HTML compatível com clientes de e-mail por tabelas e estilos inline;
- versão texto simples.

Os templates de autenticação exigem URLs reais recebidas da configuração/chamada. Não há links fictícios.

## Conteúdo de pedidos

E-mails de pedido usam somente estado comercial persistido:

- número do pedido;
- produtos;
- variante comercial, quando houver;
- cor comercial, quando houver;
- quantidade;
- subtotal;
- frete;
- total;
- modalidade de entrega;
- serviço/transportadora/prazo quando houver envio;
- status comercial quando aplicável.

Para retirada, o conteúdo mostra:

```text
Retirada no local
Grátis
```

O e-mail não recota SuperFrete e não chama integrações externas. Ele deve ser montado a partir do estado persistido do pedido.

## Dados que não pertencem aos e-mails

Não incluir:

- senha SMTP;
- tokens;
- headers `Authorization`;
- API keys;
- CPF;
- endereço completo;
- telefone completo;
- custo de produção;
- filamento;
- tempo de impressão;
- receita técnica;
- margem;
- SKU interno desnecessário;
- caixa física;
- peso/dimensões do pacote;
- `transaction_nsu`;
- `invoice_slug`;
- checkout URL de pagamento hospedado.

## Validação implementada

Testes automatizados cobrem:

- renderização HTML/texto;
- subjects;
- preheaders;
- links reais;
- remetente planejado;
- pedido com envio;
- pedido com retirada;
- pagamento confirmado;
- atualização de status;
- ausência de informações internas;
- ausência de secrets comuns;
- rejeição de links ausentes ou inválidos.

Não há envio SMTP em `go test`.

## Pendências fora do código

- configurar domínio e caixa no iCloud+;
- configurar DNS;
- revisar templates habilitados no Dashboard Supabase antes de ativar Auth de cliente;
- validar recebimento real em todos os clientes alvo que ainda não tiverem sido conferidos;
- registrar evidências sem expor credenciais, tokens, PII ou links sensíveis.
