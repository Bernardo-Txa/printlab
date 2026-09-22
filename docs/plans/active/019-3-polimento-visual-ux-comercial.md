# Fase 19.3 — Polimento visual e UX comercial

Status: Em execução

## Objetivo

Fazer com que toda a experiência PrintLab acompanhe a personalidade já presente na Home:

- criativa;
- colorida;
- tecnológica;
- profissional;
- amigável;
- consistente.

Não realizar redesign gratuito.

Reutilizar e consolidar a identidade existente da PrintLab:

- navy/azul escuro;
- azul vivo;
- magenta;
- ciano/turquesa;
- amarelo/laranja;
- cards arredondados;
- sombras suaves;
- contraste forte;
- elementos visuais criativos sem comprometer legibilidade.

A Home será a principal referência visual interna.

## 19.3.1 — Header, navegação e identidade global

Revisar o header/navigation bar global.

Problema atual: o topo possui pouca presença visual e não valoriza suficientemente a identidade PrintLab.

Direção visual: aproximar a sensação do header de referência fornecido pelo responsável:

- fundo navy/azul profundo;
- contraste maior;
- logo valorizada;
- busca integrada visualmente;
- navegação limpa;
- item ativo mais perceptível;
- conta e carrinho claramente identificáveis;
- aparência tecnológica/premium;
- funcionamento excelente em desktop e mobile.

Não copiar literalmente outro site.

Usar a referência apenas como direção visual.

Também revisar:

- hover;
- focus-visible;
- estados ativos;
- badge do carrinho;
- menu mobile;
- espaçamentos;
- alinhamentos;
- tamanho de ícones;
- contraste.

## 19.3.2 — WhatsApp e CTAs comerciais

Todos os CTAs cujo significado seja:

- Entre em contato;
- Fale conosco;
- Solicitar contato;
- Tirar dúvida;

devem direcionar para o WhatsApp oficial PrintLab:

`+55 27 99859-5125`

URL base:

`https://wa.me/5527998595125`

Usar mensagem pré-preenchida amigável.

Texto inicial sugerido:

`Olá! Vim pelo site da PrintLab 👋 Quero saber mais sobre um produto ou impressão 3D personalizada. Pode me ajudar?`

Quando fizer sentido, contextualizar a mensagem.

Exemplos:

orçamento:

`Olá! Vim pelo site da PrintLab e gostaria de fazer um orçamento de impressão 3D.`

produto personalizado:

`Olá! Vim pelo site da PrintLab e gostaria de criar uma peça personalizada em impressão 3D.`

Não espalhar números/URLs duplicados desnecessariamente pelo código.

Na futura implementação, preferir uma fonte/helper central quando arquiteturalmente adequado.

Revisar hierarquia dos CTAs:

- Comprar;
- Ver produtos;
- Pedir orçamento;
- Personalizar;
- WhatsApp;
- Acompanhar pedido;
- Retomar pagamento.

Não transformar todas as ações em botão primário.

## 19.3.3 — Consistência visual global

Levar a linguagem da Home para páginas internas.

Auditar pelo menos:

- Home;
- Produtos;
- Detalhe do produto;
- Carrinho;
- Checkout;
- Minha Conta;
- Login;
- Cadastro;
- Recuperação de senha;
- Pedido;
- Acompanhamento;
- páginas institucionais;
- contato;
- footer.

Padronizar:

- headings;
- subtítulos;
- textos auxiliares;
- cards;
- bordas;
- sombras;
- badges;
- botões;
- inputs;
- alerts;
- espaçamentos;
- divisores;
- ícones;
- estados hover/focus/active.

Evitar páginas visualmente neutras que pareçam pertencer a outro sistema.

Não transformar todas as páginas em cópias da Home.

Preservar diferenças funcionais entre catálogo, checkout, conta e Admin.

## 19.3.4 — Minha Conta

Status: validada visualmente em produção pelo responsável.

A área `/conta` precisa de revisão visual completa.

Funcionalidades existentes devem ser preservadas:

- perfil;
- endereço;
- pedidos;
- status;
- acompanhamento;
- retomada de pagamento;
- logout.

Objetivo: transformar Minha Conta em uma área de cliente profissional.

Revisar:

- arquitetura visual;
- hierarquia;
- cards;
- status;
- pedidos;
- ações;
- dados pessoais;
- endereço;
- segurança;
- responsividade.

Estrutura conceitual desejada:

- Visão geral;
- Meus pedidos;
- Dados pessoais;
- Endereço;
- Segurança/conta.

NÃO obrigar criação de subpáginas antecipadamente.

