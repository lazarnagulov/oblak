import hashlib

def handler(event):
    data = event.get("data", "").encode()
    return {"hash": hashlib.md5(data).hexdigest()}