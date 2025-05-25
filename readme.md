# Сервер запускать исключительно через Makefile!
## Запуск сервера:
## 1) Перейти в папку server
## 2) Создать пару .env файлов с такими значениями

### Настройка переменных окружения для secure_comm_service
Создайте файл `.env` в корневой директории и добавьте следующие параметры, пример:

```ini
# http
HTTP_SERVER_ADDRESS=0.0.0.0:8080

# redis
REDIS_SERVER_ADDRESS=redis:6379
REDIS_HANDSHAKE_NONCES_TTL=10m
REDIS_SESSION_NONCES_TTL=8h    # столько же, сколько живет MINIO_URL_LIFETIME
REDIS_SESSION_KEY_TTL=720h     # столько же, сколько живет refresh токен
REDIS_CLIENT_PUB_KEYS_TTL=720h # столько же, сколько живет refresh токен
REDIS_MINIO_URL_TTL=8h

# пути до серверных ключей внутри докера
KEY_DIR_PATH=/root/keys
RSA_PUB_PATH=/root/keys/server_rsa.pub
RSA_PRIV_PATH=/root/keys/server_rsa.pem
ECDSA_PUB_PATH=/root/keys/server_ecdsa.pub
ECDSA_PRIV_PATH=/root/keys/server_ecdsa.pem

# limiter для апи: /handshake/init и /handshake/finalize 
HANDSHAKE_LIMITER_RPC=1 # 1 запрос в секунду
HANDSHAKE_LIMITER_BURST=2 # разрешается разом отправить 2 запроса, далее будет ограничение сверху(LIMITER_RPC=1)
HANDSHAKE_LIMITER_EXP_TTL=1h # время когда данные о запросах клиента удалятся

# limiter для общения по защищенному каналу 
SESSION_LIMITER_RPC=20 # 20 запросов в секунду
SESSION_LIMITER_BURST=25 # разрешается разом отправить 25 запросов, далее будет ограничение сверху(LIMITER_RPC=5)
SESSION_LIMITER_EXP_TTL=1h # время когда данные о запросах клиента удалятся

#JWT параметры
JWT_PUBLIC_KEY_PATH=public_key.pem

#Minio параметры
MINIO_PORT=localhost:9000
MINIO_ROOT_USER=minioadmin
MINIO_ROOT_PASSWORD=minioadmin
MINIO_USE_SSL=false
MINIO_URL_LIFETIME=8h

#Postgres параметры
POSTGRES_USER=postgres
POSTGRES_PASSWORD=dima15042004
POSTGRES_DB=cloud_service
STORAGE_PATH=postgres://postgres:dima15042004@user_usage_db:5432/cloud_service?sslmode=disable
```
> После запуска сервер будет доступен по адресу `http://localhost:8080`

---

### Настройка переменных окружения для auth_service
Создайте файл `.env` в корневой директории и добавьте следующие параметры, пример:

```ini
# postgres параметры
POSTGRES_USER=postgres
POSTGRES_PASSWORD=dima15042004
POSTGRES_DB=auth-service
STORAGE_PATH=postgres://postgres:dima15042004@auth_db:5432/auth-service?sslmode=disable

# сервер
HTTP_SERVER_ADDRESS=0.0.0.0:8081

# jwt параметры
ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=720h

# пути до ключей(нужны для jwt)
PUBLIC_KEY_PATH=public_key.pem
PRIVATE_KEY_PATH=private_key.pem

# внешние запросы
EXTERNAL_WEB_CLIENT=http://secure_comm_service:8080/web/ks
EXTERNAL_TG_CLIENT=http://secure_comm_service:8080/tg-bot/ks
QUOTA_SERVICE_URL=http://secure_comm_service:8080

# limiter для login
LOGIN_LIMITER_MAX_REQS=5
LOGIN_LIMITER_BURST=1
LOGIN_LIMITER_PERIOD=2m
```
> **Важно:** Замените `MyPASS` на ваш реальный пароль от PostgreSQL.
> После запуска сервер будет доступен по адресу `http://localhost:8081`

---

---

## 1. Генерация ключей

Если у вас нет ключей для JWT токенов, выполните в терминале следующие команды:

```bash
# генерирует приватный ключ
openssl genpkey -algorithm RSA -out private_key.pem -pkeyopt rsa_keygen_bits:2048

# генерирует публичный ключ
openssl rsa -pubout -in private_key.pem -out public_key.pem
```

Поместите файлы `private_key.pem` и `public_key.pem` в папку auth_service. Далее !СКОПИРУЙТЕ! `public_key.pem` и поместите его в secure_comm_service.

---


## 3) Для запуска нужно вызвать команду ``` make up ```
### Если нужно пересобрать сервер ``` make up-rebuild ```

## 4) Если нужно сгенерировать документацию, находясь в папке с _service в конце найдите MakeFile и сгенерируйте по написанной команде в MakeFile, пример ``` make <gen-docs-exmaple>``` 
### Документацию ищите в папках с _service в окончании, папка docs
