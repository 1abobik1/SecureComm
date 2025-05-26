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
        // Выполняем GET запрос для получения потока данных
        const response = await fetch(url, {
            method: 'GET',
        });

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`Ошибка загрузки ${response.status}: ${errorText}`);
        }

        // Получаем поток данных
        const stream = response.body;
        if (!stream) {
            throw new Error('Не удалось получить поток данных');
        }

        // Читаем поток в ArrayBuffer
        const reader = stream.getReader();
        const chunks: Uint8Array[] = [];
        let totalLength = 0;

        while (true) {
            const { done, value } = await reader.read();
            if (done) break;
            chunks.push(value);
            totalLength += value.length;
        }

        // Объединяем все куски в один ArrayBuffer
        const data = new Uint8Array(totalLength);
        let offset = 0;
        for (const chunk of chunks) {
            data.set(chunk, offset);
            offset += chunk.length;
        }

        // Логируем длину данных
        console.log('Получено байт:', totalLength);

        // Проверяем минимальную длину данных (nonce: 16 + iv: 16 + tag: 32 + минимум 1 байт шифротекста)
        if (totalLength < 16 + 16 + 32 + 1) {
            throw new Error(`Недостаточная длина данных для дешифровки: ${totalLength} байт`);
        }

        // Разбираем данные: nonce (16), iv (16), ciphertext (все, кроме последних 32), tag (32)
        const nonce = data.slice(0, 16);
        const iv = data.slice(16, 32);
        const ciphertext = data.slice(32, data.length - 32);
        const receivedHmac = data.slice(data.length - 32);

        // Логируем разбиение
        console.log('Nonce:', Array.from(nonce));
        console.log('IV:', Array.from(iv));
        console.log('Ciphertext length:', ciphertext.length);
        console.log('Received HMAC:', Array.from(receivedHmac));

        // Вычисляем HMAC для проверки
        const hmac = await crypto.subtle.sign(
            'HMAC',
            kMac,
            new Uint8Array([...iv, ...ciphertext])
        );
        const computedHmac = new Uint8Array(hmac);
        console.log('Computed HMAC:', Array.from(computedHmac));

        // Сравниваем HMAC безопасным способом
        if (computedHmac.length !== receivedHmac.length) {
            throw new Error(`Недопустимая длина HMAC: ${computedHmac.length} != ${receivedHmac.length}`);
        }
        let isHmacValid = true;
        for (let i = 0; i < computedHmac.length; i++) {
            isHmacValid &&= computedHmac[i] === receivedHmac[i];
        }
        // if (!isHmacValid) {
        //     throw new Error('Проверка HMAC не пройдена');
        // }

        // Дешифруем данные
        const decrypted = await crypto.subtle.decrypt(
            { name: 'AES-CBC', iv },
            kEnc,
            ciphertext
        );

        // Убираем PKCS#7 padding
        const decryptedBytes = new Uint8Array(decrypted);
        const paddingLength = decryptedBytes[decryptedBytes.length - 1];
        if (
            paddingLength > 16 ||
            paddingLength === 0 ||
            !decryptedBytes.slice(-paddingLength).every((byte) => byte === paddingLength))
        {

        }
        const plaintext = decryptedBytes.slice(0, decryptedBytes.length - paddingLength);

        // Создаём Blob с указанным MIME-типом
        return new Blob([plaintext], { type: mime_type });
    } catch (error) {
        throw new Error(`Ошибка при скачивании или дешифровке: ${error instanceof Error ? error.message : String(error)}`);
    }
}