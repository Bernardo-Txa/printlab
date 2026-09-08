# Pedidos

Status: PLANEJADO.

Pedidos representarao compras realizadas ou iniciadas na PrintLab.

## Comportamento planejado

- Criar pedidos a partir de dados validados pelo backend.
- Persistir itens do pedido com valores calculados server-side.
- Registrar status de pedido.
- Relacionar pagamento e envio ao pedido.
- Permitir acompanhamento pelo cliente quando a funcionalidade for aprovada.

## Autoridade de estado

Status de pedido e pagamento nao devem ser definidos pelo navegador. Mudancas relevantes devem ocorrer por regras do backend, operacao administrativa autorizada ou eventos externos validados.

## Limites

- Nao ha pedidos implementados nesta fase.
- Nao ha maquina de estados aprovada.
- Nao ha regras finais de cancelamento, reembolso ou producao.
