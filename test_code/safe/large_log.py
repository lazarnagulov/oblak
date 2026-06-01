def handler():
    for i in range(1000):
        print(f"Log line {i}")
    longLog = "A" * 1000
    print(longLog)
    return "Done"