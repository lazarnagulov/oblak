def handler():
    print("Attempting to access /etc/shadow...")
    try:
        print(open('/etc/shadow').read())
    except Exception as e:
        print(f"Failed to hack /etc/shadow: {e}")