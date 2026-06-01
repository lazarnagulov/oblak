import asyncio
import base64
import os
import json
import logging
import struct
import sys
from firecracker_engine import execute_in_sandbox
from models import Manifest, ExecuteRequest, ExecuteResponse, OrchestratorError, Status

logging.basicConfig(
  level=logging.INFO, 
  format="%(asctime)s [%(levelname)s] %(message)s"
)
log = logging.getLogger("orchestrator")
WORKER_NODE = os.getenv("WORKER_NODE", "linux-worker")

async def read_message(reader: asyncio.StreamReader):
    length_bytes = await reader.readexactly(4)
    length = struct.unpack(">I", length_bytes)[0]
    data = await reader.readexactly(length)
    return json.loads(data.decode("utf-8"))

async def write_message(writer: asyncio.StreamWriter, payload: dict):
    data = json.dumps(payload).encode("utf-8")
    writer.write(struct.pack(">I", len(data)) + data)
    await writer.drain()

def normalize_payload(raw_payload) -> dict:
    if raw_payload is None or raw_payload == "" or raw_payload == {} or raw_payload == [] or raw_payload == "null":
        return {}
    if isinstance(raw_payload, str):
        try:
            raw_payload = json.loads(raw_payload)
            if isinstance(raw_payload, str):
                raw_payload = json.loads(raw_payload)
        except (TypeError, ValueError):
            raise OrchestratorError("payload must be an object")
        except json.JSONDecodeError as e:
            raise OrchestratorError(f"payload is not valid JSON: {str(e)}")
    if not isinstance(raw_payload, dict):
        raise OrchestratorError("payload must be an object")
    for key in raw_payload.keys():
        if not isinstance(key, str):
            raise OrchestratorError("payload keys must be strings")
    return raw_payload

async def run_in_firecracker(manifest: Manifest, artifact_bytes: bytes, payload: dict) -> ExecuteResponse:
    result = await execute_in_sandbox(manifest, artifact_bytes, payload)
    
    return ExecuteResponse(
        status=result.status,
        logs=result.logs,
        error_message=result.error_message,
        result=result.result,
        execution_time_ms=result.execution_time_ms,
        worker_node=WORKER_NODE
    )

async def handle_client(reader: asyncio.StreamReader, writer: asyncio.StreamWriter):
    log.info("New connection to Orchestrator")
    try:
        raw = await read_message(reader)
        request = ExecuteRequest(
            artifact_b64=raw.get("artifact_b64", ""),
            manifest=raw.get("manifest", {}),
            payload=raw.get("payload", {}),
        )
        payload = normalize_payload(request.payload)
        manifest = request.parse_manifest()
        log.info(f"Received request to execute script '{manifest.name}'")
        
        artifact_bytes = base64.b64decode(request.artifact_b64)
        result = await run_in_firecracker(manifest, artifact_bytes, payload)

        log.info(f"Execution of '{manifest.name}' completed with status: {result.status}")
        await write_message(writer, result.dict())

    except OrchestratorError as e:
        log.error("Orchestrator error: %s", e)
        result = ExecuteResponse(status=Status.FAILED, logs="", error_message=str(e), result=None, execution_time_ms=0, worker_node=WORKER_NODE)
        await write_message(writer, result.dict())
    except Exception as e:
        log.error("Unhandled error in orchestrator: %s", e)
        result = ExecuteResponse(status=Status.FAILED, logs="", error_message="Internal orchestrator error", result=None, execution_time_ms=0, worker_node=WORKER_NODE)
        await write_message(writer, result.dict())
    finally:
        writer.close()
        await writer.wait_closed()

async def main():
    # Orchestrator only works on Linux since it depends on the Firecracker
    if not sys.platform.startswith("linux"):
        log.error("Orchestrator must be run on a Linux system.")
        sys.exit(1)

    SOCKET_PATH = "/tmp/oblak_orchestrator.sock"
    if os.path.exists(SOCKET_PATH):
        os.remove(SOCKET_PATH)
        
    server = await asyncio.start_unix_server(handle_client, path=SOCKET_PATH)
    os.chmod(SOCKET_PATH, 0o600)
    log.info("Orchestrator listening on %s", SOCKET_PATH)

    async with server:
        await server.serve_forever()

if __name__ == "__main__":
    asyncio.run(main())