Durante a implementação, auditar se a melhor solução é:

A) página única bem organizada;

ou

B) subpáginas como:

`/conta`

`/conta/pedidos`

`/conta/dados`

`/conta/seguranca`

Decisão desta fase: manter uma dashboard única em `/conta`, sem criar subpáginas. O volume atual de funcionalidades ainda não justifica rotas separadas para pedidos, dados e segurança; a página única recebeu navegação interna por âncoras reais para preparar uma separação futura se a conta crescer.

Criar subpáginas somente se trouxer vantagem clara de organização, navegação mobile e escalabilidade em fase futura.

Não criar complexidade apenas por estética.

Pedidos devem ter melhor apresentação visual, incluindo:

- número;
- data;
- total;
- status;
- acompanhamento;
- retomada de pagamento quando aplicável.

Status devem utilizar badges claros e acessíveis.

Implementado nesta subfase:

- hero próprio da conta com identidade PrintLab;
- navegação interna para Visão geral, Pedidos, Meus dados e Segurança;
- cards de resumo sem novas consultas;
- cards próprios de pedidos com status, acompanhamento e retomada por POST;
- empty state de pedidos com CTA para produtos;
- formulário de dados agrupado em Dados pessoais e Endereço de entrega;
- refinamento visual final de Dados pessoais e Endereço de entrega com headers iconográficos, grids próprios e ações do formulário integradas à dashboard;
- seção Segurança e conta com e-mail, status, recuperação de senha e logout por POST.

## 19.3.5 — Carrinho e checkout

Status: implementada; aguardando validação visual manual.

O carrinho atual é visualmente satisfatório.

Preservar os elementos que já funcionam bem.

O maior problema está nas etapas:

- Dados;
- Frete;
- Revisão;
- Pagamento.

Reformular a experiência visual sem alterar as regras de negócio.

Objetivos:

- aparência mais premium;
- menos sensação de "formulário administrativo";
- mais identidade PrintLab;
- hierarquia clara;
- progressão do checkout óbvia;
- menor carga cognitiva.

Revisar o stepper:

1. Dados
2. Frete
3. Revisão
4. Pagamento

Implementado nesta subfase:

- stepper reutilizável com estados concluída, atual e futura, `aria-current="step"` e nomenclatura Dados, Entrega, Revisão e Pagamento;
- etapa Dados com cards de Dados pessoais e Endereço de entrega, grids próprios, ícones outline, máscaras e ViaCEP preservados;
- etapa Entrega com cards acessíveis para receber em casa, retirada no local e cotações SuperFrete, mantendo GET/POST e `delivery_method`;
- etapa Revisão com blocos de Produtos, Dados pessoais, Endereço de entrega e Entrega, links discretos de edição e CTA Confirmar pedido;
- etapa Pagamento com status do pedido, card de InfinitePay por POST e ação secundária de acompanhamento;
- resumo lateral com sticky offset compatível com header sticky, destaque de total e leitura mobile;
- ajustes de mobile, foco e contraste sem redesenhar `/carrinho`;
- testes de stepper, formulários, radios, revisão, pagamento por POST e não exposição de campos autoritativos.

Estados desejados:

- futura;
- ativa;
- concluída;
- erro quando relevante.

Etapas concluídas podem receber check visual.

Revisar:

- cabeçalho da etapa;
- cards;
- formulário;
- labels;
- inputs;
- ajuda contextual;
- erros;
- mensagens;
- resumo;
- frete;
- retirada;
- revisão;
- pagamento.

No desktop, avaliar resumo do pedido sticky quando não prejudicar layout/acessibilidade.

No mobile, priorizar leitura e ação principal.

Preservar integralmente:

- validação server-side;
- CEP;
- máscaras;
- SuperFrete;
- pickup;
- InfinitePay;
- guest checkout;
- customer profile;
- snapshots;
- segurança;
- ownership;
- payment_check.

Não alterar regra financeira durante trabalho visual.

## 19.3.6 — Revisão textual pt-BR

Executar nova varredura em todas as strings visíveis.

Foram identificados textos sem acentuação adequada, por exemplo:

- Precisao -> Precisão
- Impressao -> Impressão
- imaginacao -> imaginação
- tecnica -> técnica

Procurar outros casos equivalentes.

Revisar:

- Home;
- catálogo;
- produto;
- carrinho;
- checkout;
- conta;
- autenticação;
- pedido;
- tracking;
- páginas institucionais;
- footer;
- mensagens de erro;
- mensagens de sucesso;
- empty states;
- botões;
- placeholders;
- metadata apresentada ao usuário quando pertinente.

