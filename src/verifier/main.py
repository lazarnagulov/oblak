import asyncio
import base64
import io
import json
import logging
import struct
import zipfile

from checks import zip_safety, pattern_check, bandit_check, requirements_check, llm_check
from models import CheckResult, VerifyRequest, VerifyResponse

HOST = "127.0.0.1"
PORT = 9876

logging.basicConfig(
  level=logging.INFO,
  format="%(asctime)s [%(levelname)s] %(message)s",
)
log = logging.getLogger("verifier")


# 4-byte length-prefixed JSON protocol
async def read_message(reader: asyncio.StreamReader):
  length_bytes = await reader.readexactly(4)
  length = struct.unpack(">I", length_bytes)[0]
  data = await reader.readexactly(length)
  return json.loads(data.decode("utf-8"))


async def write_message(writer: asyncio.StreamWriter, payload: dict):
  data = json.dumps(payload).encode("utf-8")
  writer.write(struct.pack(">I", len(data)) + data)
  await writer.drain()


def run_checks(artifact_bytes: bytes) -> VerifyResponse:
  try:
    zf = zipfile.ZipFile(io.BytesIO(artifact_bytes))
  except zipfile.BadZipFile:
    return VerifyResponse(safe=False, reason="Not a valid zip file")

  all_results: list[CheckResult] = []

  # 1. Zip safety checks
  zip_results = zip_safety.run(zf)
  all_results.extend(zip_results)
  for r in zip_results:
    if not r.passed:
      return VerifyResponse(safe=False, reason=r.reason, checks=all_results)
    
  # 2. AST pattern check
  pattern_result = pattern_check.run(zf)
  all_results.append(pattern_result)
  if not pattern_result.passed:
    return VerifyResponse(safe=False, reason=pattern_result.reason, checks=all_results)
  
  # 3. Bandit static analysis
  bandit_result = bandit_check.run(zf)
  all_results.append(bandit_result)
  if not bandit_result.passed:
    return VerifyResponse(safe=False, reason=bandit_result.reason, checks=all_results)
  
  # 4. Requirements check
  req_result = requirements_check.run(zf)
  all_results.append(req_result)
  if not req_result.passed:
    return VerifyResponse(safe=False, reason=req_result.reason, checks=all_results)
  
  # 5. LLM check
  llm_result = llm_check.run(zf)
  all_results.append(llm_result)
  if not llm_result.passed:
    return VerifyResponse(safe=False, reason=llm_result.reason, checks=all_results)

  return VerifyResponse(safe=True, checks=all_results)


async def handle_client(reader: asyncio.StreamReader, writer: asyncio.StreamWriter):
  log.info("New connection")
  try:
    raw = await read_message(reader)
    request = VerifyRequest(
      artifact_b64=raw.get("artifact_b64", ""),
      manifest=raw.get("manifest", {}),
    )
    manifest = request.parse_manifest()
    log.info("Verifying function '%s'", manifest.name)

    artifact_bytes = base64.b64decode(request.artifact_b64)
    response = run_checks(artifact_bytes)

    log.info(
      "Result for '%s': safe=%s reason='%s'",
      manifest.name, response.safe, response.reason,
    )

    await write_message(writer, {
      "safe": response.safe,
      "reason": response.reason,
    })

  except Exception as e:
    log.error("Unhandled error: %s", e)
    await write_message(writer, {"safe": False, "reason": "internal verifier error"})
  finally:
    writer.close()
    await writer.wait_closed()


async def main():
  # WINDOWS (TCP)
  server = await asyncio.start_server(handle_client, host=HOST, port=PORT)
  log.info("Verifier listening on %s:%d", HOST, PORT)

  # LINUX (Unix socket)
  # SOCKET_PATH = "/tmp/oblak_verifier.sock"
  # if os.path.exists(SOCKET_PATH):
  #   os.remove(SOCKET_PATH)
  # server = await asyncio.start_unix_server(handle_client, path=SOCKET_PATH)
  # os.chmod(SOCKET_PATH, 0o600)
  # log.info("Verifier listening on %s", SOCKET_PATH)

  async with server:
    await server.serve_forever()


if __name__ == "__main__":
  asyncio.run(main())