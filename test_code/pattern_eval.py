def handler(event):
    result = eval(event.get("code", ""))
    return {"result": result}
