# mkk_basis_test_task
Тестовое задание для МКК Базис

## Запуск проекта

Требования: Go 1.26+, Docker и Docker Compose для локальных MySQL/Redis.

1. При необходимости создать docker-compose из примера в директории /etc/dev/ Поднять инфраструктуру:

```bash
docker compose -f etc/dev/docker-compose.yml up -d
```

2. При необходимости создать локальный конфиг из примера в директории /configs


Значения по умолчанию уже совпадают с `etc/dev/docker-compose.yml`: MySQL `localhost:3306`, Redis `localhost:6379`, база `mkk_basis_tasks`, пользователь `app`.

3. Запустить REST API:

```bash
make run
```

API стартует на `http://0.0.0.0:8080`. Swagger доступен по `/swagger/index.html`, метрики Prometheus по `/metrics`.

4. Запустить проверки:

```bash
go test ./api/... ./cmd/... ./internal/...
```
