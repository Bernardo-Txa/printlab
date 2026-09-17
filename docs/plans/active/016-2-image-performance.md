# Fase 16.2 — Performance de imagens, cache estatico e acessibilidade

Status: em execucao; requer nova medicao manual em producao apos deploy.

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

## Limites e próxima validação

- JSON-LD continua adiado pela ambiguidade de variantes; `og:image` continua adiado até existir semântica canônica segura.
- O produto `produto-teste-frete` está ativo, portanto é corretamente incluído no sitemap. Antes da indexação/comercialização real, produtos temporários devem ser desativados pelo Admin, sem exceção por slug.
- Não houve migration, regra de negócio, mudança em pagamentos, frete, MFA, WAF, banco, plano Supabase ou configuração Vercel.
- Reexecutar Lighthouse nas mesmas três URLs em produção após deploy e registrar bytes, LCP e warnings de image delivery/cache. A fase não busca score 100 artificial.
