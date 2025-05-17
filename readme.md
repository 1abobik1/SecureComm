# Сервер запускать исключительно через Makefile!
## Запуск сервера:
## 1)Перейти в папку server
## 2) создать .env файл с такими значениями
```
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
## 3) Вызвать команду ``` make up ```
## Если нужно пересобрать сервер ``` make up-rebuild ```

## 4) Если нужно сгенерировать документацию, находясь в папке server, нужно выполнить команду ``` make gen-docs ``` 
## Документация будет сгенерирована в папке server/docs