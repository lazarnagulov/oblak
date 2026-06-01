import asyncio
import io
import json
import os
import socket
import subprocess
import tempfile
import uuid
import zipfile
import http.client
import time
from pathlib import Path
import logging

from models import ExecuteResult, Manifest, Status

log = logging.getLogger("firecracker_engine")

WORKSPACE = Path.home() / "oblak_firecracker"
FIRECRACKER_BIN = WORKSPACE / "firecracker"
KERNEL_PATH = WORKSPACE / "vmlinux.bin"
ROOTFS_PATH = WORKSPACE / "sandbox.rootfs.ext4"
MAX_OUTPUT_SIZE = 64 * 1024  # 64KB

class UnixSocketConnection(http.client.HTTPConnection):
    """Custom HTTP client to send requests to Firecracker's Unix Socket"""
    def __init__(self, socket_path):
        super().__init__("localhost")
        self.socket_path = socket_path

    def connect(self):
        self.sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        self.sock.connect(self.socket_path)

def api_put(socket_path: str, url_path: str, payload: dict):
    """Sends a PUT request to the Firecracker API"""
    conn = UnixSocketConnection(socket_path)
    headers = {'Accept': 'application/json', 'Content-Type': 'application/json'}
    conn.request('PUT', url_path, body=json.dumps(payload), headers=headers)
    resp = conn.getresponse()
    if resp.status >= 300:
        error_msg = resp.read().decode()
        raise Exception(f"Firecracker API Error {resp.status}: {error_msg}")
    conn.close()

def create_payload_drive(artifact_bytes: bytes, manifest: Manifest, payload: dict, dest_ext4: Path):
    with tempfile.TemporaryDirectory() as tmpdir:
        tmp_path = Path(tmpdir)
        
        with zipfile.ZipFile(io.BytesIO(artifact_bytes)) as zf:
            zf.extractall(tmp_path)
            
        with open(tmp_path / "manifest.json", "w") as f:
            json.dump(manifest.dict(), f)

        with open(tmp_path / "payload.json", "w") as f:
            json.dump(payload, f)
            
        launcher_code = """
import sys
import json
import importlib

sys.path.append('/mnt')

print("__OBLAK_START__")
result = None
try:
    with open('/mnt/manifest.json', 'r') as f:
        manifest = json.load(f)

    mod_name = manifest.get('module', 'cli')
    handler_name = manifest.get('handler', 'handler')

    payload = {}
    try:
        with open('/mnt/payload.json', 'r') as f:
            payload = json.load(f)
    except FileNotFoundError:
        payload = {}

    if payload is None:
        payload = {}
    if not isinstance(payload, dict):
        raise TypeError('payload must be an object')

    mod = importlib.import_module(mod_name)
    handler = getattr(mod, handler_name)
    if payload:
        result = handler(**payload)
    else:
        result = handler()
    print("__OBLAK_END__")

except Exception as e:
    err_str = f"{type(e).__name__}: {str(e)}"
    print(f"Execution Error: {err_str}")
    print("__OBLAK_END__")
    print("__OBLAK_ERROR_START__")
    print(err_str)
    print("__OBLAK_ERROR_END__")

if result is not None:
    print("__OBLAK_RESULT_START__")
    print(json.dumps(result))
    print("__OBLAK_RESULT_END__")
"""
        with open(tmp_path / "launcher.py", "w") as f:
            f.write(launcher_code)

        run_script = """#!/bin/sh
python3 /mnt/launcher.py
"""
        with open(tmp_path / "run.sh", "w") as f:
            f.write(run_script)
            
        subprocess.run(["dd", "if=/dev/zero", f"of={dest_ext4}", "bs=1M", "count=10"], capture_output=True, check=True)
        subprocess.run(["mkfs.ext4", "-F", "-d", str(tmp_path), str(dest_ext4)], capture_output=True, check=True)

