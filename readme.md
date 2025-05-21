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
REDIS_SESSION_NONCES_TTL=1m
REDIS_SESSION_KEY_TTL=20s

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
```
> После запуска сервер будет доступен по адресу `http://localhost:8080`

---

### Настройка переменных окружения для auth_service
Создайте файл `.env` в корневой директории и добавьте следующие параметры, пример:

```ini
POSTGRES_USER=postgres
POSTGRES_PASSWORD=MyPASS
POSTGRES_DB=auth-service
STORAGE_PATH=postgres://postgres:MyPASS@auth_db:5432/auth-service?sslmode=disable
HTTP_SERVER_ADDRESS=0.0.0.0:8081
ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=720h
PUBLIC_KEY_PATH=public_key.pem
PRIVATE_KEY_PATH=private_key.pem
```
> **Важно:** Замените `MyPASS` на ваш реальный пароль от PostgreSQL.
> После запуска сервер будет доступен по адресу `http://localhost:8081`

---

---

## 1. Генерация ключей

Если у вас нет ключей, выполните в терминале следующие команды:

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

## 4) Если нужно сгенерировать документацию, находясь в папке server, нужно выполнить команду ``` make gen-docs ``` 
### Документацию ищите в папках с _service в окончании, папка docs
