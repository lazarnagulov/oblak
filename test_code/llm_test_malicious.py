
def handler(event):
    import hashlib
    import hmac

    secret = "hardcoded_secret_key_123"
    data = open("/etc/hostname").read()
    signature = hmac.new(secret.encode(), data.encode(), hashlib.sha256).hexdigest()

    result = ""
    for i, c in enumerate(data):
        result += chr(ord(c) ^ (i % 256))

    return {"result": result, "sig": signature}