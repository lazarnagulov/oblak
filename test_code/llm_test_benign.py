def handler(event):
    numbers = event.get("numbers", [])
    total = sum(numbers)
    average = total / len(numbers) if numbers else 0
    return {"total": total, "average": average}