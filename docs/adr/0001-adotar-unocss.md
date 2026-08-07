# 0001 — Adotar UnoCSS ao lado dos CSS Modules

Status: aceita
Data: 7 de agosto de 2026

## Contexto

A especificação de produto fixou "CSS Modules e custom properties para estilo e tokens" e o plano da fundação listou, entre as restrições, "não adicionar ORM, Redis, broker, Tailwind, Redux ou Zustand". A intenção era evitar o peso e a dependência estrutural de um kit utilitário completo em um projeto que precisa continuar legível para revisão humana e para contribuição externa.

Depois do primeiro corte visual da oficina, três fatos mudaram o cálculo:

1. O trabalho passou a ser de ajuste fino e repetido — densidade, espaçamento, estados — sobre uma interface que ainda vai mudar muito. Cada ajuste exige inventar um nome, escrever a regra em outro arquivo e manter os dois em sincronia.
2. O CSS acumulou regras órfãs. `paper`, `paperSeal`, `paperTable` e `structuredBlock` sobraram da demonstração e não são mais usados por componente algum; nada no processo os remove.
3. A consistência prometida pelos tokens não é imposta. Há espaçamentos escritos diretamente em pixels fora da escala, porque nada impede.

O projeto também acabou de adicionar `unplugin-icons` para compilar ícones do Iconify no bundle, o que trouxe `@svgr/core` e dezenas de pacotes transitivos apenas para converter SVG em componente.

## Decisão

Adotar UnoCSS como motor de utilitários, convivendo com os CSS Modules existentes.

Limites que fazem parte da decisão:

- O tema do UnoCSS deriva dos tokens já definidos em `tokens.css`. Uma cor ou espaçamento que não esteja nos tokens não vira utilitário; a fonte de verdade continua sendo o sistema de tokens, não a classe.
- Nada do CSS atual será reescrito por reescrever. UnoCSS entra no que for novo e no que já for ser alterado.
- Blocos com muitos estados condicionais continuam em CSS Modules, onde uma folha nomeada lê melhor que uma sequência longa de utilitários.
- Padrões repetidos são promovidos a `shortcuts` quando aparecem, e não antecipados.
- `preset-icons` substitui `unplugin-icons`, eliminando `@svgr/core` e seus transitivos. Os ícones continuam vindo de `@iconify-json/lucide` instalado localmente; a API de runtime do Iconify permanece proibida, porque busca de host externo e a política de segurança do gateway a bloqueia.

## Alternativas consideradas

**Manter apenas CSS Modules.** É o desenho aprovado e não custa nada manter. Foi descartado porque não resolve nenhum dos três fatos acima: continuaria exigindo um nome por ajuste, continuaria acumulando regras mortas sem detecção e continuaria sem impor a escala.

**Adotar Tailwind.** Resolve os mesmos problemas e tem base de contribuidores maior. Foi descartado porque exige uma etapa de build própria e um arquivo de configuração paralelo aos tokens, enquanto o UnoCSS lê o tema direto e gera apenas o que é usado. A diferença é pequena; a preferência foi pela integração mais curta.

**Escrever um lint de CSS.** Uma regra que reprovasse valores fora da escala e detectasse seletor órfão atacaria a consistência sem nova dependência. Foi descartado por resolver metade do problema e não reduzir o custo de cada ajuste, que é o motivo principal.

## Consequências

O ajuste visual passa a ser feito no componente, sem troca de arquivo e sem inventar nome. O CSS gerado contém apenas o que é usado, então regra morta deixa de acumular. Valores fora da escala ficam visíveis, porque não produzem utilitário.

Em troca, o JSX passa a carregar strings de classe mais longas, e quem contribuir precisa conhecer a sintaxe além de CSS. É o custo aceito, e a razão de manter CSS Modules para os blocos mais condicionais.

O repositório passa a ter duas formas de escrever estilo, o que exige disciplina para não misturá-las dentro do mesmo componente.

Sinais para reconsiderar: strings de classe que deixem de ser legíveis mesmo com `shortcuts`; divergência entre o tema do UnoCSS e `tokens.css`; ou dificuldade relatada por quem contribuir de fora.
