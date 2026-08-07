# Pendências do Etpeto

Etpeto está em fase pré-`v0.1`. O ciclo institucional já funciona de ponta a ponta — a administração dá acesso, o gestor monta e publica o roteiro, o autor cria um ETP a partir da revisão publicada, escreve, salva, gera o PDF oficial e exporta —, mas o domínio do ETP ainda é mínimo e **a aplicação não deve ser apresentada como pronta para produção**.

Suba tudo com `make start`.

Estado confirmado:

- o roteiro é dado versionado: o gestor edita um rascunho e publica revisões imutáveis, e cada documento fica preso à revisão com que nasceu;
- seção e subseção são a mesma coisa, aninháveis em até quatro níveis, cada uma com orientação própria e obrigatoriedade própria;
- o acesso é dado por convite de uso único, e a administração nunca conhece a senha de ninguém;
- os papéis são verificados no caso de uso, não apenas na interface;
- o editor salva com concorrência otimista e expõe conflito com escolha explícita;
- a prévia e a exportação partem sempre da revisão persistida, pelo mesmo renderer;
- força bruta por conta e abuso do renderer são contidos por limites de tentativa;
- **pesquisa de preços, mapa de riscos e tabelas ainda não existem como elementos** — o roteiro já sabe declarar quais seções os aceitariam;
- **`Finalizar` ainda não existe**: não há revisão congelada nem emissão com hashes;
- o cliente HTTP valida respostas com Zod, mas ainda é escrito à mão, não gerado do OpenAPI;
- OIDC/Keycloak ainda não foi implementado.

Os itens abaixo são critérios de conclusão, não estimativas ou promessa de ordem entre itens do mesmo grupo. Toda mudança de comportamento deve seguir TDD e preservar os limites descritos em [docs/architecture/foundation.md](docs/architecture/foundation.md).

## P0 — MVP utilizável

- [x] **Provisionar a primeira administração local com segurança.**
  - Disponibilizar comando explícito e idempotente para criar organização, usuário, identidade local e vínculo administrativo.
  - Ler a senha sem expô-la em argumentos, logs, histórico do shell ou arquivos versionados.
  - Recusar uma segunda inicialização incompatível e documentar recuperação de acesso.

- [x] **Fechar o fluxo de autenticação local.**
  - Carregar a sessão ao abrir a aplicação, proteger rotas privadas e redirecionar para `/entrar`.
  - Exibir usuário autenticado, permitir logout e tratar sessão expirada.
  - Login, credenciais inválidas, logout e acesso sem sessão estão cobertos por testes da API e pela jornada em `scripts/smoke-journey.sh`; expiração ainda só tem cobertura unitária.

- [x] **Completar o contrato de sessão com CSRF antes de escrever novas rotas mutáveis.**
  - Emitir um token CSRF vinculado à sessão, entregue fora do cookie e exigido em header próprio.
  - Recusar toda operação mutável autenticada sem token válido, comparando em tempo constante.
  - Entrar agora, e não no endurecimento final: cada rota mutável escrita antes desta decisão terá de ser reescrita junto de seus testes.

- [x] **Exigir sessão nas rotas que consomem o renderer.**
  - `POST /api/v1/previews` foi removida: renderizava conteúdo arbitrário do cliente e aceitava requisições anônimas.
  - A prévia agora parte da revisão persistida em `POST /api/v1/documents/{id}/preview`, com sessão, CSRF e escopo de organização.
  - Acesso anônimo e ausência de CSRF estão cobertos por testes e pelo smoke da stack.

- [x] **Limitar tentativas de login e de renderização.**
  - Duas camadas: por endereço, para proteger o serviço; por conta, contando só as falhas, para conter força bruta sem derrubar a organização junto.
  - Uma conta retida responde igual a uma inexistente, e `ETPETO_THROTTLE_SCALE` ajusta os limites onde muitas pessoas compartilham um endereço.

- [ ] **Gerar o cliente do frontend a partir do OpenAPI.**
  - Roteamento real e TanStack Query já estão em uso; o cliente em `web/src/shared/api/client.ts` valida com Zod, mas é escrito à mão.
  - Gerar cliente e schemas do contrato e verificar divergência na CI, para que uma mudança de API quebre o build e não a produção.

- [x] **Persistir documentos no PostgreSQL.**
  - Criar migrations e consultas tipadas para documento, revisão, seções e conteúdo estruturado versionado.
  - Implementar APIs autenticadas para listar, criar, abrir e atualizar um ETP.
  - Isolar dados por organização e impedir leitura ou alteração cruzada.

