def handler(event):
    total = sum(range(10))
    return {"message": "Hello, world!", "sum": total}