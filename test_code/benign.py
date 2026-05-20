def handler(event):
    name = event.get("name", "world")
    total = sum(range(10))
    return {"message": f"Hello, {name}!", "sum": total}
