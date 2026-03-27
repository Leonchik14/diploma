#!/usr/bin/env bash
# Вызывается certbot после успешного renew (флаг --deploy-hook).
# Нужны переменные окружения certbot: RENEWED_LINEAGE (путь к live/<домен>).
#
# Перед запуском экспортируйте:
#   export NGINX_TLS_CERT_DIR=/абсолютный/путь/к/каталогу   # тот же, что в .env для Docker
# Опционально:
#   export COMPOSE_PROJECT_DIR=/абсолютный/путь/к/репозиторию/Diploma
#
# Пример в renewal-конфиге certbot (deploy-hook =):
#   /path/to/scripts/letsencrypt-deploy-hook.sh

set -euo pipefail

if [[ -z "${RENEWED_LINEAGE:-}" ]]; then
	echo "letsencrypt-deploy-hook: нет RENEWED_LINEAGE; используйте только из certbot --deploy-hook" >&2
	exit 1
fi

if [[ -z "${NGINX_TLS_CERT_DIR:-}" ]]; then
	echo "letsencrypt-deploy-hook: задайте NGINX_TLS_CERT_DIR — каталог, куда класть fullchain.pem и privkey.pem для nginx в Docker" >&2
	exit 1
fi

mkdir -p "${NGINX_TLS_CERT_DIR}"
cp -L "${RENEWED_LINEAGE}/fullchain.pem" "${NGINX_TLS_CERT_DIR}/fullchain.pem"
cp -L "${RENEWED_LINEAGE}/privkey.pem" "${NGINX_TLS_CERT_DIR}/privkey.pem"

if [[ -n "${COMPOSE_PROJECT_DIR:-}" ]]; then
	(cd "${COMPOSE_PROJECT_DIR}" && docker compose --profile tls restart nginx-grpc)
fi
