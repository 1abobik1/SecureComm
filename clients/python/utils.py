# Определяет категорию файла по MIME-типу
def get_file_category(mime_type):
    photo_mimes = ['image/jpeg', 'image/png', 'image/gif', 'image/bmp', 'image/webp']
    video_mimes = ['video/mp4', 'video/mpeg', 'video/avi', 'video/mov', 'video/webm']
    text_mimes = [
        'text/plain', 'text/csv', 'application/pdf', 'application/msword',
        'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
        'text/x-python', 'text/javascript', 'text/html'
    ]
    if mime_type in photo_mimes:
        return 'photo'
    elif mime_type in video_mimes:
        return 'video'
    elif mime_type in text_mimes:
        return 'text'
    return 'unknown'

# Возвращает расширение файла по MIME-типу
def get_file_extension(mime_type):
    mime_to_ext = {
        'image/jpeg': '.jpg',
        'image/png': '.png',
        'image/gif': '.gif',
        'video/mp4': '.mp4',
        'video/avi': '.avi',
        'video/mkv': '.mkv',
        'application/pdf': '.pdf',
        'application/vnd.openxmlformats-officedocument.wordprocessingml.document': '.docx',
        'text/plain': '.txt'
    }
    return mime_to_ext.get(mime_type, '.unknown')