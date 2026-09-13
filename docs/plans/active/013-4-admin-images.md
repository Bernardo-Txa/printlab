# Fase 13.4 — Imagens e Supabase Storage

Status: implementacao em correcao; validacao real pendente.

## Objetivo

Permitir que o administrador gerencie imagens publicas do catalogo sem SQL Editor, Supabase Dashboard ou edicao manual de URLs.

## Schema encontrado

`public.product_images` ja possui as colunas necessarias para esta fase:

- `id`;
- `product_id`;
- `variant_id` opcional;
- `storage_path` relativo;
- `alt_text`;
- `sort_order`;
- `is_primary`;
- `created_at`.

O bucket existente `product-images` ja esta configurado pela migration da Fase 5 como publico para leitura, com limite de 5 MB e MIME types de imagem.

## Decisao de migration

Nenhuma migration foi criada.

Motivos:

- `product_images.storage_path` ja guarda o object path relativo necessario para associacao e remocao segura;
- `sort_order` e `is_primary` ja permitem ordenacao e imagem principal;
- a FK composta `(variant_id, product_id)` ja impede associacao de configuracao de outro produto;
- o bucket `product-images` ja existe.

## Escopo implementado

- Subpagina `GET /admin/produtos/{product_id}/imagens`.
- Autorizacao de upload direto em `POST /admin/produtos/{product_id}/imagens/upload-url`.
- Finalizacao de cadastro em `POST /admin/produtos/{product_id}/imagens/finalizar`.
- Autorizacao de substituicao em `POST /admin/produtos/{product_id}/imagens/{image_id}/substituir-url`.
- Finalizacao de substituicao em `POST /admin/produtos/{product_id}/imagens/{image_id}/finalizar-substituicao`.
- Remocao em `POST /admin/produtos/{product_id}/imagens/{image_id}/remover`.
- Ordenacao em `POST /admin/produtos/{product_id}/imagens/{image_id}/ordem`.
- Marcacao de imagem principal em `POST /admin/produtos/{product_id}/imagens/{image_id}/principal`.
- Provider `StorageClient` testavel para Supabase Storage via `net/http`.
- Suporte a `SUPABASE_SECRET_KEY` server-side.

## Fluxo de upload

1. Browser Admin envia ao Go apenas metadados: configuracao opcional, MIME e tamanho.
2. Go valida sessao Admin, `Origin`/`Referer`, produto, configuracao, MIME e tamanho.
3. Go gera path imprevisivel no formato `products/{product_uuid}/{random}.ext`.
4. Go solicita signed upload URL ao Supabase Storage usando `SUPABASE_SECRET_KEY`.
5. Browser envia bytes diretamente ao Supabase Storage.
6. Browser chama a finalizacao no Go.
7. Go valida novamente, confirma metadata do objeto no Storage por `GET /storage/v1/object/info/{bucket}/{path}` e grava `product_images`.

## Regras implementadas

- Aceita somente `image/jpeg`, `image/png` e `image/webp`.
- Nao aceita SVG, GIF, PDF, video ou arquivo generico.
- Extensao e definida pelo MIME validado, sem confiar no nome original.
- O limite usado pela aplicacao e 5 MB, alinhado ao bucket existente.
- Cada upload/substituicao cria path novo; nao ha overwrite.
- `Origin: null` permanece rejeitado nas mutacoes Admin.
- Mutacoes Admin sem `Origin` e sem `Referer` sao rejeitadas; `Referer` same-origin e apenas fallback quando `Origin` estiver ausente.
- Imagem especifica de configuracao exige que a configuracao pertenca ao produto.
- Falha de insert em `product_images` tenta cleanup best-effort do objeto recem-enviado.
- Falha de update em substituicao tenta cleanup best-effort apenas do objeto novo; o objeto antigo so e removido apos update bem-sucedido.
- Remocao apaga objeto fisico apenas quando o path e gerenciado no prefixo `products/{product_id}/`.
- Remocao de imagem gerenciada exige Storage administrativo configurado para evitar remover a associacao sem remover o objeto fisico.
- Imagens legadas/manuais fora desse prefixo podem ter associacao removida do banco, mas nao geram DELETE arbitrario.
- `admin_order_events` continua exclusivo para pedidos; auditoria de catalogo fica como evolucao futura.

## Fora do escopo

- Editor/crop/compressao avancada.
- Drag-and-drop ou bulk upload.
- Thumbnails persistidos multiplos.
- Bucket novo, S3, Cloudinary, Vercel Blob ou CDN proprio.
- Upload de SVG, PDF, video ou documentos.
- Auditoria generica de catalogo.
- Validacao real em producao da 13.4.
- Fase 14.

## Definition of Done

- Schema e bucket existentes inspecionados.
- Sem migration criada.
- Backend Admin, provider Storage, handlers e UI em correcao.
- Testes de config, service de imagens, provider Storage, handlers e static asset em ampliacao.
- Documentacao, roadmap, ADR, CHANGELOG e env docs atualizados.
- Validacoes locais executadas antes do commit.
- Commit e push automaticos conforme AGENTS.md.

## Pendencia

Apos deploy, configurar `SUPABASE_SECRET_KEY` na Vercel e validar com um produto real de desenvolvimento, sem registrar dados pessoais, secrets, token de signed upload ou UUID administrativo real.
