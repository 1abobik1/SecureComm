interface UploadEncryptedFileOptions {
    file: File;
    cloudURL: string;
    category: string;
    kEnc: CryptoKey; // AES-CBC ключ
    kMac: CryptoKey; // HMAC ключ
    chunkSize?: number;
    onProgress?: (progress: number) => void;
}

export async function streamingUploadEncryptedFile({
                                                       file,
                                                       cloudURL = 'http://localhost:8080/files/one/encrypted',
                                                       category = 'unknown',
                                                       kEnc,
                                                       kMac,
                                                       chunkSize = 104857600, // 100MB по умолчанию
                                                       onProgress}: UploadEncryptedFileOptions): Promise<void> {
    // Генерируем IV (16 байт для AES-CBC)
    const iv = crypto.getRandomValues(new Uint8Array(16));

    // Создаем TransformStream для шифрования и вычисления HMAC
    let hmacInitialized = false;
    let bytesProcessed = 0;

    const encryptAndHMACStream = new TransformStream({
        async transform(chunk, controller) {
            if (!hmacInitialized) {
                // Первый чанк - добавляем IV и инициализируем HMAC
                controller.enqueue(iv);
                await crypto.subtle.sign(
                    { name: 'HMAC' },
                    kMac,
                    iv
                );
                hmacInitialized = true;
            }

            // Шифруем данные
            const encrypted = await crypto.subtle.encrypt(
                {
                    name: 'AES-CBC',
                    iv: iv
                },
                kEnc,
                chunk
            );

            // Обновляем HMAC
            await crypto.subtle.sign(
                { name: 'HMAC' },
                kMac,
                encrypted
            );

            bytesProcessed += chunk.byteLength;
            if (onProgress) {
                onProgress(bytesProcessed / file.size);
            }

            controller.enqueue(new Uint8Array(encrypted));
        },

        async flush(controller) {
            // Добавляем финальный HMAC
            const hmac = await crypto.subtle.sign(
                { name: 'HMAC' },
                kMac,
                new ArrayBuffer(0) // Финализируем HMAC
            );
            controller.enqueue(new Uint8Array(hmac));
        }
    });

    // Создаем поток для чтения файла
    const fileStream = file.stream();

    // Конвейер потоков
    const encryptedStream = fileStream
        .pipeThrough(new TransformStream({
            transform(chunk, controller) {
                controller.enqueue(new Uint8Array(chunk));
            }
        }))
        .pipeThrough(encryptAndHMACStream);

    // Отправляем запрос
    const response = await fetch(cloudURL, {
        method: 'POST',
        headers: {
            'X-File-Category': category,
            'X-Orig-Filename': file.name,
            'X-Orig-Mime': file.type,
            'Content-Type': 'application/octet-stream',
            'Content-Length': file.size.toString()
        },
        body: encryptedStream
    });

    if (!response.ok) {
        const errorBody = await response.text();
        throw new Error(`Upload failed: ${response.status} - ${errorBody}`);
    }
}