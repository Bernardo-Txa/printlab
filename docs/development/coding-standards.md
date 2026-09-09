# Padroes de codigo

Status: aprovado como diretriz inicial.

## Go

- `gofmt` e obrigatorio.
- Nomes devem seguir convencoes idiomaticas de Go.
- Erros devem ser tratados explicitamente.
- `context.Context` deve ser usado quando apropriado, especialmente em I/O, banco, chamadas externas e operacoes cancelaveis.
- Dependencias devem ser explicitas.
- Globals mutaveis devem ser evitados.
- Abstracoes prematuras devem ser evitadas.
- Interfaces de um unico consumidor devem ser evitadas quando nao houver necessidade clara.
- Funcoes devem ser pequenas e focadas.
- Pacotes devem ter responsabilidades claras e coesas.
- Comentarios devem explicar o porquê, nao o obvio.
- TODO deve conter contexto suficiente para acao futura.

## Banco

- Usar `pgx/v5` e `pgxpool` para PostgreSQL.
- Nao usar `database/sql`, `lib/pq`, GORM, SQLX, ORM ou query builder sem nova decisao arquitetural.
- `DATABASE_URL` deve ser lida da configuracao e nunca logada.
- `DB_MAX_CONNS` deve limitar conexoes por instancia.
- Migrations ficam fora do startup da aplicacao.
- Testes de integracao devem usar `TEST_DATABASE_URL`, nunca `DATABASE_URL` de producao automaticamente.

## Dependencias

Antes de adicionar dependencia:

- confirme a necessidade real;
- avalie se a biblioteca padrao resolve;
- registre justificativa quando a dependencia afetar arquitetura;
- considere manutencao, seguranca e custo operacional.

## Escopo

Nao implemente funcionalidades fora da tarefa atual. Nao faca refactors amplos sem necessidade clara.

## Git workflow permanente

O fluxo padrao do projeto e:

```text
IMPLEMENTAR -> VALIDAR -> COMMIT -> PUSH
```

Antes de alterar arquivos, execute:

```sh
git status --short
git branch --show-current
```

Use esse estado inicial para identificar a branch atual, alteracoes preexistentes e arquivos fora do escopo. Nunca sobrescreva, descarte ou inclua alteracoes preexistentes nao relacionadas.

Toda tarefa deve ficar limitada ao escopo solicitado, com documentacao aplicavel atualizada na mesma alteracao. Antes do commit, execute as validacoes obrigatorias aplicaveis, por exemplo:

```sh
templ generate
npm run css:build
gofmt -w .
go test ./...
go vet ./...
go build ./...
```

Se uma validacao falhar por causa da tarefa, corrija antes de commitar. Se uma validacao nao puder ser executada por limitacao externa do ambiente, registre a limitacao claramente e use julgamento conservador; nunca faca push de codigo sabidamente quebrado.

Stage somente arquivos pertencentes a tarefa atual. Evite `git add .` e `git add -A` quando houver qualquer chance de misturar alteracoes preexistentes ou nao relacionadas. Antes do commit, revise:

```sh
git diff
git diff --cached
git status --short
```

Depois que as validacoes aplicaveis passarem, faca commit automaticamente, sem pedir confirmacao, usando mensagem clara e preferencialmente Conventional Commits, como `feat: ...`, `fix: ...`, `docs: ...`, `test: ...`, `refactor: ...` ou `chore: ...`. Uma tarefa logica normalmente deve resultar em um unico commit. Nao use `git commit --no-verify`.

Depois de um commit bem-sucedido, faca `git push` automaticamente para o upstream da branch atual. Se nao houver upstream, use `git push -u origin <branch-atual>` somente se `origin` apontar para o repositorio esperado da PrintLab:

```text
https://github.com/Bernardo-Txa/printlab
```

A forma SSH equivalente tambem e valida. Antes do primeiro push de uma tarefa, valide `git remote -v`. Se `origin` apontar para outro repositorio, nao faca push.

Nao execute automaticamente `git push --force`, `git push --force-with-lease`, `git reset --hard`, `git rebase` ou `git commit --amend`. Essas operacoes exigem instrucao explicita. Se o push normal for rejeitado porque a branch remota avancou, nao faca force push; resolva somente de forma nao destrutiva quando for seguro.

Neste estagio do projeto, commit e push diretos em `main` sao permitidos. Se no futuro houver Pull Requests obrigatorios ou branch protection, esta politica deve ser atualizada.

Pushes para GitHub podem disparar deploys da Vercel, GitHub Actions e migrations do Supabase quando houver alteracoes em `supabase/migrations/**` ou `supabase/config.toml`. Nao altere migrations remotas apenas para testar pipeline, nao faca commit/push de migration sem revisar impacto e nao inclua migration destrutiva inadvertidamente.

Antes de commit/push envolvendo configuracao, banco ou integracoes, procure possiveis secrets quando aplicavel:

```sh
rg -n "postgres://|postgresql://|password=|token=|DATABASE_URL=|SUPABASE_SERVICE_ROLE_KEY="
```

Diferencie placeholders documentais de credenciais reais. Secrets reais, tokens, passwords, connection strings reais, arquivos temporarios e dados privados nunca entram no Git.

Ao final, o relatorio deve incluir branch, hash curto do commit, mensagem do commit, resultado do push, `git status --short` e remote utilizado. O estado ideal final e working tree clean, salvo alteracoes preexistentes do responsavel que nao pertencam a tarefa.
