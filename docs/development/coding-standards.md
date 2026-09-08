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

## Dependencias

Antes de adicionar dependencia:

- confirme a necessidade real;
- avalie se a biblioteca padrao resolve;
- registre justificativa quando a dependencia afetar arquitetura;
- considere manutencao, seguranca e custo operacional.

## Escopo

Nao implemente funcionalidades fora da tarefa atual. Nao faca refactors amplos sem necessidade clara.
