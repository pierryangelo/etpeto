# Etpeto

Etpeto é uma aplicação open source para elaborar Estudos Técnicos Preliminares (ETPs) estruturados, com edição orientada por roteiro, pré-visualização no navegador e geração consistente de PDF com Typst.

> **Estado:** pré-`v0.1`, em desenvolvimento ativo. O modelo documental e as APIs ainda podem mudar sem compatibilidade retroativa.

O projeto nasce de um problema cotidiano do serviço público: o conteúdo do ETP exige análise, mas boa parte do tempo acaba consumida por estilos, numeração, sumário e comportamento imprevisível de editores de texto. No Etpeto, o documento é dado estruturado. A interface ajuda a preencher seções e elementos próprios — como pesquisa de preços e mapa de riscos — e o renderer cuida da apresentação oficial.

## O que já existe

A jornada principal funciona de ponta a ponta: entrar, criar um ETP, editar, salvar, gerar o PDF oficial e exportar.

- oficina React responsiva com roteiro, editor e prévia recolhíveis;
- texto rico com Lexical e recuperação local do rascunho;
- documentos persistidos com revisão a cada salvamento e concorrência otimista;
- prévia e exportação a partir da revisão salva, pelo mesmo renderer Typst;
- API Go documentada por OpenAPI;
- autenticação local com senha Argon2id, sessão opaca e CSRF vinculado à sessão;
- PostgreSQL com migrations versionadas;
- renderer Typst isolado, alimentado apenas por JSON validado;
- stack Docker com somente o gateway web exposto.

O roteiro do ETP ainda é fixo, e pesquisa de preços, mapa de riscos e finalização
não existem. As pendências estão em [TODO.md](TODO.md).

OIDC/Keycloak está previsto na arquitetura, mas ainda não faz parte do walking skeleton. Quando implementado, o provedor externo terminará na mesma sessão local opaca; tokens do Keycloak não serão entregues ao React.

## Início rápido

Requer Docker com Compose v2. No macOS, o fluxo é testado com Colima.

```sh
make start
```

O comando cria o `.env` se não existir, sobe a stack e provisiona a primeira
administração, pedindo a senha na entrada padrão para que ela não fique em
arquivo nem no histórico do shell. Rodar de novo não altera nada.

Ajuste `ETPETO_ADMIN_EMAIL` e `ETPETO_ADMIN_DISPLAY_NAME` no `.env` antes, se
quiser um endereço diferente do exemplo. Ao final, abra
[http://localhost:8080](http://localhost:8080) e entre com esse endereço.

Para encerrar:

```sh
make down
```

Para apagar também os dados, `docker compose down -v`.

O arquivo `.env` é apenas para desenvolvimento e não deve ser versionado. Troque todos os segredos antes de qualquer implantação compartilhada.

## Desenvolvimento

Os comandos canônicos são apresentados por `make help`. O gate local principal é:

```sh
make check
make coverage
```

Leia o [runbook de desenvolvimento](docs/runbooks/development.md) para execução sem Docker, diagnóstico do Colima e smoke tests.

## Projeto e comunidade

- [Arquitetura](docs/architecture/foundation.md) — limites, fluxo de dados e justificativa das tecnologias;
- [Especificação do produto](docs/specs/2026-08-06-etpeto-design.md) — direção funcional e visual aprovada;
- [Pendências e roadmap](TODO.md) — o que ainda é demonstrativo e os critérios para um MVP utilizável;
- [Como contribuir](CONTRIBUTING.md) — TDD, DCO e critérios para pull requests;
- [Segurança](SECURITY.md) — como comunicar vulnerabilidades de forma privada;
- [Governança](GOVERNANCE.md) e [suporte](SUPPORT.md);
- [Licença](LICENSE) — GNU AGPL v3, identificador `AGPL-3.0-only`.

Etpeto não substitui a análise técnica nem valida a conformidade jurídica de um ETP. Ele busca tornar a estrutura, a colaboração no código e a geração do documento mais confiáveis.
