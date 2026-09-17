# ADR-0020 — SMTP transacional inicial via iCloud+

Status: Proposta

Data: 2026-09-17

## Contexto

A futura conta de cliente precisa de e-mails transacionais de autenticacao e conta. A PrintLab ja possui iCloud+, cujo custo incremental inicial esperado e zero, mas isso nao garante que seja a infraestrutura definitiva em escala.

## Decisao

Planejar para a Fase 18.1 o iCloud+ Custom Email Domain com `acesso@printlab3d.com.br` e remetente PrintLab, integrado ao Custom SMTP do Supabase Auth por `smtp.mail.me.com`, porta 587, TLS/autenticacao conforme requisitos Apple.

O uso inicial sera exclusivamente transacional para autenticacao e conta, sem marketing. A senha especifica de app da Apple sera secret do provider apropriado e nunca sera versionada, documentada, enviada ao frontend ou registrada em logs.

Antes de ativacao, a fase devera validar DNS fornecido pela Apple, SPF, DKIM, DMARC, From/Reply-To e entregabilidade em Gmail, Outlook e iCloud. Os templates habilitados do Supabase Auth receberao identidade visual PrintLab em pt-BR.

## Alternativas consideradas

- Contratar SMTP transacional agora: adiado; nao ha necessidade atual nem configuracao nesta tarefa.
- Usar iCloud+ como dependencia definitiva: rejeitado; volume, entregabilidade, limites, analytics e retries podem justificar migracao futura para provedor apropriado.

## Consequencias

- A troca futura de SMTP deve preservar a arquitetura de conta do cliente.
- Nenhum dominio, DNS, SMTP, segredo ou template foi configurado por esta ADR.

## Referencias

- [Roadmap](../plans/roadmap.md)
- [Supabase Auth](../integrations/supabase-auth.md)
