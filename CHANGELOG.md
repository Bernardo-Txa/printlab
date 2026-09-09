# Changelog

Este arquivo segue a ideia de [Keep a Changelog](https://keepachangelog.com/), com secoes organizadas por versao.

## [Unreleased]

### Added

- Fundacao inicial do projeto.
- Estrutura documental.
- Arquitetura inicial.
- Fase 2 concluida com homepage server-side usando `templ`.
- Pipeline Tailwind CSS 4 via CLI npm.
- Design tokens iniciais e componentes visuais fundamentais.
- Servico de assets estaticos em `/static/`.
- Testes de homepage, health check, static CSS e rotas desconhecidas.
- Integracao da logo oficial inicial da PrintLab ao header e hero.
- Teste para entrega do asset de marca em `/static/images/branding/logo-printlab-primary.png`.
- Fase 2.1 — Brand Experience, com homepage mais editorial e linguagem grafica da marca.
- Workflow de GitHub Actions para migrations Supabase de desenvolvimento com dry-run antes da aplicacao.

### Changed

- Versao minima de Go atualizada para 1.26.0.
- Assets estaticos passaram a ser servidos via `embed.FS` para melhorar compatibilidade com deploy na Vercel.
- Tokens de design refinados com base na paleta visual da marca.
- Homepage revisada para remover copy tecnica e ampliar presenca estrutural das cores da PrintLab.
