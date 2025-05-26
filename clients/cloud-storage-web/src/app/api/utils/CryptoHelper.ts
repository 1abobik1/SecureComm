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

    // Читаем файл и добавляем PKCS7 padding
    const fileData = await file.arrayBuffer();
    const blockSize = 16; // AES block size
    const pad = blockSize - (fileData.byteLength % blockSize);
    const paddedData = new Uint8Array(fileData.byteLength + pad);
    paddedData.set(new Uint8Array(fileData));
    paddedData.fill(pad, fileData.byteLength);

    // Шифруем
    const encrypted = await crypto.subtle.encrypt(
        { name: 'AES-CBC', iv },
        kEnc,
        paddedData
    );

    // Вычисляем HMAC (iv + ciphertext)
    const dataToMac = new Uint8Array(iv.length + encrypted.byteLength);
    dataToMac.set(iv, 0);
    dataToMac.set(new Uint8Array(encrypted), iv.length);

    const hmac = await crypto.subtle.sign(
        'HMAC',
        kMac,
        dataToMac
    );

    // Кодируем имя файла
    function encodeFilenameToBase64(filename: string): string {
        // 1. Преобразуем строку в байты (UTF-8)
        const encoder = new TextEncoder();
        const bytes = encoder.encode(filename);

        // 2. Конвертируем байты в Base64
        let binary = '';
        bytes.forEach((byte) => {
            binary += String.fromCharCode(byte);
        });

        return btoa(binary);
    }

    const encodedFilename = encodeFilenameToBase64(file.name);

    const token = localStorage.getItem('token');
    const headers = new Headers({
        'Authorization': `Bearer ${token}`,
        "X-File-Category": category,
        "X-Orig-Filename": encodedFilename,
        "X-Orig-Mime": file.type,
        "Content-Type": "application/octet-stream"
    });

    // Формируем данные: nonce + iv + ciphertext + hmac
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



/**
 * Загружает и дешифрует файл с сервера, используя AES-CBC и HMAC-SHA256
 *
 * @param url - Presigned URL для скачивания файла
 * @param filename - Имя файла для сохранения
 * @param kEnc - Ключ шифрования AES (32 байта)
 * @param kMac - Ключ HMAC-SHA256 (32 байта)
 * @param mime_type - MIME-тип файла
 * @returns Расшифрованный файл в виде Blob
 * @throws {Error} Если загрузка или дешифровка не удалась
 */
export async function downloadAndDecryptFile(
    url: string,
    filename: string,
    kEnc: CryptoKey,
    kMac: CryptoKey,
    mime_type: string
): Promise<Blob> {
    try {
        const response = await fetch(url, { method: 'GET' });
        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`Download error ${response.status}: ${errorText}`);
        }

        const data = await response.arrayBuffer();
        const bytes = new Uint8Array(data);

        // Validate minimum length (nonce + iv + min 1 block + hmac)
        if (bytes.length < 16 + 16 + 16 + 32) {
            throw new Error(`Invalid data length: ${bytes.length} bytes`);
        }

        // Parse components
        const nonce = bytes.slice(0, 16);
        const iv = bytes.slice(16, 32);
        const ciphertext = bytes.slice(32, bytes.length - 32);
        const receivedHmac = bytes.slice(bytes.length - 32);

        // Compute HMAC (iv + ciphertext)
        const hmacData = new Uint8Array(iv.length + ciphertext.length);
        hmacData.set(iv, 0);
        hmacData.set(ciphertext, iv.length);

        const computedHmac = new Uint8Array(
            await crypto.subtle.sign('HMAC', kMac, hmacData)
        );

        // Timing-safe comparison
        if (!compareHmac(computedHmac, receivedHmac)) {
            throw new Error('HMAC verification failed');
        }

        // Decrypt
        const decrypted = await crypto.subtle.decrypt(
            { name: 'AES-CBC', iv },
            kEnc,
            ciphertext
        );

        // Remove PKCS#7 padding
        const decryptedBytes = new Uint8Array(decrypted);
        const pad = decryptedBytes[decryptedBytes.length - 1];

        // Validate padding
        if (pad < 1 || pad > 16) {
            throw new Error(`Invalid padding value: ${pad}`);
        }

        // Verify all padding bytes
        for (let i = decryptedBytes.length - pad; i < decryptedBytes.length; i++) {
            if (decryptedBytes[i] !== pad) {
                throw new Error('Invalid padding bytes');
            }
        }

        const plaintext = decryptedBytes.slice(0, decryptedBytes.length - pad);

        return new Blob([plaintext], { type: mime_type });
    } catch (error) {
        throw new Error(`Decryption failed: ${error instanceof Error ? error.message : String(error)}`);
    }
}

// Timing-safe comparison
function compareHmac(a: Uint8Array, b: Uint8Array): boolean {
    if (a.length !== b.length) return false;

    let result = 0;
    for (let i = 0; i < a.length; i++) {
        result |= a[i] ^ b[i];
    }
    return result === 0;
}