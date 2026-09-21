# E-mail transacional PrintLab

Status: fundação de código implementada na Fase 18.1; envio real e Custom SMTP ainda dependem de configuração externa e validação manual.

## Objetivo

A PrintLab usará e-mails transacionais com identidade própria para autenticação futura de clientes e comunicação de pedidos. Esta fase prepara templates e documentação; ela não envia e-mail SMTP real durante testes, não altera checkout, não cria login de cliente e não muda Supabase Auth em produção.

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

Supabase Auth está implementado somente para o Admin:

- `POST /admin/login` autentica e-mail/senha no Supabase Auth;
- a PrintLab autoriza somente por `ADMIN_SUPABASE_USER_ID`, nunca por e-mail;
- MFA TOTP é obrigatório para criar a sessão administrativa própria;
- não há signup, login de cliente, Magic Link de cliente, Minha Conta ou recuperação pública de acesso implementados.

A Fase 18.1 não altera esse comportamento. Os templates de autenticação criados agora são fundação para fases futuras.

## Configuração externa necessária

Checklist antes de habilitar envio real:

- [ ] domínio configurado no iCloud+ Custom Email Domain;
- [ ] endereço `acesso@printlab3d.com.br` criado;
- [ ] DNS exigido pelo iCloud+ configurado;
- [ ] SPF configurado;
- [ ] DKIM configurado;
- [ ] DMARC configurado;
- [ ] Custom SMTP configurado no Supabase Auth;
- [ ] SMTP host definido no Supabase;
- [ ] SMTP port definido no Supabase;
- [ ] SMTP username definido no Supabase;
- [ ] SMTP password definido como secret externo, nunca no Git;
- [ ] sender name `PrintLab` configurado;
- [ ] sender email `acesso@printlab3d.com.br` configurado;
- [ ] reply-to configurado;
- [ ] URLs de redirect do Supabase Auth revisadas para o domínio real;
- [ ] e-mail de teste recebido;
- [ ] Gmail validado;
- [ ] Outlook validado;
- [ ] iCloud Mail validado.

Não afirmar que SMTP, DNS ou entregabilidade estão funcionando até que esses itens sejam concluídos e registrados.

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
- configurar Custom SMTP no Supabase Auth;
- revisar templates habilitados no Dashboard Supabase antes de ativar Auth de cliente;
- validar recebimento real em Gmail, Outlook e iCloud Mail;
- registrar evidências sem expor credenciais, tokens, PII ou links sensíveis.
