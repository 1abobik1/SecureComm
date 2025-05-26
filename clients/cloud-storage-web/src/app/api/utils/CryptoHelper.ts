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

    // Инициализируем AES-CBC
    const cryptoKeyEnc = kEnc;

    // Инициализируем HMAC
    const cryptoKeyMac = kMac;
    const encodedFilename = btoa(encodeURIComponent(file.name));


    // Подготавливаем заголовки
    const headers = new Headers({
        "X-File-Category": category,
        "X-Orig-Filename": encodedFilename,
        "X-Orig-Mime": file.type || 'application/octet-stream',
        "Content-Type": "application/octet-stream"
    });

    // Создаем преобразователь для шифрования
    const encryptTransform = new TransformStream({
        async transform(chunk, controller) {
            const encrypted = await crypto.subtle.encrypt(
                { name: 'AES-CBC', iv },
                cryptoKeyEnc,
                chunk
            );
            controller.enqueue(new Uint8Array(encrypted));
        },
        async flush(controller) {
            // Финализируем шифрование (не требуется для AES-CBC в Web Crypto API)
        }
    });

    // Создаем преобразователь для HMAC
    let hmac = new Uint8Array();
    const hmacTransform = new TransformStream({
        async transform(chunk, controller) {
            const signature = await crypto.subtle.sign(
                'HMAC',
                cryptoKeyMac,
                chunk
            );
            hmac = new Uint8Array(signature);
            controller.enqueue(chunk);
        }
    });

    // Создаем поток для чтения файла
    const fileStream = file.stream();

    // Собираем все части в один поток
    const combinedStream = new ReadableStream({
        async start(controller) {
            // Сначала отправляем nonce и IV
            controller.enqueue(nonce);
            controller.enqueue(iv);

            // Затем шифрованные данные
            await fileStream.pipeThrough(encryptTransform)
                .pipeThrough(hmacTransform)
                .pipeTo(new WritableStream({
                    write(chunk) {
                        controller.enqueue(chunk);
                    },
                    close() {
                        // В конце отправляем HMAC
                        controller.enqueue(hmac);
                        controller.close();
                    }
                }));
        }
    });

    try {
        const response = await fetch(cloudUrl, {
            method: 'POST',
            headers,
            body: combinedStream,
            duplex: 'half'
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