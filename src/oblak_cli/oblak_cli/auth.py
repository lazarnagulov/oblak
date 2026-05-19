import keyring
import os
import json
from pathlib import Path
from typing import Optional

CONFIG_DIR = Path.home() / ".oblak"
CONFIG_FILE = CONFIG_DIR / "config.json"

DEFAULT_SERVER_URL = "http://localhost:8080/api/v1"
SERVICE_NAME = "oblak_cli"

def _load_config_file() -> dict:
    if not CONFIG_FILE.exists():
        return {}
    try:
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
    keyring.set_password(SERVICE_NAME, username, token)
    keyring.set_password(SERVICE_NAME, "current_active_user", username)

def get_token() -> Optional[str]:
    username = keyring.get_password(SERVICE_NAME, "current_active_user")
    if not username:
        return None
    return keyring.get_password(SERVICE_NAME, username)

def get_username() -> Optional[str]:
    return keyring.get_password(SERVICE_NAME, "current_active_user")

def delete_auth_data():
    username = keyring.get_password(SERVICE_NAME, "current_active_user")
    if username:
        try:
            keyring.delete_password(SERVICE_NAME, username)
            keyring.delete_password(SERVICE_NAME, "current_active_user")
        except keyring.errors.PasswordDeleteError:
            pass

def get_server_url() -> str:
    return _load_config_file().get("server_url", DEFAULT_SERVER_URL)

def set_server_url(url: str):
    config = _load_config_file()
    config["server_url"] = url
    _save_config_file(config)