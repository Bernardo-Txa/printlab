# Fase 17.1 — Cadastro e slugs

Status: em execução; aguardando validação manual em produção.

Esta fase torna o slug uma responsabilidade do servidor no cadastro administrativo. Produtos, categorias, configurações (variantes), materiais, cores e caixas recebem um slug derivado do nome com canonicalização ASCII. Quando o candidato já existe, o servidor tenta sufixos determinísticos (`-2`, `-3` e assim por diante), limitados a 100 tentativas; a constraint `UNIQUE` do PostgreSQL continua sendo a autoridade final em condições de concorrência.

Renomeações preservam o slug persistido e, portanto, as URLs existentes. O fluxo comum do Admin não exibe um campo editável de slug: em edições o valor atual aparece apenas como informação. Um slug inválido ou um nome formado somente por símbolos produz erro no campo nome.

O tratamento cobre as constraints existentes, inclusive a unicidade de variantes no escopo do produto. Erros de unicidade de SKU ou outras falhas não são convertidos em colisões de slug. Nenhum slug existente é recalculado em lote e esta fase não cria migration.

Fases 17.2 e 17.3 permanecem planejadas e não fazem parte deste plano.
