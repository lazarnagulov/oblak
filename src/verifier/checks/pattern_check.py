import ast
import zipfile
from models import CheckResult

BANNED_IMPORTS = {
  "socket",
  "subprocess",
  "multiprocessing",
  "ctypes",
  "pty",
  "telnetlib",
  "ftplib",
  "smtplib",
  "http.server",
  "xmlrpc",
  "paramiko",
  "fabric",
}

BANNED_BUILTINS = {
  "eval",
  "exec",
  "compile",
  "__import__",
}

BANNED_ATTRIBUTES = {
  ("os", "system"),
  ("os", "popen"),
  ("os", "execv"),
  ("os", "execve"),
  ("os", "execvp"),
  ("os", "fork"),
  ("os", "spawnl"),
  ("shutil", "rmtree"),
  ("importlib", "import_module"),
}


def analyse_source(filename: str, source: str) -> list[str]:
  try:
    tree = ast.parse(source, filename=filename)
  except SyntaxError as e:
    return [f"Syntax error: {e}"]

  violations = []

  for node in ast.walk(tree):
    match node:
      case ast.Import(names=aliases, lineno=ln):
        for alias in aliases:
          if alias.name.split(".")[0] in BANNED_IMPORTS:
            violations.append(f"Line {ln}: forbidden import '{alias.name}'")

      case ast.ImportFrom(module=mod, lineno=ln) if mod:
        if mod.split(".")[0] in BANNED_IMPORTS:
          violations.append(f"Line {ln}: forbidden import from '{mod}'")

      case ast.Call(func=ast.Name(id=name), lineno=ln) if name in BANNED_BUILTINS:
        violations.append(f"Line {ln}: forbidden call '{name}()'")

      case ast.Call(func=ast.Attribute(value=ast.Name(id=obj), attr=attr), lineno=ln):
        if (obj, attr) in BANNED_ATTRIBUTES:
          violations.append(f"Line {ln}: forbidden call '{obj}.{attr}()'")

  return violations


def run(zf: zipfile.ZipFile) -> CheckResult:
  for name in zf.namelist():
    if not name.endswith(".py"):
      continue
    try:
      source = zf.read(name).decode("utf-8", errors="replace")
    except Exception as e:
      return CheckResult(
          name="pattern_check",
          passed=False,
          reason=f"Could not read '{name}': {e}",
      )

    violations = analyse_source(name, source)
    if violations:
      return CheckResult(
          name="pattern_check",
          passed=False,
          reason=f"In '{name}': {violations[0]}",
      )

  return CheckResult(name="pattern_check", passed=True)