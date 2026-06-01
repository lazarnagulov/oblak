def handler():
    print("Attempting to generate large output...")
    try:
        for i in range(1000000):
            print(f"Log line {i}")
        print("Danger: Generated too many logs.")
        return "Generated 1 million log lines."
    except Exception as e:
        print(f"Safe: Blocked by log limits. Error: {e}")
        return f"Blocked by log limits"