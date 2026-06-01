def handler(data: str):
    words = data.split()
    return {
        "word_count": len(words),
        "character_count": len(data),
    }