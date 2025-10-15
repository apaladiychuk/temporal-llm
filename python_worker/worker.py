import asyncio
import logging
import os
from typing import Any, Dict

from temporalio import activity
from temporalio.client import Client
from temporalio.worker import Worker

logger = logging.getLogger("llm_worker")
logging.basicConfig(level=logging.INFO)


@activity.defn(name="RunLLMOnGPU")
async def run_llm_on_gpu(input_payload: Dict[str, Any]) -> Dict[str, Any]:
    """Мінімальна імплементація GPU-активності."""
    user_id = input_payload.get("user_id")
    request_id = input_payload.get("request_id")
    model = input_payload.get("model")

    logger.info("starting generation user=%s request=%s model=%s", user_id, request_id, model)

    progress_payload = {
        "percent": 5,
        "stage": "initializing",
        "message": "warming up GPU",
    }
    await activity.heartbeat(progress_payload)

    # Тут потрібно викликати реальну LLM inference.
    await asyncio.sleep(2)

    await activity.heartbeat({
        "percent": 60,
        "stage": "generating",
        "message": "decoding tokens",
    })

    await asyncio.sleep(2)

    if activity.is_cancelled():
        logger.info("activity cancelled user=%s request=%s", user_id, request_id)
        raise asyncio.CancelledError()

    result_text = f"Generated text for {user_id}:{request_id} using {model}"

    await activity.heartbeat({
        "percent": 100,
        "stage": "finalizing",
        "message": "writing outputs",
    })

    result = {
        "output": result_text,
        "meta": {
            "model": model or "unknown",
            "duration_ms": "4000",
        },
    }
    return result


async def main() -> None:
    temporal_address = os.getenv("TEMPORAL_ADDRESS", "temporal:7233")
    namespace = os.getenv("TEMPORAL_NAMESPACE", "default")

    client = await Client.connect(temporal_address, namespace=namespace)

    worker = Worker(
        client,
        task_queue="llm-gpu-activities",
        activities=[run_llm_on_gpu],
        max_concurrent_activities=int(os.getenv("MAX_CONCURRENT_ACTIVITIES", "1")),
    )

    logger.info("Python worker connected to %s namespace=%s", temporal_address, namespace)
    await worker.run()


if __name__ == "__main__":
    asyncio.run(main())

