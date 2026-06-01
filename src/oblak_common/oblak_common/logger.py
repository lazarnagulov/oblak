import logging
import logging.handlers
import os, sys

class SanitizeFilter(logging.Filter):
    def filter(self, record: logging.LogRecord) -> bool:
        record.msg = str(record.msg).replace("\n", "\\n").replace("\r", "\\r")
        if record.args:
            if isinstance(record.args, dict):
                record.args = {
                    k: str(v).replace("\n", "\\n").replace("\r", "\\r")
                    for k, v in record.args.items()
                }
            else:
                record.args = tuple(
                    str(a).replace("\n", "\\n").replace("\r", "\\r")
                    for a in record.args
                )
        return True
    
def _make_file_handler(log_dir: str, name: str) -> logging.Handler:
    os.makedirs(log_dir, exist_ok=True)
    handler = logging.handlers.RotatingFileHandler(
        os.path.join(log_dir, f"{name}.log"),
        maxBytes=10 * 1024 * 1024,  # 10MB
        backupCount=5,
        encoding="utf-8",
    )
    return handler

def setup_logger(name: str) -> logging.Logger:
    log = logging.getLogger(name)
    LOG_LEVEL = os.environ.get("LOG_LEVEL", "INFO").upper()
    log.setLevel(getattr(logging, LOG_LEVEL, logging.INFO))

    formatter = logging.Formatter(
        "%(asctime)s [%(levelname)s] [PID:%(process)d] %(message)s",
        datefmt="%Y-%m-%dT%H:%M:%S"
    )

    console_handler = logging.StreamHandler()
    console_handler.setFormatter(formatter)
    log.addHandler(console_handler)

    if sys.platform == "linux":
        log_dir = f"/var/log/oblak/{name}"
    else:
        log_dir = os.path.join(os.path.dirname(__file__), "..", "..", "logs", name)

    file_handler = _make_file_handler(log_dir, name)
    file_handler.setFormatter(formatter)
    log.addHandler(file_handler)

    log.addFilter(SanitizeFilter())

    return log