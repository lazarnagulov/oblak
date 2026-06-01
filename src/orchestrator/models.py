from dataclasses import dataclass, field
from typing import Dict


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
  

class Status(Dict):
    SUCCESS: str = "SUCCESS"
    FAILED: str = "FAILED"
    TIMEOUT: str = "TIMEOUT"
    

@dataclass
class ExecuteResult:
  status: str = Status.SUCCESS
  logs: str = ""
  error_message: str = ""
  result: any = None
  execution_time_ms: int = 0


@dataclass
class ExecuteResponse:
  status: str = Status.SUCCESS
  logs: str = ""
  error_message: str = ""
  result: any = None
  execution_time_ms: int = 0
  worker_node: str = ""
  
  def dict(self):
    return {
      "status": self.status,
      "logs": self.logs,
      "error_message": self.error_message,
      "result": self.result,
      "execution_time_ms": self.execution_time_ms,
      "worker_node": self.worker_node,
    }
  

class OrchestratorError(Exception):
    pass

class RequirementsError(OrchestratorError):
    pass