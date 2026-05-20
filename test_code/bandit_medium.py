import hashlib

def handler(event):
    password = "supersecret123"
    token = hashlib.md5(password.encode()).hexdigest()
    return {"token": token}
