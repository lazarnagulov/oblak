import os
import json
import requests
from pathlib import Path
from typing import Optional

CONFIG_DIR = Path.home() / ".oblak"
CONFIG_FILE = CONFIG_DIR / "config.json"

DEFAULT_SERVER_URL = "http://localhost:8080/api/v1"

def _load_config_file() -> dict:
    if not CONFIG_FILE.exists():
        return {}
    try:
        print(CONFIG_FILE)
        with open(CONFIG_FILE, "r") as f:
            return json.load(f)
    except Exception:
        return {}

def _save_config_file(data: dict):
    CONFIG_DIR.mkdir(exist_ok=True)
    with open(CONFIG_FILE, "w") as f:
        json.dump(data, f, indent=4)
    os.chmod(CONFIG_FILE, 0o600)

def save_token(token: str, username: str):
    config = _load_config_file()
    config["username"] = username
    config["token"] = token
    _save_config_file(config)

def get_token() -> Optional[str]:
    return _load_config_file().get("token")

def get_username() -> Optional[str]:
    return _load_config_file().get("username")

def delete_auth_data():
    config = _load_config_file()
    config.pop("token", None)
    config.pop("username", None)
    _save_config_file(config)

def get_server_url() -> str:
    return _load_config_file().get("server_url", DEFAULT_SERVER_URL)

def set_server_url(url: str):
    config = _load_config_file()
    config["server_url"] = url
    _save_config_file(config)