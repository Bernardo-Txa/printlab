# Fase 16.2 — Performance de imagens, cache estatico e acessibilidade

Status: concluida; validada em producao.

## Resultado apos a primeira rodada de branding

| URL | Performance anterior → atual | Accessibility anterior → atual | Best Practices / SEO |
| --- | --- | --- | --- |
| `/` | 80 → 100 | 91 → 95 | 100 / 100 |
| `/produtos` | 75 → 77 | 100 → 100 | 100 / 100 |
| `/produtos/produto-teste-frete` | 75 → 79 | 100 → 100 | 100 / 100 |

Branding local deixou de ser gargalo relevante. SSR, JavaScript e CSS não justificam refactor nesta etapa; as imagens públicas de produto no Storage são o alvo restante.

## Validacao final em producao

| URL | Performance | Accessibility | Best Practices | SEO |
| --- | ---: | ---: | ---: | ---: |
| `/` | 100 | 100 | 100 | 100 |
| `/produtos` | 100 | 100 | 100 | 100 |
| `/produtos/produto-teste-frete` | 100 | 100 | 100 | 100 |

Na observacao final com cache desabilitado, o catálogo transferiu aproximadamente 45,9 KB (cerca de 150 KB de recursos), incluindo imagem WebP de produto de aproximadamente 20,8 KB. O detalhe transferiu aproximadamente 45,4 KB (cerca de 150 KB de recursos), incluindo imagem principal WebP de aproximadamente 21 KB. Sao resultados observados nessa validacao, nao garantias universais.

O Admin foi validado com imagem 4K de aproximadamente 142 MB: o processamento client-side executou, enviou WebP otimizado e preservou qualidade visual adequada no catálogo e no detalhe. Nenhuma Image Transformation paga foi usada; o backend continuou validando MIME, tamanho e objeto no Storage.

## Patch final de contraste da Home

O Lighthouse identificou contraste insuficiente somente nos parágrafos dos cards `dna-card-blue` e `dna-card-pink`. A cor anterior era `#071c36` com opacidade de 75%, composta sobre os fundos. O patch usa `#071c36` opaco no azul (4,72:1 sobre `#1187f4`) e o novo token navy `#000f20` no rosa (4,53:1 sobre `#e42b7b`), ambos WCAG AA para texto normal. Fundos, tipografia, tamanho e identidade visual permanecem inalterados. A validacao final da Home confirmou Accessibility 100.

## Baseline de producao (2026-09-16)

| URL | Performance | Accessibility | SEO | LCP | Diagnostico principal |
| --- | ---: | ---: | ---: | ---: | --- |
| `/` | 80 | 91 | 100 | 5,4 s | logo local, cache e contraste/ARIA |
| `/produtos` | 75 | 100 | 100 | 11,2 s | logo local e imagem de produto do Storage |
| `/produtos/produto-teste-frete` | 75 | 100 | 100 | 10,6 s | logo local e imagem principal do Storage |

TTFB/FCP aparentes estavam bons, TBT foi 0–20 ms e CLS foi 0. CSS render-blocking (110–230 ms) fica fora desta entrega enquanto P0/P1 nao forem reavaliados.

## Inventario de imagens carregadas

| Pagina | Recurso | Origem | Formato/dimensoes | Bytes | Papel |
| --- | --- | --- | --- | ---: | --- |
| `/` | `logo-printlab-primary.png` | estatico local | PNG, 1448×1086 | 897.969 | hero/LCP; repetida em header, footer e favicon pelo mesmo URL |
| `/produtos` | `logo-printlab-primary.png` | estatico local | PNG, 1448×1086 | 897.969 | header, footer e favicon; repetida pelo mesmo URL |
| `/produtos` | `999b…f20.png` | Supabase Storage | PNG, 1920×520 | 1.014.148 | card lazy do produto teste |
| `/produtos/produto-teste-frete` | `logo-printlab-primary.png` | estatico local | PNG, 1448×1086 | 897.969 | header, footer e favicon; repetida pelo mesmo URL |
| `/produtos/produto-teste-frete` | `bcfaa…921.png` | Supabase Storage | PNG, 1448×1086 | 897.969 | imagem principal/LCP do produto |

As URLs públicas do Storage observadas respondem `Cache-Control: no-cache`. A arquitetura atual apenas monta URLs de objetos públicos; não há evidência nesta subfase de Image Transformations gratuitas ou de variantes reais no provider. Portanto não houve mudança em upload, transformação, provider ou arquivos existentes do Storage.

## Entregas aplicadas

- [x] Mantido o PNG original como master no repositório.
- [x] Criadas variantes locais versionadas: hero WebP 640×480 (16.118 bytes), logo pequena WebP 256×192 (5.990 bytes) e favicon PNG 64×48 (2.977 bytes).
- [x] Hero usa variante adequada com dimensões explícitas, `fetchpriority="high"` e `decoding="async"`; header/footer usam a variante pequena e favicon usa arquivo próprio.
- [x] Assets com sufixo `-v1` recebem `Cache-Control: public, max-age=31536000, immutable`; arquivos mutáveis, como `app.css` e o PNG master, permanecem em uma hora.
- [x] Removidos ARIA labels redundantes de `div`s da Home e corrigido o contraste do texto introdutório no hero.
- [x] Não há `srcset` artificial: variantes locais atendem usos distintos e imagens do Storage não possuem variantes reais.
- [x] Novos uploads administrativos usam, quando a API do navegador estiver disponível, canvas client-side para preservar proporção/orientação, limitar largura a 1200 px sem ampliar imagens menores e converter para WebP quality 0,82. Falha legítima mantém o arquivo original; validações server-side continuam obrigatórias.
- [x] O upload direto envia cache longo para objeto novo de caminho aleatório e não faz upsert; replacement recebe outro caminho antes de remover o anterior, portanto não reutiliza URL com conteúdo diferente.

## Limites e próxima validação

- JSON-LD continua adiado pela ambiguidade de variantes; `og:image` continua adiado até existir semântica canônica segura.
- O produto `produto-teste-frete` está ativo, portanto é corretamente incluído no sitemap. Antes da indexação/comercialização real, produtos temporários devem ser desativados pelo Admin, sem exceção por slug.
- Não houve migration, regra de negócio, mudança em pagamentos, frete, MFA, WAF, banco, plano Supabase ou configuração Vercel.
- A fase foi encerrada com Lighthouse 100 nas tres URLs e validacao manual do Admin. Imagens existentes não foram alteradas em lote.