- [x] **Transformar o editor demonstrativo em editor de um documento real.**
  - Remover título, identificador, roteiro, progresso e conteúdo fixos do componente React.
  - Carregar o documento pela API e navegar entre seções reais.
  - Exibir estados de carregamento, vazio, erro, conflito e falta de permissão.

- [x] **Implementar salvamento confiável.**
  - Enviar alterações ao backend com revisão para concorrência otimista.
  - Mostrar estados reais: alterado, salvando, salvo, conflito e falha.
  - Conciliar a recuperação local com a última revisão do servidor sem sobrescrever conteúdo silenciosamente.

- [x] **Exibir a prévia Typst verdadeira.**
  - Converter o documento persistido para o contrato JSON versionado do renderer.
  - `Atualizar` chama `/api/v1/documents/{id}/preview` e mostra o PDF retornado, com carregamento e erro recuperável.
  - As URLs temporárias são revogadas e a prévia avisa quando está atrás da revisão atual.

- [x] **Exportar o PDF.**
  - Habilitar o botão somente quando existir uma renderização válida.
  - Baixar um arquivo com nome seguro e previsível, preservando o PDF gerado pelo Typst.
  - Cobrir resposta inválida, indisponibilidade do renderer e nova tentativa.

- [ ] **Definir e implementar a finalização.**
  - Registrar uma revisão final imutável ou definir explicitamente como uma finalização pode ser reaberta.
  - Validar campos obrigatórios antes de finalizar e informar pendências por seção.
  - Gerar o PDF a partir da mesma revisão marcada como final.

- [x] **Executar as suítes de integração já escritas.**
  - `migration_integration_test.go` e `runner_integration_test.go` exigem a tag `integration` e não são chamados pelo `Makefile` nem pela CI.
  - Expor um comando canônico e um job dedicado, com Docker disponível e Typst real.
  - Falhar a CI quando a suíte de integração quebrar, e não apenas a unitária.

- [x] **Cobrir a jornada principal de ponta a ponta pela API.**
  - `scripts/smoke-journey.sh` provisiona, entra, cria, salva, comprova o 409 do salvamento obsoleto, renderiza o PDF e sai.
  - A CI executa a jornada contra a stack Docker com banco limpo, sem credenciais reais.

- [ ] **Implementar a pesquisa de preços, hoje escrita e desligada.**
  - `internal/document/pricing.go` tem o modelo, a validação e o cálculo — itens, fontes, datas, valores em centavos, e estimativa reproduzível por média, mediana ou menor preço — mas nada o referencia: o validador do documento ainda recusa qualquer elemento que não seja texto rico.
  - Aceitar o elemento no documento, normalizar para o Typst como tabela e escrever o componente de edição.
  - Respeitar o "conteúdo permitido" que o roteiro já declara por seção, e registrar no documento a metodologia adotada.

- [ ] **Criar a tela inicial por papel.**
  - Hoje a raiz cai direto na lista de documentos; o desenho aprovado mostra o trabalho parado e, abaixo, apenas as áreas que o papel permite.

- [ ] **Cobrir os repositórios novos por teste de integração.**
  - Roteiro e convite têm testes de domínio com dublês, mas nenhum exercita o PostgreSQL real.
  - Publicação concorrente, unicidade do convite pendente e aceitação simultânea só se comprovam contra o banco.

- [ ] **Cobrir a jornada principal pelo navegador.**
  - A jornada atual usa apenas a API; nada garante que a interface a percorra de fato.
  - Adicionar Playwright para o fluxo crítico no desktop e em viewport móvel.
  - Incluir digitação no Lexical, autosave por debounce, prévia e exportação.

## P1 — Domínio do ETP

- [x] **Implementar roteiros configuráveis por organização.**
  - O gestor monta a árvore de seções, escreve a orientação, marca obrigatoriedade e escolhe que elementos cada seção aceita.
  - Publicar cria revisão imutável e recusa rascunho idêntico ao publicado; documentos existentes não são tocados.
  - Falta permitir mais de um roteiro por tipo de documento e histórico navegável de revisões.

- [ ] **Modelar elementos documentais discriminados e versionados.**
  - Manter o JSON do Etpeto independente de Lexical e Typst.
  - Validar cada variante no frontend e no backend.
  - Preservar round-trip sem perda entre edição, persistência e renderização.

- [ ] **Implementar mapa de riscos.**
  - Registrar evento, causa, consequência, probabilidade, impacto, resposta e responsáveis.
  - Derivar o nível de risco por regra explícita e versionada.
  - Renderizar matriz e tabela acessíveis no editor e no PDF.

