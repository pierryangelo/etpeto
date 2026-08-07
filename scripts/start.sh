#!/bin/sh
# Brings the Etpeto up from nothing: configuration, containers, and the first
# administrator. Safe to run again — every step is idempotent.
set -eu

port="${ETPETO_HTTP_PORT:-8080}"
url="http://localhost:${port}"

if ! docker compose version >/dev/null 2>&1; then
  echo "Docker com Compose v2 é necessário. No macOS, o fluxo é testado com Colima." >&2
  exit 1
fi

# The build failure people actually hit is a Docker credential helper that the
# configuration names but the machine does not have. The buildkit output buries
# that under a stack trace, so it is translated here.
explain_failure() {
  log="$1"
  if grep -q "docker-credential-.*executable file not found" "$log" 2>/dev/null; then
    helper=$(grep -o 'docker-credential-[a-z]*' "$log" | head -1)
    cat >&2 <<MESSAGE

O Docker não encontrou o auxiliar de credenciais "${helper}".

O arquivo ~/.docker/config.json pede esse auxiliar, mas ele não está instalado —
é o caso comum de quem usa Colima ou Podman em vez do Docker Desktop.

Duas saídas:

  1. Remover a linha "credsStore" de ~/.docker/config.json.
     Só é seguro se "auths" estiver vazio, o que acontece quando você não faz
     login em registro privado.

  2. Instalar o auxiliar:
     macOS   brew install docker-credential-helper
     Linux   pelo gerenciador de pacotes da distribuição

MESSAGE
    return
  fi
  echo >&2
  echo "Falha ao subir a stack:" >&2
  tail -20 "$log" >&2
}

if [ ! -f .env ]; then
  cp .env.example .env
  echo "Criado .env a partir de .env.example."
  echo "Os segredos ali servem apenas para desenvolvimento; troque todos antes de compartilhar a instalação."
  echo
fi

echo "Subindo a stack…"
if ! docker compose up --build -d --wait 2>/tmp/etpeto-start.log; then
  explain_failure /tmp/etpeto-start.log
  exit 1
fi

# A published script has to exist before anyone can create a document, and the
# bootstrap also publishes it. Running it again changes nothing.
echo
echo "Provisionando a primeira administração."
echo "A senha é lida agora e não fica em arquivo nem no histórico do shell."
echo "Precisa de ao menos 12 caracteres."
printf 'Senha: '
stty -echo 2>/dev/null || true
read -r password
stty echo 2>/dev/null || true
echo

printf '%s\n' "$password" | docker compose run --rm --build --no-TTY bootstrap

echo
echo "Etpeto no ar em ${url}"
echo "Entre com o endereço em ETPETO_ADMIN_EMAIL do seu .env."
echo
echo "Para encerrar:            make down"
echo "Para apagar tudo:         docker compose down -v"