async def execute_in_sandbox(manifest: Manifest, artifact_bytes: bytes, payload: dict) -> ExecuteResult:
    run_id = str(uuid.uuid4())
    socket_path = f"/tmp/fc_{run_id}.sock"
    payload_path = WORKSPACE / f"payload_{run_id}.ext4"
    
    timeout = manifest.timeout
    memory = manifest.memory
    module = manifest.module
    handler = manifest.handler

    fc_process = None
    function_logs = ""
    function_result = None
    error_message = None
    status = Status.SUCCESS
    execution_time_ms = 0

    try:
        create_payload_drive(artifact_bytes, manifest, payload, payload_path)

        if os.path.exists(socket_path):
            os.remove(socket_path)
            
        fc_process = await asyncio.create_subprocess_exec(
            str(FIRECRACKER_BIN), "--api-sock", socket_path,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.STDOUT
        )
        
        await asyncio.sleep(0.1)

        # Boot Source
        api_put(socket_path, '/boot-source', {
            "kernel_image_path": str(KERNEL_PATH),
            "boot_args": "console=ttyS0 reboot=k panic=1 pci=off init=/oblak_init quiet loglevel=1"
        })

        # Root Filesystem
        api_put(socket_path, '/drives/rootfs', {
            "drive_id": "rootfs",
            "path_on_host": str(ROOTFS_PATH),
            "is_root_device": True,
            "is_read_only": True # RootOS cannot be altered
        })
        
        # Payload Drive (The User Code)
        api_put(socket_path, '/drives/payload', {
            "drive_id": "payload",
            "path_on_host": str(payload_path),
            "is_root_device": False,
            "is_read_only": True # Code cannot modify itself during execution
        })
        
        # Machine Config
        api_put(socket_path, '/machine-config', {
            "vcpu_count": 1,
            "mem_size_mib": memory
        })

        api_put(socket_path, '/actions', {"action_type": "InstanceStart"})

        async def read_stream(stream):
            stdout_data = bytearray()
            while True:
                chunk = await stream.read(8192)
                if not chunk:
                    break
                stdout_data.extend(chunk)

                if len(stdout_data) > MAX_OUTPUT_SIZE:
                    log.warning(f"[{run_id}] Output limit exceeded. Terminating VM.")
                    if fc_process and fc_process.returncode is None:
                        fc_process.kill()
                    return stdout_data[:MAX_OUTPUT_SIZE].decode('utf-8', errors='ignore') + "\n...[output truncated]..." + "__OBLAK_END__"
            return stdout_data.decode('utf-8', errors='ignore')

        start_time = time.perf_counter()
        raw_output = await asyncio.wait_for(read_stream(fc_process.stdout), timeout=timeout + 1)
        end_time = time.perf_counter()
        execution_time_ms = int((end_time - start_time) * 1000)

        if "__OBLAK_START__" in raw_output and "__OBLAK_END__" in raw_output:
            start_idx = raw_output.index("__OBLAK_START__") + len("__OBLAK_START__")
            end_idx = raw_output.index("__OBLAK_END__")
            function_logs = raw_output[start_idx:end_idx].strip()
            status = Status.SUCCESS
        else:
            log.error(f"[{run_id}] Execution did not produce output markers. Log snippet:\n{raw_output[:2000]}")
            error_message = "Error: Sandbox execution failed unexpectedly."
            status = Status.FAILED

        if "__OBLAK_ERROR_START__" in raw_output and "__OBLAK_ERROR_END__" in raw_output:
            err_start = raw_output.index("__OBLAK_ERROR_START__") + len("__OBLAK_ERROR_START__")
            err_end = raw_output.index("__OBLAK_ERROR_END__")
            error_raw = raw_output[err_start:err_end].strip()
            if error_raw:
                error_message = error_raw
                status = Status.FAILED

        if status == Status.SUCCESS and "__OBLAK_RESULT_START__" in raw_output and "__OBLAK_RESULT_END__" in raw_output:
            res_start = raw_output.index("__OBLAK_RESULT_START__") + len("__OBLAK_RESULT_START__")
            res_end = raw_output.index("__OBLAK_RESULT_END__")
            result_raw = raw_output[res_start:res_end].strip()
            try:
                function_result = json.loads(result_raw)
            except json.JSONDecodeError:
                function_result = result_raw
        


    except asyncio.TimeoutError:
        log.warning(f"[{run_id}] Execution timeout. Terminating sandbox.")
        if fc_process:
            fc_process.kill()
        error_message = f"Error: Function timed out after {timeout} seconds."
        status = Status.TIMEOUT
        execution_time_ms = timeout * 1000
        
    except Exception as e:
        log.error(f"[{run_id}] Infrastructure/System Error: {str(e)}", exc_info=True)
        error_message = "Error: Internal infrastructure error during sandbox initialization."
        status = Status.FAILED

    finally:
        if fc_process and fc_process.returncode is None:
            try:
                fc_process.kill()
            except ProcessLookupError:
                pass
                
        if os.path.exists(socket_path):
            os.remove(socket_path)
            
        if os.path.exists(payload_path):
            os.remove(payload_path)

    return ExecuteResult(
         status=status, 
         logs=function_logs, 
         error_message=error_message, 
         result=function_result, 
         execution_time_ms=execution_time_ms
    )