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

if [ ! -f .env ]; then
  cp .env.example .env
  echo "Criado .env a partir de .env.example."
  echo "Os segredos ali servem apenas para desenvolvimento; troque todos antes de compartilhar a instalação."
  echo
fi

echo "Subindo a stack…"
docker compose up --build -d --wait

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
