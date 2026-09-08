# AGENTS.md

Este arquivo e um mapa rapido para agentes que trabalharem no projeto PrintLab.

Antes de trabalhar:

- Leia [README.md](README.md).
- Leia [ARCHITECTURE.md](ARCHITECTURE.md).
- Leia a documentacao especifica da area alterada em [docs/](docs/).

Referencias obrigatorias por area:

- Banco: [docs/architecture/database.md](docs/architecture/database.md), [docs/database/schema.md](docs/database/schema.md) e [docs/database/migrations.md](docs/database/migrations.md).
- Frontend: [docs/architecture/frontend.md](docs/architecture/frontend.md).
- Backend: [docs/architecture/backend.md](docs/architecture/backend.md).
- Seguranca: [docs/architecture/security.md](docs/architecture/security.md).
- Integracoes: documentos correspondentes em [docs/integrations/](docs/integrations/).

Regras obrigatorias:

- Codigo + testes aplicaveis + documentacao = tarefa concluida.
- Se codigo mudar comportamento documentado, atualize a documentacao na mesma alteracao.
- Nao mude arquitetura silenciosamente.
- Mudancas arquiteturais relevantes exigem ADR em [docs/decisions/](docs/decisions/).
- Nao implemente funcoes fora do escopo da tarefa atual.
- Nao faca refactors amplos sem necessidade.
- Nao apague documentacao sem justificativa.
- Nao adicione dependencias sem justificar.
- Execute testes e validacoes antes de considerar uma tarefa concluida.
