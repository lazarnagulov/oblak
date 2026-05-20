import os
import zipfile

from models import CheckResult

MAX_FILES = 50
MAX_FILE_SIZE_MB = 5
ALLOWED_EXTENSIONS = {".py", ".txt", ".json", ".cfg", ".toml"}

def check_zip_slip(zf: zipfile.ZipFile) -> CheckResult:
  for name in zf.namelist():
    normalized = os.path.normpath(name)
    if normalized.startswith("..") or os.path.isabs(normalized):
      return CheckResult(
        name="Zip Slip",
        passed=False,
        reason=f"File '{name}' is trying to escape the target directory."
      )
  return CheckResult(name="Zip Slip", passed=True)

def check_file_count(zf: zipfile.ZipFile) -> CheckResult:
  count = len(zf.namelist())
  if count > MAX_FILES:
    return CheckResult(
      name="file_count",
      passed=False,
      reason=f"Too many files: {count} (max {MAX_FILES})",
    )
  return CheckResult(name="file_count", passed=True)

def check_file_sizes(zf: zipfile.ZipFile) -> CheckResult:
  for info in zf.infolist():
    size_mb = info.file_size / (1024 * 1024)
    if size_mb > MAX_FILE_SIZE_MB:
      return CheckResult(
        name="file_sizes",
        passed=False,
        reason=f"File too large: '{info.filename}' ({size_mb:.1f} MB, max {MAX_FILE_SIZE_MB} MB)",
      )
  return CheckResult(name="file_sizes", passed=True)

def check_extensions(zf: zipfile.ZipFile) -> CheckResult:
  for name in zf.namelist():
    if name.endswith("/"):
      continue  # dir
    _, ext = os.path.splitext(name.lower())
    if ext not in ALLOWED_EXTENSIONS:
      return CheckResult(
        name="extensions",
        passed=False,
        reason=f"Disallowed file type: '{name}'",
      )
  return CheckResult(name="extensions", passed=True)

def run(zf: zipfile.ZipFile) -> list[CheckResult]:
  return [
    check_zip_slip(zf),
    check_file_count(zf),
    check_file_sizes(zf),
    check_extensions(zf),
  ]