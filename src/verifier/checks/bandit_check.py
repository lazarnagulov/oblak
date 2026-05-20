import json
import subprocess
import tempfile
import zipfile

from models import CheckResult

BLOCKED_SEVERITIES = {"HIGH", "CRITICAL"}


def run(zf: zipfile.ZipFile) -> CheckResult:
  py_files = [name for name in zf.namelist() if name.endswith(".py")]
  if not py_files:
    return CheckResult(name="bandit", passed=True, reason="no Python files to scan")

  with tempfile.TemporaryDirectory() as tmpdir:
    for name in py_files:
      zf.extract(name, tmpdir)

    try:
      result = subprocess.run(
        [
          "bandit",
          "-r", tmpdir,
          "--severity-level", "high",
          "--confidence-level", "medium",
          "--format", "json",
          "--quiet",
        ],
        capture_output=True,
        text=True,
        timeout=30,
      )
    except FileNotFoundError:
      return CheckResult(name="bandit", passed=True, reason="bandit not installed, skipped")
    except subprocess.TimeoutExpired:
      return CheckResult(name="bandit", passed=False, reason="bandit scan timed out")

    if result.returncode == 0:
      return CheckResult(name="bandit", passed=True)

    try:
      report = json.loads(result.stdout)
    except json.JSONDecodeError:
      return CheckResult(name="bandit", passed=False, reason="bandit returned unparseable output")

    issues = report.get("results", [])
    blocked = [
      i for i in issues
      if i.get("issue_severity", "").upper() in BLOCKED_SEVERITIES
    ]

    if not blocked:
      return CheckResult(name="bandit", passed=True)

    reasons = [
      f"[{i['issue_severity']}] {i['issue_text']} in {i['filename']}:{i['line_number']}"
      for i in blocked
    ]
    return CheckResult(name="bandit", passed=False, reason="\n".join(reasons))