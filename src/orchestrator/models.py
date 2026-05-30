from dataclasses import dataclass, field


@dataclass
class Manifest:
  name: str
  runtime: str
  module: str
  handler: str
  timeout: int
  memory: int

  def dict(self):
    return {
      "name": self.name,
      "runtime": self.runtime,
      "module": self.module,
      "handler": self.handler,
      "timeout": self.timeout,
      "memory": self.memory,
    }


@dataclass
class ExecuteRequest:
  artifact_b64: str
  manifest: dict
  payload: dict = field(default_factory=dict)

  def parse_manifest(self) -> Manifest:
    m = self.manifest
    return Manifest(
      name=m.get("name", ""),
      runtime=m.get("runtime", ""),
      module=m.get("module", ""),
      handler=m.get("handler", ""),
      timeout=m.get("timeout", 5),
      memory=m.get("memory", 128),
    )


@dataclass
class ExecuteResponse:
  success: bool
  output: str = ""