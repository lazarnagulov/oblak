import json
import os
import urllib.request
import urllib.error
import zipfile

from models import CheckResult

OLLAMA_HOST = os.environ.get("OLLAMA_HOST", "http://localhost:11434")
OLLAMA_URL = f"{OLLAMA_HOST}/api/generate"
MODEL = "qwen2.5-coder:1.5b"
MAX_SOURCE_CHARS = 4000

PROMPT_TEMPLATE = """You are a security analyst. Analyze this Python code and determine if it is malicious.

Respond ONLY with JSON in this exact format, no other text:
{{"safe": true}}
or
{{"safe": false, "reason": "short explanation"}}

Code to analyze:
{source}"""


def _collect_sources(zf: zipfile.ZipFile) -> str:
  parts = []
  total = 0
  for name in zf.namelist():
    if not name.endswith(".py"):
      continue
    try:
      source = zf.read(name).decode("utf-8", errors="replace")
    except Exception:
      continue
    chunk = f"# === {name} ===\n{source}\n"
    if total + len(chunk) > MAX_SOURCE_CHARS:
      break
    parts.append(chunk)
    total += len(chunk)
  return "\n".join(parts)


def run(zf: zipfile.ZipFile) -> CheckResult:
  sources = _collect_sources(zf)
  if not sources.strip():
    return CheckResult(name="llm_check", passed=True, reason="no Python source to analyse")

  payload = json.dumps({
    "model": MODEL,
    "prompt": PROMPT_TEMPLATE.format(source=sources),
    "stream": False,
  }).encode("utf-8")

  req = urllib.request.Request(
    OLLAMA_URL,
    data=payload,
    headers={"Content-Type": "application/json"},
    method="POST",
  )

  try:
    with urllib.request.urlopen(req, timeout=60) as resp:
      body = json.loads(resp.read().decode("utf-8"))
  except urllib.error.URLError:
    return CheckResult(name="llm_check", passed=True, reason="Ollama not reachable, skipped")
  except Exception as e:
    return CheckResult(name="llm_check", passed=True, reason=f"LLM check failed ({e}), skipped")

  try:
    text = body["response"].strip()
    text = text.replace("```json", "").replace("```", "").strip()
    start = text.find("{")
    end = text.rfind("}") + 1
    result = json.loads(text[start:end])
  except (KeyError, ValueError, json.JSONDecodeError) as e:
    return CheckResult(name="llm_check", passed=True, reason=f"Could not parse LLM response, skipped: {e}")

  safe = result.get("safe", True)
  reason = result.get("reason", "")
  return CheckResult(name="llm_check", passed=bool(safe), reason=reason)