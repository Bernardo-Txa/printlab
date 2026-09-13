# ADR-0016 — Upload direto administrativo para Supabase Storage

Status: aprovado.

## Contexto

A Fase 13.4 precisa permitir que o Admin gerencie imagens publicas do catalogo no bucket `product-images`.

Enviar o arquivo inteiro pelo backend Go/Vercel aumenta custo operacional e esbarra no limite de payload das Functions. Ao mesmo tempo, o navegador nao deve escolher livremente bucket/path nem receber credencial administrativa.

## Decisao

A PrintLab usara upload direto ao Supabase Storage com autorizacao server-side:

```text
Browser Admin
  -> Go Admin: metadados do arquivo
  -> Go valida sessao, Origin/Referer, produto, configuracao, MIME e tamanho
  -> Go gera object path seguro e solicita signed upload URL ao Supabase
Browser Admin
  -> Supabase Storage: bytes do arquivo
Browser Admin
  -> Go Admin: finalizacao
  -> Go confirma objeto e grava public.product_images
```

`SUPABASE_SECRET_KEY` fica somente no backend. Ela bypassa RLS e por isso nunca e autorizacao do usuario: a autorizacao real continua sendo a sessao Admin da PrintLab, vinculada a `ADMIN_SUPABASE_USER_ID`.

## Consequencias

- O backend nao recebe os bytes da imagem.
- Signed upload URL/token pode ser entregue ao Admin autenticado, pois e temporario e limitado ao path gerado pelo servidor.
- O cliente nao escolhe bucket nem path.
- Cada upload/substituicao usa path novo, evitando overwrite e problemas de cache/CDN.
- A finalizacao revalida produto, configuracao, path, MIME e tamanho antes de gravar `product_images`.
- Falha de insert tenta cleanup best-effort do objeto recem-enviado.
- Remocao fisica ocorre somente para paths gerenciados no prefixo `products/{product_id}/`.
- Objetos legados/manuais fora desse prefixo nao sao deletados por tentativa de inferencia.

## Alternativas rejeitadas

- Enviar o arquivo pelo Go/Vercel: rejeitado pelo limite de payload e por eficiencia.
- Criar outro bucket: rejeitado porque `product-images` ja existe para imagens publicas de catalogo.
- Usar service role antiga como primeira opcao: rejeitado; a fase usa a nova `SUPABASE_SECRET_KEY` com prefixo `sb_secret_`.
- Criar auditoria de catalogo nesta fase: rejeitado para evitar sistema amplo fora do escopo; `admin_order_events` permanece exclusivo para pedidos.