IMPORTANTE: corrigir somente texto apresentado.

Não renomear por acentuação:

- slugs;
- rotas;
- nomes de coluna;
- constantes;
- IDs;
- eventos;
- APIs;
- arquivos técnicos;
- valores persistidos cuja alteração quebre contratos.

## 19.3.7 — Tipografia e espaçamento

Auditar a hierarquia tipográfica global.

Padronizar:

- H1;
- H2;
- H3;
- body;
- small/help text;
- labels;
- preços;
- badges;
- botões.

Auditar também:

- padding de cards;
- gaps;
- margens entre seções;
- largura máxima de conteúdo;
- densidade de formulários.

Evitar mudanças de fonte sem justificativa.

Priorizar consistência antes de adicionar dependência/font nova.

## 19.3.8 — Estados da interface

Revisar estados:

- vazio;
- erro;
- sucesso;
- indisponível;
- aguardando pagamento;
- pago;
- loading quando existente;
- sem pedidos;
- carrinho vazio;
- frete indisponível;
- autenticação.

Esses estados devem parecer parte do design PrintLab.

Não esconder falhas importantes apenas para deixar a interface bonita.

## 19.3.9 — Mobile e acessibilidade

Toda alteração da 19.3 deve ser mobile-first.

Validar:

- header;
- menus;
- busca;
- catálogo;
- produto;
- carrinho;
- checkout;
- conta;
- formulários;
- modais quando existentes;
- footer.

Preservar:

- navegação por teclado;
- focus-visible;
- contraste WCAG;
- labels;
- semântica;
- reduced motion;
- SSR funcional sem JavaScript obrigatório nas jornadas críticas.

## 19.3.10 — Footer e áreas institucionais

Revisar footer e páginas institucionais para acompanhar o novo acabamento.

O footer deve ajudar em:

- navegação;
- contato;
- WhatsApp;
- confiança;
- marca.

Não adicionar conteúdo jurídico, certificações ou promessas que não existam.

## 19.3.11 — O que não fazer

Fora do escopo da 19.3:

- migrations;
- mudança financeira;
- alteração de preços;
- mudança nas regras de frete;
- nova integração externa;
- mudança de autenticação;
- mudança de ownership;
- refactor amplo de backend sem necessidade;
- SPA;
- framework frontend novo;
- dependências pesadas sem justificativa;
- otimização prematura de performance.

Mudanças de estrutura devem existir somente quando melhorarem experiência/manutenção de forma demonstrável.

## Ordem de execução futura da 19.3

### P0 — Conversão e identidade

- WhatsApp;
- CTAs;
- header/navigation;
- correções textuais mais visíveis.

### P1 — Jornadas críticas

- Minha Conta;
- checkout;
- carrinho quando necessário;
- produto.

### P2 — Consistência global

- componentes;
- tipografia;
- cards;
- espaçamentos;
- estados;
- páginas institucionais;
- footer.

### P3 — Regressão visual

- mobile;
- desktop;
- teclado;
- acessibilidade;
- reduced motion.

Depois:

Fase 19.4 — Performance e regressão para produção.

## Definition of Done da futura 19.3

- Home e páginas internas parecem pertencer à mesma marca.
- Header possui presença visual compatível com PrintLab.
- CTAs de contato usam WhatsApp correto.
- Minha Conta possui experiência profissional.
- Checkout possui experiência consistente e clara.
- Carrinho preserva a qualidade visual atual.
- Strings visíveis relevantes estão em pt-BR correto.
- Mobile não apresenta overflow ou ações inacessíveis.
- Teclado e foco continuam funcionais.
- Contraste não regride.
- Reduced motion continua respeitado.
- Guest checkout continua funcional.
- Cliente autenticado continua funcional.
- SuperFrete, pickup e InfinitePay não sofrem alteração de regra.
- Admin não sofre regressão funcional.
- Nenhuma migration é necessária por motivo puramente visual.
- `templ generate`, testes e build deverão passar na implementação.
- Validação manual do responsável será necessária antes de declarar concluída.

## Status de implementação

- 19.3.1 — Header, navegação e identidade global: Implementada; aguardando validação manual.
- 19.3.2 — WhatsApp e CTAs comerciais: Implementada; aguardando validação manual.
- 19.3.4 — Minha Conta: Validada visualmente em produção pelo responsável.
- 19.3.5 — Carrinho e checkout: Implementada; aguardando validação visual manual.
- 19.3.6 — Revisão textual pt-BR: Planejada; não iniciada neste pacote.

A Fase 19.3 permanece em execução. A Fase 19.4 permanece planejada e não foi iniciada neste pacote.
