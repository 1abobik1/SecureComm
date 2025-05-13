# Сервер запускать исключительно через Makefile!
## Запуск сервера:
## 1)Перейти в папку server
## 2) создать .env файл с такими значениями
```
# http
HTTP_SERVER_ADDRESS=0.0.0.0:8080

# redis
REDIS_SERVER_ADDRESS=redis:6379
REDIS_NONCES_TTL=10m

# пути до серверных ключей внутри докера
KEY_DIR_PATH=/root/keys
RSA_PUB_PATH=/root/keys/server_rsa.pub
RSA_PRIV_PATH=/root/keys/server_rsa.pem
ECDSA_PUB_PATH=/root/keys/server_ecdsa.pub
ECDSA_PRIV_PATH=/root/keys/server_ecdsa.pem

# limiter для апи: /handshake/init и /handshake/finalize 1 запрос в секунду
LIMITER_RPC=1
LIMITER_BURST=2
LIMITER_EXP_TTL=1h
```
## 3) Вызвать команду ``` make up ```
## Если нужно пересобрать сервер ``` make up-rebuild ```
