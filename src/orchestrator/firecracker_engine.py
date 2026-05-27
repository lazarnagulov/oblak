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
from pathlib import Path
import logging

from models import Manifest

log = logging.getLogger("firecracker_engine")

WORKSPACE = Path.home() / "oblak_firecracker"
FIRECRACKER_BIN = WORKSPACE / "firecracker"
KERNEL_PATH = WORKSPACE / "vmlinux.bin"
ROOTFS_PATH = WORKSPACE / "sandbox.rootfs.ext4"

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

def create_payload_drive(artifact_bytes: bytes, manifest: Manifest, dest_ext4: Path):
    with tempfile.TemporaryDirectory() as tmpdir:
        tmp_path = Path(tmpdir)
        
        with zipfile.ZipFile(io.BytesIO(artifact_bytes)) as zf:
            zf.extractall(tmp_path)
            
        with open(tmp_path / "manifest.json", "w") as f:
            json.dump(manifest.dict(), f)
            
        launcher_code = """
import sys
import json
import importlib

sys.path.append('/mnt')

print("__OBLAK_START__")
try:
    with open('/mnt/manifest.json', 'r') as f:
        manifest = json.load(f)

    mod_name = manifest.get('module', 'cli')
    handler_name = manifest.get('handler', 'handler')

    mod = importlib.import_module(mod_name)
    getattr(mod, handler_name)()
except Exception as e:
    print(f"Execution Error: {type(e).__name__}: {str(e)}")
print("__OBLAK_END__")
"""
        with open(tmp_path / "launcher.py", "w") as f:
            f.write(launcher_code)

        run_script = """#!/bin/sh
mount -t tmpfs -o size=10m tmpfs /tmp

cd /mnt
python3 /mnt/launcher.py > /tmp/out.txt 2>&1

cat /tmp/out.txt
"""
        with open(tmp_path / "run.sh", "w") as f:
            f.write(run_script)
            
        subprocess.run(["dd", "if=/dev/zero", f"of={dest_ext4}", "bs=1M", "count=10"], capture_output=True, check=True)
        subprocess.run(["mkfs.ext4", "-F", "-d", str(tmp_path), str(dest_ext4)], capture_output=True, check=True)

async def execute_in_sandbox(manifest: Manifest, artifact_bytes: bytes) -> dict:
    run_id = str(uuid.uuid4())
    socket_path = f"/tmp/fc_{run_id}.sock"
    payload_path = WORKSPACE / f"payload_{run_id}.ext4"
    
    timeout = manifest.timeout
    memory = manifest.memory
    module = manifest.module
    handler = manifest.handler

    fc_process = None
    output_text = ""
    success = False

    try:
        create_payload_drive(artifact_bytes, manifest, payload_path)

        if os.path.exists(socket_path):
            os.remove(socket_path)
            
        fc_process = await asyncio.create_subprocess_exec(
            str(FIRECRACKER_BIN), "--api-sock", socket_path,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE
        )
        
        await asyncio.sleep(0.1)

        # Boot Source
        api_put(socket_path, '/boot-source', {
            "kernel_image_path": str(KERNEL_PATH),
            "boot_args": "console=ttyS0 reboot=k panic=1 pci=off init=/oblak_init"
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

        # Wait until the process finishes or timeout occurs
        await asyncio.wait_for(fc_process.wait(), timeout=timeout)
        
        stdout, _ = await fc_process.communicate()
        raw_output = stdout.decode('utf-8', errors='ignore')
        
        if "__OBLAK_START__" in raw_output and "__OBLAK_END__" in raw_output:
            output_text = raw_output.split("__OBLAK_START__")[1].split("__OBLAK_END__")[0].strip()
            success = True
        else:
            output_text = "Execution failed or crashed. Raw Output:\n" + raw_output
            success = False

    except asyncio.TimeoutError:
        log.error(f"[{run_id}] Execution timeout!")
        if fc_process:
            fc_process.kill()
        output_text = f"Error: Function timed out after {timeout} seconds."
        success = False
        
    except Exception as e:
        log.error(f"[{run_id}] Sandbox error: {str(e)}")
        output_text = f"Internal sandbox error: {str(e)}"
        success = False

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

    return {"success": success, "output": output_text}