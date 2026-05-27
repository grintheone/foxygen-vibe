#!/bin/sh
set -eu

PATH="/usr/local/bin:/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin:${PATH:-}"

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
ROOT_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/../.." && pwd)

ENV_FILE=${ENV_FILE:-"$SCRIPT_DIR/.env"}
COMPOSE_FILE=${COMPOSE_FILE:-"$ROOT_DIR/docker-compose.production.yml"}
BACKUP_DIR=${BACKUP_DIR:-"$SCRIPT_DIR/backups"}
BACKUP_PREFIX=${BACKUP_PREFIX:-foxygen-db}
BACKUP_KEEP_DAYS=${BACKUP_KEEP_DAYS:-14}
DB_SERVICE=${DB_SERVICE:-postgres}

if [ ! -f "$ENV_FILE" ]; then
    echo "missing env file: $ENV_FILE" >&2
    exit 1
fi

set -a
. "$ENV_FILE"
set +a

: "${POSTGRES_USER:?POSTGRES_USER is required in $ENV_FILE}"
: "${POSTGRES_DB:?POSTGRES_DB is required in $ENV_FILE}"

if docker compose version >/dev/null 2>&1; then
    compose() {
        docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" "$@"
    }
elif command -v docker-compose >/dev/null 2>&1; then
    compose() {
        docker-compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" "$@"
    }
else
    echo "docker compose is not available on this host" >&2
    exit 1
fi

mkdir -p "$BACKUP_DIR"

timestamp=$(date +"%Y-%m-%d_%H-%M-%S")
backup_file="$BACKUP_DIR/${BACKUP_PREFIX}_${timestamp}.sql.gz"
tmp_file="${backup_file}.tmp"

cleanup() {
    rm -f "$tmp_file"
}

trap cleanup INT TERM EXIT

compose ps "$DB_SERVICE" >/dev/null

compose exec -T "$DB_SERVICE" sh -lc \
    'exec pg_dump --clean --if-exists --no-owner --no-privileges -U "$POSTGRES_USER" -d "$POSTGRES_DB"' \
    | gzip -9 > "$tmp_file"

mv "$tmp_file" "$backup_file"
trap - INT TERM EXIT

find "$BACKUP_DIR" -type f -name "${BACKUP_PREFIX}_*.sql.gz" -mtime +"$BACKUP_KEEP_DAYS" -delete

echo "backup created: $backup_file"
