from dataclasses import dataclass, field


@dataclass
class Manifest:
  name: str
  runtime: str
  module: str
  handler: str
  timeout: int
  memory: int


@dataclass
class CheckResult:
  name: str
  passed: bool
  reason: str = ""


@dataclass
class VerifyRequest:
  artifact_b64: str
  manifest: dict

  def parse_manifest(self) -> Manifest:
    m = self.manifest
    return Manifest(
      name=m.get("name", ""),
      runtime=m.get("runtime", ""),
      module=m.get("module", ""),
      handler=m.get("handler", ""),
      timeout=m.get("timeout", 0),
      memory=m.get("memory", 0),
    )


@dataclass
class VerifyResponse:
  safe: bool
  reason: str = ""
  checks: list[CheckResult] = field(default_factory=list)