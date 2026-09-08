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

### Changed

- Versao minima de Go atualizada para 1.26.0.
- Assets estaticos passaram a ser servidos via `embed.FS` para melhorar compatibilidade com deploy na Vercel.
