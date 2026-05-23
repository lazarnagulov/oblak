import asyncio
import base64
import io
import os
import json
import logging
import struct
import sys
import zipfile
import uuid
from models import Manifest, ExecuteRequest, ExecuteResponse

logging.basicConfig(
  level=logging.INFO, 
  format="%(asctime)s [%(levelname)s] %(message)s"
)
log = logging.getLogger("orchestrator")

async def read_message(reader: asyncio.StreamReader):
    length_bytes = await reader.readexactly(4)
    length = struct.unpack(">I", length_bytes)[0]
    data = await reader.readexactly(length)
    return json.loads(data.decode("utf-8"))

async def write_message(writer: asyncio.StreamWriter, payload: dict):
    data = json.dumps(payload).encode("utf-8")
    writer.write(struct.pack(">I", len(data)) + data)
    await writer.drain()

async def run_in_firecracker(manifest: Manifest, artifact_bytes: bytes) -> ExecuteResponse:
    """This function handles the entire Firecracker lifecycle"""
    run_id = str(uuid.uuid4())
    log.info(f"[{run_id}] Starting sandbox for {manifest.name}")
    
    try:
      zf = zipfile.ZipFile(io.BytesIO(artifact_bytes))
    except zipfile.BadZipFile:
      return ExecuteResponse(success=False, output="Not a valid zip file")

    timeout = manifest.timeout
    memory = manifest.memory
    handler = manifest.handler
    module = manifest.module
    runtime = manifest.runtime

    try:
        # await asyncio.wait_for(wait_for_firecracker_to_finish(run_id), timeout=timeout)
        
        # Mock result for now
        await asyncio.sleep(1) # Simulate execution time
        result_output = "Hello from MVM."
        success = True
        
    except asyncio.TimeoutError:
        log.error(f"[{run_id}] Execution timeout. Killing MVM.")
        # TODO: kill -9 firecracker_process
        result_output = "Error: Function timed out."
        success = False
        
    finally:
        # TODO: Delete sockets, temporary payload drives, and ensure process is dead
        log.info(f"[{run_id}] Cleaning up resources.")

    return ExecuteResponse(
        success=success,
        output=result_output
    )

async def handle_client(reader: asyncio.StreamReader, writer: asyncio.StreamWriter):
    log.info("New connection to Orchestrator")
    try:
        raw = await read_message(reader)
        request = ExecuteRequest(
            artifact_b64=raw.get("artifact_b64", ""),
            manifest=raw.get("manifest", {}),
        )
        manifest = request.parse_manifest()
        log.info(f"Received request to execute script '{manifest.name}'")
        
        artifact_bytes = base64.b64decode(request.artifact_b64)
        result = await run_in_firecracker(manifest, artifact_bytes)

        await write_message(writer, {
            "success": result.success,
            "output": result.output
        })

    except Exception as e:
        log.error("Unhandled error in orchestrator: %s", e)
        await write_message(writer, {"success": False, "output": str(e)})
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