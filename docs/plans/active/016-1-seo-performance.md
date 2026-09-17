# Fase 16.1 — SEO tecnico e baseline de performance

Status: em execucao.

## Objetivo

Estabelecer descoberta tecnica segura para as paginas publicas SSR e uma linha de base de performance antes de qualquer otimizacao orientada por dados.

## Entregas desta subfase

- [x] Auditoria de rotas: `/`, `/produtos` e `/produtos/{slug}` sao indexaveis; carrinho, checkout, pedido, retorno de pagamento, acompanhamento, Admin, `/health` e `/ready` recebem `noindex`.
- [x] Canonical absoluto baseado somente em `SITE_URL` valido, sem confiar no header `Host`; filtro de categoria e variante selecionada canonicalizam para a URL-base correspondente.
- [x] Metadados basicos: `title`, `description`, canonical, Open Graph e Twitter card nas paginas publicas canonicas.
- [x] `GET /robots.txt` com referencia ao sitemap e prefixos de rotas privadas; robots nao e fronteira de seguranca.
- [x] `GET /sitemap.xml` com home, catalogo e slugs validos retornados pelo catalogo publico ativo; sem query strings, checkout, pedido ou Admin.
- [x] Falha segura do sitemap com HTTP 503 generico e evento operacional sem dados sensiveis.
- [x] Cache conservador de uma hora para assets estaticos embutidos; HTML dinamico nao recebeu cache publico novo.
- [x] Cards e thumbnails usam `loading="lazy"` e `decoding="async"`; a imagem principal do detalhe usa `loading="eager"`, `fetchpriority="high"` e o layout existente. JavaScript de checkout/Admin continua `defer` e especifico.
- [x] Testes de canonical, metadados, `noindex`, robots, sitemap (incluindo escaping XML), estrategia de imagens e cache estatico.

## Decisoes e limites

- `SITE_URL` ausente ou invalida nao gera canonical absoluto nem linha `Sitemap` em `robots.txt`; o sitemap responde 503. O runtime configurado deve fornecer uma origem HTTP(S) valida.
- O sitemap nao consulta repositorio proprio: reutiliza o service de catalogo, cuja semantica publica ja retorna somente produtos ativos.
- JSON-LD `Product` foi adiado. As variantes podem ter preco efetivo diferente e a pagina ainda nao possui um contrato de oferta canonica que evite markup incorreto. A implementacao futura deve incluir dados reais de oferta e testes antes de emitir schema.
- `og:image` tambem foi adiado: imagens podem vir de produto ou variante e a escolha deve respeitar a URL canonica, sem apontar para configuracao opcional por query string.
- Nao houve migration, alteracao de regra de negocio, mudanca de Vercel, Supabase, pagamentos, frete, MFA ou WAF.

## Baseline local

Em 2026-09-16, os artefatos embutidos medidos localmente foram:

| Recurso | Tamanho |
| --- | ---: |
| `web/static/css/app.css` | 115.804 bytes |
| `web/static/js/checkout.js` | 5.275 bytes |
| `web/static/js/admin-images.js` | 4.475 bytes |
| `web/static/images/branding/logo-printlab-primary.png` | 897.969 bytes |

Antes de otimizar imagens reais do catalogo, medir em ambiente publico com Lighthouse/WebPageTest pelo menos `/`, `/produtos` e um detalhe de produto com imagens: LCP, INP, CLS, TTFB, tamanho transferido e requests. Registrar data, URL, dispositivo/rede e resultado; nao assumir que uma imagem de Storage ou o logo e gargalo sem essa medicao.

## Validacao prevista

- `templ generate`
- `npm run css:build`
- `gofmt -w .`
- `go mod tidy`
- `go test ./...`
- `go test ./... -cover`
- `go vet ./...`
- `go build ./...`
- `govulncheck ./...`
- `npm audit`