- [ ] **Adicionar tabelas, listas estruturadas e anexos quando houver contrato definido.**
  - Estabelecer limites de tipo, tamanho e armazenamento antes de aceitar upload.
  - Não incorporar HTML, Typst ou caminhos arbitrários fornecidos pelo usuário.

- [ ] **Implementar histórico e trilha de auditoria.**
  - Registrar autor, instante, revisão e natureza das alterações relevantes.
  - Permitir comparar revisões e restaurar conteúdo sem apagar o histórico.

- [ ] **Elevar progressivamente os pisos de cobertura até os alvos aprovados.**
  - Os pisos atuais são 55% no Go, já sem código gerado e sem os processos de entrada, e 75/72/68/78 no frontend; a especificação define 90% em domínio e aplicação, 80% no backend, 85/80 no frontend e 90% de patch.
  - O que mais puxa o número para baixo são os repositórios PostgreSQL, hoje exercitados apenas pela suíte de integração e pela jornada.
  - Subir o piso a cada corte vertical concluído, sem permitir regressão.

- [x] **Administrar usuários, vínculos, papéis e templates.**
  - Convite de uso único com validade de 72 horas, senha escolhida por quem aceita, e recusa de remover a última administração.
  - Telas e rotas restritas ao papel, com teste negativo para cada rota de escrita.
  - Falta auditar as mudanças administrativas, o que depende da trilha de auditoria abaixo.

- [ ] **Evoluir o template oficial do PDF.**
  - Gerar capa, sumário, numeração, cabeçalhos, tabelas e quebras de página a partir de testes de referência.
  - Permitir identidade institucional sem executar templates enviados pelo navegador.
  - Validar documentos extensos, caracteres portugueses e conteúdo sem texto.

## P2 — Produção, Keycloak e comunidade

- [ ] **Implementar OIDC/Keycloak.**
  - Usar Authorization Code com PKCE no backend, validar issuer, state, nonce e assinatura.
  - Associar a identidade externa a um usuário local e emitir a mesma sessão opaca do login local.
  - Nunca entregar tokens do provedor ao React; suportar logout e rotação de chaves.
  - Permitir que cada organização habilite login local, OIDC ou ambos conforme política explícita.

- [ ] **Completar autorização e isolamento multi-organização.**
  - Aplicar papéis em todos os casos de uso e não apenas na interface.
  - Adicionar testes negativos para acesso horizontal e vertical.
  - Avaliar defesa adicional no banco antes de declarar suporte multi-tenant em produção.

- [ ] **Endurecer a segurança operacional.**
  - Definir política de senha, limitação global de tentativas e revogação administrativa de sessões; a proteção CSRF é tratada em P0.
  - Automatizar análise de dependências, imagens, segredos, SBOM e proveniência de releases.
  - Revisar headers, cookies e configuração TLS no ambiente real de implantação.

- [ ] **Preparar operação e recuperação.**
  - Publicar métricas, logs estruturados, correlação de requisições e alertas sem conteúdo sensível.
  - Documentar e testar backup, restauração, migrations, rollback e atualização sem perda de dados.
  - Definir limites de CPU, memória, armazenamento, concorrência e tempo de renderização.

- [ ] **Validar acessibilidade, responsividade e compatibilidade.**
  - Testar teclado, foco, leitores de tela, contraste, zoom e telas estreitas.
  - Definir navegadores suportados e executar testes automatizados do fluxo crítico.
  - Verificar a legibilidade do PDF e o comportamento de impressão.

- [ ] **Estabelecer releases e estabilidade de contratos.**
  - Publicar changelog, versionamento, imagens assinadas e instruções de upgrade.
  - Definir política de compatibilidade para API, schema documental, banco e templates.
  - Só remover o rótulo pré-`v0.1` após uma instalação limpa concluir a jornada P0.

- [ ] **Amadurecer a governança pública.**
  - Criar templates de issue, rótulos, roadmap e processo de triagem quando o remote estiver publicado.
  - Ativar private vulnerability reporting, proteção de branch, revisão obrigatória e DCO no GitHub.
  - Revisar critérios de manutenção e sucessão antes da primeira versão estável.

## Fora do escopo atual

- Coedição simultânea em tempo real, cursores compartilhados e CRDT.
- Aplicativo móvel nativo.
- Assinatura eletrônica ou integração com processos administrativos específicos sem ADR própria.

Esses itens só devem entrar no roadmap quando houver necessidade validada e capacidade de manutenção.
