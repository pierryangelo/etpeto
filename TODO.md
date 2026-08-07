# Pendências do Etpeto

Etpeto está em fase pré-`v0.1`. A fundação técnica funciona, mas a interface atual ainda é uma demonstração parcialmente interativa e **não deve ser apresentada como um produto utilizável**.

Estado confirmado da interface:

- o texto rico e a preferência dos painéis são recuperados apenas pelo `localStorage` do navegador;
- título, roteiro, progresso, pesquisa de preços e prévia exibem dados demonstrativos;
- o indicador `Salvo agora`, `Atualizar`, `Finalizar` e `Exportar PDF` ainda não formam um fluxo funcional;
- a API gera PDF com Typst, mas o editor ainda não chama esse endpoint;
- a autenticação local possui API e sessões, mas falta criar o primeiro usuário e proteger as rotas;
- OIDC/Keycloak ainda não foi implementado.

Os itens abaixo são critérios de conclusão, não estimativas ou promessa de ordem entre itens do mesmo grupo. Toda mudança de comportamento deve seguir TDD e preservar os limites descritos em [docs/architecture/foundation.md](docs/architecture/foundation.md).

## P0 — MVP utilizável

- [ ] **Provisionar a primeira administração local com segurança.**
  - Disponibilizar comando explícito e idempotente para criar organização, usuário, identidade local e vínculo administrativo.
  - Ler a senha sem expô-la em argumentos, logs, histórico do shell ou arquivos versionados.
  - Recusar uma segunda inicialização incompatível e documentar recuperação de acesso.

- [ ] **Fechar o fluxo de autenticação local.**
  - Carregar a sessão ao abrir a aplicação, proteger rotas privadas e redirecionar corretamente para `/login`.
  - Exibir usuário autenticado, permitir logout e tratar sessão expirada.
  - Cobrir login, credenciais inválidas, logout, expiração e acesso sem sessão por testes de integração.

- [ ] **Persistir documentos no PostgreSQL.**
  - Criar migrations e consultas tipadas para documento, revisão, seções e conteúdo estruturado versionado.
  - Implementar APIs autenticadas para listar, criar, abrir e atualizar um ETP.
  - Isolar dados por organização e impedir leitura ou alteração cruzada.

- [ ] **Transformar o editor demonstrativo em editor de um documento real.**
  - Remover título, identificador, roteiro, progresso e conteúdo fixos do componente React.
  - Carregar o documento pela API e navegar entre seções reais.
  - Exibir estados de carregamento, vazio, erro, conflito e falta de permissão.

- [ ] **Implementar salvamento confiável.**
  - Enviar alterações ao backend com revisão para concorrência otimista.
  - Mostrar estados reais: alterado, salvando, salvo, conflito e falha.
  - Conciliar a recuperação local com a última revisão do servidor sem sobrescrever conteúdo silenciosamente.

- [ ] **Exibir a prévia Typst verdadeira.**
  - Converter o documento persistido para o contrato JSON versionado do renderer.
  - Fazer `Atualizar` chamar `/api/v1/previews` e mostrar o PDF retornado, com loading e erro recuperável.
  - Revogar URLs temporárias do navegador e indicar quando a prévia estiver desatualizada.

- [ ] **Exportar o PDF.**
  - Habilitar o botão somente quando existir uma renderização válida.
  - Baixar um arquivo com nome seguro e previsível, preservando o PDF gerado pelo Typst.
  - Cobrir resposta inválida, indisponibilidade do renderer e nova tentativa.

- [ ] **Definir e implementar a finalização.**
  - Registrar uma revisão final imutável ou definir explicitamente como uma finalização pode ser reaberta.
  - Validar campos obrigatórios antes de finalizar e informar pendências por seção.
  - Gerar o PDF a partir da mesma revisão marcada como final.

- [ ] **Cobrir a jornada principal de ponta a ponta.**
  - Provisionar usuário, entrar, criar ETP, editar, salvar, atualizar prévia, exportar e sair.
  - Executar a jornada contra a stack Docker na CI, sem credenciais reais ou dados pessoais.

## P1 — Domínio do ETP

- [ ] **Implementar roteiros configuráveis por organização.**
  - Versionar templates e fixar cada documento à revisão usada em sua criação.
  - Permitir seções obrigatórias, opcionais e personalizadas sem alterar código.
  - Definir migração explícita quando um roteiro publicado mudar.

- [ ] **Modelar elementos documentais discriminados e versionados.**
  - Manter o JSON do Etpeto independente de Lexical e Typst.
  - Validar cada variante no frontend e no backend.
  - Preservar round-trip sem perda entre edição, persistência e renderização.

- [ ] **Implementar pesquisa de preços.**
  - Registrar itens, fontes, datas, valores, evidências e critérios de cálculo.
  - Calcular estimativas de forma reproduzível e registrar a metodologia adotada.
  - Renderizar o elemento de modo próprio no editor e no PDF.

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

- [ ] **Administrar usuários, vínculos, papéis e templates.**
  - Disponibilizar telas e APIs somente para os papéis autorizados.
  - Auditar mudanças administrativas e impedir autoelevação de privilégios.

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
  - Definir proteção CSRF, limitação de tentativas, política de senha e revogação administrativa de sessões.
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
