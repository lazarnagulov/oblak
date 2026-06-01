
import io
from pathlib import Path
import tomllib
from typing import Optional
import zipfile

import rich


def get_or_create_manifest(
    path: Path,
    name: Optional[str],
    manifest_path: Optional[Path],
    runtime: str,
    handler: str,
    timeout: int,
    memory: int
) -> dict:
    if manifest_path and manifest_path.exists():
        try:
            with open(manifest_path, "rb") as f:
                data = tomllib.load(f)
            rich.print(f"Loaded explicit manifest: {manifest_path.name}")
            return data
        except Exception as e:
            raise ValueError(f"Error parsing manifest: {e}")

    func_name = name or path.name
    rich.print("Using generated manifest based on command options.")
    return {
        "name": func_name,
        "runtime": runtime,
        "module": path.stem if path.is_file() else "main",
        "handler": handler,
        "timeout": timeout,
        "memory": memory,
    }
    
def build_zip_artifact(
    path: Path,
    manifest_data: dict,
    requirements_path: Optional[Path] = None
) -> io.BytesIO:
    zip_buffer = io.BytesIO()
    
    with zipfile.ZipFile(zip_buffer, "w", zipfile.ZIP_DEFLATED) as zip_file:
        if path.is_file():
            zip_file.write(path, path.name)
        elif path.is_dir():
            for file_path in path.rglob("*"):
                if file_path.is_file() and ".git" not in file_path.parts and "__pycache__" not in file_path.parts:
                    arcname = file_path.relative_to(path)
                    zip_file.write(file_path, arcname)

        req_path = requirements_path if requirements_path else (path / "requirements.txt" if path.is_dir() else None)
        if req_path and req_path.exists():
            zip_file.write(req_path, "requirements.txt")
            rich.print("Included requirements.txt")

        toml_content = f"""
name = "{manifest_data['name']}"
runtime = "{manifest_data['runtime']}"
module = "{manifest_data['module']}"
handler = "{manifest_data['handler']}"
timeout = {manifest_data['timeout']}
memory = {manifest_data['memory']}
"""
        zip_file.writestr("oblak.toml", toml_content.strip())
    
    zip_buffer.seek(0)
    return zip_buffer