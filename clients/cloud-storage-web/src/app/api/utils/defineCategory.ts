export function determineFileCategory(mimeType: string): "photo" | "video" | "text" | "unknown" {
    if (!mimeType) return 'unknown';

    const normalizedMime = mimeType.toLowerCase();

    // Фото и изображения
    const photoMimes = [
        'image/jpeg',
        'image/png',
        'image/gif',
        'image/webp',
        'image/svg+xml',
        'image/bmp',
        'image/tiff',
        'image/x-icon'
    ];

    // Видео файлы
    const videoMimes = [
        'video/mp4',
        'video/webm',
        'video/ogg',
        'video/quicktime',
        'video/x-msvideo',
        'video/x-ms-wmv',
        'video/mpeg',
        'video/3gpp',
        'video/3gpp2'
    ];

    // Текстовые файлы
    const textMimes = [
        'text/plain',
        'text/csv',
        'text/html',
        'text/css',
        'text/javascript',
        'application/json',
        'application/xml',
        'application/pdf',
        'application/msword',
        'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
        'application/vnd.ms-excel',
        'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
        'application/vnd.ms-powerpoint',
        'application/vnd.openxmlformats-officedocument.presentationml.presentation',
        'application/rtf'
    ];

    if (photoMimes.includes(normalizedMime)) {
        return 'photo';
    }

    if (videoMimes.includes(normalizedMime)) {
        return 'video';
    }

    if (textMimes.includes(normalizedMime)) {
        return 'text';
    }

    return 'unknown';
}
