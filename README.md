# Anti Brute Force 

Учебный REST-сервис для ограничения частоты попыток авторизации по login/password/IP.

## Архитектура

- bucket-ы rate limiter хранятся в Redis;
- whitelist/blacklist  хранятся в Redis;
- `docker-compose.yml` поднимает и приложение, и Redis;
- CI/CD на GitLab через `.gitlab-ci.yml`.


### Redis keys

- `antibf:bucket:login:<login>`
- `antibf:bucket:password:<password>`
- `antibf:bucket:ip:<ip>`
- `antibf:whitelist`
- `antibf:blacklist`

Для bucket-ов используется Redis Lua script:
- в одном атомарном вызове проверяются три bucket-а;
- если хотя бы один пустой, запрос блокируется;
- если все bucket-ы допускают запрос, у всех трёх списывается по токену и выставляется TTL.

## Запуск локально

```bash
docker compose up --build
```

Проверка:
```bash
curl -X POST http://localhost:8080/api/v1/auth/check \
  -H 'Content-Type: application/json' \
  -d '{"login":"alice","password":"secret","ip":"192.168.1.10"}'
```

## CLI

```bash
go run ./cmd/antibf reset --login alice --ip 192.168.1.10
go run ./cmd/antibf whitelist-add --cidr 192.168.1.0/24
go run ./cmd/antibf blacklist-add --cidr 10.0.0.0/8
```

## GitLab CI

GitLab CI/CD использует `.gitlab-ci.yml` в корне проекта
В пайплайне сейчас есть:
- `lint`
- `test`
- `build`

