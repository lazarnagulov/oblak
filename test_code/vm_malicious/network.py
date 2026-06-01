def handler():
    import urllib.request
    print("Attempting to access google.com...")
    try:
        urllib.request.urlopen('http://google.com', timeout=2)
        print("Danger: We have internet access.")
    except Exception as e:
        print(f"Safe: No internet access. Error: {e}")