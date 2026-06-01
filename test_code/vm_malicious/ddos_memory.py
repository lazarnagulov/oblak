def handler():
    print("Attempting to allocate memory...")
    try:
        memory = bytearray(1024 * 1024 * 1024) 
        print("Danger: Allocated 1GB.")
    except MemoryError:
        print("Safe: Blocked by memory limits.")
    except Exception as e:
        print(f"Safe: Crash due to memory limits: {e}")