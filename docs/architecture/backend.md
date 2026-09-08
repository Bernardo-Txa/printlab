# Backend

Status: fundacao minima IMPLEMENTADA; funcionalidades de negocio PLANEJADAS.

## Responsabilidade

O backend Go sera a camada autoritativa da aplicacao. Ele recebera requisicoes HTTP, validara entradas, aplicara regras de negocio, acessara o PostgreSQL e renderizara respostas server-side.

Nesta fase, o backend implementa apenas `GET /health`.

## Limites

- Nao ha acesso a banco implementado.
- Nao ha produtos, carrinho, checkout, pedidos, pagamentos ou admin.
- Nao ha integracoes externas.

## Decisoes

- Usar `net/http` como base HTTP.
- Manter dependencias externas fora do projeto ate haver necessidade real.
- Separar areas futuras em `internal/`, sem codigo artificial.
- Centralizar regras financeiras no backend.

## Praticas recomendadas

- Handlers pequenos e explicitos.
- Validacao de entrada antes de chamar regras de dominio.
- Erros tratados explicitamente.
- `context.Context` quando a operacao envolver I/O, banco, chamadas externas ou cancelamento.
- Testes deterministico para regras criticas.
- Pacotes coesos por responsabilidade.

## Praticas proibidas

- Confiar em valores financeiros vindos do navegador.
- Adicionar dependencia sem justificativa.
- Criar interfaces prematuras sem multiplos consumidores ou necessidade clara de teste.
- Implementar integracoes reais sem plano aprovado.
- Colocar secrets no codigo, testes ou documentacao.
