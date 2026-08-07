# Runbook de desenvolvimento

## Pré-requisitos

- Docker Engine compatível com Compose v2;
- Colima no macOS, ou daemon Docker equivalente;
- Go e Bun somente para executar gates fora dos containers;
- portas `8080`, `5173` e, em testes de integração locais, uma porta PostgreSQL efêmera disponíveis.

Versões canônicas estão em `go.mod` e nos Dockerfiles. Confira os comandos com `make help`.

## Stack completa no Colima

```sh
colima start
docker context use colima
cp .env.example .env
make up
make smoke
```

Serviços esperados:

- web/gateway em `http://localhost:8080`;
- API, PostgreSQL e renderer sem portas publicadas;
- `migrate` concluído com status `Exited (0)`.

Inspecione com `docker compose ps -a` e encerre com `make down`. Para apagar também o banco estritamente local, execute `docker compose down -v`; esse comando remove o volume do projeto.

### Helper de credenciais ausente

Se o CLI informar que `docker-credential-osxkeychain` não existe, o problema é a configuração do cliente Docker, não o Colima. Instale o helper esperado ou remova/corrija `credsStore` em `~/.docker/config.json` conforme a política da sua máquina. Não versione uma configuração pessoal nem desative o armazenamento seguro de credenciais em ambiente compartilhado.

## Execução nativa

Suba apenas o PostgreSQL e configure `ETPETO_DATABASE_URL` apontando para ele, ou use a stack completa. Em terminais separados:

```sh
go run ./cmd/renderer
go run ./cmd/api
cd web && bun run dev
```

Para esse modo, ajuste `ETPETO_RENDERER_URL` para `http://localhost:8081`. A interface Vite fica em `http://localhost:5173` e encaminha `/api` para a API local.

## Gates

```sh
make check       # contrato, formatação, testes, vet, tipos e builds
make coverage    # pisos de cobertura Go e frontend
make smoke       # contrato HTTP do stack já iniciado
```

Testes Go que usam Testcontainers precisam acessar o daemon do Colima. O teste real do Typst exige o binário `typst` no `PATH`; quando indisponível, o teste informa skip, e o smoke do container continua obrigatório para fechar a integração.

## Diagnóstico rápido

1. `docker compose ps -a` confirma ordem e healthchecks.
2. `docker compose logs --tail=100 <serviço>` mostra o erro sem despejar todo o histórico.
3. `curl -i http://localhost:8080/livez` verifica processo da API.
4. `curl -i http://localhost:8080/readyz` verifica dependências da API.
5. `curl -i http://localhost:8080/api/v1/service-info` verifica proxy e contrato público.

Não cole arquivos `.env`, tokens, URLs com senha nem documentos reais em issues. Sanitize logs antes de compartilhá-los.

## Migrações e SQL

Adicione migrations em `db/migrations` sem alterar migrations já publicadas. Atualize consultas em `db/queries`, execute `make generate` e revise o diff gerado. A aplicação valida a disponibilidade do banco; o container `migrate` aplica o schema antes de iniciar a API.

## Renderer

O endpoint do renderer é interno e exige `ETPETO_RENDERER_TOKEN`. Nunca publique sua porta. Para mudanças no contrato, comece em `internal/rendercontract`, escreva testes de validação, adapte renderer e cliente e execute o smoke completo.
