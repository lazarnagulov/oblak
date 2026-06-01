def handler(event):
    total = sum(range(10))
    print(f"Calculated total: {total}")
    return {"message": "Hello, world!", "sum": total}