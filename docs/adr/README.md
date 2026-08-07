# Registros de decisão arquitetural

Use uma ADR para decisões difíceis de reverter: dependências estruturais, autenticação, formatos persistidos, contratos externos, limites entre processos e mudanças relevantes de segurança.

Nomeie o arquivo como `NNNN-titulo-curto.md` e use:

```md
# NNNN — Título

Status: proposta | aceita | substituída por NNNN
Data: AAAA-MM-DD

## Contexto

Qual problema e quais restrições tornam a decisão necessária?

## Decisão

O que será feito e quais limites fazem parte da decisão?

## Alternativas consideradas

Quais opções reais foram comparadas e por que não foram escolhidas agora?

## Consequências

Quais benefícios, custos, riscos, migrações e sinais para reconsideração existem?
```

Uma ADR registra o raciocínio disponível na época; ela pode ser substituída, mas não reescrita para apagar o histórico.
