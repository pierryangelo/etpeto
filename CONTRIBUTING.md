# Como contribuir

Obrigado por ajudar a construir o Etpeto. O projeto quer acolher pessoas do serviço público, design, direito, produto e engenharia — não apenas desenvolvedores profissionais.

## Antes de começar

1. Leia a [arquitetura](docs/architecture/foundation.md) e a especificação ativa.
2. Procure uma discussão ou issue existente antes de iniciar uma mudança ampla.
3. Para novo contrato público, dependência estrutural ou mudança de segurança, proponha primeiro uma ADR em `docs/adr/`.
4. Nunca use dados pessoais, documentos reais sigilosos ou credenciais em testes e exemplos.

## Fluxo de desenvolvimento

Mudanças de comportamento seguem TDD: teste falhando, implementação mínima, teste verde e refatoração. Prefira commits pequenos e preserve os limites entre modelo documental, editor Lexical e renderer Typst.

Os pisos de cobertura são um ratchet contra regressão, não uma meta de qualidade isolada. Código novo deve exercitar caminhos relevantes mesmo quando o percentual global já estiver acima do mínimo.

```sh
cp .env.example .env
make check
make coverage
```

Mudanças de integração também devem passar por:

```sh
make up
make smoke
make down
```

Atualize a documentação quando alterar configuração, API, modelo documental ou comportamento observável.

## Pull requests

O PR deve explicar o problema, a decisão tomada, como foi testado e riscos conhecidos. A CI precisa passar. Alterações visuais devem incluir evidência da interface em desktop e tela estreita quando aplicável.

Contribuições usam o [Developer Certificate of Origin 1.1](https://developercertificate.org/). Todos os commits devem conter a linha `Signed-off-by`, adicionada com:

```sh
git commit -s
```

Ao assinar, você declara ter direito de enviar a contribuição sob a licença do projeto. Não é exigida cessão de copyright.

## Compatibilidade

Até `v0.1`, mudanças incompatíveis podem ocorrer, mas precisam ser explícitas na descrição do PR. Rotas, schemas persistidos e contratos de renderização devem ser versionados antes de dependerem de estabilidade.

Participantes devem seguir o [Código de Conduta](CODE_OF_CONDUCT.md).
