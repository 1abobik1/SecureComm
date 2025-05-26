/**
 * Потоково загружает зашифрованный файл на сервер, используя AES-CBC и HMAC-SHA256
 *
 * @param file - Файл для загрузки (из input или Drag'n'Drop)
 * @param cloudUrl - URL эндпоинта для загрузки
 * @param category - Категория файла (photo, video, text, unknown)
 * @param kEnc - Ключ шифрования AES (32 байта)
 * @param kMac - Ключ HMAC-SHA256 (32 байта)
 * @returns Ответ сервера с метаданными
 * @throws {Error} Если загрузка не удалась
 */
export async function streamUploadEncryptedFile(
    file: File,
    cloudUrl: string,
    category: string,
    kEnc: CryptoKey,
    kMac: CryptoKey
): Promise<any> {

    // Генерируем IV и nonce
    const iv = crypto.getRandomValues(new Uint8Array(16));
    const nonce = crypto.getRandomValues(new Uint8Array(16));

    const fileData = await file.arrayBuffer();
    const encrypted = await crypto.subtle.encrypt(
        { name: 'AES-CBC', iv },
        kEnc,
        fileData
    );
    const hmac = await crypto.subtle.sign(
        'HMAC',
        kMac,
        encrypted
    );
    const encodedFilename = btoa(encodeURIComponent(file.name));
    console.log(encodedFilename)

    const token = localStorage.getItem('token');
    // Подготавливаем заголовки
    const headers = new Headers({
        'Authorization': `Bearer ${token}`,
        "X-File-Category": category,
        "X-Orig-Filename": encodedFilename,
        "X-Orig-Mime": file.type,
        "Content-Type": "application/octet-stream"
    });


    const combinedData = new Blob([
        nonce,
        iv,
        new Uint8Array(encrypted),
        new Uint8Array(hmac)
    ]);

    try {
        const response = await fetch(cloudUrl, {
            method: 'POST',
            headers,
            body: combinedData,
        });

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`Ошибка загрузки ${response.status}: ${errorText}`);
        }

        return await response.json();
    } catch (error) {
        throw new Error(`Ошибка при загрузке: ${error instanceof Error ? error.message : String(error)}`);
    }
}