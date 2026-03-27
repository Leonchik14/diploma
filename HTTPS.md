# HTTPS для gRPC: nginx + Certbot (Let’s Encrypt)

Схема: **certbot** получает сертификат на хосте → в каталог для Docker кладутся **`fullchain.pem`** и **`privkey.pem`** (такие имена ждёт `nginx/grpc-tls.conf`) → контейнер **`nginx-grpc`** монтирует этот каталог в `/etc/nginx/certs`.

**api-gateway** по-прежнему без TLS на `9090` внутри сети; снаружи клиент ходит на **443** с TLS.

---

## Куда что попадает (шпаргалка)

| Что | Где оказывается |
|-----|------------------|
| Сертификат от Let’s Encrypt (оригинал) | `/etc/letsencrypt/live/<ВАШ_ДОМЕН>/fullchain.pem` и `privkey.pem` (симлинки на `archive/`) |
| То, что должен читать nginx в Docker | В **одном каталоге на хосте** два файла с **именами** `fullchain.pem` и `privkey.pem` (копия с `cp -L`, см. ниже) |
| Проперти в `.env` | `NGINX_TLS_CERT_DIR` = **абсолютный путь** к этому каталогу на хосте |
| Порт HTTPS | `NGINX_GRPC_HTTPS_PORT=443` (если не занят другим процессом) |

Имена **`fullchain.pem`** и **`privkey.pem`** менять нельзя — они зашиты в `nginx/grpc-tls.conf`.

---

## 1. Проперти в `.env` (в корне репозитория)

Скопируйте из `.env.example` и заполните:

```env
# Домен, на который выписываете сертификат (для ваших заметок и команд certbot)
TLS_DOMAIN=grpc.example.com

# Каталог НА ХОСТЕ, где лежат (или будут лежать) fullchain.pem и privkey.pem для Docker
NGINX_TLS_CERT_DIR=/var/lib/diploma/nginx-certs

# Проброс порта HTTPS (по умолчанию 443)
NGINX_GRPC_HTTPS_PORT=443
```

Создайте каталог под сертификаты (один раз):

```bash
sudo mkdir -p /var/lib/diploma/nginx-certs
sudo chown "$USER:$USER" /var/lib/diploma/nginx-certs
```

(Путь может быть любым; главное — тот же в `.env` и в командах ниже.)

---

## 2. Первый выпуск сертификата (certbot)

Нужны **A/AAAA-записи** домена на IP этого сервера. На время **`certonly --standalone`** порты **80** и **443** должны быть **свободны** (остановите nginx/systemd-службы, которые их занимают, и при необходимости `docker compose down` для контейнеров с 443).

Установите certbot (пример Debian/Ubuntu):

```bash
sudo apt update && sudo apt install -y certbot
```

Выпуск (подставьте свой домен и email):

```bash
sudo certbot certonly --standalone \
  -d grpc.example.com \
  --email you@example.com \
  --agree-tos \
  --non-interactive
```

После успеха файлы будут здесь:

- `/etc/letsencrypt/live/grpc.example.com/fullchain.pem`
- `/etc/letsencrypt/live/grpc.example.com/privkey.pem`

**В Docker нельзя смонтировать только `live/<домен>`** — симлинки указывают в `archive/` и в контейнере «ломаются». Поэтому **копируем с разыменованием** (`-L`) в каталог из `NGINX_TLS_CERT_DIR`:

```bash
sudo cp -L /etc/letsencrypt/live/grpc.example.com/fullchain.pem /var/lib/diploma/nginx-certs/fullchain.pem
sudo cp -L /etc/letsencrypt/live/grpc.example.com/privkey.pem  /var/lib/diploma/nginx-certs/privkey.pem
sudo chown root:root /var/lib/diploma/nginx-certs/*.pem
sudo chmod 644 /var/lib/diploma/nginx-certs/fullchain.pem
sudo chmod 600 /var/lib/diploma/nginx-certs/privkey.pem
```

(Если каталог принадлежит пользователю из шага выше — `chown` можно к этому пользователю; важно, чтобы контейнер nginx читал файлы.)

---

## 3. Запуск стека с TLS

Из корня репозитория:

```bash
docker compose --profile tls up -d
```

Клиенты gRPC: **`grpc.example.com:443`** с обычной проверкой TLS (как у браузера для Let’s Encrypt).

Локальная диагностика:

```bash
grpcurl -plaintext=false grpc.example.com:443 list
```

---

## 4. Продление (renew) и автоматическое обновление файлов для Docker

При `certbot renew` Let’s Encrypt обновляет каталог `live/…`. Нужно снова скопировать PEM в `NGINX_TLS_CERT_DIR` и перезапустить nginx.

Подготовьте скрипт из репозитория и сделайте исполняемым:

```bash
chmod +x scripts/letsencrypt-deploy-hook.sh
```

Один раз добавьте **deploy-hook** к сертификату (подставьте домен и **абсолютные** пути):

```bash
sudo certbot certonly --standalone -d grpc.example.com \
  --deploy-hook 'env NGINX_TLS_CERT_DIR=/var/lib/diploma/nginx-certs COMPOSE_PROJECT_DIR=/path/to/Diploma /path/to/Diploma/scripts/letsencrypt-deploy-hook.sh'
```

Если сертификат уже выпущен **без** hook, отредактируйте файл renewal (удобно через `certbot renew --dry-run` после правок):

`/etc/letsencrypt/renewal/grpc.example.com.conf` — в секции добавьте строку:

```ini
deploy_hook = env NGINX_TLS_CERT_DIR=/var/lib/diploma/nginx-certs COMPOSE_PROJECT_DIR=/path/to/Diploma /path/to/Diploma/scripts/letsencrypt-deploy-hook.sh
```

Проверка продления без записи сертификата:

```bash
sudo certbot renew --dry-run
```

Реальное продление (по cron/systemd обычно раз в 12 ч):

```bash
sudo certbot renew
```

Hook сам скопирует `fullchain.pem` / `privkey.pem` и выполнит `docker compose --profile tls restart nginx-grpc`, если задан **`COMPOSE_PROJECT_DIR`**.

---

## 5. Certbot в Docker (альтернатива)

Если certbot не ставите на хост:

```bash
docker run -it --rm \
  -p 80:80 -p 443:443 \
  -v /etc/letsencrypt:/etc/letsencrypt \
  certbot/certbot certonly --standalone \
  -d grpc.example.com \
  --email you@example.com \
  --agree-tos \
  --non-interactive
```

Дальше те же **`cp -L`** из `/etc/letsencrypt/live/<домен>/` в **`NGINX_TLS_CERT_DIR`**.

---

## 6. Webroot вместо standalone (если порт 80 уже отдаёт HTTP)

Нужен ответ по `http://<домен>/.well-known/acme-challenge/`. Пример:

```bash
sudo certbot certonly --webroot -w /var/www/certbot \
  -d grpc.example.com \
  --email you@example.com \
  --agree-tos \
  --non-interactive
```

Каталог `-w` должен совпадать с тем, куда ваш веб-сервер отдаёт файлы для ACME. Копирование в `NGINX_TLS_CERT_DIR` — как в п. 2.

---

## Локальная разработка без Let’s Encrypt

```bash
docker compose up
```

Доступ к шлюзу: `localhost:9090` (plain). Профиль **`tls`** не используйте без реальных PEM в `NGINX_TLS_CERT_DIR`.
