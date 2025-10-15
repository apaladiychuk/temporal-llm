# Архітектура Temporal LLM пайплайна

Цей документ описує референсну реалізацію пайплайна для запуску LLM inference через Temporal.

## Компоненти

| Компонент | Мова | Призначення |
|-----------|------|-------------|
| Temporal Server | - | Оркестрація workflow та activity. |
| Go Gateway Service | Go | REST API для UI, запуск Workflow, worker для orchestration та notifications. |
| Python GPU Worker | Python | Виконує довгі GPU-активності (LLM inference). |
| UI | будь-яка | Надсилає запити, отримує нотифікації, опитує статус. |

## Temporal Namespace та Task Queues

- Namespace: `prod` (або `dev` на staging).
- Workflow Task Queue: `go-gateway-workflows` — воркер у Go виконує workflow `LLMJobWorkflow`.
- GPU Activity Task Queue: `llm-gpu-activities` — Python worker на GPU хості виконує activity `RunLLMOnGPU`.
- Notification Activity Task Queue: `notifications-activities` — Go worker пушить подію в UI (webhook/WS/Kafka).

## Потік подій

1. UI відправляє `POST /jobs` у Go gateway. Payload включає бізнес-ключ `user_id` + `request_id`, модель, prompt та параметри.
2. Gateway створює workflow `LLMJobWorkflow` зі `workflowId = llmjob-<user>-<request>` (ідемпотентність).
3. Workflow викликає activity `RunLLMOnGPU` у task queue `llm-gpu-activities`. Python worker heartbeat-ить прогрес.
4. Після успішного завершення workflow викликає activity `NotifyUI`, яка пушить повідомлення у UI hub/webhook.
5. UI може викликати Temporal Query `GetStatus` через REST `GET /jobs/{workflowId}/status` або Signal `Cancel` через `POST /jobs/{workflowId}/cancel`.

## Контракти даних

У репозиторії використовується JSON payload (сумісний між Go та Python за замовчуванням). За потреби легко замінити на Protobuf: додайте payload converter у Go і Python та генеруйте типи із `proto/llm.proto`.

Основні структури:

```jsonc
{
  "user_id": "user-123",
  "request_id": "req-456",
  "model": "llama-3.1-70b-instruct",
  "prompt": "...",
  "params": {"temperature": "0.1"}
}
```

Прогрес (`JobProgress`): `%`, `stage`, `message`, `updated_at`.

Результат (`JobResult`): `output` (текст) + метадані (`tokens`, `latency`, `gpu` тощо).

## Таймаути та ретраї

- `RunLLMOnGPU`:
  - `StartToCloseTimeout`: 2 години (налаштовується).
  - `HeartbeatTimeout`: 30 секунд (Python воркер heartbeat-ить частіше).
  - `ScheduleToStartTimeout`: 5 хвилин (щоб таска не висіла без GPU).
  - `RetryPolicy`: `InitialInterval=10s`, `Backoff=2.0`, `MaximumAttempts=3` (налаштовується).
- Workflow timeout: 8 годин (через `WorkflowExecutionTimeout`).
- Cancel виконується через Signal, Python activity повинна обробити `activity.is_cancelled()` та коректно завершитись.

## REST API Gateway

- `POST /jobs`: стартує workflow, повертає `workflow_id`, `run_id`.
- `GET /jobs/{workflowId}/status`: виконує Temporal Query `GetStatus`.
- `POST /jobs/{workflowId}/cancel`: відправляє Signal `Cancel`.

Notification worker може виконати webhook, WS broadcast або публікацію у Kafka/NATS.

## Observability

- Heartbeat прогрес доступний в Temporal Web UI.
- Метрики/логи варто скеровувати у Prometheus/Grafana або Loki.
- Додаткові Search Attributes: `UserID`, `RequestID`, `Model`, `ProjectId`, `LatencyMs` — додаються через Temporal CLI.

