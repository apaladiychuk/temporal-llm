# Temporal LLM Reference Stack

Репозиторій показує, як зібрати пайплайн для запуску LLM inference через Temporal із трьома сервісами:

1. **Go gateway service** — REST API + Temporal workflow/notification workers.
2. **Python GPU worker** — довгі GPU-активності.
3. **Temporal server** — оркестрація workflow/activities.

## Швидкий старт

```bash
docker compose up --build
```

Команда підніме Temporal (`temporalio/auto-setup`), Temporal Web UI, Go gateway (порт `8080`) та Python worker.

Після запуску можна:

1. Створити job:
   ```bash
   curl -X POST http://localhost:8080/jobs \
     -H 'Content-Type: application/json' \
     -d '{"user_id":"user-1","request_id":"req-1","model":"llama-3.1","prompt":"Hello"}'
   ```
2. Перевірити статус:
   ```bash
   curl http://localhost:8080/jobs/llmjob-user-1-req-1/status
   ```
3. Скасувати job:
   ```bash
   curl -X POST http://localhost:8080/jobs/llmjob-user-1-req-1/cancel
   ```

Temporal Web UI доступний на http://localhost:8088.

Деталі архітектури та контрактів описані в [docs/architecture.md](docs/architecture.md).

## Конфігурація середовища

- `TEMPORAL_ADDRESS`: адреса Temporal (`temporal:7233` за замовчуванням у docker-compose).
- `TEMPORAL_NAMESPACE`: namespace (за замовчуванням `default`).
- `NOTIFICATIONS_WEBHOOK_URL`: опціональний webhook для NotifyUI activity.
- `MAX_CONCURRENT_ACTIVITIES`: для Python worker, кількість паралельних GPU задач.

## Розробка

Go код використовує модуль `github.com/example/temporal-llm`. Для локального запуску поза Docker достатньо мати Temporal (Cloud або локальний) та виконати:

```bash
go run ./cmd/gateway
```

Python worker:

```bash
cd python_worker
pip install -r requirements.txt
python worker.py
```

## Docker образи

- `gateway` будується з `cmd/gateway`.
- `python-worker` будується з `python_worker`.

Docker Compose файли налаштовані для локального дев-середовища та не призначені для продакшн без доопрацювань (mTLS, secrets, observability).

