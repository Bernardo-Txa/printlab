# Fase 13.3 — Catalogo, variantes, materiais, cores e caixas

Status: implementacao concluida; validacao real pendente.

## Objetivo

Tornar o catalogo e os dados operacionais basicos da PrintLab administraveis pelo painel protegido existente.

## Escopo implementado

- Produtos em `GET /admin/produtos`, `GET /admin/produtos/novo`, `POST /admin/produtos`, `GET /admin/produtos/{product_id}` e `POST /admin/produtos/{product_id}`.
- Categorias em `GET /admin/categorias`, `GET /admin/categorias/nova`, `POST /admin/categorias`, `GET /admin/categorias/{id}` e `POST /admin/categorias/{id}`.
- Variantes em `GET /admin/produtos/{product_id}/variantes/nova`, `POST /admin/produtos/{product_id}/variantes`, `GET /admin/produtos/{product_id}/variantes/{variant_id}` e `POST /admin/produtos/{product_id}/variantes/{variant_id}`.
- Receita de producao em `POST /admin/produtos/{product_id}/variantes/{variant_id}/receita`, `POST /admin/produtos/{product_id}/variantes/{variant_id}/receita/{component_id}` e `POST /admin/produtos/{product_id}/variantes/{variant_id}/receita/{component_id}/remover`.
- Materiais em `GET /admin/materiais`, `GET /admin/materiais/novo`, `POST /admin/materiais`, `GET /admin/materiais/{id}` e `POST /admin/materiais/{id}`.
- Cores em `GET /admin/cores`, `GET /admin/cores/nova`, `POST /admin/cores`, `GET /admin/cores/{id}` e `POST /admin/cores/{id}`.
- Caixas em `GET /admin/caixas`, `GET /admin/caixas/nova`, `POST /admin/caixas`, `GET /admin/caixas/{id}` e `POST /admin/caixas/{id}`.

## Regras implementadas

- Sem hard delete de categorias, produtos, variantes, materiais, cores e caixas.
- `is_active` controla disponibilidade para novas ofertas, escolhas e cotacoes.
- Componentes de `variant_filaments` podem ser removidos porque pedidos antigos possuem snapshots historicos.
- Alteracoes de catalogo nao alteram pedidos historicos, carrinhos convertidos, snapshots de itens, snapshots de receita ou frete congelado.
- Preco Admin usa BRL amigavel e persiste centavos inteiros sem `float`.
- Peso de receita usa gramas na UI e persiste miligramas inteiros.
- Material/cor inativo permanece visivel em receita existente e nao aparece em novas escolhas.
- Perfil logistico de produto e variante e atomico: completo ou ausente.
- Variante default precisa estar ativa; troca de default remove as demais na mesma transacao.
- Desativar variante default remove `is_default` sem escolher outra automaticamente.
- Caixa valida dimensoes positivas, peso positivo, `sort_order >= 0` e dimensoes externas maiores ou iguais as internas.
- Violações conhecidas de unique viram mensagens amigaveis.
- Erros inesperados de banco exibem mensagem generica.

## Seguranca

- Todas as paginas Admin preservam `Cache-Control: private, no-store`, `X-Robots-Tag: noindex, nofollow, noarchive` e `Referrer-Policy: same-origin`.
- Todas as mutacoes usam POST, sessao administrativa obrigatoria e validacao centralizada de `Origin`/`Referer`.
- `Origin: null` continua rejeitado independentemente de `Referer`.
- Formularios administrativos possuem limite de body de 256 KiB.
- Nao ha upload de arquivos, Supabase Storage, `SUPABASE_SECRET_KEY` ou service role nesta fase.
- Catalogo Admin nao consulta PII de cliente, dados InfinitePay, URL de checkout, UUID interno de pedido ou eventos de pedido.

## Fora do escopo

- Upload, remocao, reordenacao ou definicao de imagem primaria.
- Supabase Storage administrativo.
- Estoque fisico de filamento, marcas, lotes, carretel, custo por kg e reserva de material.
- Custos calculados.
- Etiqueta SuperFrete, codigo de rastreamento ou postagem externa.
- Novos meios de pagamento.
- Multiplos admins, papeis e permissoes granulares.
- Auditoria de catalogo.
- Alteracoes em pedidos historicos.

## Definition of Done

- Rotas Admin implementadas: concluido.
- Templates SSR sem JavaScript obrigatorio: concluido.
- Service e repository Admin sobre tabelas existentes: concluido.
- Testes de parsers, validacoes, handlers e regressao de repository: concluido.
- Documentacao de produto, schema, seguranca, roadmap, plano e ADR atualizada: concluido.
- Nenhuma migration criada: concluido.
- Validacoes locais executadas: pendente ate fechamento do commit.
- Validacao real em producao: pendente.
