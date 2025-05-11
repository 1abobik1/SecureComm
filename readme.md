# Сервер запускать исключительно через Makefile!
## Запуск сервера:
## 1)Перейти в папку server
## 2) создать .env файл с такими значениями
```
HTTP_SERVER_ADDRESS=0.0.0.0:8080
REDIS_SERVER_ADDRESS=redis:6379
REDIS_NONCES_TTL=10m
KEY_DIR_PATH=/root/keys
RSA_PUB_PATH=/root/keys/server_rsa.pub
RSA_PRIV_PATH=/root/keys/server_rsa.pem
ECDSA_PUB_PATH=/root/keys/server_ecdsa.pub
ECDSA_PRIV_PATH=/root/keys/server_ecdsa.pem
```
## 3) Вызвать команду ``` make up ```
## Если нужно пересобрать сервер ``` make up-rebuild ```
