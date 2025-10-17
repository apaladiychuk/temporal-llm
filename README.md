# Temporal LLM Reference Stack

Репозиторій показує, як зібрати пайплайн для запуску LLM inference через Temporal із чотирма сервісами:

1. **Go gateway API** — REST API для UI, стартує Temporal workflow.
2. **Go Temporal worker** — виконує workflow `LLMJobWorkflow` та activity `NotifyUI`.
3. **Python GPU worker** — довгі GPU-активності.
4. **Temporal server** — оркестрація workflow/activities.

## Швидкий старт

```bash
docker compose up --build
```

Команда підніме Temporal (`temporalio/auto-setup`), Temporal Web UI, Go gateway API (порт `8080`), Go Temporal worker та Python worker.

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
go run ./cmd/gateway-worker
```

Python worker:

```bash
cd python_worker
pip install -r requirements.txt
python worker.py
```

## Docker образи

- `gateway-api` будується з `cmd/gateway`.
- `gateway-worker` будується з `cmd/gateway-worker`.
- `python-worker` будується з `python_worker`.

Docker Compose файли налаштовані для локального дев-середовища та не призначені для продакшн без доопрацювань (mTLS, secrets, observability).

