import re
import urllib.request
import urllib.error
import zipfile

from models import CheckResult

# typosquatting - popular packages which are often targeted by typosquatting attacks, normalized to lowercase and without dashes/underscores
POPULAR_PACKAGES = {
  "requests", "numpy", "pandas", "flask", "django", "fastapi",
  "sqlalchemy", "pydantic", "boto3", "pillow", "scipy", "matplotlib",
  "tensorflow", "torch", "scikit-learn", "colorama", "cryptography",
  "paramiko", "urllib3", "certifi", "setuptools", "pip",
}

# minimal Levenshtein distance to consider as typosquatting
TYPOSQUATTING_THRESHOLD = 2

VALID_LINE = re.compile(
  r"^([A-Za-z0-9]([A-Za-z0-9._-]*[A-Za-z0-9])?)"  # package name
  r"\s*==\s*"                                     # only allow '==' operator for pinned versions
  r"[0-9][A-Za-z0-9.*+-]*$"                       # version
)

# line that has a name but no version (just the package name)
UNPINNED_LINE = re.compile(r"^([A-Za-z0-9][A-Za-z0-9._-]*)$")


def _levenshtein(a: str, b: str) -> int:
  if len(a) < len(b):
    a, b = b, a
  if not b:
    return len(a)

  prev = list(range(len(b) + 1))
  for i, ca in enumerate(a):
    curr = [i + 1]
    for j, cb in enumerate(b):
      curr.append(min(prev[j + 1] + 1, curr[j] + 1, prev[j] + (ca != cb)))
    prev = curr
  return prev[-1]


def _is_typosquatting(name: str) -> str | None:
  name_lower = name.lower().replace("-", "").replace("_", "")
  for popular in POPULAR_PACKAGES:
    popular_norm = popular.lower().replace("-", "").replace("_", "")
    if name_lower == popular_norm:
      return None
    dist = _levenshtein(name_lower, popular_norm)
    if 0 < dist <= TYPOSQUATTING_THRESHOLD:
      return popular
  return None


def _exists_on_pypi(package_name: str) -> bool:
  url = f"https://pypi.org/pypi/{package_name}/json"
  try:
    req = urllib.request.Request(url, method="HEAD")
    with urllib.request.urlopen(req, timeout=5):
      return True
  except urllib.error.HTTPError as e:
    if e.code == 404:
      return False
    return True
  except Exception:
    return True


def _parse_requirements(content: str) -> list[tuple[int, str]]:
  packages = []
  for i, raw_line in enumerate(content.splitlines(), start=1):
    line = raw_line.strip()
    if not line or line.startswith("#"):
      continue
    line = line.split("#")[0].strip()
    if line:
      packages.append((i, line))
  return packages


def run(zf: zipfile.ZipFile) -> CheckResult:
  req_files = [n for n in zf.namelist() if n.lower().endswith("requirements.txt")]

  if not req_files:
    return CheckResult(name="requirements", passed=True, reason="no requirements.txt found")

  req_file = req_files[0]
  try:
    content = zf.read(req_file).decode("utf-8", errors="replace")
  except Exception as e:
    return CheckResult(
      name="requirements",
      passed=False,
      reason=f"Could not read '{req_file}': {e}",
    )

  lines = _parse_requirements(content)

  for lineno, line in lines:
    if UNPINNED_LINE.match(line):
      return CheckResult(
        name="requirements",
        passed=False,
        reason=f"Line {lineno}: '{line}' has no pinned version (use ==)",
      )

    if not VALID_LINE.match(line):
      return CheckResult(
        name="requirements",
        passed=False,
        reason=f"Line {lineno}: invalid format '{line}'",
      )

    package_name = re.split(r"[><=!~]", line)[0].strip()

    similar_to = _is_typosquatting(package_name)
    if similar_to:
      return CheckResult(
        name="requirements",
        passed=False,
        reason=(
          f"Line {lineno}: '{package_name}' looks like typosquatting of '{similar_to}'"
        ),
      )

    if not _exists_on_pypi(package_name):
      return CheckResult(
        name="requirements",
        passed=False,
        reason=f"Line {lineno}: '{package_name}' not found on PyPI",
      )

  return CheckResult(name="requirements", passed=True)