# Fase 18.1 — E-mail transacional PrintLab

Status: Implementada no código; aguardando configuração externa e validação manual de SMTP/entregabilidade.

## Objetivo

Preparar a fundação dos e-mails transacionais da PrintLab para autenticação futura de clientes e comunicação de pedidos, sem implementar login de cliente, Minha Conta, alteração de checkout, SuperFrete, InfinitePay ou envio SMTP real nesta fase.

## Escopo implementado

- Pacote `internal/email` com renderer de mensagens transacionais HTML e texto simples.
- Remetente planejado centralizado: `PrintLab <acesso@printlab3d.com.br>` e reply-to planejado no mesmo endereço.
- Templates preparados:
  - confirmação de cadastro;
  - Magic Link;
  - recuperação de acesso;
  - confirmação de pedido;
  - atualização de status do pedido;
  - pagamento confirmado.
- Layout de e-mail compatível com clientes comuns, usando tabelas, estilos inline, CTA com fallback textual e identidade visual PrintLab.
- E-mails de pedido usam somente dados comerciais necessários e estado persistido; retirada mostra “Retirada no local” e “Grátis”.
- Testes de renderização, assunto, preheader, links, pedidos com envio/retirada e ausência de dados internos/secrets.

## Limites preservados

- Nenhum envio SMTP real.
- Nenhuma migration.
- Nenhuma alteração em checkout, pedidos, pagamentos, SuperFrete, InfinitePay, Auth funcional ou Admin.
- Nenhum segredo, senha SMTP, token ou API key versionado.
- Nenhum login, signup, Magic Link funcional, Minha Conta ou recuperação de acesso pública implementados.

## Configuração externa pendente

- Domínio e endereço `acesso@printlab3d.com.br` no iCloud+ Custom Email Domain.
- DNS, SPF, DKIM e DMARC.
- Custom SMTP no Supabase Auth.
- Sender name, sender email e reply-to.
- Teste real de recebimento e validação em Gmail, Outlook e iCloud Mail.

## Validação técnica

- `go test ./internal/email`
- `templ generate`
- `gofmt`
- `go test ./...`
- `go vet ./...`
- `go build ./...`
- `git diff --check`

## Validação manual pendente

A infraestrutura externa ainda não foi configurada nem validada. Não declarar envio transacional validado em produção até haver configuração SMTP real e recebimento confirmado.
